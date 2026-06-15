package controllers

import (
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type MonitorController struct {
	executorService *tasks.ExecutorService

	// 缓存物理机状态
	hostMu     sync.RWMutex
	lastUpdate time.Time
	cpuPercent float64
	vMem       *mem.VirtualMemoryStat
	diskUsage  *disk.UsageStat
	hostInfo   *host.InfoStat
}

func NewMonitorController(executorService *tasks.ExecutorService) *MonitorController {
	return &MonitorController{
		executorService: executorService,
	}
}

var monitorUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有跨域，生产环境可根据配置限制
	},
}

// GetSystemMonitor 获取系统和内存监控信息 (HTTP)
func (mc *MonitorController) GetSystemMonitor(c *gin.Context) {
	data := mc.getMonitorData()
	utils.Success(c, data)
}

// MonitorWS WebSocket 获取系统监控数据
func (mc *MonitorController) MonitorWS(c *gin.Context) {
	ws, err := monitorUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// 初始发送一次数据
	if err := mc.sendMonitorData(ws); err != nil {
		return
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := mc.sendMonitorData(ws); err != nil {
				return // 客户端断开连接或发送失败
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}

func (mc *MonitorController) sendMonitorData(ws *websocket.Conn) error {
	data := mc.getMonitorData()
	return ws.WriteJSON(gin.H{
		"code": 200,
		"data": data,
		"msg":  "success",
	})
}

func (mc *MonitorController) updateHostMetrics() {
	mc.hostMu.Lock()
	defer mc.hostMu.Unlock()

	// 缓存 2 秒
	if time.Since(mc.lastUpdate) < 2*time.Second && mc.vMem != nil {
		return
	}

	if constant.DemoMode {
		mc.updateDemoMetrics()
		return
	}

	cpuPercents, _ := cpu.Percent(0, false)
	if len(cpuPercents) > 0 {
		mc.cpuPercent = cpuPercents[0]
	}
	mc.vMem, _ = mem.VirtualMemory()
	mc.diskUsage, _ = disk.Usage("/")
	mc.hostInfo, _ = host.Info()
	mc.lastUpdate = time.Now()
}

func (mc *MonitorController) updateDemoMetrics() {
	mc.cpuPercent = 10 + rand.Float64()*40 // 10% - 50% 的随机 CPU 波动

	totalMem := uint64(8 * 1024 * 1024 * 1024) // 8GB 内存
	usedMem := uint64(float64(totalMem) * (0.3 + rand.Float64()*0.3)) // 30% - 60% 随机使用率
	mc.vMem = &mem.VirtualMemoryStat{
		Total:       totalMem,
		Used:        usedMem,
		UsedPercent: float64(usedMem) / float64(totalMem) * 100,
	}

	totalDisk := uint64(500 * 1024 * 1024 * 1024) // 500GB 硬盘
	usedDisk := uint64(float64(totalDisk) * 0.45) // 固定 45% 使用率
	mc.diskUsage = &disk.UsageStat{
		Total:       totalDisk,
		Used:        usedDisk,
		UsedPercent: float64(usedDisk) / float64(totalDisk) * 100,
	}

	mc.hostInfo = &host.InfoStat{
		Platform: "Demo Environment", 
		OS:       "linux",
		Uptime:   uint64(time.Now().Unix() - 1700000000), // 生成一个较长且持续增加的运行时间
	}
	mc.lastUpdate = time.Now()
}

func (mc *MonitorController) getMonitorData() gin.H {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 更新并读取缓存的物理主机指标
	mc.updateHostMetrics()
	
	mc.hostMu.RLock()
	cpuPercent := mc.cpuPercent
	vMem := mc.vMem
	diskUsage := mc.diskUsage
	hostInfo := mc.hostInfo
	mc.hostMu.RUnlock()

	// 提供默认值防空指针
	if vMem == nil {
		vMem = &mem.VirtualMemoryStat{}
	}
	if diskUsage == nil {
		diskUsage = &disk.UsageStat{}
	}
	if hostInfo == nil {
		hostInfo = &host.InfoStat{}
	}

	return gin.H{
		"env": gin.H{
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"go_version": runtime.Version(),
			"num_cpu":    runtime.NumCPU(),
			"goroutines": runtime.NumGoroutine(),
		},
		"host": gin.H{
			"cpu_percent":  cpuPercent,
			"mem_total":    vMem.Total,
			"mem_used":     vMem.Used,
			"mem_percent":  vMem.UsedPercent,
			"disk_total":   diskUsage.Total,
			"disk_used":    diskUsage.Used,
			"disk_percent": diskUsage.UsedPercent,
			"uptime":       hostInfo.Uptime,
			"platform":     hostInfo.Platform + " " + hostInfo.PlatformVersion,
		},
		"mem": gin.H{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"lookups":     m.Lookups,
			"mallocs":     m.Mallocs,
			"frees":       m.Frees,
		},
		"heap": gin.H{
			"heap_alloc":    m.HeapAlloc,
			"heap_sys":      m.HeapSys,
			"heap_idle":     m.HeapIdle,
			"heap_inuse":    m.HeapInuse,
			"heap_released": m.HeapReleased,
			"heap_objects":  m.HeapObjects,
		},
		"gc": gin.H{
			"next_gc":        m.NextGC,
			"last_gc":        m.LastGC,
			"pause_total_ns": m.PauseTotalNs,
			"num_gc":         m.NumGC,
		},
		"scheduler": gin.H{
			"scheduled":    mc.executorService.GetScheduledCount(),
			"running":      mc.executorService.GetRunningCount(),
			"queue_size":   mc.executorService.GetScheduler().GetQueueSize(),
			"worker_count": mc.executorService.GetScheduler().GetConfig().WorkerCount,
			"workers":      mc.executorService.GetScheduler().GetWorkerStatuses(),
		},
	}
}
