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
}
type IndexType = int8

const (
	// Btree 索引
	Btree IndexType = iota + 1

	// ART 自适应基数树索引
	ART
)

// BtreeCRUD节点重写
type Item struct {
	key   []byte
	value *data.LogRecordPos
}

func (it *Item) Less(bi btree.Item) bool {
	return bytes.Compare(it.key, bi.(*Item).key) == -1
}
func NewIndexer(indexType IndexType) Indexer {
	switch indexType {
	case Btree:
		return NewBTree(32)
	case ART:
		return nil
	default:
		panic("unsupported index type")
	}
}
