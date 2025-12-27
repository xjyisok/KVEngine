package redis

import (
	bitcaskgo "bitcask-go"
	"bitcask-go/utils"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Redis_String_Set(t *testing.T) {
	options := bitcaskgo.DefaultOptions
	dirPath, _ := os.MkdirTemp("", "bitcask-go-redis")
	options.DirPath = dirPath
	rds, err := newRedisDataStructure(options)
	assert.Nil(t, err)
	err = rds.Set(utils.GetTestKey(1), 0, utils.RandomValue(100))
	assert.Nil(t, err)
	err = rds.Set(utils.GetTestKey(2), time.Second*5, utils.RandomValue(100))
	assert.Nil(t, err)

	val1, err := rds.Get(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val1)

	time.Sleep(time.Second * 6)

	val2, err := rds.Get(utils.GetTestKey(2))
	assert.Equal(t, err, ErrExpired)
	assert.Nil(t, val2)

	_, err = rds.Get(utils.GetTestKey(33))
	assert.Equal(t, bitcaskgo.ErrKeyIsUnfound, err)
}
func Test_Redis_String_Get(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-del-type")
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	// del
	err = rds.Del(utils.GetTestKey(11))
	assert.Nil(t, err)

	err = rds.Set(utils.GetTestKey(1), 0, utils.RandomValue(100))
	assert.Nil(t, err)

	// type
	typ, err := rds.Type(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, String, typ)

	err = rds.Del(utils.GetTestKey(1))
	assert.Nil(t, err)

	_, err = rds.Get(utils.GetTestKey(1))
	assert.Equal(t, bitcaskgo.ErrKeyIsUnfound, err)
}
