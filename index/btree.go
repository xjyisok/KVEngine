package index

import (
	"sync"

	"bitcask-go/data"

	"github.com/google/btree"
)

type BTree struct {
	tree *btree.BTree
	lock *sync.RWMutex
}

// NewBTree 创建一个BTree索引实例
func NewBTree(degree int) *BTree {
	return &BTree{
		tree: btree.New(degree),
		lock: new(sync.RWMutex),
	}
}

// BTree插入
func (bt *BTree) Put(key []byte, record *data.LogRecordPos) bool {
	bt.lock.Lock()
	defer bt.lock.Unlock()
	item := &Item{
		key:   key,
		value: record,
	}
	bt.tree.ReplaceOrInsert(item)
	return true
}

// Btree查找
func (bt *BTree) Get(key []byte) (*data.LogRecordPos, bool) {
	bt.lock.RLock()
	defer bt.lock.RUnlock()
	item := &Item{
		key: key,
	}
	result := bt.tree.Get(item)
	if result == nil {
		return nil, false
	}
	return result.(*Item).value, true
}

// BTree删除
func (bt *BTree) Delete(key []byte) bool {
	bt.lock.Lock()
	defer bt.lock.Unlock()
	item := &Item{
		key: key,
	}
	oldItem := bt.tree.Delete(item)
	if oldItem == nil {
		return false
	}
	return true
}
