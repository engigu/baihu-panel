package idgen

import (
	"github.com/rs/xid"
)

// GenerateID 生成一个新的 ID (使用 xid，20位字符)
func GenerateID() string {
	return xid.New().String()
}
