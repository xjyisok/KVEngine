package data

import (
	"bitcask-go/fio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

var (
	ErrInvalidCRC = errors.New("the log record crc is invalid")
)

type DataFile struct {
	Fid      uint32 //文件id
	WriteOff int64  //文件写入偏移量
	IO       fio.IOManager
}

// crc type keySize valueSize
// 4 + 1+5+5=15	keySize和valueSize使用变长编码节省磁盘空间
const maxLogRecordHeaderSize = binary.MaxVarintLen32*2 + 1 + 4 //日志记录最大头部大小

const DataFileNameSuffix = ".data"

// 初始化数据文件
func OpenDataFile(dirPath string, fid uint32) (*DataFile, error) {
	fileName := fmt.Sprintf("%09d", fid) + DataFileNameSuffix
	ioManager, err := fio.NewIOManager(dirPath + "/" + fileName)
	if err != nil {
		return nil, err
	}
	return &DataFile{
		Fid:      fid,
		WriteOff: 0, //初始化写入偏移量为0,后续写入数据时更新,在初始化indexer的时候是从文件头开始读数据创建索引的
		IO:       ioManager,
	}, nil
}
func (df *DataFile) Sync() error {
	return df.IO.Sync()
}
func (df *DataFile) Write(buf []byte) error {
	n, err := df.IO.Write(buf)
	if err != nil {
		return err
	}
	df.WriteOff += int64(n)
	return nil
}
func (df *DataFile) Close() error {
	return df.IO.Close()
}

// 从数据文件中读取一条日志记录
func (df *DataFile) ReadLogRecord(offset int64) (*LogRecord, int64, error) {
	//获取文件大小
	fileSize, err := df.IO.Size()
	if err != nil {
		return nil, 0, err
	}
	headerReadBytes := maxLogRecordHeaderSize
	//如果文件偏移量加上数据条目最大长度超过文件大小则只读到文件末尾
	//如果每次都是读取maxLogRecordHeaderSize长度的数据，可能会读到文件末尾之外，导致读取失败
	//因为keySize和valueSize是变长编码的，所以需要动态读取
	//实际上logRecordHeader最大也就15字节，所以是可能存在读超文件末尾的，例如文件只剩12字节了，crc+type+keySize+valueSize的变长编码占了8字节，kV四字节刚好给文件填满
	//这时候如果还是按照15字节去读就会读超文件末尾
	if offset+maxLogRecordHeaderSize > fileSize {
		headerReadBytes = int(fileSize - offset)
	}
	// 读取头部信息
	headerbuf, err := df.readNBytes(offset, headerReadBytes)
	if err != nil {
		return nil, 0, err
	}
	//这里需要注释掉，因为offset在加载索引时会直达文件末尾最后一次读取是空的
	// if len(headerbuf) == 0 {
	// 	return nil, 0, nil
	// }
	logRecordHeader, headerSize := DecodeLogRecordHeader(headerbuf)
	if logRecordHeader == nil {
		return nil, 0, io.EOF
	}
	//读到文件末尾了
	if logRecordHeader.KeySize == 0 && logRecordHeader.ValueSize == 0 && logRecordHeader.crc == 0 {
		return nil, 0, io.EOF
	}
	// 取出对应的 key 和 value 的长度
	keySize, valueSize := int64(logRecordHeader.KeySize), int64(logRecordHeader.ValueSize)
	var recordSize = headerSize + keySize + valueSize
	LogRecord := &LogRecord{
		Type: logRecordHeader.Type,
	}
	if keySize > 0 && valueSize >= 0 {
		// 读取 key 和 value
		kvBuf, err := df.readNBytes(offset+headerSize, int(keySize)+int(valueSize))
		if err != nil {
			return nil, 0, err
		}
		key := kvBuf[:keySize]
		value := kvBuf[keySize:]
		LogRecord.Key = key
		LogRecord.Value = value
	}
	crc := getLogRecordCRC(LogRecord, headerbuf[crc32.Size:headerSize])
	if crc != logRecordHeader.crc {
		return nil, 0, ErrInvalidCRC
	}
	return LogRecord, recordSize, nil
}
func (df *DataFile) readNBytes(offset int64, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := df.IO.Read(buf, offset)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
