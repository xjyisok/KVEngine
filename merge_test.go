package bitcaskgo

import (
	"bitcask-go/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_Merge(t *testing.T) {
	opts := DefaultOptions
	//dir, _ := os.MkdirTemp("", "bitcask-go-get")
	dir := "/tmp/bitcask-go-merge-1"
	opts.DirPath = dir
	opts.DataFileSizeThreshold = 64 * 1024 * 1024
	db, err := Open(opts)
	//defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	//keys := db.ListKeys()
	//t.Log(len(keys))

	// val, err := db.Get(utils.GetTestKey(99033))
	// t.Log(string(val))
	// t.Log(err)

	for i := 0; i < 50000; i++ {
		db.Put(utils.GetTestKey(i), utils.RandomValue(128))
	}
	for i := 0; i < 50000; i++ {
		if i == 9033 {
			db.Put(utils.GetTestKey(i), utils.RandomValue(128))
		} else {
			db.Delete(utils.GetTestKey(i))
		}
	}
	db.Merge()
	db.Close()
}
