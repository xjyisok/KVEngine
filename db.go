package bitcaskgo

import (
	"bitcask-go/data"
	"sync"

	"bitcask-go/index"
)

// 主要实现面向用户的操作接口
type DB struct {
	mu             *sync.RWMutex
	activeDataFile *data.DataFile            //当前活跃的数据文件
	olderDataFiles map[uint32]*data.DataFile //旧的数据文件集合
	options        Options
	indexer        index.Indexer //内存索引
}

func (db *DB) Put(key []byte, value []byte) error {
	//如果key为空，返回错误
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}
	//构造logRecord结构体
	record := &data.LogRecord{
		Key:   key,
		Value: value,
		Type:  data.LogRecordTypeNormal, //正常数据
	}
	LogRecordPos, err := db.appnedLogRecord(record)
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
func (db *DB) appnedLogRecord(record *data.LogRecord) (*data.LogRecordPos, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
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
	db.activeDataFile.WriteOff += size
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
