package bitcaskgo

type Options struct {
	DirPath               string //数据文件存储路径
	DataFileSizeThreshold uint32 //数据文件大小阈值
	SyncWrites            bool   //是否每次写入都进行持久化
}
