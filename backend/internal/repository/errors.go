package repository

import "errors"

// 仓储层哨兵错误，上层使用 errors.Is 判断。
var (
	ErrNotFound      = errors.New("record not found")
	ErrDuplicateKey  = errors.New("duplicate key")
)
