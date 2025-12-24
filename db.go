package bitcaskgo

import (
	"bitcask-go/data"
	"bitcask-go/index"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	seqNoKey     = "seq.no"
	fileLockName = "flock"
)

// 主要实现面向用户的操作接口
type DB struct {
	mu              *sync.RWMutex
	activeDataFile  *data.DataFile            //当前活跃的数据文件
	olderDataFiles  map[uint32]*data.DataFile //旧的数据文件集合
	options         Options
	indexer         index.Indexer //内存索引
	fileIds         []int         // 文件 id，只能在加载索引的时候使用，不能在其他的地方更新和使用
	seqNum          uint64        //全局事务序列号
	isMerging       bool          //是否正在进行merge操作的标记，0表示没有，1表示正在进行
	seqNoFileExists bool          // 存储事务序列号的文件是否存在isSeq
	isInitial       bool          // 是否是第一次初始化此数据目录
}

// Close 关闭数据库
func (db *DB) Close() error {
	if db.activeDataFile == nil {
		return nil
	}
	db.mu.Lock()
	//defer db.mu.Unlock()
	// if err := db.indexer.Close(); err != nil {
	// 	return err
	// }

	// 保存当前事务序列号
	seqNoFile, err := data.OpenSeqNoFile(db.options.DirPath)
	if err != nil {
		return err
	}
	record := &data.LogRecord{
		Key:   []byte(seqNoKey),
		Value: []byte(strconv.FormatUint(db.seqNum, 10)),
	}
	encRecord, _ := data.EncodeLogRecord(record)
	if err := seqNoFile.Write(encRecord); err != nil {
		return err
	}
	if err := seqNoFile.Sync(); err != nil {
		return err
	}
	//	关闭当前活跃文件
	if err := db.activeDataFile.Close(); err != nil {
		return err
	}
	// 关闭旧的数据文件
	for _, file := range db.olderDataFiles {
		if err := file.Close(); err != nil {
			return err
		}
	}
	db.mu.Unlock()
	if err := db.indexer.Close(); err != nil {
		return err
	}
	return nil
}

// Sync 持久化数据文件
func (db *DB) Sync() error {
	if db.activeDataFile == nil {
		return nil
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.activeDataFile.Sync()
}

// Open 打开 bitcask 存储引擎实例
func Open(options Options) (*DB, error) {
	// 对用户传入的配置项进行校验
	if err := checkOptions(options); err != nil {
		return nil, err
	}
	var isInitial bool
	// 判断数据目录是否存在，如果不存在的话，则创建这个目录
	if _, err := os.Stat(options.DirPath); os.IsNotExist(err) {
		isInitial = true
		if err := os.MkdirAll(options.DirPath, os.ModePerm); err != nil {
			return nil, err
		}
	}
	entries, err := os.ReadDir(options.DirPath)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		isInitial = true
	}

	// 初始化 DB 实例结构体
	db := &DB{
		options:        options,
		mu:             new(sync.RWMutex),
		olderDataFiles: make(map[uint32]*data.DataFile),
		indexer:        index.NewIndexer(options.IndexType, options.DirPath, options.SyncWrites),
		isInitial:      isInitial,
	}
	//加载数据文件之前加载merge数据目录
	if err := db.loadMergeFiles(); err != nil {
		return nil, err
	}
	// 加载数据文件
	if err := db.loadDataFiles(); err != nil {
		return nil, err
	}
	//非持久化B+树索引需要从hintfile和数据文件中加载索引
	if options.IndexType != index.BPlustree {
		//从hintfile加载索引
		if err := db.loadIndexFromHintFile(); err != nil {
			return nil, err
		}
		// 从数据文件中加载索引
		if err := db.loadIndexFromDataFiles(); err != nil {
			return nil, err
		}
	}
	if options.IndexType == index.BPlustree {
		//持久化B+树索引需要从seqno文件中加载事务序列号
		if err := db.loadSeqNo(); err != nil {
			return nil, err
		}
		if db.activeDataFile != nil {
			size, err := db.activeDataFile.IO.Size()
			if err != nil {
				return nil, err
			}
			db.activeDataFile.WriteOff = size
		}
	}
	//如果是持久化B+树索引需要从seqno文件中加载事务序列号

	return db, nil
}

