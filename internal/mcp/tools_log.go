package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerLogTools 注册所有日志相关的 MCP 工具
func registerLogTools(s *server.MCPServer, deps *Deps) {
	registerListTaskLogsTool(s, deps)
	registerGetLogDetailTool(s, deps)
	registerCleanTaskLogsTool(s, deps)
}

// 1. list_task_logs
func registerListTaskLogsTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("list_task_logs",
		mcp.WithDescription("分页查询任务执行历史记录列表，可按任务ID、名称、运行状态或日期筛选"),
		mcp.WithString("task_id", mcp.Description("任务主键 ID")),
		mcp.WithString("task_name", mcp.Description("任务名称搜索关键字")),
		mcp.WithString("status", mcp.Description("执行状态：running (运行中), success (成功), failed (失败), cancelled (已取消)")),
		mcp.WithString("date", mcp.Description("查询日期，如 'today' 或 '2026-10-07'")),
		mcp.WithInteger("page", mcp.Description("页码，默认为 1")),
		mcp.WithInteger("page_size", mcp.Description("每页大小，默认为 10，最大 100")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTaskLogs(ctx, deps, req)
	})
}

func handleListTaskLogs(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID := req.GetString("task_id", "")
	taskName := req.GetString("task_name", "")
	status := req.GetString("status", "")
	date := req.GetString("date", "")
	page := req.GetInt("page", 1)
	pageSize := req.GetInt("page_size", 10)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	} else if pageSize > 100 {
		pageSize = 100
	}

	resultList, total := deps.TaskLogService.GetLogsWithPagination(page, pageSize, taskID, taskName, status, date)
	res := map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"logs":      resultList,
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 2. get_log_detail
func registerGetLogDetailTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("get_log_detail",
		mcp.WithDescription("获取指定任务执行记录的完整详细信息与控制台输出日志"),
		mcp.WithString("log_id", mcp.Required(), mcp.Description("日志主键 ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetLogDetail(ctx, deps, req)
	})
}

func handleGetLogDetail(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logID, err := req.RequireString("log_id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 log_id"), nil
	}

	logVO, err := deps.TaskLogService.GetLogDetailByID(logID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, _ := json.MarshalIndent(logVO, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 3. clean_task_logs
func registerCleanTaskLogsTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("clean_task_logs",
		mcp.WithDescription("清理历史任务执行日志，释放数据库存储空间。支持指定任务或按保留天数清理"),
		mcp.WithString("task_id", mcp.Description("仅清理指定任务的日志；留空表示全局清理")),
		mcp.WithInteger("days", mcp.Description("清理早于指定天数的旧日志（如 7 表示清理 7 天前日志）；传 0 或未传则清空全部匹配日志")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCleanTaskLogs(ctx, deps, req)
	})
}

func handleCleanTaskLogs(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID := req.GetString("task_id", "")
	days := req.GetInt("days", 0)
	if days < 0 {
		days = 0
	}

	count, err := deps.TaskLogService.ClearLogs(taskID, days)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	res := map[string]any{
		"success":       true,
		"deleted_count": count,
		"task_id":       taskID,
		"days":          days,
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
