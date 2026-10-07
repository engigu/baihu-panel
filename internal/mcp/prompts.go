package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPrompts(s *server.MCPServer, deps *Deps) {
	// 1. 任务失败诊断 Prompt
	s.AddPrompt(mcp.NewPrompt("diagnose_task_failure",
		mcp.WithPromptDescription("拉取指定任务的执行日志与配置，自动引导 AI 排查报错原因并生成修复建议"),
		mcp.WithArgument("log_id", mcp.ArgumentDescription("运行异常的日志 ID"), mcp.RequiredArgument()),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		logID := req.Params.Arguments["log_id"]
		if logID == "" {
			return nil, fmt.Errorf("必须提供参数 log_id")
		}

		logVO, err := deps.TaskLogService.GetLogDetailByID(logID)
		if err != nil {
			return nil, fmt.Errorf("未找到 ID 为 %s 的任务日志: %w", logID, err)
		}

		task := deps.TaskService.GetTaskByID(logVO.TaskID)
		command := logVO.Command
		taskName := logVO.TaskName
		triggerType := "cron"
		if task != nil {
			if task.Name != "" {
				taskName = task.Name
			}
			if command == "" {
				command = string(task.Command)
			}
			if task.TriggerType != "" {
				triggerType = task.TriggerType
			}
		}

		userPrompt := fmt.Sprintf(`你是一位精通自动化运维与脚本调试的技术专家。请帮我分析白虎面板中以下任务执行失败的具体原因，并给出排查步骤与代码修复建议：

【任务基本信息】
- 任务名称：%s
- 任务 ID：%s
- 触发方式：%s
- 执行指令：%s
- 执行状态：%s (耗时: %ds)

【控制台输出日志】
%s

【异常错误信息】
%s

请根据上述日志：
1. 明确指出导致任务失败的直接原因与核心报错行；
2. 解释产生该错误的环境依赖、语法或配置缺陷；
3. 给出切实可行的修复建议或可直接运行的替换代码/命令。`,
			taskName,
			logVO.TaskID,
			triggerType,
			command,
			logVO.Status,
			logVO.Duration,
			logVO.Output,
			logVO.Error,
		)

		return mcp.NewGetPromptResult("任务失败排查与诊断建议", []mcp.PromptMessage{
			mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(userPrompt)),
		}), nil
	})

	// 2. 自动化定时任务创建助手 Prompt
	s.AddPrompt(mcp.NewPrompt("create_scheduled_task",
		mcp.WithPromptDescription("引导 AI 根据业务需求生成符合白虎面板规范的 6 位标准 Cron 表达式与执行指令"),
		mcp.WithArgument("task_description", mcp.ArgumentDescription("任务目标描述（如：'每天凌晨3点抓取数据并推送到飞书'）"), mcp.RequiredArgument()),
	), func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		desc := req.Params.Arguments["task_description"]
		if desc == "" {
			return nil, fmt.Errorf("必须提供参数 task_description")
		}

		userPrompt := fmt.Sprintf(`用户想要在白虎面板中创建一个自动化任务，其需求描述如下：
"%s"

请帮助用户设计该任务，并提供：
1. 规范的 6 位秒级 Cron 定时表达式（例如：0 0 3 * * * 表示每天凌晨 3:00 执行）；
2. 推荐的执行命令（支持 bash / pwsh / node / python / dotnet）；
3. 如果需要脚本文件，提供完整的示例源码与存放路径建议（如 main.js / run.py）；
4. 指引用户调用 create_task 或 save_script 工具自动写入面板。`, desc)

		return mcp.NewGetPromptResult("自动化任务创建方案建议", []mcp.PromptMessage{
			mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(userPrompt)),
		}), nil
	})
}
