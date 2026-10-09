//go:build !linux

package memopt

import "fmt"

// GetContainerMemoryStat 在非 Linux 系统下的空实现
func GetContainerMemoryStat() (*ContainerMemoryStat, error) {
	return nil, fmt.Errorf("当前操作系统不支持 cgroup 账本统计")
}

// GetContainerFileCacheBytes 在非 Linux 系统下的空实现
func GetContainerFileCacheBytes() (uint64, error) {
	return 0, fmt.Errorf("当前操作系统不支持 cgroup 账本统计")
}
