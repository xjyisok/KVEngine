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

var DefaultOptions = Options{
	DirPath:               os.TempDir(),
	DataFileSizeThreshold: 256 * 1024 * 1024, // 256MB
	SyncWrites:            false,
	IndexType:             BTree,
}
