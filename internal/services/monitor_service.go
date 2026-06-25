package services

import (
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type HostMetrics struct {
	CPUPercent float64
	VMem       *mem.VirtualMemoryStat
	DiskUsage  *disk.UsageStat
	HostInfo   *host.InfoStat
}

type MonitorService struct {
	hostMu     sync.RWMutex
	lastUpdate time.Time
	metrics    HostMetrics
}

var (
	monitorServiceInstance *MonitorService
	monitorServiceOnce     sync.Once
)

// GetMonitorService 获取系统监控服务单例
func GetMonitorService() *MonitorService {
	monitorServiceOnce.Do(func() {
		monitorServiceInstance = &MonitorService{}
	})
	return monitorServiceInstance
}

// GetHostMetrics 获取并返回物理机状态（带有缓存和演示模式伪装）
func (ms *MonitorService) GetHostMetrics() HostMetrics {
	ms.hostMu.Lock()
	defer ms.hostMu.Unlock()

	// 缓存 2 秒
	if time.Since(ms.lastUpdate) < 2*time.Second && ms.metrics.VMem != nil {
		return ms.metrics
	}

	if constant.DemoMode {
		ms.updateDemoMetrics()
		return ms.metrics
	}

	cpuPercents, _ := cpu.Percent(0, false)
	if len(cpuPercents) > 0 {
		ms.metrics.CPUPercent = cpuPercents[0]
	}
	ms.metrics.VMem, _ = mem.VirtualMemory()
	ms.metrics.DiskUsage, _ = disk.Usage("/")
	ms.metrics.HostInfo, _ = host.Info()
	ms.lastUpdate = time.Now()

	// 提供默认值防空指针
	if ms.metrics.VMem == nil {
		ms.metrics.VMem = &mem.VirtualMemoryStat{}
	}
	if ms.metrics.DiskUsage == nil {
		ms.metrics.DiskUsage = &disk.UsageStat{}
	}
	if ms.metrics.HostInfo == nil {
		ms.metrics.HostInfo = &host.InfoStat{}
	}

	return ms.metrics
}

type RuntimeMetrics struct {
	NumGoroutine int
	MemStats     runtime.MemStats
}

var (
	runtimeMu     sync.RWMutex
	lastRuntime   time.Time
	cachedRuntime RuntimeMetrics
)

// GetRuntimeMetrics 获取 Go 运行时指标（缓存 2 秒，防止高并发下频繁触发 STW）
func (ms *MonitorService) GetRuntimeMetrics() RuntimeMetrics {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()

	if time.Since(lastRuntime) < 2*time.Second && cachedRuntime.NumGoroutine > 0 {
		return cachedRuntime
	}

	cachedRuntime.NumGoroutine = runtime.NumGoroutine()
	runtime.ReadMemStats(&cachedRuntime.MemStats)
	lastRuntime = time.Now()

	return cachedRuntime
}

func (ms *MonitorService) updateDemoMetrics() {
	ms.metrics.CPUPercent = 10 + rand.Float64()*40 // 10% - 50% 的随机 CPU 波动

	totalMem := uint64(8 * 1024 * 1024 * 1024) // 8GB 内存
	usedMem := uint64(float64(totalMem) * (0.3 + rand.Float64()*0.3)) // 30% - 60% 随机使用率
	ms.metrics.VMem = &mem.VirtualMemoryStat{
		Total:       totalMem,
		Used:        usedMem,
		UsedPercent: float64(usedMem) / float64(totalMem) * 100,
	}

	totalDisk := uint64(500 * 1024 * 1024 * 1024) // 500GB 硬盘
	usedDisk := uint64(float64(totalDisk) * 0.45) // 固定 45% 使用率
	ms.metrics.DiskUsage = &disk.UsageStat{
		Total:       totalDisk,
		Used:        usedDisk,
		UsedPercent: float64(usedDisk) / float64(totalDisk) * 100,
	}

	ms.metrics.HostInfo = &host.InfoStat{
		Platform: "Demo Environment",
		OS:       "linux",
		Uptime:   uint64(time.Now().Unix() - 1700000000), // 生成一个较长且持续增加的运行时间
	}
	ms.lastUpdate = time.Now()
}
