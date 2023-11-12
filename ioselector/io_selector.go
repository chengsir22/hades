package ioselector

const DataFilePerm = 0644

type FileIOType = byte

const (
	// StandardFIO 标准文件 IO
	StandardFIO FileIOType = iota
	// MemoryMap 内存文件映射
	MemoryMap
)

// IOSelector 抽象 IO 管理接口，可以接入不同的 IO 类型，目前支持标准文件 IO
type IOSelector interface {
	// Read 从文件的给定位置读取对应的数据
	Read([]byte, int64) (int, error)

	// Write 写入字节数组到文件中
	Write([]byte) (int, error)

	// Sync 持久化数据
	Sync() error

	// Close 关闭文件
	Close() error

	// Size 获取到文件大小
	Size() (int64, error)
}

// NewIOSelector 初始化 IOManager，目前只支持标准 FileIO
func NewIOSelector(fileName string, ioType FileIOType) (IOSelector, error) {
	switch ioType {
	case StandardFIO:
		return NewFileIOSelector(fileName)
	case MemoryMap:
		return NewMMapIOSelector(fileName)
	default:
		panic("unsupported io type")
	}
}
