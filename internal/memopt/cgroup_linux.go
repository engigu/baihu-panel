//go:build linux

package memopt

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
)

// GetContainerMemoryStat 完整采集容器内存指标（严格对齐 Docker CLI 标准：docker_used = total_usage - inactive_file）
func GetContainerMemoryStat() (*ContainerMemoryStat, error) {
	// 1. 优先尝试 cgroup v2
	if stat, err := readCgroupV2MemoryStat(); err == nil {
		return stat, nil
	}

	// 2. 回退尝试 cgroup v1
	if stat, err := readCgroupV1MemoryStat(); err == nil {
		return stat, nil
	}

	return nil, fmt.Errorf("未探测到容器 cgroup 内存账本")
}

func readCgroupV2MemoryStat() (*ContainerMemoryStat, error) {
	statData, err := os.ReadFile(constant.CgroupV2MemoryStatPath)
	if err != nil {
		return nil, err
	}

	totalUsage := readUint64File(constant.CgroupV2MemoryCurrentPath)
	limit := readUint64File(constant.CgroupV2MemoryMaxPath)

	dockerUsed, inactive, _ := CalculateDockerMemory(totalUsage, string(statData))

	return &ContainerMemoryStat{
		TotalUsageBytes: totalUsage,
		DockerUsedBytes: dockerUsed,
		InactiveFile:    inactive,
		LimitBytes:      limit,
	}, nil
}

func readCgroupV1MemoryStat() (*ContainerMemoryStat, error) {
	statData, err := os.ReadFile(constant.CgroupV1MemoryStatPath)
	if err != nil {
		return nil, err
	}

	totalUsage := readUint64File(constant.CgroupV1MemoryUsagePath)
	limit := readUint64File(constant.CgroupV1MemoryLimitPath)

	dockerUsed, inactive, _ := CalculateDockerMemory(totalUsage, string(statData))

	return &ContainerMemoryStat{
		TotalUsageBytes: totalUsage,
		DockerUsedBytes: dockerUsed,
		InactiveFile:    inactive,
		LimitBytes:      limit,
	}, nil
}

func readUint64File(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(data))
	if s == "" || s == "max" {
		return 0
	}
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// GetContainerFileCacheBytes 读取当前容器 cgroup 记录的文件缓存 (Page Cache) 占用大小（字节）
func GetContainerFileCacheBytes() (uint64, error) {
	stat, err := GetContainerMemoryStat()
	if err != nil {
		return 0, err
	}
	return stat.InactiveFile, nil
}
