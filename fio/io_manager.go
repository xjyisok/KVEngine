package fio

type IOType = byte

const (
	//标准文件IO
	StandardIO IOType = iota
	//内存映射IO
	MMapIO
)

type IOManager interface {
	// 代表一个文件IO操作的接口
	//代表从文件的指定位置读写删除数据
	Read([]byte, int64) (int, error)
	Write([]byte) (int, error)
	Sync() error
	Close() error
	Size() (int64, error)
}

// 创建一个新的IO管理器，方便对接后续不同的IO实现
func NewIOManager(fileName string, ioType IOType) (IOManager, error) {
	switch ioType {
	case MMapIO:
		return NewMMapIOManager(fileName)
	case StandardIO:
		return newFileIOManager(fileName)
	default:
		panic("ioType unsupported")
	}
}
