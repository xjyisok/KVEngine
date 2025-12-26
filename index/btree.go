package index

import (
	"bytes"
	"sort"
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
func (bt *BTree) Put(key []byte, record *data.LogRecordPos) (*data.LogRecordPos, bool) {
	bt.lock.Lock()
	defer bt.lock.Unlock()
	item := &Item{
		key:   key,
		value: record,
	}
	oldItem := bt.tree.ReplaceOrInsert(item)
	if oldItem == nil {
		return nil, false
	}
	return oldItem.(*Item).value, true
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

// Btree中所有的key
func (bt *BTree) Size() int64 {
	bt.lock.RLock()
	defer bt.lock.RUnlock()
	return int64(bt.tree.Len())
}

// BTree删除
func (bt *BTree) Delete(key []byte) (*data.LogRecordPos, bool) {
	bt.lock.Lock()
	defer bt.lock.Unlock()
	item := &Item{
		key: key,
	}
	oldItem := bt.tree.Delete(item)
	if oldItem == nil {
		return nil, false
	}
	return oldItem.(*Item).value, true
}
func (bt *BTree) Iterator(reverse bool) Iterator {
	if bt.tree == nil {
		return nil
	}
	bt.lock.RLock()
	defer bt.lock.RUnlock()
	return newBTreeIterator(bt.tree, reverse)
}
func (bt *BTree) Close() error {
	return nil
}

// BTree 索引迭代器
type btreeIterator struct {
	currIndex int     // 当前遍历的下标位置
	reverse   bool    // 是否是反向遍历
	values    []*Item // key+位置索引信息
}

func newBTreeIterator(tree *btree.BTree, reverse bool) *btreeIterator {
	var idx int
	values := make([]*Item, tree.Len())

	// 将所有的数据存放到数组中
	saveValues := func(it btree.Item) bool {
		values[idx] = it.(*Item)
		idx++
		return true
	}
	if reverse {
		tree.Descend(saveValues)
	} else {
		tree.Ascend(saveValues)
	}

	return &btreeIterator{
		currIndex: 0,
		reverse:   reverse,
		values:    values,
	}
}
func (it *btreeIterator) Rewind() {
	it.currIndex = 0
}

func (it *btreeIterator) Seek(key []byte) {
	if it.reverse {
		it.currIndex = sort.Search(len(it.values), func(i int) bool {
			return bytes.Compare(it.values[i].key, key) <= 0
		})
	} else {
		it.currIndex = sort.Search(len(it.values), func(i int) bool {
			return bytes.Compare(it.values[i].key, key) >= 0
		})
	}
}
func (it *btreeIterator) Next() {
	it.currIndex++
}

func (it *btreeIterator) Valid() bool {
	return it.currIndex < len(it.values)
}

func (it *btreeIterator) Key() []byte {
	return it.values[it.currIndex].key
}

func (it *btreeIterator) Value() *data.LogRecordPos {
	return it.values[it.currIndex].value
}

func (it *btreeIterator) Close() {
	it.values = nil
}
