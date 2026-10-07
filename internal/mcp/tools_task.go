package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTaskTools 注册任务编排与生命周期管理相关 MCP 工具
func registerTaskTools(s *server.MCPServer, deps *Deps) {
	registerListTasksTool(s, deps)
	registerGetTaskTool(s, deps)
	registerCreateTaskTool(s, deps)
	registerUpdateTaskTool(s, deps)
	registerDeleteTaskTool(s, deps)
	registerExecuteTaskTool(s, deps)
	registerStopTaskTool(s, deps)
}

// 1. list_tasks: 获取任务列表
func registerListTasksTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("list_tasks",
		mcp.WithDescription("获取白虎面板的任务列表，支持按名称、标签、类型和启用状态筛选"),
		mcp.WithString("name", mcp.Description("任务名称搜索关键字")),
		mcp.WithString("tags", mcp.Description("按标签筛选（逗号分隔）")),
		mcp.WithString("type", mcp.Description("任务类型：task (普通任务), app (声明式应用主任务), repo (仓库任务)")),
		mcp.WithBoolean("enabled", mcp.Description("是否启用：true 或 false")),
		mcp.WithInteger("page", mcp.Description("页码，默认为 1")),
		mcp.WithInteger("page_size", mcp.Description("每页大小，默认为 20")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListTasks(ctx, deps, req)
	})
}

func handleListTasks(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	tags := req.GetString("tags", "")
	taskType := req.GetString("type", "")
	page := req.GetInt("page", 1)
	pageSize := req.GetInt("page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var enabledPtr *bool
	if val, err := req.RequireBool("enabled"); err == nil {
		enabledPtr = &val
	}

	taskList, total := deps.TaskService.GetTasksWithPagination(page, pageSize, name, nil, tags, taskType, "", enabledPtr, "", "")
	voList := vo.ToTaskVOListFromModels(taskList)

	res := map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"tasks":     voList,
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 2. get_task: 获取特定任务详情
func registerGetTaskTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("get_task",
		mcp.WithDescription("根据任务 ID 获取特定任务的详细配置"),
		mcp.WithString("id", mcp.Required(), mcp.Description("任务的主键 ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetTask(ctx, deps, req)
	})
}

func handleGetTask(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 id"), nil
	}

	task := deps.TaskService.GetTaskByID(id)
	if task == nil {
		return mcp.NewToolResultError(fmt.Sprintf("未找到 ID 为 %s 的任务", id)), nil
	}

	taskVO := vo.ToTaskVO(task)
	data, _ := json.MarshalIndent(taskVO, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 3. create_task: 创建新定时任务
func registerCreateTaskTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("create_task",
		mcp.WithDescription("创建新的自动化定时任务"),
		mcp.WithString("name", mcp.Required(), mcp.Description("任务名称")),
		mcp.WithString("command", mcp.Required(), mcp.Description("执行指令（例如: node test.js 或 python3 main.py）")),
		mcp.WithString("schedule", mcp.Description("Cron 定时规则（标准 6 位秒级或 5 位标准 Cron，如 0 0 8 * * *）")),
		mcp.WithString("work_dir", mcp.Description("运行工作目录，留空默认脚本根目录")),
		mcp.WithString("tags", mcp.Description("任务标签（逗号分隔）")),
		mcp.WithString("remark", mcp.Description("任务备注说明")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateTask(ctx, deps, req)
	})
}

func handleCreateTask(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("缺少任务名称 name"), nil
	}
	command, err := req.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError("缺少执行指令 command"), nil
	}

	param := tasks.TaskParam{
		Name:     name,
		Command:  command,
		Schedule: req.GetString("schedule", ""),
		WorkDir:  req.GetString("work_dir", ""),
		Tags:     req.GetString("tags", ""),
		Remark:   req.GetString("remark", ""),
		Type:     constant.TaskTypeNormal,
		Enabled:  true,
	}

	createdTask, err := deps.TaskService.CreateTaskWithSchedule(&param, deps.ExecutorService)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("创建任务失败: %v", err)), nil
	}

	taskVO := vo.ToTaskVO(createdTask)
	data, _ := json.MarshalIndent(taskVO, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("任务创建成功:\n%s", string(data))), nil
}

// 4. update_task: 更新已有任务配置
func registerUpdateTaskTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("update_task",
		mcp.WithDescription("更新已有任务的配置（名称、指令、定时规则、启停状态等）"),
		mcp.WithString("id", mcp.Required(), mcp.Description("待更新的任务 ID")),
		mcp.WithString("name", mcp.Description("新任务名称")),
		mcp.WithString("command", mcp.Description("新执行指令")),
		mcp.WithString("schedule", mcp.Description("新 Cron 表达式")),
		mcp.WithString("work_dir", mcp.Description("新工作目录")),
		mcp.WithBoolean("enabled", mcp.Description("启停状态")),
		mcp.WithString("tags", mcp.Description("新标签")),
		mcp.WithString("remark", mcp.Description("新备注")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateTask(ctx, deps, req)
	})
}

