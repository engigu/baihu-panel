package memopt

import (
	"strconv"
	"strings"
)

// parseCgroupFileCache 解析 cgroup memory.stat 内容获取文件缓存字节数
func parseCgroupFileCache(content string) (uint64, bool) {
	lines := strings.Split(content, "\n")
	var fileBytes, totalCacheBytes, cacheBytes uint64
	hasFile := false
	hasTotalCache := false
	hasCache := false

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		k := parts[0]
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}

		switch k {
		case "file": // cgroup v2: 纯文件 Page Cache
			fileBytes = val
			hasFile = true
		case "total_cache": // cgroup v1: 包含子 cgroup 的全局 cache
			totalCacheBytes = val
			hasTotalCache = true
		case "cache": // cgroup v1: 本 cgroup cache
			cacheBytes = val
			hasCache = true
		}
	}

	if hasFile {
		return fileBytes, true
	}
	if hasTotalCache {
		return totalCacheBytes, true
	}
	if hasCache {
		return cacheBytes, true
	}

	return 0, false
}
