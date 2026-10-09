//go:build !linux

package memopt

import "fmt"

// GetContainerFileCacheBytes 在非 Linux 系统下的空实现
func GetContainerFileCacheBytes() (uint64, error) {
	return 0, fmt.Errorf("当前操作系统不支持 cgroup 账本统计")
}
