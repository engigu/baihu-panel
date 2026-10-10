package memopt

import (
	"strconv"
	"strings"
)

// ContainerMemoryStat 容器 cgroup 结构化内存指标（对齐 Docker CLI 标准）
type ContainerMemoryStat struct {
	TotalUsageBytes uint64 // 物理总使用量 (memory.current / memory.usage_in_bytes)
	DockerUsedBytes uint64 // 对齐 Docker stats 口径: TotalUsageBytes - InactiveFile
	InactiveFile    uint64 // inactive_file (未激活文件页，Docker 视为随时可释放丢弃的缓存)
	ActiveFile      uint64 // active_file (活跃文件页，重型任务反复读取的类库/文件缓存)
	TotalFileCache  uint64 // 总文件页缓存 (file / cache，包含 active + inactive)
	SlabReclaimable uint64 // slab_reclaimable (可回收内核 Slab 目录项与 inode 缓存)
	LimitBytes      uint64 // 内存上限 (0 代表未限制)
}

// CalculateDockerMemory 计算对齐 Docker stats 标准的内存占用
// Docker 官方计算规则 (Docker CLI stats_helpers):
// docker_stats_used = total_usage - inactive_file
// 扣除内核中随时可无损丢弃的未激活文件缓存，得出容器真实的物理工作集内存占用
func CalculateDockerMemory(totalUsage uint64, statContent string) (dockerUsed uint64, inactiveFile uint64, ok bool) {
	inactive, hasInactive := parseCgroupInactiveFile(statContent)
	if !hasInactive {
		return totalUsage, 0, false
	}
	if totalUsage >= inactive {
		return totalUsage - inactive, inactive, true
	}
	return 0, inactive, true
}

// ParseCgroupMemoryDetails 完整解析 cgroup memory.stat 内容获取结构化指标
func ParseCgroupMemoryDetails(totalUsage uint64, content string) ContainerMemoryStat {
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

	stat := ContainerMemoryStat{
		TotalUsageBytes: totalUsage,
	}

	// 1. InactiveFile (用于 Docker stats 标准相减计算)
	if v, ok := metrics["inactive_file"]; ok {
		stat.InactiveFile = v
	} else if v, ok := metrics["total_inactive_file"]; ok {
		stat.InactiveFile = v
	}

	// 2. ActiveFile
	if v, ok := metrics["active_file"]; ok {
		stat.ActiveFile = v
	} else if v, ok := metrics["total_active_file"]; ok {
		stat.ActiveFile = v
	}

	// 3. TotalFileCache
	if v, ok := metrics["file"]; ok {
		stat.TotalFileCache = v
	} else if v, ok := metrics["total_cache"]; ok {
		stat.TotalFileCache = v
	} else if v, ok := metrics["cache"]; ok {
		stat.TotalFileCache = v
	} else {
		stat.TotalFileCache = stat.InactiveFile + stat.ActiveFile
	}

	// 4. SlabReclaimable
	if v, ok := metrics["slab_reclaimable"]; ok {
		stat.SlabReclaimable = v
	} else if v, ok := metrics["total_slab_reclaimable"]; ok {
		stat.SlabReclaimable = v
	}

	// 5. 对齐 Docker Stats 真实内存相减口径
	if stat.InactiveFile > 0 {
		if totalUsage >= stat.InactiveFile {
			stat.DockerUsedBytes = totalUsage - stat.InactiveFile
		} else {
			stat.DockerUsedBytes = 0
		}
	} else {
		stat.DockerUsedBytes = totalUsage
	}

	return stat
}

// parseCgroupInactiveFile 解析 cgroup memory.stat 内容获取 inactive_file（不活跃文件缓存）
func parseCgroupInactiveFile(content string) (uint64, bool) {
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
	if inactive, ok := metrics["inactive_file"]; ok {
		return inactive, true
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

// parseCgroupFileCache 解析 cgroup memory.stat 内容获取真实可释放的文件缓存字节数
func parseCgroupFileCache(content string) (uint64, bool) {
	return parseCgroupInactiveFile(content)
}
