package memopt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
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
		_, _ = DropCache(constant.ResolveMiseDataDir())
	}()
}

var isTrimming atomic.Bool

// filterValidTargets 过滤出真实存在的目录路径并进行路径去重
func filterValidTargets(targets []string) []string {
	seen := make(map[string]bool)
	validTargets := make([]string, 0, len(targets))
	for _, p := range targets {
		if p == "" {
			continue
		}
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

// GetContainerTrimTargets 统一集中获取容器常驻运行期需重点回收 Page Cache 的目标清单
// 将重点单体 CLI、多语言环境、基础底座、工作区与任务目录集中收敛定义，杜绝分散割裂
func GetContainerTrimTargets(extraDirs ...string) []string {
	targets := make([]string, 0, 10+len(extraDirs))

	// 1. 显式传入的任务工作目录
	for _, d := range extraDirs {
		if d != "" {
			targets = append(targets, d)
		}
	}

	// 2. 重点单体 CLI 二进制文件：mise 与白虎主程序等（单体120MB+），运行任务后常驻 Active File
	if utils.IsRunningInDocker() {
		targets = append(targets, "/usr/local/bin/mise", "/app/baihu")
	}

	// 3. 用户环境目录（node、dotnet、python等产物，由 walkDirLimited 自动遍历其子目录）
	if miseDataDir := constant.ResolveMiseDataDir(); miseDataDir != "" {
		targets = append(targets, miseDataDir)
	}

	// 4. 容器预置基础语言底座（针对启动期与基础镜像环境）
	if constant.ContainerMiseBaseDir != "" {
		targets = append(targets, constant.ContainerMiseBaseDir)
	}

	// 5. 任务脚本工作目录（所有应用与脚本仓库存放地）
	if constant.ScriptsWorkDir != "" {
		targets = append(targets, constant.ScriptsWorkDir)
	}

	// 6. 系统与任务执行日志目录（高频磁盘落盘产生 Page Cache）
	if constant.LogsDir != "" {
		targets = append(targets, constant.LogsDir)
	}

	// 7. 支持通过环境变量附加扩展目录（以逗号、分号或冒号分隔）
	if envExtra := os.Getenv(constant.EnvKeyCacheTrimDirs); envExtra != "" {
		for _, dir := range strings.FieldsFunc(envExtra, func(r rune) bool {
			return r == ',' || r == ';' || r == ':'
		}) {
			dir = strings.TrimSpace(dir)
			if dir != "" {
				targets = append(targets, dir)
			}
		}
	}

	return filterValidTargets(targets)
}

// TrimContainerCache 立即同步执行一次动态工作区 Page Cache 与 Go 运行时堆内存回收
// 返回本次释放的文件总数量
func TrimContainerCache() int {
	if !utils.IsRunningInDocker() {
		return 0
	}

	targets := GetContainerTrimTargets()
	if len(targets) == 0 {
		return 0
	}

	// 1. 同步遍历高频动态目录，调用 posix_fadvise 卸载已读入内存的文件缓存
	totalFiles, err := DropCache(targets...)
	if err != nil {
		logger.Debugf("[MemOpt] 释放容器文件缓存出现提示: %v", err)
	}

	// 2. 联动触发运行时堆内存收缩与物理页退还
	Free()

	return totalFiles
}

// TrimContainerStartupCache 系统启动就绪时的一次性全量回收
// 包含一次性初始化产生的只读基础底座 (/opt/mise-base) 以及全量工作区
func TrimContainerStartupCache() int {
	if !utils.IsRunningInDocker() {
		return 0
	}

	targets := GetContainerTrimTargets()
	if len(targets) == 0 {
		return 0
	}

	totalFiles, err := DropCache(targets...)
	if err != nil {
		logger.Debugf("[MemOpt] 释放启动文件缓存出现提示: %v", err)
	}

	Free()
	return totalFiles
}

// TryReclaimCgroupMemory 尝试通过 cgroup v2 memory.reclaim 请求内核主动回收可回收内存（含 Slab 与 PageCache）
// 若当前容器具备该文件写权限，内核将以毫秒级速度直接完成回收，无需应用层递归遍历文件系统
func TryReclaimCgroupMemory(reclaimBytes uint64) bool {
	if !utils.IsRunningInDocker() {
		return false
	}
	if reclaimBytes == 0 {
		reclaimBytes = 50 * 1024 * 1024
	}
	payload := []byte(strconv.FormatUint(reclaimBytes, 10))
	err := os.WriteFile(constant.CgroupV2MemoryReclaimPath, payload, 0644)
	if err == nil {
		return true
	}
	// 在 Linux 内核 cgroup v2 规范中，若实际回收的字节数小于请求量，内核仍会尽最大努力回收，但系统调用会返回 -EAGAIN。
	// 针对此类情况，内核已实际完成部分回收，视为生效，避免误判降级产生无效的磁盘遍历。
	if errors.Is(err, syscall.EAGAIN) {
		return true
	}
	return false
}

// AutoTrimContainerCache 智能检测并自适应释放容器内的 Page Cache
// 规则：
// 1. 仅在 Docker 容器环境中执行；
// 2. 原子防重入保护：避免上一轮耗时较长时造成定时任务 IO 堆叠；
// 3. 高水位线自适应：配置了内存限额时，仅在总内存达到安全警戒线（默认 80%）且存在大量 Page Cache 时介入；
//    平时充裕时静默放行，让 Page Cache 充分发挥磁盘加速作用，避免无谓的磁盘扫描开销；
// 4. 清理时优先使用 cgroup memory.reclaim 原生回收（一并卸载 Slab 与 PageCache），不可用时降级为动态目录 DropCache。
func AutoTrimContainerCache() bool {
	if !utils.IsRunningInDocker() {
		return false
	}

	cfg := GetConfig()
	if !cfg.Enabled {
		return false
	}

	// 防重入原子控制
	if !isTrimming.CompareAndSwap(false, true) {
		logger.Debugf("[MemOpt] 上一次容器缓存回收任务仍在进行中，跳过本次触发")
		return false
	}
	defer isTrimming.Store(false)

	stat, err := GetContainerMemoryStat()
	if err != nil {
		logger.Debugf("[MemOpt] 未能获取 cgroup 内存指标 (%v)，跳过本次自适应回收", err)
		return false
	}

	thresholdMB := cfg.ThresholdMB
	thresholdBytes := uint64(thresholdMB) * 1024 * 1024
	watermarkRate := cfg.WatermarkRate

	// 智能判定策略：
	// 若配置了容器内存限额 (LimitBytes > 0)：
	// 只要出现以下任一情况，即启动自适应回收：
	//   1. 达到高警戒水位线 (默认 80%): 容器总物理内存或 Docker 真实占用 >= watermarkRate，且存在可回收资源 (hasReclaimable)；
	//   2. 活跃缓存重型任务半载防护: 总文件缓存 (TotalFileCache = active + inactive) 达到阈值 (>= thresholdBytes)，
	//      且系统整体负载已过半 (totalUsageRate >= 0.60)，避免像 .NET/Python/Node 任务产生上百兆 active_file 时漏判。
	// 若未配置限额 (LimitBytes == 0)：
	//   只要总文件缓存 TotalFileCache >= thresholdBytes 即触发
	shouldTrim := false
	if stat.LimitBytes > 0 {
		totalUsageRate := float64(stat.TotalUsageBytes) / float64(stat.LimitBytes)
		dockerUsageRate := float64(stat.DockerUsedBytes) / float64(stat.LimitBytes)
		hasReclaimable := stat.TotalFileCache >= thresholdBytes || stat.SlabReclaimable >= thresholdBytes

		isHighWatermark := hasReclaimable && (totalUsageRate >= watermarkRate || dockerUsageRate >= watermarkRate)
		isCacheHeavy := stat.TotalFileCache >= thresholdBytes && totalUsageRate >= 0.60

		if isHighWatermark || isCacheHeavy {
			shouldTrim = true
		}
	} else if stat.TotalFileCache >= thresholdBytes {
		shouldTrim = true
	}

	if !shouldTrim {
		return false
	}

	// 格式化输出容器内存状况，让 Docker Stats 真实占用与 PageCache 缓存清晰透明对齐
	usedStr := FormatBytes(stat.DockerUsedBytes)
	cacheStr := FormatBytes(stat.TotalFileCache)
	if stat.LimitBytes > 0 {
		limitStr := FormatBytes(stat.LimitBytes)
		pct := float64(stat.DockerUsedBytes) / float64(stat.LimitBytes) * 100
		totalPct := float64(stat.TotalUsageBytes) / float64(stat.LimitBytes) * 100
		logger.Infof("[MemOpt] 容器内存达到警戒水位线 (当前总占用: %.1f%%, 总文件缓存: %s [活跃: %s, 非活跃: %s], Docker真实占用: %s / %s [%.1f%%])，启动自适应回收",
			totalPct, cacheStr, FormatBytes(stat.ActiveFile), FormatBytes(stat.InactiveFile), usedStr, limitStr, pct)
	} else {
		logger.Infof("[MemOpt] 容器 PageCache 达到水位线 (当前总缓存: %s, 阈值: %d MB, Docker真实占用: %s)，启动自适应回收",
			cacheStr, thresholdMB, usedStr)
	}

	// 优先尝试 cgroup v2 memory.reclaim 原生回收（毫秒级极速，且可回收 Slab 目录项）
	if TryReclaimCgroupMemory(thresholdBytes) {
		Free()
		logger.Infof("[MemOpt] 容器自适应回收完成 (内核 cgroup 原生释放 Slab 与缓存，并归还物理内存)")
		return true
	}

	// 降级回退方案：精准同步遍历动态工作区目录回收并联动堆退还
	filesTrimmed := TrimContainerCache()
	if filesTrimmed > 0 {
		logger.Infof("[MemOpt] 容器自适应回收完成 (已释放 %d 个文件缓存，并归还物理内存)", filesTrimmed)
	} else {
		logger.Debugf("[MemOpt] 容器自适应巡检完成 (无待释放文件缓存)")
	}

	return true
}

// DirDebouncer 目录任务结束清理防抖器（Trailing Edge 尾部合并防抖）
// 用于处理同目录下多任务高频/连续结束场景，消除重复遍历开销，由最后结束的任务在静默期触发唯一一次收尾
type DirDebouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
}

// NewDirDebouncer 创建新的目录防抖器
func NewDirDebouncer() *DirDebouncer {
	return &DirDebouncer{
		timers: make(map[string]*time.Timer),
	}
}

var defaultDirDebouncer = NewDirDebouncer()

// Cancel 显式取消指定目录正在挂起的防抖任务
func (d *DirDebouncer) Cancel(dir string) {
	if dir == "" {
		return
	}
	cleanDir := filepath.Clean(dir)

	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, ok := d.timers[cleanDir]; ok {
		timer.Stop()
		delete(d.timers, cleanDir)
	}
}

// Debounce 在指定静默延迟后执行清理；若在延迟期内有新任务结束，自动顺延重置计时器
func (d *DirDebouncer) Debounce(dir string, delay time.Duration, fn func(dir string)) {
	if dir == "" {
		return
	}
	cleanDir := filepath.Clean(dir)

	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. 若该目录已有挂起的计时器，直接取消并顺延重置
	if timer, ok := d.timers[cleanDir]; ok {
		timer.Stop()
	}

	// 2. 注册延迟执行：静默期到达后由最后一个任务执行，并自动从 map 中释放计时器句柄
	var timer *time.Timer
	timer = time.AfterFunc(delay, func() {
		// 隔离协程 Panic，防止底层异常拖垮主进程
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("[MemOpt] 目录清理防抖回调发生异常: %v", r)
			}
		}()

		d.mu.Lock()
		if cur, ok := d.timers[cleanDir]; ok && cur == timer {
			delete(d.timers, cleanDir)
			d.mu.Unlock()
			fn(cleanDir)
			return
		}
		d.mu.Unlock()
	})
	d.timers[cleanDir] = timer
}

