package fio

import (
	"os"
)

type FileIO struct {
	fd *os.File
}

func newFileIOManager(fileName string) (*FileIO, error) {
	fd, err := os.OpenFile(
		fileName,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}
	return &FileIO{
		fd: fd,
	}, nil
}
func (f *FileIO) Read(buf []byte, offset int64) (int, error) {
	return f.fd.ReadAt(buf, offset)
}
func (f *FileIO) Write(buf []byte) (int, error) {
	return f.fd.Write(buf)
}
func (f *FileIO) Sync() error {
	return f.fd.Sync()
}
func (f *FileIO) Close() error {
	return f.fd.Close()
}
