package utils

import (
	"os"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/engigu/baihu-panel/internal/logger"
)

// InitRuntime 设置运行时内存和性能优化参数
func InitRuntime() {
	// 从环境变量或配置读取 GOGC
	if gogc := os.Getenv("GOGC"); gogc == "" {
		// 默认设为 80，比 100 更激进一点，减少 RSS 峰值
		debug.SetGCPercent(80)
	}

	// 从环境变量读取 GOMEMLIMIT (Go 1.19+)
	// 建议用户在系统层面设置，例如 BH_MEM_LIMIT=256MiB
	if memLimit := os.Getenv("BH_MEM_LIMIT"); memLimit != "" {
		// 这里可以解析一下单位，但简单处理可以直接读取字节
		if limit, err := strconv.ParseInt(memLimit, 10, 64); err == nil {
			debug.SetMemoryLimit(limit)
			logger.Infof("[Runtime] 已设置内存上限: %d 字节", limit)
		}
	}

	// 打印当前运行时信息
	logger.Infof("[Runtime] CPU 核心数: %d, Goroutine 数量: %d", runtime.NumCPU(), runtime.NumGoroutine())
}

// FreeMemory 显式触发内存回收，释放物理资源给 OS
// 仅建议在执行了超大批量任务或处理了大型文件后调用
func FreeMemory() {
	// 1. 触发 GC 标记并清理堆不可达对象
	runtime.GC()
	// 2. 尽可能将虚拟内存归还 OS (Decommit)
	debug.FreeOSMemory()
	// 3. 针对 OS (如 Windows) 主动收缩物理常驻内存工作集 (RSS)
	trimWorkingSet()
}
