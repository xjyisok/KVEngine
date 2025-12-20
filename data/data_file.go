package data

import (
	"bitcask-go/fio"
)

type DataFile struct {
	Fid      uint32 //文件id
	WriteOff int64  //文件写入偏移量
	IO       fio.IOManager
}

func OpenDataFile(dirPath string, fid uint32) (*DataFile, error) {
	return nil, nil
}
func (df *DataFile) Sync() error {
	return nil
}
func (df *DataFile) Write(byte []byte) error {
	return nil
}
func (df *DataFile) ReadLogRecord(offset int64) (*LogRecord, int64, error) {
	return nil, 0, nil
}
