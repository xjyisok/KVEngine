package data

// 代表一个日志文件中的一条记录在文件中的位置
type LogRecordPos struct {
	Fid    uint32 //描述key指向的数据所在的文件id
	Offset int64  //描述key指向的数据在文件中的偏移量
	Size   uint32 //描述key指向的数据的大小
}
