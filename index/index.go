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

// BtreeCRUD节点重写
type Item struct {
	key   []byte
	value *data.LogRecordPos
}

func (it *Item) Less(bi btree.Item) bool {
	return bytes.Compare(it.key, bi.(*Item).key) == -1
}
