package utils

import (
	"github.com/rs/xid"
)

// GenerateID 生成一个新的 ID (使用 xid，20位字符)
func GenerateID() string {
	return xid.New().String()
}
// IsNumeric 检查字符串是否全为数字
func IsNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}
