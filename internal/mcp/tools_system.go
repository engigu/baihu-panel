package mcp

import (
	"context"
	"encoding/json"
	"runtime"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var startTime = time.Now()

func registerSystemTools(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("get_system_status",
		mcp.WithDescription("获取白虎面板系统运行状态概览（版本号、架构、任务统计、调度中/运行中任务数、环境变量数及系统时间）"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		overview := deps.ExecutorService.GetSystemStatsOverview()

		status := map[string]any{
			"version":         constant.Version,
			"os":              runtime.GOOS,
			"arch":            runtime.GOARCH,
			"num_cpu":         runtime.NumCPU(),
			"uptime":          time.Since(startTime).String(),
			"current_time":    time.Now().Format("2006-01-02 15:04:05"),
			"total_tasks":     overview.Tasks,
			"scheduled_tasks": overview.Scheduled,
			"running_tasks":   overview.Running,
			"today_execs":     overview.TodayExecs,
			"total_envs":      overview.Envs,
			"total_logs":      overview.Logs,
		}

		data, _ := json.MarshalIndent(status, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	})
}
