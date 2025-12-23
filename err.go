package bitcaskgo

import "errors"

// 自定义error类型
var (
	ErrKeyIsEmpty             = errors.New("key is empty")
	ErrIndexUpdateFailed      = errors.New("index update failed")
	ErrKeyIsUnfound           = errors.New("key is unfound")
	ErrDataFileNotFound       = errors.New("data file not found")
	ErrDataDirectoryCorrupted = errors.New("the database directory maybe corrupted")
	ErrInvalidCRC             = errors.New("the log record crc is invalid")
	ErrExceedMaxBatchSize     = errors.New("the batch size exceeds the maximum limit")
	ErrMergeIsRunning		= errors.New("merge operation is already running")
)
