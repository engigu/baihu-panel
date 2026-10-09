package memopt

import (
	"strconv"
	"strings"
)

// parseCgroupFileCache 解析 cgroup memory.stat 内容获取真实可释放的文件缓存字节数
// 策略：优先读取 inactive_file（未激活可安全回收页）；若无则取 file 扣除被只读映射的代码段 (file - file_mapped)，对齐 Docker stats 标准
func parseCgroupFileCache(content string) (uint64, bool) {
	lines := strings.Split(content, "\n")
	metrics := make(map[string]uint64, 32)

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}
		metrics[parts[0]] = val
	}

	// 1. 优先读取 cgroup v2 指标
	if inactive, ok := metrics["inactive_file"]; ok {
		return inactive, true
	}
	if file, ok := metrics["file"]; ok {
		if mapped, okMapped := metrics["file_mapped"]; okMapped && file >= mapped {
			return file - mapped, true
		}
		return file, true
	}

	// 2. cgroup v1 兼容逻辑
	if totalInactive, ok := metrics["total_inactive_file"]; ok {
		return totalInactive, true
	}
	if totalCache, ok := metrics["total_cache"]; ok {
		if mapped, okMapped := metrics["total_mapped_file"]; okMapped && totalCache >= mapped {
			return totalCache - mapped, true
		}
		return totalCache, true
	}
	if cache, ok := metrics["cache"]; ok {
		if mapped, okMapped := metrics["mapped_file"]; okMapped && cache >= mapped {
			return cache - mapped, true
		}
		return cache, true
	}

	return 0, false
}