func (db *DB) Delete(key []byte) error {
	//如果key为空，返回错误
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}
	//构造logRecord结构体
	record := &data.LogRecord{
		Key:   logRecordKeyWithSeq(nonTransaction, key),
		Value: nil,
		Type:  data.LogRecordTypeDelete, //删除标记
	}
	if pos, _ := db.indexer.Get(key); pos == nil {
		return nil
	}
	_, err := db.appnedLogRecordWithLock(record)
	if err != nil {
		return err
	}
	//更新内存索引
	if ok := db.indexer.Delete(key); !ok {
		return ErrIndexUpdateFailed
	}
	return nil
}

func (db *DB) Put(key []byte, value []byte) error {
	//如果key为空，返回错误
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}
	//构造logRecord结构体
	record := &data.LogRecord{
		Key:   logRecordKeyWithSeq(nonTransaction, key),
		Value: value,
		Type:  data.LogRecordTypeNormal, //正常数据
	}
	LogRecordPos, err := db.appnedLogRecordWithLock(record)
	if err != nil {
		return err
	}
	//更新内存索引
	if ok := db.indexer.Put(key, LogRecordPos); !ok {
		return ErrIndexUpdateFailed
	}
	return nil
}
func (db *DB) Get(key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	//如果key为空，返回错误
	if len(key) == 0 {
		return nil, ErrKeyIsEmpty
	}
	logRecordPos, _ := db.indexer.Get(key)
	//若key不存在则返回错误
	if logRecordPos == nil {
		return nil, ErrKeyIsUnfound
	} // 根据文件 id 找到对应的数据文件
	return db.getValueByPosition(logRecordPos)
	// var dataFile *data.DataFile
	// if db.activeDataFile.Fid == logRecordPos.Fid {
	// 	dataFile = db.activeDataFile
	// } else {
	// 	dataFile = db.olderDataFiles[logRecordPos.Fid]
	// }
	// // 数据文件为空
	// if dataFile == nil {
	// 	return nil, ErrDataFileNotFound
	// }

	// // 根据偏移读取对应的数据
	// logRecord, _, err := dataFile.ReadLogRecord(logRecordPos.Offset)
	// if err != nil {
	// 	return nil, err
	// }

	// if logRecord.Type == data.LogRecordTypeDelete {
	// 	return nil, ErrKeyIsUnfound
	// }

	// return logRecord.Value, nil
}
func (db *DB) ListKeys() [][]byte {
	iterator := db.indexer.Iterator(false)
	keys := make([][]byte, db.indexer.Size())
	var idx int
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		keys[idx] = iterator.Key()
		idx++
	}
	iterator.Close()
	return keys
}
func (db *DB) Fold(fn func(key []byte, value []byte) bool) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	iterator := db.indexer.Iterator(false)
	defer iterator.Close()
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		value, err := db.getValueByPosition(iterator.Value())
		if err != nil {
			return err
		}
		if !fn(iterator.Key(), value) {
			break
		}
	}
	return nil
}

