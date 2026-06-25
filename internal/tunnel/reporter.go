package tunnel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/robfig/cron/v3"
)

var (
	reporterCronID      cron.EntryID
	isReporting         bool
	localTunnelURL      string
	localTunnelURLMutex sync.RWMutex
)

// GetLocalTunnelURL 返回当前子节点被分配的隧道地址
func GetLocalTunnelURL() string {
	localTunnelURLMutex.RLock()
	defer localTunnelURLMutex.RUnlock()
	return localTunnelURL
}

// StartReporter 启动子节点上报守护进程
func StartReporter(parentURL, token string) {
	if isReporting {
		return
	}
	isReporting = true

	// 修整 parentURL 确保正确指向 /api/v1/interconnect/report
	// 如果用户填的是 http://192.168.1.100:8000，我们要加上路径
	baseURL := strings.TrimRight(parentURL, "/")
	// 注意兼容如果用户带了 /api/v1
	if !strings.HasSuffix(baseURL, "/api/v1") && !strings.Contains(baseURL, "/api/v1") {
		baseURL = baseURL + "/api/v1"
	}
	reportURL := baseURL + "/interconnect/report"

	// 使用全局 SysCron 并立即上报一次，随后每 45 秒循环一次
	id, err := executor.GetSysCron().AddJobWithRun("@every 45s", func() {
		reportMonitorData(reportURL, token)
	})

	if err == nil {
		reporterCronID = id
	} else {
		logger.Warnf("[Tunnel] 无法将上报任务加入 SysCron: %v", err)
	}
}

// StopReporter 停止子节点上报守护进程
func StopReporter() {
	if isReporting {
		if reporterCronID != 0 {
			executor.GetSysCron().RemoveJob(reporterCronID)
			reporterCronID = 0
		}
		isReporting = false
	}
}

func reportMonitorData(reportURL, token string) {
	// 调用单例监控服务获取底层系统状态
	metrics := services.GetMonitorService().GetHostMetrics()
	
	payload := map[string]interface{}{
		"cpu_percent":  metrics.CPUPercent,
		"mem_percent":  metrics.VMem.UsedPercent,
		"disk_percent": metrics.DiskUsage.UsedPercent,
		"tx_bytes":     GetTxBytes(),
		"rx_bytes":     GetRxBytes(),
	}
	
	data, _ := json.Marshal(payload)
	
	req, err := http.NewRequest("POST", reportURL, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warnf("[Tunnel] 上报监控数据失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warnf("[Tunnel] 上报监控数据返回异常状态码: %d", resp.StatusCode)
		return
	}

	var res struct {
		Data struct {
			TunnelURL string `json:"tunnel_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		if res.Data.TunnelURL != "" {
			localTunnelURLMutex.Lock()
			localTunnelURL = res.Data.TunnelURL
			localTunnelURLMutex.Unlock()
		}
	}
}
