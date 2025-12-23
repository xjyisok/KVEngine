package bitcaskgo

import (
	"bitcask-go/data"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
)

const (
	mergeDirName     = "-merge"
	mergeFinishedKey = "merge.finished"
)

// 创建merge文件删除无效数据同属生成hintfile
func (db *DB) Merge() error {
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
	db.isMerging = true
	defer func() {
		db.isMerging = false
	}()
	//持久化当前活跃文件
	if err := db.activeDataFile.Sync(); err != nil {
		db.mu.Unlock()
		return err
	}
	nonMergeFileId := db.activeDataFile.Fid + 1
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
	mergePath := db.getMergePath()
	if _, err := os.Stat(mergePath); err == nil {
		if err := os.RemoveAll(mergePath); err != nil {
			return err
		}
	}
	//新建一个merge目录
	if err := os.MkdirAll(mergePath, os.ModePerm); err != nil {
		return err
	}
	//打开一个新的bitcask实例用于merge操作
	mergeOpts := db.options
	mergeOpts.DirPath = mergePath
	mergeOpts.SyncWrites = false
	mergeDB, err := Open(mergeOpts)
	if err != nil {
		return err
	}
	//打开hint文件存储索引
	hintFile, err := data.OpenHintFile(mergePath)
	if err != nil {
		return err
	}
	//遍历所有需要merge的文件
	for _, file := range mergeFileIds {
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
				_, err := mergeDB.appnedLogRecord(logRecord)
				if err != nil {
					return err
				}
				//将当前位置索引写入到Hint文件中
				if err := hintFile.WriteHintRecord(realKey, logRecordPos); err != nil {
					return err
				}
			}
			offset += size
		}
	}
	//sync 保证持久化
	if err := mergeDB.Sync(); err != nil {
		return err
	}
	if err := hintFile.Sync(); err != nil {
		return err
	}
	//写标识完成的文件
	// 写标识 merge 完成的文件
	mergeFinFile, err := data.OpenMergeFinishedFile(mergeDB.options.DirPath)
	if err != nil {
		return err
	}
	mergeFinRecord := &data.LogRecord{
		Key:   []byte(mergeFinishedKey),
		Value: []byte(strconv.Itoa(int(nonMergeFileId))),
	}
	encRecord, _ := data.EncodeLogRecord(mergeFinRecord)
	if err := mergeFinFile.Write(encRecord); err != nil {
		return err
	}
	if err := mergeFinFile.Sync(); err != nil {
		return err
	}
	return nil
}
func (db *DB) getMergePath() string {
	dir := path.Dir(path.Clean(db.options.DirPath))
	base := path.Base(db.options.DirPath)
	return filepath.Join(dir, base) + mergeDirName
}
//数据库启动时自动加载merge数据目录
func (db *DB) loadMergeFiles() error {
	mergePath := db.getMergePath()
	if _, err := os.Stat(mergePath); os.IsNotExist(err) {
		return nil
	}
	defer func() {
		_ = os.RemoveAll(mergePath)
	}()

	dirEntries, err := os.ReadDir(mergePath)
	if err != nil {
		return err
	}

	//	遍历查找是否完成了 merge
	var mergeFinished bool
	var fileNames []string
	for _, entry := range dirEntries {
		if entry.Name() == data.MergeFinishedFileName {
			mergeFinished = true
		}
		fileNames = append(fileNames, entry.Name())
	}
	if !mergeFinished {
		return nil
	}

	nonMergeFileId, err := db.getNonMergeFileId(mergePath)
	if err != nil {
		return err
	}
	//	先删除旧的数据文件
	var fileId uint32 = 0
	for ; fileId < nonMergeFileId; fileId++ {
		fileName := data.GetDataFileName(db.options.DirPath, fileId)
		if _, err := os.Stat(fileName); err == nil {
			if err := os.Remove(fileName); err != nil {
				return err
			}
		}
	}

	// 移动文件
	for _, fileName := range fileNames {
		srcPath := filepath.Join(mergePath, fileName)
		destPath := filepath.Join(db.options.DirPath, fileName)
		if err := os.Rename(srcPath, destPath); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) loadIndexFromHintFile() error {
	hintFileName := filepath.Join(db.options.DirPath, data.HintFileName)
	if _, err := os.Stat(hintFileName); os.IsNotExist(err) {
		return nil
	}

	hintFile, err := data.OpenHintFile(db.options.DirPath)
	if err != nil {
		return err
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
		db.indexer.Put(logRecord.Key, logReordPos)
		offset += size
	}
	return nil
}

func (db *DB) getNonMergeFileId(dirPath string) (uint32, error) {
	mergeFinishedFile, err := data.OpenMergeFinishedFile(dirPath)
	if err != nil {
		return 0, err
	}
	record, _, err := mergeFinishedFile.ReadLogRecord(0)
	if err != nil {
		return 0, err
	}
	nonMergeFileId, err := strconv.Atoi(string(record.Value))
	if err != nil {
		return 0, err
	}
	return uint32(nonMergeFileId), nil
}