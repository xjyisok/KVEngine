package data

import (
	"encoding/binary"
	"hash/crc32"
	"sync"
)

type LogRecordType = byte

const (
	LogRecordTypeNormal   LogRecordType = 0 //正常数据
	LogRecordTypeDelete   LogRecordType = 1 //删除标记
	LogRecordTypeBatchEnd LogRecordType = 2 //批量写入标记
)

var LogRecordBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 4*1024)
	},
}
var LogRecordPosPool = sync.Pool{
	New: func() any {
		return new(LogRecordPos)
	},
}

// const DataFileNameSuffix = ".data"

// 数据内存索引
// 代表一个日志文件中的一条记录在文件中的位置
type LogRecordPos struct {
	Fid    uint32 //描述key指向的数据所在的文件id
	Offset int64  //描述key指向的数据在文件中的偏移量
	Size   uint32 //描述key指向的数据的大小
}
type LogRecordHeader struct {
	crc       uint32        //校验和
	Type      LogRecordType //记录类型，正常数据还是删除标记
	KeySize   uint32        //key的大小
	ValueSize uint32        //value的大小
}

// 数据文件中的一条记录
type LogRecord struct {
	Key   []byte
	Value []byte
	Type  LogRecordType //记录类型，正常数据还是删除标记
}

// 【新增】事务记录结构体
type TransactionRecord struct {
	Record *LogRecord
	Pos    *LogRecordPos
}

func EncodeLogRecord(logRecord *LogRecord) ([]byte, int64) {
	// 初始化一个 header 部分的字节数组
	header := make([]byte, maxLogRecordHeaderSize)

	// 第五个字节存储 Type
	header[4] = logRecord.Type
	var index = 5
	// 5 字节之后，存储的是 key 和 value 的长度信息
	// 使用变长类型，节省空间
	index += binary.PutVarint(header[index:], int64(len(logRecord.Key)))
	index += binary.PutVarint(header[index:], int64(len(logRecord.Value)))

	var size = index + len(logRecord.Key) + len(logRecord.Value)
	encBytes := make([]byte, size)

	// 将 header 部分的内容拷贝过来
	copy(encBytes[:index], header[:index])
	// 将 key 和 value 数据拷贝到字节数组中
	copy(encBytes[index:], logRecord.Key)
	copy(encBytes[index+len(logRecord.Key):], logRecord.Value)

	// 对整个 LogRecord 的数据进行 crc 校验
	crc := crc32.ChecksumIEEE(encBytes[4:])
	binary.LittleEndian.PutUint32(encBytes[:4], crc)
	//fmt.Printf("header Length:%d,crc:%d\n", index, crc)
	return encBytes, int64(size)
}

// 零 alloc 编码
func EncodeLogRecordTo(buf []byte, record *LogRecord) ([]byte, int64) {
	start := len(buf)

	// crc 预留 4 字节
	buf = append(buf, 0, 0, 0, 0)

	// type
	buf = append(buf, record.Type)

	// key size
	buf = binary.AppendVarint(buf, int64(len(record.Key)))
	// value size
	buf = binary.AppendVarint(buf, int64(len(record.Value)))

	// key + value
	buf = append(buf, record.Key...)
	buf = append(buf, record.Value...)

	// crc（从 Type 开始）
	crc := crc32.ChecksumIEEE(buf[start+4:])
	binary.LittleEndian.PutUint32(buf[start:start+4], crc)

	size := int64(len(buf) - start)
	return buf, size
}
func DecodeLogRecordHeader(data []byte) (*LogRecordHeader, int64) {
	if len(data) <= 4 {
		return nil, 0
	}

	header := &LogRecordHeader{
		crc:  binary.LittleEndian.Uint32(data[:4]),
		Type: data[4],
	}

	var index = 5
	// 取出实际的 key size
	keySize, n := binary.Varint(data[index:])
	header.KeySize = uint32(keySize)
	index += n

	// 取出实际的 value size
	valueSize, n := binary.Varint(data[index:])
	header.ValueSize = uint32(valueSize)
	index += n

	return header, int64(index)
}
func EncodeLogRecordPos(pos *LogRecordPos) []byte {
	buf := make([]byte, binary.MaxVarintLen32*2+binary.MaxVarintLen64)
	var index = 0
	index += binary.PutVarint(buf[index:], int64(pos.Fid))
	index += binary.PutVarint(buf[index:], pos.Offset)
	index += binary.PutUvarint(buf[index:], uint64(pos.Size))
	return buf[:index]
}

func DecodeLogReordPos(buf []byte) *LogRecordPos {
	var index = 0
	fileId, n := binary.Varint(buf[index:])
	index += n
	offset, n1 := binary.Varint(buf[index:])
	index += n1
	Size, _ := binary.Varint(buf[index:])
	return &LogRecordPos{Fid: uint32(fileId), Offset: offset, Size: uint32(Size)}
}
func getLogRecordCRC(logRecord *LogRecord, headerbuf []byte) uint32 {
	if logRecord == nil {
		return 0
	}

	crc := crc32.ChecksumIEEE(headerbuf[:])
	crc = crc32.Update(crc, crc32.IEEETable, logRecord.Key)
	crc = crc32.Update(crc, crc32.IEEETable, logRecord.Value)

	return crc
}
