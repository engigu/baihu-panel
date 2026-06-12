package controllers

import (
	"net/http"
	"runtime"
	"sync"
	"time"

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

// MonitorWS WebSocket实时推送系统监控信息
func (mc *MonitorController) MonitorWS(c *gin.Context) {
	ws, err := monitorUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// 先立即发送一次
	mc.sendMonitorData(ws)

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

	cpuPercents, _ := cpu.Percent(0, false)
	if len(cpuPercents) > 0 {
		mc.cpuPercent = cpuPercents[0]
	}
	mc.vMem, _ = mem.VirtualMemory()
	mc.diskUsage, _ = disk.Usage("/")
	mc.hostInfo, _ = host.Info()
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
