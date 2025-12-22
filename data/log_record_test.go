package data

import (
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_EncoderLogRecord(t *testing.T) {
	logRecord := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("bitcask"),
		Type:  LogRecordTypeNormal, //正常数据
	}
	encBytes, size := EncodeLogRecord(logRecord)
	assert.NotNil(t, encBytes)
	assert.Greater(t, size, int64(5))
	t.Log(encBytes)

	logRecord2 := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("bitcask"),
		Type:  LogRecordTypeDelete, //删除数据
	}
	encBytes2, size2 := EncodeLogRecord(logRecord2)
	assert.NotNil(t, encBytes2)
	assert.Greater(t, size2, int64(5))
	t.Log(encBytes2)
	//测试value为空的情况
	logRecord3 := &LogRecord{
		Key: []byte("name"),
		//Value: []byte("bitcask"),
		Type: LogRecordTypeNormal, //正常数据
	}
	encBytes3, size3 := EncodeLogRecord(logRecord3)
	assert.NotNil(t, encBytes3)
	assert.Greater(t, size3, int64(5))
	t.Log(encBytes3)
}
func Test_DecodeLogRecordHeader(t *testing.T) {
	headerBuf := []byte{166, 77, 93, 234, 0, 8, 14}
	header, size := DecodeLogRecordHeader(headerBuf)
	assert.NotNil(t, header)
	assert.Equal(t, int64(7), size)
	assert.Equal(t, LogRecordTypeNormal, header.Type)
	assert.Equal(t, uint32(4), header.KeySize)
	assert.Equal(t, uint32(7), header.ValueSize)
	t.Log(header)

	headerBuf2 := []byte{208, 172, 82, 119, 1, 8, 14}
	header2, size2 := DecodeLogRecordHeader(headerBuf2)
	assert.NotNil(t, header2)
	assert.Equal(t, int64(7), size2)
	assert.Equal(t, LogRecordTypeDelete, header2.Type)
	assert.Equal(t, uint32(4), header2.KeySize)
	assert.Equal(t, uint32(7), header2.ValueSize)
	t.Log(header2)

	headerBuf3 := []byte{9, 252, 88, 14, 0, 8, 0}
	header3, size3 := DecodeLogRecordHeader(headerBuf3)
	assert.NotNil(t, header3)
	assert.Equal(t, int64(7), size3)
	assert.Equal(t, LogRecordTypeNormal, header3.Type)
	assert.Equal(t, uint32(4), header3.KeySize)
	assert.Equal(t, uint32(0), header3.ValueSize)
	t.Log(header3)

}
func Test_getCRC(t *testing.T) {
	logRecord := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("bitcask"),
		Type:  LogRecordTypeNormal, //正常数据
	}
	headerBuf := []byte{166, 77, 93, 234, 0, 8, 14}
	crc := getLogRecordCRC(logRecord, headerBuf[crc32.Size:])
	assert.Equal(t, uint32(3931983270), crc)

	logRecord2 := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("bitcask"),
		Type:  LogRecordTypeNormal, //正常数据
	}
	headerBuf2 := []byte{208, 172, 82, 119, 1, 8, 14}
	crc2 := getLogRecordCRC(logRecord2, headerBuf2[crc32.Size:])
	assert.Equal(t, uint32(2001906896), crc2)

	logRecord3 := &LogRecord{
		Key: []byte("name"),
		//Value: []byte("bitcask"),
		Type: LogRecordTypeNormal, //正常数据
	}
	headerBuf3 := []byte{9, 252, 88, 14, 0, 8, 0}
	crc3 := getLogRecordCRC(logRecord3, headerBuf3[crc32.Size:])
	assert.Equal(t, uint32(240712713), crc3)
}
