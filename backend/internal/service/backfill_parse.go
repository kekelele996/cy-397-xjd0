package service

import (
	"errors"
	"strconv"
	"strings"
)

// errInvalidContractNo 表示描述里的合同编号无法对应合同库的数字主键。
var errInvalidContractNo = errors.New("invalid contract no")

// parseContractNo 把描述中提取出的编号归一成合同库主键。
// 合同库主键为自增数字，因此只接受纯数字（允许 HT-123 / C123 这类带字母前缀的写法，取尾部数字段）。
func parseContractID(raw string) (uint64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, errInvalidContractNo
	}
	// 形如 HT-2026-123 / C123：取最后一段连续数字。
	if !isDigits(s) {
		idx := strings.LastIndexAny(s, "-_")
		tail := s
		if idx >= 0 {
			tail = s[idx+1:]
		} else {
			tail = digitsSuffix(s)
		}
		s = tail
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil || id == 0 {
		return 0, errInvalidContractNo
	}
	return id, nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// digitsSuffix 取字符串尾部连续数字（如 C123 -> 123）。
func digitsSuffix(s string) string {
	i := len(s)
	for i > 0 {
		c := s[i-1]
		if c < '0' || c > '9' {
			break
		}
		i--
	}
	return s[i:]
}
