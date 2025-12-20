package data

type LogRecordType = byte

const (
	LogRecordTypeNormal LogRecordType = 0 //正常数据
	LogRecordTypeDelete LogRecordType = 1 //删除标记
)

// 数据内存索引
// 代表一个日志文件中的一条记录在文件中的位置
type LogRecordPos struct {
	Fid    uint32 //描述key指向的数据所在的文件id
	Offset int64  //描述key指向的数据在文件中的偏移量
	Size   uint32 //描述key指向的数据的大小
}

// 数据文件中的一条记录
type LogRecord struct {
	Key   []byte
	Value []byte
	Type  LogRecordType //记录类型，正常数据还是删除标记
}

func EncodeLogRecord(record *LogRecord) ([]byte, int64) {
	return nil, 0
}
