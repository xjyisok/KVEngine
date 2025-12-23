package index

import (
	"bytes"

	"bitcask-go/data"

	"github.com/google/btree"
)

type Indexer interface {
	Put(key []byte, record *data.LogRecordPos) bool
	Get(key []byte) (*data.LogRecordPos, bool)
	Delete(key []byte) bool
	Iterator(reverse bool) Iterator
	Size() int64
	Close()error
}
type IndexType = int8

const (
	// Btree 索引
	Btree IndexType = iota + 1

	// ART 自适应基数树索引
	ART
	// BPlusTree B+树索引
	BPlustree
)

// BtreeCRUD节点重写
type Item struct {
	key   []byte
	value *data.LogRecordPos
}

func (it *Item) Less(bi btree.Item) bool {
	return bytes.Compare(it.key, bi.(*Item).key) == -1
}
func NewIndexer(indexType IndexType, dirPath string,sync bool) Indexer {
	switch indexType {
	case Btree:
		return NewBTree(32)
	case ART:
		return NewART()
	case BPlustree:
		return NewBPlusTree(dirPath, sync)
	default:
		panic("unsupported index type")
	}
}

// Iterator 通用索引迭代器
type Iterator interface {
	// Rewind 重新回到迭代器的起点，即第一个数据
	Rewind()

	// Seek 根据传入的 key 查找到第一个大于（或小于）等于的目标 key，根据从这个 key 开始遍历
	Seek(key []byte)

	// Next 跳转到下一个 key
	Next()

	// Valid 是否有效，即是否已经遍历完了所有的 key，用于退出遍历
	Valid() bool

	// Key 当前遍历位置的 Key 数据
	Key() []byte

	// Value 当前遍历位置的 Value 数据
	Value() *data.LogRecordPos

	// Close 关闭迭代器，释放相应资源
	Close()
}
