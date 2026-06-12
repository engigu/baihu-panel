package controllers

import (
	"net/http"
	"runtime"
	"time"

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

func (mc *MonitorController) getMonitorData() gin.H {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return gin.H{
		"env": gin.H{
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"go_version": runtime.Version(),
			"num_cpu":    runtime.NumCPU(),
			"goroutines": runtime.NumGoroutine(),
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
