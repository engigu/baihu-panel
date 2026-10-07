package memopt

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/shirou/gopsutil/v3/process"
)

/*
var (
	daemonOnce sync.Once
)
*/

// Init 运行时自适应初始化
// 统一设置垃圾回收阈值与软内存上限，保障低配机器与容器下的平稳运行
func Init() {
	// 1. 设置 GOGC 比例，未指定时默认设为 80（较标准的 100 更激进回收，显著降低常驻内存）
	if gogc := os.Getenv("GOGC"); gogc == "" {
		debug.SetGCPercent(80)
	}

	// 2. 检查并设置软内存上限 (Go 1.19+ GOMEMLIMIT)
	if memLimit := os.Getenv("BH_MEM_LIMIT"); memLimit != "" {
		if limit, err := strconv.ParseInt(memLimit, 10, 64); err == nil && limit > 0 {
			debug.SetMemoryLimit(limit)
			logger.Infof("[MemOpt] 已加载自定义内存上限: %d 字节", limit)
		}
	}

	logger.Infof("[MemOpt] 运行时内存优化中心已激活 (CPU: %d, 初始协程: %d)", runtime.NumCPU(), runtime.NumGoroutine())
}

// Free 统一跨平台物理内存归还
// 结合多轮标记清理、空闲物理页退还与操作系统底层工作集裁减
func Free() {
	// 1. 第一轮 GC 标记并清理不可达对象
	runtime.GC()
	// 2. 将空闲堆内存归还给操作系统 (Decommit)
	debug.FreeOSMemory()
	// 3. 针对不同操作系统调用平台专属机制（Windows工作集裁减 / Linux二次强制退还）
	trimPlatformWorkingSet()
}

// Checkpoint 分阶段启动内存检查点
// 用于在数据库迁移、大批量任务扫描等重量级操作完成后，切断瞬时堆对象并释放物理页，防止高水位叠加
func Checkpoint(phase string) {
	beforeRSS := GetRSS()
	Free()
	afterRSS := GetRSS()

	if beforeRSS > 0 && afterRSS > 0 && beforeRSS > afterRSS {
		diffMB := float64(beforeRSS-afterRSS) / 1024 / 1024
		logger.Infof("[MemOpt] 阶段检查点 [%s] 触发内存回收，释放物理常驻内存约: %.2f MB", phase, diffMB)
	} else {
		logger.Infof("[MemOpt] 阶段检查点 [%s] 触发内存回收已完成", phase)
	}
}

/*
// StartDaemon 启动静默期后台内存守护协程（暂时保留注释）
// 在系统长时间待机或定时任务间歇期，周期性轻量收缩物理常驻内存，防止长时间挂机出现内存漂移
func StartDaemon(interval time.Duration) {
	daemonOnce.Do(func() {
		if interval <= 0 {
			interval = 15 * time.Minute
		}
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				Free()
			}
		}()
		logger.Infof("[MemOpt] 静默期后台内存守护协程已就绪 (巡检周期: %v)", interval)
	})
}
*/

// Metrics 进程物理与运行时内存指标
type Metrics struct {
	RSS          uint64 `json:"rss"`           // 真实物理常驻内存字节数
	NumGoroutine int    `json:"num_goroutine"` // 活跃协程数
	HeapAlloc    uint64 `json:"heap_alloc"`    // 当前堆分配字节数
	HeapInuse    uint64 `json:"heap_inuse"`    // 正在使用的堆内存字节数
	HeapSys      uint64 `json:"heap_sys"`      // 向系统申请的堆总空间
}

// GetMetrics 统一采集当前进程的内存与运行时指标
func GetMetrics() Metrics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return Metrics{
		RSS:          GetRSS(),
		NumGoroutine: runtime.NumGoroutine(),
		HeapAlloc:    ms.HeapAlloc,
		HeapInuse:    ms.HeapInuse,
		HeapSys:      ms.HeapSys,
	}
}

// GetRSS 获取当前进程真实物理常驻内存 (Resident Set Size)
func GetRSS() uint64 {
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if memInfo, err := p.MemoryInfo(); err == nil {
			return memInfo.RSS
		}
	}
	return 0
}

// FormatBytes 工具辅助函数：人性化字节格式化
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

/*
// TuneDB 数据库连接池自动瘦身配置（根据用户要求暂时注释保留）
// 用于控制空闲连接常驻与连接生命周期，避免 GORM 启动膨胀连接长期占用内存
func TuneDB(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil || sqlDB == nil {
		return
	}
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxIdleTime(30 * time.Second)
	sqlDB.SetMaxOpenConns(10)
}
*/