// OnTaskFinished 任务执行完成联动回收
// 针对大任务（耗时较长或处理大量依赖/产物文件）结束后的轻量级异步回收
// 采用尾部防抖（Trailing Debounce）：同目录多任务密集接连结束时自动顺延合并，在最后一个任务平息后执行唯一一次彻底清理
func OnTaskFinished(workDir string) {
	if !utils.IsRunningInDocker() {
		return
	}

	cfg := GetConfig()
	if !cfg.Enabled || !cfg.TaskFinishedTrim {
		return
	}

	// 若未指定具体工作目录，轻量延迟后直接检测容器内存水位
	if workDir == "" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			AutoTrimContainerCache()
		}()
		return
	}

	// 尾部防抖窗口设为 1.5 秒：多个密集子任务接连结束时自动顺延，在最后一个任务完全平息后执行收尾
	defaultDirDebouncer.Debounce(workDir, 1500*time.Millisecond, func(dir string) {
		_, _ = DropCache(GetContainerTrimTargets(dir)...)
		AutoTrimContainerCache()
	})
}

// RegisterTaskCompletionListener 注册全局事件总线监听（仅在 Docker 容器环境下激活）
// 监听任务成功/失败/超时/取消等全部结束事件，解耦触发工作区 Page Cache 与高水位内存异步自适应回收
func RegisterTaskCompletionListener() {
	if !utils.IsRunningInDocker() {
		return
	}

	handler := func(e eventbus.Event) {
		var workDir string
		if payload, ok := e.Payload.(map[string]interface{}); ok {
			if wd, okStr := payload["work_dir"].(string); okStr {
				workDir = wd
			}
		}
		OnTaskFinished(workDir)
	}

	eventbus.DefaultBus.Subscribe(constant.EventTaskSuccess, handler)
	eventbus.DefaultBus.Subscribe(constant.EventTaskFailed, handler)
	eventbus.DefaultBus.Subscribe(constant.EventTaskTimeout, handler)
	eventbus.DefaultBus.Subscribe(constant.EventTaskCancelled, handler)

	logger.Infof("[MemOpt] 已挂载 EventBus 任务结束事件监听器 (工作区 PageCache 异步自适应回收)")
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
