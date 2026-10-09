package memopt

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/utils"
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

// DropMiseCacheAsync 异步延迟释放 mise 相关目录的文件缓存 (Page Cache)
// 仅在 Docker 容器环境中真正执行，避免占用容器内存限额
func DropMiseCacheAsync(delays ...time.Duration) {
	delay := 500 * time.Millisecond
	if len(delays) > 0 {
		delay = delays[0]
	}

	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		_, _ = DropCache(constant.ContainerMiseBaseDir, constant.ResolveMiseDataDir())
	}()
}

var isTrimming atomic.Bool

// getContainerTrimTargets 获取容器内需重点回收 Page Cache 的目标路径清单
// 覆盖：任务脚本工作目录、系统与任务日志目录、用户动态环境目录，以及用户自定义扩展目录（自动排除静态只读的系统底座）
func getContainerTrimTargets() []string {
	targets := make([]string, 0, 8)

	// 1. Mise 运行时基础底座与数据存储目录
	if constant.ContainerMiseBaseDir != "" {
		targets = append(targets, constant.ContainerMiseBaseDir)
	}
	if miseDataDir := constant.ResolveMiseDataDir(); miseDataDir != "" && miseDataDir != constant.ContainerMiseBaseDir {
		targets = append(targets, miseDataDir)
	}

	// 2. 任务脚本工作目录（大量定时任务代码与依赖）
	if constant.ScriptsWorkDir != "" {
		targets = append(targets, constant.ScriptsWorkDir)
	}

	// 3. 系统与任务执行日志目录（高频磁盘落盘产生 Page Cache）
	if constant.LogsDir != "" {
		targets = append(targets, constant.LogsDir)
	}

	// 4. 支持通过环境变量附加扩展目录（以逗号、分号或冒号分隔）
	if extraDirs := os.Getenv(constant.EnvKeyCacheTrimDirs); extraDirs != "" {
		for _, dir := range strings.FieldsFunc(extraDirs, func(r rune) bool {
			return r == ',' || r == ';' || r == ':'
		}) {
			dir = strings.TrimSpace(dir)
			if dir != "" {
				targets = append(targets, dir)
			}
		}
	}

	// 去重并过滤出真实存在的路径，避免无效扫描
	seen := make(map[string]bool)
	validTargets := make([]string, 0, len(targets))
	for _, p := range targets {
		cleanPath := filepath.Clean(p)
		if seen[cleanPath] {
			continue
		}
		seen[cleanPath] = true
		if _, err := os.Stat(cleanPath); err == nil {
			validTargets = append(validTargets, cleanPath)
		}
	}

	return validTargets
}

// TrimContainerCache 立即同步执行一次容器内 Page Cache 与 Go 运行时堆内存回收
// 返回本次释放的文件总数量
func TrimContainerCache() int {
	if !utils.IsRunningInDocker() {
		return 0
	}

	targets := getContainerTrimTargets()
	if len(targets) == 0 {
		return 0
	}

	// 1. 同步遍历高频目录，调用 posix_fadvise 卸载已读入内存的文件缓存
	totalFiles, err := DropCache(targets...)
	if err != nil {
		logger.Debugf("[MemOpt] 释放容器文件缓存出现提示: %v", err)
	}

	// 2. 联动触发运行时堆内存收缩与物理页退还
	Free()

	return totalFiles
}

// AutoTrimContainerCache 智能检测并自适应释放容器内的 Page Cache
// 规则：
// 1. 仅在 Docker 容器环境中执行；
// 2. 原子防重入保护：避免上一轮耗时较长时造成定时任务 IO 堆叠；
// 3. 动态读取 cgroup 文件缓存占用，若超过水位线（默认 60MB，可通过 BH_CACHE_MAX_MB 配置）才触发释放；
// 4. 低于水位线时静默跳过，无额外 CPU 或磁盘 IO 损耗；
// 5. 触发释放时，同步遍历脚本/日志等动态高频目录卸载 Page Cache，并联动归还 Go 堆内存给操作系统。
func AutoTrimContainerCache() bool {
	if !utils.IsRunningInDocker() {
		return false
	}

	// 防重入原子控制
	if !isTrimming.CompareAndSwap(false, true) {
		logger.Debugf("[MemOpt] 上一次容器缓存回收任务仍在进行中，跳过本次触发")
		return false
	}
	defer isTrimming.Store(false)

	thresholdMB := constant.DefaultCacheMaxMB
	if envVal := os.Getenv(constant.EnvKeyCacheMaxMB); envVal != "" {
		if v, err := strconv.Atoi(envVal); err == nil && v > 0 {
			thresholdMB = v
		}
	}
	thresholdBytes := uint64(thresholdMB) * 1024 * 1024

	cacheBytes, err := GetContainerFileCacheBytes()
	if err != nil {
		// 未能精准读取 cgroup 时（如部分老旧宿主机或受限无权读取），直接按 debug 记录并跳过，避免盲目频繁报警
		logger.Debugf("[MemOpt] 未能获取 cgroup 缓存指标 (%v)，跳过本次自适应回收", err)
		return false
	}

	if cacheBytes < thresholdBytes {
		// 未达到水位线，保持现状，不进行多余操作
		return false
	}

	logger.Infof("[MemOpt] 容器 PageCache 达到智能水位线 (当前: %s, 阈值: %d MB)，启动自适应回收", FormatBytes(cacheBytes), thresholdMB)

	// 同步执行目标目录回收与运行时内存退还
	filesTrimmed := TrimContainerCache()
	if filesTrimmed > 0 {
		logger.Infof("[MemOpt] 容器自适应回收完成，清理了 %d 个文件的 Page Cache 并归还物理内存", filesTrimmed)
	} else {
		logger.Debugf("[MemOpt] 容器自适应回收完成，动态工作区无待释放文件缓存")
	}

	return true
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
