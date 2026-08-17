package controllers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type MonitorController struct {
	executorService *tasks.ExecutorService
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

// MonitorSSE Server-Sent Events 获取系统监控数据
func (mc *MonitorController) MonitorSSE(c *gin.Context) {
	// 设置 SSE 响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	// 初始发送一次数据
	if err := mc.sendMonitorDataSSE(c); err != nil {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := mc.sendMonitorDataSSE(c); err != nil {
				return // 客户端断开连接或发送失败
			}
		case <-c.Request.Context().Done():
			return // 连接已断开，立即退出
		}
	}
}

func (mc *MonitorController) sendMonitorDataSSE(c *gin.Context) error {
	data := mc.getMonitorData()
	// 使用 Gin 提供的 SSE 方法
	c.SSEvent("message", gin.H{
		"code": 200,
		"data": data,
		"msg":  "success",
	})
	c.Writer.Flush()
	return nil
}

func (mc *MonitorController) getMonitorData() gin.H {
	rt := services.GetMonitorService().GetRuntimeMetrics()
	m := rt.MemStats

	// 调用统一的监控服务获取物理机指标
	metrics := services.GetMonitorService().GetHostMetrics()

	return gin.H{
		"env": gin.H{
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"go_version": runtime.Version(),
			"num_cpu":    runtime.NumCPU(),
			"goroutines": rt.NumGoroutine,
		},
		"host": gin.H{
			"cpu_percent":  metrics.CPUPercent,
			"mem_total":    metrics.VMem.Total,
			"mem_used":     metrics.VMem.Used,
			"mem_percent":  metrics.VMem.UsedPercent,
			"disk_total":   metrics.DiskUsage.Total,
			"disk_used":    metrics.DiskUsage.Used,
			"disk_percent": metrics.DiskUsage.UsedPercent,
			"uptime":       metrics.HostInfo.Uptime,
			"platform":     metrics.HostInfo.Platform + " " + metrics.HostInfo.PlatformVersion,
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
