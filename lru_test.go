package bitcaskgo

import (
	"bitcask-go/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_Put_LRU(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-put")
	opts.DirPath = dir
	opts.DataFileSizeThreshold = 8 * 1024 * 1024
	db, err := Open(opts)
	//defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)
	var val0 []byte
	var val1 []byte
	for i := 0; i < 101000; i++ {
		if i == 101 {
			val0 = utils.RandomValue(128)
			db.Put(utils.GetTestKey(i), val0)
			continue
		}
		if i == 100000 {
			val1 = utils.RandomValue(128)
			db.Put(utils.GetTestKey(i), val1)
			continue
		}
		//fmt.Printf("cur:%d\n", i)
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(128))
		assert.Nil(t, err)
	}
	val, err := db.Get(utils.GetTestKey(101))
	assert.Nil(t, err)
	assert.Equal(t, val, val0)
	val2, err := db.Get(utils.GetTestKey(100000))
	assert.Nil(t, err)
	assert.Equal(t, val1, val2)
}
