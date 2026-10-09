//go:build linux

package memopt

import (
	"fmt"
	"os"

	"github.com/engigu/baihu-panel/internal/constant"
)

// GetContainerFileCacheBytes 读取当前容器 cgroup 记录的文件缓存 (Page Cache) 占用大小（字节）
// 优先探测 cgroup v2 (constant.CgroupV2MemoryStatPath)，其次探测 cgroup v1 (constant.CgroupV1MemoryStatPath)
func GetContainerFileCacheBytes() (uint64, error) {
	paths := []string{
		constant.CgroupV2MemoryStatPath,
		constant.CgroupV1MemoryStatPath,
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		if val, ok := parseCgroupFileCache(string(data)); ok {
			return val, nil
		}
	}

	return 0, fmt.Errorf("未探测到容器 cgroup 内存账本")
}
