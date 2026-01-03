package bitcaskgo

import (
	"os"
)

type Options struct {
	DirPath               string //数据文件存储路径
	DataFileSizeThreshold uint32 //数据文件大小阈值
	SyncWrites            bool   //是否每次写入都进行持久化
	// 索引类型
	IndexType          IndexerType
	BytesPerSync       uint    //每次同步写入的字节数
	MMapIsOpen         bool    //是否使用内存映射IO
	DataFileMergeRatio float32 //数据文件merge操作的无效数据merge比例
	keyNumThreshhod    int64   //内存索引中允许的最大key的数量
}
type IndexerType = int8

const (
	// BTree 索引
	BTree IndexerType = iota + 1

	// ART Adpative Radix Tree 自适应基数树索引
	ART
	// BPlusTree B+树索引
	BPlusTree
)

// IteratorOptions 索引迭代器配置项
type IteratorOptions struct {
	// 遍历前缀为指定值的 Key，默认为空
	Prefix []byte
	// 是否反向遍历，默认 false 是正向
	Reverse bool
}
type WriteBatchOptions struct {
	MaxBatchSize int  //最大批量写入数量
	SyncWrites   bool //是否每次写入都进行持久化
}

var DefaultOptions = Options{
	DirPath:               os.TempDir(),
	DataFileSizeThreshold: 256 * 1024 * 1024, // 256MB
	SyncWrites:            false,
	IndexType:             BTree,
	BytesPerSync:          0, //默认不启用每隔多少字节同步写入
	DataFileMergeRatio:    0.5,
	keyNumThreshhod:       1000000, //默认最多允许100万个key(测试用)
}
var DefaultIteratorOptions = IteratorOptions{
	Prefix:  nil,
	Reverse: false,
}
var DefaultWriteBatchOptions = WriteBatchOptions{
	MaxBatchSize: 10000,
	SyncWrites:   true,
}