// 根据索引信息获取对应的 value
func (db *DB) getValueByPosition(logRecordPos *data.LogRecordPos) ([]byte, error) {
	// 根据文件 id 找到对应的数据文件
	var dataFile *data.DataFile
	if db.activeDataFile.Fid == logRecordPos.Fid {
		dataFile = db.activeDataFile
	} else {
		dataFile = db.olderDataFiles[logRecordPos.Fid]
	}
	// 数据文件为空
	if dataFile == nil {
		return nil, ErrDataFileNotFound
	}

	// 根据偏移读取对应的数据
	logRecord, _, err := dataFile.ReadLogRecord(logRecordPos.Offset)
	if err != nil {
		return nil, err
	}

	if logRecord.Type == data.LogRecordTypeDelete {
		return nil, ErrKeyIsUnfound
	}

	return logRecord.Value, nil
}
func (db *DB) appnedLogRecordWithLock(record *data.LogRecord) (*data.LogRecordPos, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.appnedLogRecord(record)
}
func (db *DB) appnedLogRecord(record *data.LogRecord) (*data.LogRecordPos, error) {
	// db.mu.Lock()
	// defer db.mu.Unlock()
	//数据库在没有文件写入时是没有活跃文件的，所以需要初始化文件id
	//如果当前的活跃文件为空则初始化活跃文件
	if db.activeDataFile == nil {
		err := db.setActiveDataFile()
		if err != nil {
			return nil, err
		}
	}
	//写入数据编码
	encodedRecord, size := data.EncodeLogRecord(record)
	//如果超过文件大小阈值就关闭文件打开一个新文件
	if db.activeDataFile.WriteOff+size > int64(db.options.DataFileSizeThreshold) {
		//持久化旧文件数据
		if err := db.activeDataFile.Sync(); err != nil {
			return nil, err
		}
		//将当前活跃文件加入旧文件集合
		db.olderDataFiles[db.activeDataFile.Fid] = db.activeDataFile
		//创建新的活跃文件
		err := db.setActiveDataFile()
		if err != nil {
			return nil, err
		}
	}
	//写入数据到活跃文件
	writeOff := db.activeDataFile.WriteOff
	if err := db.activeDataFile.Write(encodedRecord); err != nil {
		return nil, err
	}
	if db.options.SyncWrites {
		if err := db.activeDataFile.Sync(); err != nil {
			return nil, err
		}
	}
	//构造LogRecordPos返回
	logRecordPos := &data.LogRecordPos{
		Fid:    db.activeDataFile.Fid,
		Offset: writeOff,
		Size:   uint32(size),
	}
	//更新活跃文件的写入偏移量
	//TODO 后续继承到datafile的Write方法中
	//db.activeDataFile.WriteOff += size
	return logRecordPos, nil
}

// 访问此方法前必须持有锁不然会并发问题，导致activeDataFile被多次创建
func (db *DB) setActiveDataFile() error {
	var fid uint32 = 0
	//如果旧文件集合不为空，则找到最大的文件id，并在此基础上加1作为新的活跃文件id
	if db.activeDataFile != nil {
		fid = db.activeDataFile.Fid + 1
	}
	//创建新的数据文件
	dataFile, err := data.OpenDataFile(db.options.DirPath, fid)
	if err != nil {
		panic(err)
	}
	db.activeDataFile = dataFile
	return nil
}

// 从磁盘中加载数据文件
func (db *DB) loadDataFiles() error {
	dirEntries, err := os.ReadDir(db.options.DirPath)
	if err != nil {
		return err
	}

	var fileIds []int
	// 遍历目录中的所有文件，找到所有以 .data 结尾的文件
	for _, entry := range dirEntries {
		if strings.HasSuffix(entry.Name(), data.DataFileNameSuffix) {
			splitNames := strings.Split(entry.Name(), ".")
			fileId, err := strconv.Atoi(splitNames[0])
			// 数据目录有可能被损坏了
			if err != nil {
				return ErrDataDirectoryCorrupted
			}
			fileIds = append(fileIds, fileId)
		}
	}

	//	对文件 id 进行排序，从小到大依次加载
	sort.Ints(fileIds)
	db.fileIds = fileIds

	// 遍历每个文件id，打开对应的数据文件
	for i, fid := range fileIds {
		dataFile, err := data.OpenDataFile(db.options.DirPath, uint32(fid))
		if err != nil {
			return err
		}
		if i == len(fileIds)-1 { // 最后一个，id是最大的，说明是当前活跃文件
			db.activeDataFile = dataFile
		} else { // 说明是旧的数据文件
			db.olderDataFiles[uint32(fid)] = dataFile
		}
	}
	return nil
}

