package bitcaskgo

import (
	"bitcask-go/data"
	"bitcask-go/fio"
	"bitcask-go/index"
	"bitcask-go/utils"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	mergeSyncDirName     = "-mergeSync"
	mergeSyncFinishedKey = "mergeSync.finished"
	mergeTmpFolder       = "mergefolder"
)

func (db *DB) Merge_Sync() error {
	//merge只对oldfile进行merge如果不存在activefile则直接返回
	if db.activeDataFile == nil {
		return nil
	}
	db.mu.Lock()
	//标记正在进行merge操作
	if db.isMerging {
		db.mu.Unlock()
		return ErrMergeIsRunning
	}
	totalSize, err := utils.DirSize(db.options.DirPath)
	if err != nil {
		return err
	}
	useLessRatio := db.reclaimableSize / totalSize
	if db.options.DataFileMergeRatio > float32(useLessRatio) {
		db.mu.Unlock()
		return ErrMergeTime
	} // 查看剩余的空间容量是否可以容纳 Merge 后的数据
	availableDiskSize, err := utils.AvailableDiskSize()
	if err != nil {
		db.mu.Unlock()
		return err
	}
	if uint64(totalSize-db.reclaimableSize) >= availableDiskSize {
		db.mu.Unlock()
		return ErrNoEnoughDiskForMerge
	}
	db.isMerging = true
	defer func() {
		db.isMerging = false
	}()
	//持久化当前活跃文件
	if err := db.activeDataFile.Sync(); err != nil {
		db.mu.Unlock()
		return err
	}
	//将当前活跃文件转化为旧的
	db.olderDataFiles[db.activeDataFile.Fid] = db.activeDataFile
	//创建新的活跃文件
	if err := db.setActiveDataFile(); err != nil {
		db.mu.Unlock()
		return err
	}
	//取出所有需要merge的文件
	var mergeFileIds []*data.DataFile
	for _, file := range db.olderDataFiles {
		mergeFileIds = append(mergeFileIds, file)
	}
	db.mu.Unlock()
	//将merge的文件按照id进行排序
	sort.Slice(mergeFileIds, func(i, j int) bool {
		return mergeFileIds[i].Fid < mergeFileIds[j].Fid
	})
	mergeSyncPath := db.getMergeSyncPath()
	if _, err := os.Stat(mergeSyncPath); err == nil {
		if err := os.RemoveAll(mergeSyncPath); err != nil {
			return err
		}
	}
	//新建一个merge目录
	if err := os.MkdirAll(mergeSyncPath, os.ModePerm); err != nil {
		return err
	}
	//打开一个新的bitcask实例用于merge操作
	mergeOpts := db.options
	mergeOpts.DirPath = mergeSyncPath
	mergeOpts.SyncWrites = false
	mergeDB, err := Open(mergeOpts)
	if err != nil {
		return err
	}
	defer func() {
		_ = mergeDB.Close()
	}()
	//遍历所有需要merge的文件
	mergeSyncFinFile, err := data.OpenMergeSyncFinishedFile(mergeDB.options.DirPath)
	if err != nil {
		return err
	}
	for _, file := range mergeFileIds {
		hintFileSync, err := data.OpenHintSyncFile(mergeSyncPath, file.Fid)
		if err != nil {
			return err
		}
		var offset int64 = 0
		for {
			logRecord, size, err := file.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			realKey, _ := parseLogRecordKey(logRecord.Key)
			//查询当前key在原db中的位置
			logRecordPos, _ := db.indexer.Get(realKey)
			if logRecordPos != nil && logRecordPos.Fid == file.Fid && logRecordPos.Offset == offset {
				//清除事务标记，能持久化到磁盘中的数据肯定已经是一个完整的事务
				logRecord.Key = logRecordKeyWithSeq(nonTransaction, realKey)
				//将符合条件的数据写入到mergeDB中
				hintpos, err := mergeDB.appnedLogRecord(logRecord)
				if err != nil {
					return err
				}
				hinRecordPos := &data.LogRecordPos{
					Fid:    mergeDB.activeDataFile.Fid,
					Offset: hintpos.Offset,
					Size:   hintpos.Size,
				}
				//将当前位置索引写入到Hint文件中
				//fmt.Printf("HintFileOOOOOOOOOOFFFFFFFFset:%d Size:%d\n", offsetHint, size)
				if err := hintFileSync.WriteHintRecord(realKey, hinRecordPos); err != nil {
					return err
				}
			}
			offset += size
			//count++
		}
		//写标识完成的文件
		// 写标识 merge 完成的文件
		//每merge完一个文件就保存一个hintfile同时在mergeSyncFinFile中记录已经merge完成的文件
		mergeSyncFinRecord := &data.LogRecord{
			Key:   []byte(mergeSyncFinishedKey),
			Value: []byte(strconv.Itoa(int(file.Fid + 1))),
		}
		encRecord, _ := data.EncodeLogRecord(mergeSyncFinRecord)
		if err := mergeSyncFinFile.Write(encRecord); err != nil {
			return err
		}
	}
	if err := mergeSyncFinFile.Sync(); err != nil {
		return err
	}
	return nil
}
func (db *DB) loadMergeSyncFiles() error {
	mergeSyncPath := db.getMergeSyncPath()
	fmt.Printf("mergeSyncPath:%s", mergeSyncPath)
	if _, err := os.Stat(mergeSyncPath); os.IsNotExist(err) {
		return nil
	}
	defer func() {
		_ = os.RemoveAll(mergeSyncPath)
	}()
	tmpDir := filepath.Join(db.options.DirPath, mergeTmpFolder)
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return err
	}
	dirEntries, err := os.ReadDir(mergeSyncPath)
	if err != nil {
		return err
	}
	nonMergedSyncFileId, err := db.getNonMergeSyncFileId(mergeSyncPath)
	if err != nil {
		return err
	}
	if nonMergedSyncFileId == uint32(0) {
		return nil
	}
	var fileNames []string
	//遍历所有已经完成merge的文件名字
	for _, entry := range dirEntries {
		if entry.Name() == data.SeqNoFileName {
			continue
		}
		fileNames = append(fileNames, entry.Name())
	}
	//将mergeSync中merge完成的文件以及其hintfile拷贝到数据文件夹中
	//这里暂时将数据文件保存在dirpath/mergefolder中防止文件名冲突
	for _, fileName := range fileNames {
		srcPath := filepath.Join(mergeSyncPath, fileName)
		desPath := filepath.Join(db.options.DirPath, mergeTmpFolder, fileName)
		if err := os.Rename(srcPath, desPath); err != nil {
			return err
		}
	}
	if err := db.switchIndexAfterMerge(nonMergedSyncFileId); err != nil {
		return err
	}
	return nil
}
func (db *DB) getMergeSyncPath() string {
	dir := path.Dir(path.Clean(db.options.DirPath))
	base := path.Base(db.options.DirPath)
	return filepath.Join(dir, base) + mergeSyncDirName
}
func (db *DB) getNonMergeSyncFileId(dirPath string) (uint32, error) {
	mergeSyncFinishedFile, err := data.OpenMergeSyncFinishedFile(dirPath)
	if err != nil {
		return 0, err
	}

	var (
		offset     int64
		lastRecord *data.LogRecord
	)

	for {
		logRecord, size, err := mergeSyncFinishedFile.ReadLogRecord(offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		lastRecord = logRecord
		offset += size
	}

	// 文件为空，说明没有任何 merge 记录
	if lastRecord == nil {
		return 0, nil
	}

	// Value 中存的是 “第一个未 merge 的 fileId”
	nonMergeFileId, err := strconv.Atoi(string(lastRecord.Value))
	if err != nil {
		return 0, err
	}

	return uint32(nonMergeFileId), nil
}
func (db *DB) switchIndexAfterMerge(nonMergedSyncFileId uint32) error {
	//构建新索引
	newIndexer := index.NewIndexer(db.options.IndexType, db.options.DirPath, db.options.SyncWrites)
	//构建新的数据文件集合
	tmpOlderDataFiles := make(map[uint32]*data.DataFile)
	//设置文件IO类型
	ioType := fio.StandardIO
	if db.options.MMapIsOpen {
		ioType = fio.MMapIO
	}
	updateIndex := func(key []byte, Type data.LogRecordType, logRecordPos *data.LogRecordPos) error {
		var oldPos *data.LogRecordPos
		if Type == data.LogRecordTypeDelete {
			//NOTE这里由于bitcask是追加写入所以删除操作只是添加了一个删除标记，并没有真正删除数据文件中的数据
			//所以在加载索引时需要将该key从内存索引中删除
			//比如用户先执行了put(k1,v1)然后delete(k1)最后再put(k1,v2)
			//那么数据文件中会有三条记录，k1:v1、k1:delete标记、k1:v2
			//在加载索引时会先将k1指向v1的位置，然后再将k1从内存索引中删除，最后再将k1指向v2的位置
			//这样就保证了内存索引中的数据是最新的
			oldPos, _ = newIndexer.Delete(key)
			//对于删除的数据来说删除的数据本身也是需要被删除的
			//db.reclaimableSize += int64(logRecordPos.Size)
			//db.reclaimableSize += int64(oldPos.Size)
		} else if Type == data.LogRecordTypeNormal {
			oldPos, _ = newIndexer.Put(key, logRecordPos)
		}
		if oldPos != nil {
			//db.reclaimableSize += int64(oldPos.Size)
		}
		return nil
	}
	//fmt.Printf("dirPath:%s\n", db.options.DirPath)
	dirEntries, err := os.ReadDir(db.options.DirPath)
	if err != nil {
		return err
	}

	var fileIds []int
	// 遍历目录中的所有文件，找到所有以 .data 结尾的文件
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), data.DataFileNameSuffix) {
			splitNames := strings.Split(entry.Name(), ".")
			fileId, err := strconv.Atoi(splitNames[0])
			// 数据目录有可能被损坏了
			if err != nil {
				return ErrDataDirectoryCorrupted
			}
			fileIds = append(fileIds, fileId)
			fmt.Printf("fileId IN fileIds:%d\n", fileId)
		}
	}

	//	对文件 id 进行排序，从小到大依次加载
	sort.Ints(fileIds)
	// 1.1 从 hint 文件加载 merge 后的 data files
	for _, fid := range fileIds {
		if fid >= int(nonMergedSyncFileId) {
			continue
		}

		hintFile, err := data.OpenHintSyncFile(filepath.Join(db.options.DirPath, mergeTmpFolder), uint32(fid))
		if err != nil {
			hintFile.Close()
			continue // 文件可能不存在，跳过
		}
		mergeDataFile, err := data.OpenDataFile(filepath.Join(db.options.DirPath, mergeTmpFolder), uint32(fid), ioType)
		if err != nil {
			mergeDataFile.Close()
			continue // 文件可能不存在，跳过
		}
		var offset int64 = 0
		for {
			logRecord, size, err := hintFile.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			logReordPos := data.DecodeLogReordPos(logRecord.Value)
			newIndexer.Put(logRecord.Key, logReordPos)
			offset += size
		}
		tmpOlderDataFiles[uint32(fid)] = mergeDataFile
	}

	// 开启写屏障，等待 activeFile 中正在进行的写完成
	db.writePaused.Store(true)
	defer db.writePaused.Store(false)
	for db.inFlightWrite.Load() != 0 {
		runtime.Gosched()
	}
	// 扫描 active data file 补齐 merge 期间写入
	//暂存批量写入的记录
	batchRecords := make(map[uint64][]*data.TransactionRecord)
	for fid := nonMergedSyncFileId; fid <= db.activeDataFile.Fid; fid++ {
		var offset int64 = 0
		df, err := data.OpenDataFile(db.options.DirPath, uint32(fid), ioType)
		if fid != db.activeDataFile.Fid {
			tmpOlderDataFiles[fid] = df
		}
		if err != nil {
			return err
		}
		for {
			logRecord, size, err := df.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			realKey, seqNum := parseLogRecordKey(logRecord.Key)
			logRecordPos := &data.LogRecordPos{Fid: fid, Offset: offset, Size: uint32(size)}
			if seqNum == nonTransaction {
				//非批量写数据索引构建
				updateIndex(realKey, logRecord.Type, logRecordPos)
			} else {
				//批量写数据索引构建
				if logRecord.Type == data.LogRecordTypeBatchEnd {
					//批量写结束标记，处理该批次的所有记录
					for _, transaction := range batchRecords[seqNum] {
						updateIndex(transaction.Record.Key, transaction.Record.Type, transaction.Pos)
					}
					delete(batchRecords, seqNum)
				} else {
					//暂存批量写入的记录
					batchRecords[seqNum] = append(batchRecords[seqNum], &data.TransactionRecord{
						Record: logRecord,
						Pos:    logRecordPos,
					})
				}
			}
			offset += size
		}
	}

	// 原子切换索引
	db.mu.Lock()
	oldIndexer := db.indexer
	db.indexer = newIndexer
	//将dirpath/mergefolder中的文件移动到dirpath中
	//	先删除旧的数据文件
	//fmt.Printf("noMergedFileId:%d", nonMergeFileId)
	for fileId := uint32(0); fileId < nonMergedSyncFileId; fileId++ {
		fileName := data.GetDataFileName(db.options.DirPath, fileId)
		if _, err := os.Stat(fileName); err == nil {
			if err := os.Remove(fileName); err != nil {
				return err
			}
		}
	}
	for fileId := uint32(0); fileId < nonMergedSyncFileId; fileId++ {
		srcPath := filepath.Join(filepath.Join(db.options.DirPath, mergeTmpFolder), fmt.Sprintf("%09d", fileId)+data.DataFileNameSuffix)
		desPath := data.GetDataFileName(db.options.DirPath, fileId)
		if err := os.Rename(srcPath, desPath); err != nil {
			return err
		}
	}
	//原子切换文件IO映射
	db.olderDataFiles = tmpOlderDataFiles
	db.mu.Unlock()

	// 关闭旧索引
	oldIndexer.Close()
	os.RemoveAll(filepath.Join(db.options.DirPath, mergeTmpFolder))

	// 解除写屏障

	return nil
}
