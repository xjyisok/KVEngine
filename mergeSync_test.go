package bitcaskgo

import (
	"bitcask-go/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_MergeSync(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-merge-2")
	opts.DataFileSizeThreshold = 32 * 1024 * 1024
	opts.DataFileMergeRatio = 0
	opts.DirPath = dir
	db, err := Open(opts)
	//defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	for i := 0; i < 105000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(128))
		assert.Nil(t, err)
	}
	valInit, err0 := db.Get(utils.GetTestKey(80000))
	assert.Nil(t, err0)
	for i := 0; i < 5000; i++ {
		err := db.Delete(utils.GetTestKey(i))
		assert.Nil(t, err)
	}
	_, err101 := db.Get(utils.GetTestKey(101))
	assert.Equal(t, ErrKeyIsUnfound, err101)
	err1 := db.Merge_Sync()
	assert.Nil(t, err1)
	err2 := db.loadMergeSyncFiles()
	assert.Nil(t, err2)
	_, err3 := db.Get(utils.GetTestKey(100))
	assert.Equal(t, ErrKeyIsUnfound, err3)
	valAfterMerge, err4 := db.Get(utils.GetTestKey(80000))
	assert.Nil(t, err4)
	assert.Equal(t, valInit, valAfterMerge)
	db.Close()
}
