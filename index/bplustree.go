package index

import (
	"bitcask-go/data"
	"path/filepath"

	"go.etcd.io/bbolt"
)

const indexFileName = "bptree-index"

var indexBucketName = []byte("bitcask-index")

// BPlusTree B+树索引，将索引存储到磁盘上
// 使用 etcd 的 bbolt 库
type BPlusTree struct {
	tree *bbolt.DB
}

// NewBPlusTree 打开一个 B+ 树实例
func NewBPlusTree(dirPath string, sync bool) *BPlusTree {
	// 打开 bbolt 实例
	opts := bbolt.DefaultOptions
	opts.NoSync = !sync
	bptree, err := bbolt.Open(filepath.Join(dirPath, indexFileName), 0644, opts)
	if err != nil {
		panic("failed to open bptree at startup")
	}

	// 创建一个对应的 bucket
	if err := bptree.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(indexBucketName)
		return err
	}); err != nil {
		panic("failed to create bptree bucket at startup")
	}
	return &BPlusTree{tree: bptree}
}

func (bpt *BPlusTree) Put(key []byte, pos *data.LogRecordPos) (*data.LogRecordPos, bool) {
	var oldval []byte
	if err := bpt.tree.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucketName)
		oldval = bucket.Get(key)
		return bucket.Put(key, data.EncodeLogRecordPos(pos))
	}); err != nil {
		panic("failed to put index in bptree")
	}
	if len(oldval) == 0 {
		return nil, false
	}
	return data.DecodeLogReordPos(oldval), true
}

func (bpt *BPlusTree) Get(key []byte) (*data.LogRecordPos, bool) {
	var pos *data.LogRecordPos
	if err := bpt.tree.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucketName)
		value := bucket.Get(key)
		if len(value) != 0 {
			pos = data.DecodeLogReordPos(value)
		}
		return nil
	}); err != nil {
		panic("failed to get index in bptree")
	}
	return pos, pos != nil
}

func (bpt *BPlusTree) Delete(key []byte) (*data.LogRecordPos, bool) {
	var oldVal []byte
	if err := bpt.tree.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucketName)
		if oldVal = bucket.Get(key); len(oldVal) != 0 {
			return bucket.Delete(key)
		}
		return nil
	}); err != nil {
		panic("failed to delete value in bptree")
	}
	if len(oldVal) == 0 {
		return nil, false
	}
	return data.DecodeLogReordPos(oldVal), true
}

func (bpt *BPlusTree) Size() int64 {
	var size int
	if err := bpt.tree.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucketName)
		size = bucket.Stats().KeyN
		return nil
	}); err != nil {
		panic("failed to size in bptree")
	}
	return int64(size)
}

func (bpt *BPlusTree) Iterator(reverse bool) Iterator {
	return newBptreeIterator(bpt.tree, reverse)
}
func (bpt *BPlusTree) Close() error {
	return bpt.tree.Close()
}

// B+树迭代器
type bptreeIterator struct {
	tx        *bbolt.Tx
	cursor    *bbolt.Cursor
	reverse   bool
	currKey   []byte
	currValue []byte
}

func newBptreeIterator(tree *bbolt.DB, reverse bool) *bptreeIterator {
	tx, err := tree.Begin(false)
	if err != nil {
		panic("failed to begin a transaction")
	}

	bi := &bptreeIterator{
		tx:      tx,
		cursor:  tx.Bucket(indexBucketName).Cursor(),
		reverse: reverse,
	}
	bi.Rewind()
	return bi
}

func (bi *bptreeIterator) Rewind() {
	if bi.reverse {
		bi.currKey, bi.currValue = bi.cursor.Last()
	} else {
		bi.currKey, bi.currValue = bi.cursor.First()
	}
}

func (bi *bptreeIterator) Seek(key []byte) {
	bi.currKey, bi.currValue = bi.cursor.Seek(key)
}

func (bi *bptreeIterator) Next() {
	if bi.reverse {
		bi.currKey, bi.currValue = bi.cursor.Prev()
	} else {
		bi.currKey, bi.currValue = bi.cursor.Next()
	}
}

func (bi *bptreeIterator) Valid() bool {
	return len(bi.currKey) != 0
}

func (bi *bptreeIterator) Key() []byte {
	return bi.currKey
}

func (bi *bptreeIterator) Value() *data.LogRecordPos {
	return data.DecodeLogReordPos(bi.currValue)
}

func (bi *bptreeIterator) Close() {
	_ = bi.tx.Rollback()
}
