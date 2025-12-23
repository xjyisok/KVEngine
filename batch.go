package bitcaskgo

import (
	"bitcask-go/data"
	"bitcask-go/index"
	"encoding/binary"
	"sync"
	"sync/atomic"
)

var signEndBatch = []byte("__BATCH_END__")

const nonTransaction uint64 = 0

type WriteBatch struct {
	mu          *sync.RWMutex
	db          *DB
	opts        WriteBatchOptions
	pendingPuts map[string]*data.LogRecord //即将写入DB的批量数据
}

func (db *DB) NewWriteBatch(opts WriteBatchOptions) *WriteBatch {
	if db.options.IndexType == index.BPlustree && !db.seqNoFileExists && !db.isInitial {
		panic("cannot create WriteBatch when using persistent BPlusTree index before loading seqNo")
	}
	return &WriteBatch{
		mu:          new(sync.RWMutex),
		db:          db,
		opts:        opts,
		pendingPuts: make(map[string]*data.LogRecord),
	}
}

// 批量写数据
func (wb *WriteBatch) Put(key []byte, value []byte) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}
	logRecord := &data.LogRecord{
		Key:   key,
		Value: value,
		Type:  data.LogRecordTypeNormal,
	}
	wb.pendingPuts[string(key)] = logRecord
	return nil
}
func (wb *WriteBatch) Delete(key []byte) error {
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}
	wb.mu.Lock()
	defer wb.mu.Unlock()
	logRecordPos, _ := wb.db.indexer.Get(key)
	if logRecordPos == nil {
		if wb.pendingPuts[string(key)] != nil {
			delete(wb.pendingPuts, string(key))
		}
		return nil
	}
	logRecord := &data.LogRecord{
		Key:  key,
		Type: data.LogRecordTypeDelete,
	}
	wb.pendingPuts[string(key)] = logRecord
	return nil
}
func (wb *WriteBatch) Commit() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	if len(wb.pendingPuts) == 0 {
		return nil
	}
	if len(wb.pendingPuts) > wb.opts.MaxBatchSize {
		return ErrExceedMaxBatchSize
	}
	//获取当前最新的事务seqNum
	wb.db.mu.Lock()
	defer wb.db.mu.Unlock()
	seqNum := atomic.AddUint64(&wb.db.seqNum, 1) //获取最新的序列号
	pos := make(map[string]*data.LogRecordPos)
	for _, logRecord := range wb.pendingPuts {
		// TODO: 实现批量写入逻辑
		logRecordPos, err := wb.db.appnedLogRecord(&data.LogRecord{
			Key:   logRecordKeyWithSeq(seqNum, logRecord.Key),
			Value: logRecord.Value,
			Type:  logRecord.Type,
		})
		if err != nil {
			return err
		}
		pos[string(logRecord.Key)] = logRecordPos
	}
	//写入最后一条数据时需要添加批次写结束标记
	_, err := wb.db.appnedLogRecord(&data.LogRecord{
		Key:   logRecordKeyWithSeq(seqNum, signEndBatch),
		Value: nil,
		Type:  data.LogRecordTypeBatchEnd,
	})
	if err != nil {
		return err
	}
	//根据配置决定是否持久化
	if wb.opts.SyncWrites {
		err = wb.db.activeDataFile.Sync()
		if err != nil {
			return err
		}
	}
	//更新内存索引
	for _, record := range wb.pendingPuts {
		logRecordPos := pos[string(record.Key)]
		if record.Type == data.LogRecordTypeDelete {
			if ok := wb.db.indexer.Delete(record.Key); !ok {
				return ErrIndexUpdateFailed
			}
		} else {
			if ok := wb.db.indexer.Put(record.Key, logRecordPos); !ok {
				return ErrIndexUpdateFailed
			}
		}
	}
	//清空批量数据
	wb.pendingPuts = make(map[string]*data.LogRecord)
	return nil
}
func logRecordKeyWithSeq(seqNum uint64, key []byte) []byte {
	//将seqNum转换为字节数组
	seqBytes := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(seqBytes, seqNum)
	enc := make([]byte, n+len(key))
	copy(enc[:n], seqBytes[:n])
	copy(enc[n:], key)
	return enc
}
func parseLogRecordKey(key []byte) ([]byte, uint64) {
	seqNo, n := binary.Uvarint(key)
	realKey := key[n:]
	return realKey, seqNo
}
