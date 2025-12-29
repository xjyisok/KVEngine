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
func TestRedisDataStructure_HGet(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	// dir, _ := os.MkdirTemp("", "bitcask-go-redis-hget")
	dir := "/tmp/bitcask-go-redis-hget"
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	rds.HSet([]byte("my_hash"), []byte("hash-f1"), []byte("hash-val1"))
	rds.HSet([]byte("my_hash"), []byte("hash-f2"), []byte("hash-val2"))

	rds.HSet([]byte("my_hash"), []byte("hash-f3"), []byte("hash-val3"))
	rds.HSet([]byte("my_hash"), []byte("hash-f3"), []byte("hash-val-new"))

	v1, err := rds.HGet([]byte("my_hash"), []byte("hash-f3"))
	t.Log(string(v1))
	t.Log(err)

	v2, err := rds.HGet([]byte("my_hash"), []byte("hash-f5"))
	t.Log(string(v2))
	t.Log(err)
}

func TestRedisDataStructure_HDel(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	// dir, _ := os.MkdirTemp("", "bitcask-go-redis-hget")
	dir := "/tmp/bitcask-go-redis-hdel"
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	rds.HSet([]byte("my_hash"), []byte("hash-f1"), []byte("hash-val1"))
	rds.HSet([]byte("my_hash"), []byte("hash-f2"), []byte("hash-val2"))

	rds.HSet([]byte("my_hash"), []byte("hash-f3"), []byte("hash-val3"))
	rds.HSet([]byte("my_hash"), []byte("hash-f3"), []byte("hash-val-new"))

	v1, err := rds.HGet([]byte("my_hash"), []byte("hash-f3"))
	t.Log(string(v1))
	t.Log(err)

	// rds.HDel([]byte("my_hash"), []byte("hash-f3"))

	// v2, err := rds.HGet([]byte("my_hash"), []byte("hash-f3"))
	// t.Log(string(v2))
	// t.Log(err)

	v2, err := rds.HDel([]byte("my_hash1"), []byte("hash-f3"))
	t.Log(v2)
	t.Log(err)
}
func TestRedisDataStructure_SIsMember(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-sismember")
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	ok, err := rds.SAdd(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.True(t, ok)
	ok, err = rds.SAdd(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.False(t, ok)
	ok, err = rds.SAdd(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.True(t, ok)

	ok, err = rds.SIsMember(utils.GetTestKey(2), []byte("val-1"))
	assert.Nil(t, err)
	assert.False(t, ok)
	ok, err = rds.SIsMember(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.True(t, ok)
	ok, err = rds.SIsMember(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.True(t, ok)
	ok, err = rds.SIsMember(utils.GetTestKey(1), []byte("val-not-exist"))
	assert.Nil(t, err)
	assert.False(t, ok)
}

func TestRedisDataStructure_SRem(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-srem")
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	ok, err := rds.SAdd(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.True(t, ok)
	ok, err = rds.SAdd(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.False(t, ok)
	ok, err = rds.SAdd(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.True(t, ok)

	ok, err = rds.SRem(utils.GetTestKey(2), []byte("val-1"))
	assert.Nil(t, err)
	assert.False(t, ok)
	ok, err = rds.SRem(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.True(t, ok)

	ok, err = rds.SIsMember(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.False(t, ok)
}
func TestRedisDataStructure_LPop(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-lpop")
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	res, err := rds.LPush(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), res)
	res, err = rds.LPush(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.Equal(t, uint32(2), res)
	res, err = rds.LPush(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.Equal(t, uint32(3), res)

	val, err := rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val)
	val, err = rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val)
	val, err = rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val)
}

func TestRedisDataStructure_RPop(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-rpop")
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	res, err := rds.RPush(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), res)
	res, err = rds.RPush(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.Equal(t, uint32(2), res)
	res, err = rds.RPush(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.Equal(t, uint32(3), res)

	val, err := rds.RPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val)
	val, err = rds.RPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val)
	val, err = rds.RPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val)
}
func TestRedisDataStructure_ZScore(t *testing.T) {
	opts := bitcaskgo.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-zset")
	opts.DirPath = dir
	rds, err := newRedisDataStructure(opts)
	assert.Nil(t, err)

	ok, err := rds.ZAdd(utils.GetTestKey(1), 113, []byte("val-1"))
	assert.Nil(t, err)
	assert.True(t, ok)
	ok, err = rds.ZAdd(utils.GetTestKey(1), 333, []byte("val-1"))
	assert.Nil(t, err)
	assert.False(t, ok)
	ok, err = rds.ZAdd(utils.GetTestKey(1), 98, []byte("val-2"))
	assert.Nil(t, err)
	assert.True(t, ok)

	score, err := rds.ZScore(utils.GetTestKey(1), []byte("val-1"))
	assert.Nil(t, err)
	assert.Equal(t, float64(333), score)
	score, err = rds.ZScore(utils.GetTestKey(1), []byte("val-2"))
	assert.Nil(t, err)
	assert.Equal(t, float64(98), score)
}