func handleUpdateTask(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 id"), nil
	}

	oldTask := deps.TaskService.GetTaskByID(id)
	if oldTask == nil {
		return mcp.NewToolResultError(fmt.Sprintf("未找到 ID 为 %s 的任务", id)), nil
	}

	param := tasks.TaskParam{
		Name:          oldTask.Name,
		Remark:        oldTask.Remark,
		Command:       string(oldTask.Command),
		PreCommand:    string(oldTask.PreCommand),
		PostCommand:   string(oldTask.PostCommand),
		Tags:          oldTask.Tags,
		Type:          oldTask.Type,
		UnifiedConfig: string(oldTask.UnifiedConfig),
		Schedule:      oldTask.Schedule,
		Timeout:       oldTask.Timeout,
		WorkDir:       oldTask.WorkDir,
		CleanConfig:   string(oldTask.CleanConfig),
		Envs:          string(oldTask.Envs),
		Languages:     oldTask.Languages,
		AgentID:       oldTask.AgentID,
		TriggerType:   oldTask.TriggerType,
		RetryCount:    oldTask.RetryCount,
		RetryInterval: oldTask.RetryInterval,
		RandomRange:   oldTask.RandomRange,
		PinType:       oldTask.PinType,
		Enabled:       utils.DerefBool(oldTask.Enabled, true),
		SourceID:      oldTask.SourceID,
	}

	if val := req.GetString("name", ""); val != "" {
		param.Name = val
	}
	if val := req.GetString("command", ""); val != "" {
		param.Command = val
	}
	if val := req.GetString("schedule", ""); val != "" {
		param.Schedule = val
	}
	if val := req.GetString("work_dir", ""); val != "" {
		param.WorkDir = val
	}
	if val := req.GetString("tags", ""); val != "" {
		param.Tags = val
	}
	if val := req.GetString("remark", ""); val != "" {
		param.Remark = val
	}
	if val, err := req.RequireBool("enabled"); err == nil {
		param.Enabled = val
	}

	updatedTask, err := deps.TaskService.UpdateTaskWithSchedule(id, &param, deps.ExecutorService)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("更新任务失败: %v", err)), nil
	}

	taskVO := vo.ToTaskVO(updatedTask)
	data, _ := json.MarshalIndent(taskVO, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("任务更新成功:\n%s", string(data))), nil
}

// 5. delete_task: 级联删除任务
func registerDeleteTaskTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("delete_task",
		mcp.WithDescription("删除指定任务并停止其定时计划与受控依赖"),
		mcp.WithString("id", mcp.Required(), mcp.Description("要删除的任务 ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteTask(ctx, deps, req)
	})
}

func handleDeleteTask(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 id"), nil
	}

	task := deps.TaskService.GetTaskByID(id)
	if task == nil {
		return mcp.NewToolResultError(fmt.Sprintf("未找到 ID 为 %s 的任务", id)), nil
	}

	success := deps.TaskService.DeleteTaskWithCascade(id, deps.ExecutorService)
	if !success {
		return mcp.NewToolResultError("删除任务失败"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("成功删除任务: %s (ID: %s)", task.Name, id)), nil
}

// 6. execute_task: 立即执行指定任务
func registerExecuteTaskTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("execute_task",
		mcp.WithDescription("立即执行指定的任务，并返回本次执行结果与日志 ID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("任务主键 ID")),
		mcp.WithString("extra_envs", mcp.Description("临时附加环境变量，多项可用换行或逗号分隔（如: KEY1=VAL1,KEY2=VAL2）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleExecuteTask(ctx, deps, req)
	})
}

func handleExecuteTask(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 id"), nil
	}

	task := deps.TaskService.GetTaskByID(id)
	if task == nil {
		return mcp.NewToolResultError(fmt.Sprintf("未找到 ID 为 %s 的任务", id)), nil
	}

	var extraEnvs []string
	if envStr := req.GetString("extra_envs", ""); envStr != "" {
		parts := strings.FieldsFunc(envStr, func(r rune) bool {
			return r == '\n' || r == ',' || r == ';'
		})
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				extraEnvs = append(extraEnvs, part)
			}
		}
	}

	result := deps.ExecutorService.ExecuteTask(id, extraEnvs)
	if result == nil {
		return mcp.NewToolResultError("触发任务执行异常"), nil
	}

	resultVO := vo.ToExecutionResultVO(result)
	data, _ := json.MarshalIndent(resultVO, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("任务已触发执行:\n%s\n提示: 可使用 get_log_detail 查看执行日志 (log_id: %s)", string(data), result.LogID)), nil
}

// 7. stop_task: 停止运行中的任务
func registerStopTaskTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("stop_task",
		mcp.WithDescription("停止正在运行中的任务。支持通过 log_id 停止单次运行，或通过 task_id 停止该任务的所有并发运行副本"),
		mcp.WithString("log_id", mcp.Description("单次运行的执行日志 ID")),
		mcp.WithString("task_id", mcp.Description("任务主键 ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleStopTask(ctx, deps, req)
	})
}

func handleStopTask(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logID := req.GetString("log_id", "")
	taskID := req.GetString("task_id", "")

	if logID == "" && taskID == "" {
		return mcp.NewToolResultError("必须提供 log_id 或 task_id 之一"), nil
	}

	if logID != "" {
		if err := deps.ExecutorService.StopTaskExecution(logID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("停止执行失败: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("成功停止执行实例 (log_id: %s)", logID)), nil
	}

	stopped := deps.ExecutorService.GetScheduler().StopTask(taskID)
	if !stopped {
		return mcp.NewToolResultText(fmt.Sprintf("任务 (task_id: %s) 当前没有正在运行的副本，或已自行退出", taskID)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("成功停止任务 (task_id: %s) 正在运行的副本", taskID)), nil
}
