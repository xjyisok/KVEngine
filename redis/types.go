package redis

import (
	bitcaskgo "bitcask-go"
	"bitcask-go/utils"
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

// ======================= Hash 数据结构 =======================

func (rds *RedisDataStructure) HSet(key, field, value []byte) (bool, error) {
	// 先查询元数据
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return false, err
	}

	// 构造数据部分的 key
	hk := &hashInternalKey{
		key:     key,
		version: meta.version,
		field:   field,
	}
	encKey := hk.encode()

	var exist = true
	// 先查找是否存在
	if _, err = rds.db.Get(encKey); err == bitcaskgo.ErrKeyIsUnfound {
		exist = false
	}

	// 更新数据和元数据
	wb := rds.db.NewWriteBatch(bitcaskgo.DefaultWriteBatchOptions)
	if !exist {
		meta.size++
		_ = wb.Put(key, meta.encode())
	}
	_ = wb.Put(encKey, value)
	if err = wb.Commit(); err != nil {
		return false, err
	}

	return !exist, nil
}

func (rds *RedisDataStructure) HGet(key, field []byte) ([]byte, error) {
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return nil, nil
	}

	hk := &hashInternalKey{
		key:     key,
		version: meta.version,
		field:   field,
	}

	return rds.db.Get(hk.encode())
}

func (rds *RedisDataStructure) HDel(key, field []byte) (bool, error) {
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return false, err
	}
	if meta.size == 0 {
		return false, nil
	}

	// 构造数据部分的 key
	hk := &hashInternalKey{
		key:     key,
		version: meta.version,
		field:   field,
	}
	encKey := hk.encode()

	// 先查看是否存在
	var exist = true
	if _, err = rds.db.Get(encKey); err == bitcaskgo.ErrKeyIsUnfound {
		exist = false
	}

	// 更新数据和元数据
	if exist {
		wb := rds.db.NewWriteBatch(bitcaskgo.DefaultWriteBatchOptions)
		meta.size--
		_ = wb.Put(key, meta.encode())
		_ = wb.Delete(encKey)
		if err = wb.Commit(); err != nil {
			return false, err
		}
	}
	return !exist, nil
}

// ======================= Set 数据结构 =======================

func (rds *RedisDataStructure) SAdd(key, member []byte) (bool, error) {
	// 查找元数据
	meta, err := rds.findMetadata(key, Set)
	if err != nil {
		return false, err
	}

	// 构造一个数据部分的 key
	sk := &setInternalKey{
		key:     key,
		version: meta.version,
		member:  member,
	}

	var ok bool
	if _, err = rds.db.Get(sk.encode()); err == bitcaskgo.ErrKeyIsUnfound {
		// 不存在的话则更新
		wb := rds.db.NewWriteBatch(bitcaskgo.DefaultWriteBatchOptions)
		meta.size++
		_ = wb.Put(key, meta.encode())
		_ = wb.Put(sk.encode(), nil)
		if err = wb.Commit(); err != nil {
			return false, err
		}
		ok = true
	}

	return ok, nil
}

func (rds *RedisDataStructure) SIsMember(key, member []byte) (bool, error) {
	meta, err := rds.findMetadata(key, Set)
	if err != nil {
		return false, err
	}
	if meta.size == 0 {
		return false, nil
	}

	// 构造一个数据部分的 key
	sk := &setInternalKey{
		key:     key,
		version: meta.version,
		member:  member,
	}

	_, err = rds.db.Get(sk.encode())
	if err != nil && err != bitcaskgo.ErrKeyIsUnfound {
		return false, err
	}
	if err == bitcaskgo.ErrKeyIsUnfound {
		return false, nil
	}
	return true, nil
}

