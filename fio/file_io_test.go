package fio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func DestoryFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		panic(err)
	}
}
func TestNewFileIOManager(t *testing.T) {
	fd, err := newFileIOManager(filepath.Join("/tmp", "a.data"))
	assert.NoError(t, err)
	assert.NotNil(t, fd)
}
func TestFileIO_WriteRead(t *testing.T) {
	fd, err := newFileIOManager(filepath.Join("/tmp", "a.data"))
	assert.NoError(t, err)
	defer fd.Close()
	defer DestoryFile(filepath.Join("/tmp", "a.data"))
	writeData := []byte("hello world")
	n, err := fd.Write(writeData)
	assert.NoError(t, err)
	assert.Equal(t, len(writeData), n)

	readBuf := make([]byte, len(writeData))
	n, err = fd.Read(readBuf, 0)
	assert.NoError(t, err)
	assert.Equal(t, len(writeData), n)
	assert.Equal(t, writeData, readBuf)
	t.Log(string(readBuf), err)

	readBuf2 := make([]byte, 7)
	n, err = fd.Write([]byte("xjyisOK"))
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
	n, err = fd.Read(readBuf2, 11)
	assert.NoError(t, err)
	assert.Equal(t, 7, n)
	t.Log(string(readBuf2), err)
}
func TestFileIO_Sync(t *testing.T) {
	fd, err := newFileIOManager(filepath.Join("/tmp", "a.data"))
	assert.NoError(t, err)
	defer fd.Close()
	defer DestoryFile(filepath.Join("/tmp", "a.data"))
	err = fd.Sync()
	assert.NoError(t, err)
}
func TestFileIO_Close(t *testing.T) {
	fd, err := newFileIOManager(filepath.Join("/tmp", "a.data"))
	defer DestoryFile(filepath.Join("/tmp", "a.data"))
	assert.NoError(t, err)
	err = fd.Close()
	assert.NoError(t, err)
}
