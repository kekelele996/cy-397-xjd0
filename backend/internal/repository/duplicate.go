package repository

import "strings"

// isDuplicate 判断 MySQL 错误是否为主键/唯一键冲突。
func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "1062")
}