func (rds *RedisDataStructure) SRem(key, member []byte) (bool, error) {
	meta, err := rds.findMetadata(key, Set)
	if err != nil {
		return false, err
	}
	if meta.size == 0 {
		return false, nil
	}

	// 构造一个数据部分的 key
	sk := &setInternalKey{
		key:     key,
		version: meta.version,
		member:  member,
	}

	if _, err = rds.db.Get(sk.encode()); err == bitcaskgo.ErrKeyIsUnfound {
		return false, nil
	}

	// 更新
	wb := rds.db.NewWriteBatch(bitcaskgo.DefaultWriteBatchOptions)
	meta.size--
	_ = wb.Put(key, meta.encode())
	_ = wb.Delete(sk.encode())
	if err = wb.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// ======================= List 数据结构 =======================
func (rds *RedisDataStructure) LPush(key []byte, elements []byte) (uint32, error) {
	return rds.redisPush(key, elements, true)
}
func (rds *RedisDataStructure) RPush(key []byte, elements []byte) (uint32, error) {
	return rds.redisPush(key, elements, false)
}
func (rds *RedisDataStructure) LPop(key []byte) ([]byte, error) {
	return rds.redisPop(key, true)
}
func (rds *RedisDataStructure) RPop(key []byte) ([]byte, error) {
	return rds.redisPop(key, false)
}
func (rds *RedisDataStructure) redisPush(key []byte, element []byte, isLeft bool) (uint32, error) {
	meta, err := rds.findMetadata(key, List)
	if err != nil {
		return 0, err
	}
	//构造数据部分key
	lk := &listInternalKey{
		key:     key,
		version: meta.expire,
	}
	if isLeft {
		lk.index = meta.head - 1
	} else {
		lk.index = meta.tail
	}
	wb := rds.db.NewWriteBatch(bitcaskgo.DefaultWriteBatchOptions)
	meta.size++
	if isLeft {
		meta.head--
	} else {
		meta.tail++
	}
	_ = wb.Put(key, meta.encode())
	_ = wb.Put(lk.encode(), element)
	err = wb.Commit()
	if err != nil {
		return 0, err
	}
	return meta.size, nil
}
func (rds *RedisDataStructure) redisPop(key []byte, isLeft bool) ([]byte, error) {
	meta, err := rds.findMetadata(key, List)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return nil, nil
	}
	lk := &listInternalKey{
		key:     key,
		version: meta.expire,
	}
	if isLeft {
		lk.index = meta.head
	} else {
		lk.index = meta.tail - 1
	}
	element, err := rds.db.Get(lk.encode())
	if err != nil {
		return nil, nil
	}
	meta.size--
	if isLeft {
		meta.head++
	} else {
		meta.tail--
	}
	wb := rds.db.NewWriteBatch(bitcaskgo.DefaultWriteBatchOptions)
	_ = wb.Put(key, meta.encode())
	err = wb.Commit()
	if err != nil {
		return nil, err
	}
	return element, nil
}

// ======================= ZSet 数据结构 =======================

func (rds *RedisDataStructure) ZAdd(key []byte, score float64, member []byte) (bool, error) {
	meta, err := rds.findMetadata(key, ZSet)
	if err != nil {
		return false, err
	}

	// 构造数据部分的key
	zk := &zsetInternalKey{
		key:     key,
		version: meta.version,
		score:   score,
		member:  member,
	}

	var exist = true
	// 查看是否已经存在
	value, err := rds.db.Get(zk.encodeWithMember())
	if err != nil && err != bitcaskgo.ErrKeyIsUnfound {
		return false, err
	}
	if err == bitcaskgo.ErrKeyIsUnfound {
		exist = false
	}
	if exist {
		if score == utils.FloatFromBytes(value) {
			return false, nil
		}
	}

	// 更新元数据和数据
	wb := rds.db.NewWriteBatch(bitcaskgo.DefaultWriteBatchOptions)
	if !exist {
		meta.size++
		_ = wb.Put(key, meta.encode())
	}
	if exist {
		oldKey := &zsetInternalKey{
			key:     key,
			version: meta.version,
			member:  member,
			score:   utils.FloatFromBytes(value),
		}
		_ = wb.Delete(oldKey.encodeWithScore())
	}
	_ = wb.Put(zk.encodeWithMember(), utils.Float64ToBytes(score))
	_ = wb.Put(zk.encodeWithScore(), nil)
	if err = wb.Commit(); err != nil {
		return false, err
	}

	return !exist, nil
}

func (rds *RedisDataStructure) ZScore(key []byte, member []byte) (float64, error) {
	meta, err := rds.findMetadata(key, ZSet)
	if err != nil {
		return -1, err
	}
	if meta.size == 0 {
		return -1, nil
	}

	// 构造数据部分的key
	zk := &zsetInternalKey{
		key:     key,
		version: meta.version,
		member:  member,
	}

	value, err := rds.db.Get(zk.encodeWithMember())
	if err != nil {
		return -1, err
	}

	return utils.FloatFromBytes(value), nil
}

func (rds *RedisDataStructure) findMetadata(key []byte, dataType redisDataType) (*metadata, error) {
	metaBuf, err := rds.db.Get(key)
	if err != nil && err != bitcaskgo.ErrKeyIsUnfound {
		return nil, err
	}

	var meta *metadata
	// 元数据不存在，则初始化
	var exist = true
	if err == bitcaskgo.ErrKeyIsUnfound {
		exist = false
	} else {
		meta = decodeMetadata(metaBuf)
		// 存在，判断类型是否匹配
		if meta.dataType != dataType {
			return nil, ErrWrongDataType
		}
		// 判断是否过期
		if meta.expire != 0 && meta.expire <= time.Now().UnixNano() {
			exist = false
		}
	}

	if !exist {
		meta = &metadata{
			dataType: dataType,
			expire:   0,
			version:  time.Now().UnixNano(),
			size:     0,
		}
		if dataType == List {
			meta.head = initialListMark
			meta.tail = initialListMark
		}
	}
	return meta, nil
}
