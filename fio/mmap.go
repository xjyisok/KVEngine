package fio

import (
	"os"

	"golang.org/x/exp/mmap"
)

// MMap (Memory Map a File) IO 类型
type MMap struct {
	readerAt *mmap.ReaderAt
}

func NewMMapIOManager(fileName string) (*MMap, error) {
	// 创建文件如果不存在
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return nil, err
		}
		file.Close()
	}
	readerAt, err := mmap.Open(fileName)
	if err != nil {
		return nil, err
	}
	return &MMap{readerAt: readerAt}, nil
}

func (mmap *MMap) Read(b []byte, offset int64) (int, error) {
	return mmap.readerAt.ReadAt(b, offset)
}

func (mmap *MMap) Write([]byte) (int, error) {
	panic("not implemented")
}

func (mmap *MMap) Sync() error {
	panic("not implemented")
}

func (mmap *MMap) Close() error {
	return mmap.readerAt.Close()
}

func (mmap *MMap) Size() (int64, error) {
	return int64(mmap.readerAt.Len()), nil
}
