package redis

import (
	bitcaskgo "bitcask-go"
	"encoding/binary"
	"errors"
	"time"
)

var (
	ErrWrongDataType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	ErrExpired       = errors.New("key has expired")
)

type RedisDataStructure struct {
	db *bitcaskgo.DB
}
type redisDataType = byte

const (
	String redisDataType = iota
	Set
	Hash
	List
	ZSet
)

func newRedisDataStructure(options bitcaskgo.Options) (*RedisDataStructure, error) {
	db, err := bitcaskgo.Open(options)
	if err != nil {
		return nil, err
	}
	return &RedisDataStructure{db: db}, err
}

// String 数据结构
func (rds *RedisDataStructure) Set(key []byte, ttl time.Duration, value []byte) error {
	if value == nil {
		return nil
	}
	//type+expire+value编码
	buf := make([]byte, binary.MaxVarintLen64+1)
	buf[0] = String
	var index = 1
	var expire int64 = 0
	if ttl != 0 {
		expire = time.Now().Add(ttl).UnixNano()
	}
	index += binary.PutVarint(buf[index:], expire)
	recordValue := make([]byte, index+len(value))
	copy(recordValue[:index], buf[:index])
	copy(recordValue[index:], buf[index:])
	err := rds.db.Put(key, recordValue)
	if err != nil {
		return err
	}
	return nil
}
func (rds *RedisDataStructure) Get(key []byte) ([]byte, error) {
	recordValue, err := rds.db.Get(key)
	if err != nil {
		return nil, err
	}
	dataType := recordValue[0]
	if dataType != String {
		return nil, ErrWrongDataType
	}
	var index = 1
	expireTime, n := binary.Varint(recordValue[index:])
	if expireTime > 0 && expireTime <= time.Now().UnixNano() {
		return nil, ErrExpired
	}
	index += n
	return recordValue[index:], nil
}
