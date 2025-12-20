package bitcaskgo

import "errors"

// 自定义error类型
var (
	ErrKeyIsEmpty        = errors.New("key is empty")
	ErrIndexUpdateFailed = errors.New("index update failed")
	ErrKeyIsUnfound      = errors.New("key is unfound")
	ErrDataFileNotFound  = errors.New("data file not found")
)