// 从数据文件中加载索引
// 遍历文件中的所有记录，并更新到内存索引中
func (db *DB) loadIndexFromDataFiles() error {
	// 没有文件，说明数据库是空的，直接返回
	if len(db.fileIds) == 0 {
		return nil
	}
	//查看是否发生过merge操作
	hasMerge, nonMergeFileId := false, uint32(0)
	mergeFinFileName := filepath.Join(db.options.DirPath, data.MergeFinishedFileName)
	if _, err := os.Stat(mergeFinFileName); err == nil {
		fid, err := db.getNonMergeFileId(db.options.DirPath)
		if err != nil {
			return err
		}
		hasMerge = true
		nonMergeFileId = fid
	}
	updateIndex := func(key []byte, Type data.LogRecordType, logRecordPos *data.LogRecordPos) error {
		var ok bool
		if Type == data.LogRecordTypeDelete {
			//NOTE这里由于bitcask是追加写入所以删除操作只是添加了一个删除标记，并没有真正删除数据文件中的数据
			//所以在加载索引时需要将该key从内存索引中删除
			//比如用户先执行了put(k1,v1)然后delete(k1)最后再put(k1,v2)
			//那么数据文件中会有三条记录，k1:v1、k1:delete标记、k1:v2
			//在加载索引时会先将k1指向v1的位置，然后再将k1从内存索引中删除，最后再将k1指向v2的位置
			//这样就保证了内存索引中的数据是最新的
			ok = db.indexer.Delete(key)
		} else if Type == data.LogRecordTypeNormal {
			ok = db.indexer.Put(key, logRecordPos)
		}
		if !ok {
			return ErrIndexUpdateFailed
		}
		return nil
	}
	//暂存批量写入的记录
	batchRecords := make(map[uint64][]*data.TransactionRecord)
	curSeqNum := nonTransaction
	// 遍历所有的文件id，处理文件中的记录
	for i, fid := range db.fileIds {
		//如果发生过merge操作则跳过非merge文件id之前的文件
		if hasMerge && uint32(fid) < nonMergeFileId {
			continue
		}
		var fileId = uint32(fid)
		var dataFile *data.DataFile
		if fileId == db.activeDataFile.Fid {
			dataFile = db.activeDataFile
		} else {
			dataFile = db.olderDataFiles[fileId]
		}

		var offset int64 = 0
		for {
			logRecord, size, err := dataFile.ReadLogRecord(offset)

			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			//fmt.Printf("type:%d,keySize:%d,valueSize:%d\n", logRecord.Type, len(logRecord.Key), len(logRecord.Value))
			// 构造内存索引并保存
			logRecordPos := &data.LogRecordPos{Fid: fileId, Offset: offset}
			realKey, seqNum := parseLogRecordKey(logRecord.Key)
			fmt.Printf("type:%d,key:%s,seqNum:%d\n", logRecord.Type, string(realKey), seqNum)
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
			if seqNum > curSeqNum {
				curSeqNum = seqNum
			}
			offset += size

			// 如果是当前活跃文件，更新这个文件的 WriteOff
		}
		// 如果是当前活跃文件，更新这个文件的 WriteOff
		if i == len(db.fileIds)-1 {
			db.activeDataFile.WriteOff = offset
		}
	}
	db.seqNum = curSeqNum
	return nil
}
func checkOptions(options Options) error {
	if options.DirPath == "" {
		return errors.New("database dir path is empty")
	}
	if options.DataFileSizeThreshold <= 0 {
		return errors.New("database data file size must be greater than 0")
	}
	return nil
}
func (db *DB) loadSeqNo() error {
	fileName := filepath.Join(db.options.DirPath, data.SeqNoFileName)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil
	}

	seqNoFile, err := data.OpenSeqNoFile(db.options.DirPath)
	if err != nil {
		return err
	}
	record, _, err := seqNoFile.ReadLogRecord(0)
	seqNo, err := strconv.ParseUint(string(record.Value), 10, 64)
	if err != nil {
		return err
	}
	db.seqNum = seqNo
	db.seqNoFileExists = true

	return os.Remove(fileName)
}
