package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerResources(s *server.MCPServer, deps *Deps) {
	// 静态资源: baihu://status
	s.AddResource(mcp.NewResource("baihu://status", "白虎面板运行概览",
		mcp.WithMIMEType("application/json"),
		mcp.WithResourceDescription("白虎面板当前服务状态与统计摘要"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		overview := deps.ExecutorService.GetSystemStatsOverview()

		status := map[string]any{
			"version":         constant.Version,
			"os":              runtime.GOOS,
			"arch":            runtime.GOARCH,
			"total_tasks":     overview.Tasks,
			"scheduled_tasks": overview.Scheduled,
			"running_tasks":   overview.Running,
			"today_execs":     overview.TodayExecs,
			"total_envs":      overview.Envs,
			"total_logs":      overview.Logs,
			"current_time":    time.Now().Format("2006-01-02 15:04:05"),
		}
		data, _ := json.MarshalIndent(status, "", "  ")

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// 动态模板资源: task://{id}
	s.AddResourceTemplate(mcp.NewResourceTemplate("task://{id}", "任务配置资源",
		mcp.WithTemplateMIMEType("application/json"),
		mcp.WithTemplateDescription("读取指定 ID 的任务配置元数据"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		taskID := strings.TrimPrefix(req.Params.URI, "task://")
		task := deps.TaskService.GetTaskByID(taskID)
		if task == nil {
			return nil, fmt.Errorf("未找到任务 ID: %s", taskID)
		}

		data, _ := json.MarshalIndent(vo.ToTaskVO(task), "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// 动态模板资源: log://{id}
	s.AddResourceTemplate(mcp.NewResourceTemplate("log://{id}", "任务执行日志资源",
		mcp.WithTemplateMIMEType("text/plain"),
		mcp.WithTemplateDescription("读取指定 ID 的任务执行日志输出"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		logID := strings.TrimPrefix(req.Params.URI, "log://")
		logVO, err := deps.TaskLogService.GetLogDetailByID(logID)
		if err != nil {
			return nil, err
		}

		text := fmt.Sprintf("=== 任务: %s (状态: %s, 耗时: %ds) ===\n%s\n", logVO.TaskName, logVO.Status, logVO.Duration, logVO.Output)
		if logVO.Error != "" {
			text += fmt.Sprintf("\n[错误异常]: %s\n", logVO.Error)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     text,
			},
		}, nil
	})

	// 动态模板资源: file://{path}
	s.AddResourceTemplate(mcp.NewResourceTemplate("file://{path}", "脚本源码资源",
		mcp.WithTemplateMIMEType("text/plain"),
		mcp.WithTemplateDescription("读取指定相对路径的脚本文件内容"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		relPath := strings.TrimPrefix(req.Params.URI, "file://")
		fileVO, err := deps.FileService.GetFileContent(relPath)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     fileVO.Content,
			},
		}, nil
	})

	// 动态模板资源: app://{id}
	s.AddResourceTemplate(mcp.NewResourceTemplate("app://{id}", "声明式应用配置资源",
		mcp.WithTemplateMIMEType("application/json"),
		mcp.WithTemplateDescription("读取指定已安装应用的配置详情与受控子任务清单"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		appID := strings.TrimPrefix(req.Params.URI, "app://")
		if deps.AppService == nil {
			return nil, fmt.Errorf("AppService 未初始化")
		}
		appDTO, err := deps.AppService.GetApp(appID)
		if err != nil {
			var masterTask models.Task
			if dbErr := database.DB.Where("source_id = ? AND type = ?", "app:"+appID, constant.TaskTypeApp).First(&masterTask).Error; dbErr == nil {
				appDTO, err = deps.AppService.GetApp(masterTask.ID)
			}
		}
		if err != nil || appDTO == nil {
			return nil, fmt.Errorf("未找到应用: %s", appID)
		}

		var childTasks []models.Task
		database.DB.Where("source_id = ? AND type = ?", appDTO.ID, constant.TaskTypeNormal).Order("created_at asc").Find(&childTasks)

		resData := map[string]interface{}{
			"app":         appDTO,
			"child_tasks": childTasks,
		}
		data, _ := json.MarshalIndent(resData, "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// 动态模板资源: repo://{id}
	s.AddResourceTemplate(mcp.NewResourceTemplate("repo://{id}", "代码仓库配置资源",
		mcp.WithTemplateMIMEType("application/json"),
		mcp.WithTemplateDescription("读取指定代码仓库任务的完整 Git 属性与同步状态"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		repoID := strings.TrimPrefix(req.Params.URI, "repo://")
		var task models.Task
		res := database.DB.Where("id = ? AND type = ?", repoID, constant.TaskTypeRepo).Limit(1).Find(&task)
		if res.Error != nil || res.RowsAffected == 0 {
			return nil, fmt.Errorf("未找到代码仓库: %s", repoID)
		}

		repoCfg := task.GetRepoConfig()
		var latestLog models.TaskLog
		lastStatus := "none"
		if dbRes := database.DB.Where("task_id = ?", task.ID).Order("created_at desc").Limit(1).Find(&latestLog); dbRes.Error == nil && dbRes.RowsAffected > 0 {
			lastStatus = latestLog.Status
		}

		resData := map[string]interface{}{
			"id":          task.ID,
			"name":        task.Name,
			"remark":      task.Remark,
			"schedule":    task.Schedule,
			"enabled":     utils.DerefBool(task.Enabled, true),
			"repo_config": repoCfg,
			"last_run":    task.LastRun,
			"last_status": lastStatus,
		}
		data, _ := json.MarshalIndent(resData, "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})

	// 静态资源: notify://channels
	s.AddResource(mcp.NewResource("notify://channels", "通知渠道概览",
		mcp.WithMIMEType("application/json"),
		mcp.WithResourceDescription("查看当前已配置的通知渠道清单与启停状态"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if deps.NotifyService == nil {
			return nil, fmt.Errorf("NotifyService 未初始化")
		}
		channels := deps.NotifyService.GetChannels()
		type SafeChan struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Enabled bool   `json:"enabled"`
		}
		safeList := make([]SafeChan, 0, len(channels))
		for _, ch := range channels {
			safeList = append(safeList, SafeChan{
				ID:      ch.ID,
				Name:    ch.Name,
				Type:    ch.Type,
				Enabled: ch.Enabled,
			})
		}
		data, _ := json.MarshalIndent(safeList, "", "  ")
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	})
}

