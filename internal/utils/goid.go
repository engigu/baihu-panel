package utils

import (
	"runtime"
	"strconv"
	"strings"
)

// GetGoroutineID 获取当前 Goroutine ID
// 注意：这只是为了调试和日志目的，不应该用于业务逻辑
func GetGoroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.ParseInt(idField, 10, 64)
	if err != nil {
		return -1
	}
	return id
}
