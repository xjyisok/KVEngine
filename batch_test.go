package bitcaskgo

import (
	"bitcask-go/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_WriteBatch1(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-batch-1")
	opts.DirPath = dir
	db, err := Open(opts)
	//defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	// 写数据之后并不提交
	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb.Put(utils.GetTestKey(1), utils.RandomValue(10))
	assert.Nil(t, err)
	err = wb.Delete(utils.GetTestKey(2))
	assert.Nil(t, err)

	_, err = db.Get(utils.GetTestKey(1))
	assert.Equal(t, ErrKeyIsUnfound, err)

	// 正常提交数据
	err = wb.Commit()
	assert.Nil(t, err)

	val1, err := db.Get(utils.GetTestKey(1))
	assert.NotNil(t, val1)
	assert.Nil(t, err)

	// 删除有效的数据
	wb2 := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb2.Delete(utils.GetTestKey(1))
	assert.Nil(t, err)
	err = wb2.Commit()
	assert.Nil(t, err)

	_, err = db.Get(utils.GetTestKey(1))
	assert.Equal(t, ErrKeyIsUnfound, err)
	db.Close()
}

func TestDB_WriteBatch2(t *testing.T) {
	opts := DefaultOptions
	dir := "/tmp/bitcask-go-batch-2"
	opts.DirPath = dir
	db, err := Open(opts)
	//defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	err = db.Put(utils.GetTestKey(1), utils.RandomValue(10))
	assert.Nil(t, err)

	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb.Put(utils.GetTestKey(2), utils.RandomValue(10))
	assert.Nil(t, err)
	err = wb.Delete(utils.GetTestKey(1))
	assert.Nil(t, err)

	err = wb.Commit()
	assert.Nil(t, err)

	err = wb.Put(utils.GetTestKey(11), utils.RandomValue(10))
	assert.Nil(t, err)
	err = wb.Commit()
	assert.Nil(t, err)
	err = db.Close()
	assert.Nil(t, err)

	_, err = Open(opts)
	assert.Nil(t, err)
	_, err = db.Get(utils.GetTestKey(1))
	assert.Equal(t, ErrKeyIsUnfound, err)

	// 校验序列号
	assert.Equal(t, uint64(2), db.seqNum)
	db.Close()
}

func TestDB_WriteBatch3(t *testing.T) {
	opts := DefaultOptions
	//dir, _ := os.MkdirTemp("", "bitcask-go-wb")
	dir := "/tmp/batch-test-1"
	opts.DirPath = dir
	db, err := Open(opts)
	//defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	//	提交之后再提交
	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb.Put(utils.GetTestKey(11), utils.RandomValue(12))
	assert.Nil(t, err)
	err = wb.Put(utils.GetTestKey(12), utils.RandomValue(12))
	assert.Nil(t, err)
	err = wb.Put(utils.GetTestKey(13), utils.RandomValue(12))
	assert.Nil(t, err)

	err = wb.Commit()
	t.Log(err)

	err = wb.Put(utils.GetTestKey(14), utils.RandomValue(12))
	assert.Nil(t, err)
	err = wb.Commit()
	t.Log(err)

	keys := db.ListKeys()
	for _, k := range keys {
		t.Log(string(k))
	}
	db.Close()
}

func TestDB_WriteBatch4(t *testing.T) {
	opts := DefaultOptions
	//dir, _ := os.MkdirTemp("", "bitcask-go-wb")
	dir := "/tmp/batch-test-3"
	opts.DirPath = dir
	db, err := Open(opts)
	//defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	for i := 0; i < 7000; i++ {
		wb.Put(utils.GetTestKey(i), utils.RandomValue(40960))
	}

	err = wb.Commit()
	t.Log(err)

	keys := db.ListKeys()
	t.Log(len(keys))
	t.Log(db.seqNum)
	db.Close()
}
