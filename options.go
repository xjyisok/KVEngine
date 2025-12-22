package bitcaskgo

import "os"

type Options struct {
	DirPath               string //数据文件存储路径
	DataFileSizeThreshold uint32 //数据文件大小阈值
	SyncWrites            bool   //是否每次写入都进行持久化
	// 索引类型
	IndexType IndexerType
}
type IndexerType = int8

const (
	// BTree 索引
	BTree IndexerType = iota + 1

	// ART Adpative Radix Tree 自适应基数树索引
	ART
)

// IteratorOptions 索引迭代器配置项
type IteratorOptions struct {
	// 遍历前缀为指定值的 Key，默认为空
	Prefix []byte
	// 是否反向遍历，默认 false 是正向
	Reverse bool
}

var DefaultOptions = Options{
	DirPath:               os.TempDir(),
	DataFileSizeThreshold: 256 * 1024 * 1024, // 256MB
	SyncWrites:            false,
	IndexType:             BTree,
}
var DefaultIteratorOptions = IteratorOptions{
	Prefix:  nil,
	Reverse: false,
}
