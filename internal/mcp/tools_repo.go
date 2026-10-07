package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerRepoTools 注册代码仓库同步 (Repo Sync) 相关 MCP 工具
func registerRepoTools(s *server.MCPServer, deps *Deps) {
	registerListReposTool(s, deps)
	registerSyncRepoTool(s, deps)
}

// 1. list_repos
func registerListReposTool(s *server.MCPServer, _ *Deps) {
	s.AddTool(mcp.NewTool("list_repos",
		mcp.WithDescription("列出白虎面板中已绑定的所有 Git 脚本代码仓库配置与最近同步状态"),
		mcp.WithString("keyword", mcp.Description("按仓库名称或 Git URL 搜索关键字")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListRepos(ctx, req)
	})
}

func handleListRepos(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword := strings.TrimSpace(strings.ToLower(req.GetString("keyword", "")))

	var repoTasks []models.Task
	query := database.DB.Where("type = ?", constant.TaskTypeRepo).Order("created_at desc")
	if err := query.Find(&repoTasks).Error; err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查询代码仓库失败: %v", err)), nil
	}

	type RepoDTO struct {
		ID         string            `json:"id"`
		Name       string            `json:"name"`
		Remark     string            `json:"remark,omitempty"`
		SourceURL  string            `json:"source_url"`
		Branch     string            `json:"branch"`
		TargetPath string            `json:"target_path"`
		Schedule   string            `json:"schedule"`
		Enabled    bool              `json:"enabled"`
		LastRun    *models.LocalTime `json:"last_run,omitempty"`
		NextRun    *models.LocalTime `json:"next_run,omitempty"`
		LastStatus string            `json:"last_status,omitempty"`
		LastLogID  string            `json:"last_log_id,omitempty"`
	}

	var result []RepoDTO
	for _, task := range repoTasks {
		repoCfg := task.GetRepoConfig()
		sourceURL := ""
		branch := ""
		targetPath := ""
		if repoCfg != nil {
			sourceURL = repoCfg.SourceURL
			branch = repoCfg.Branch
			targetPath = repoCfg.TargetPath
		}

		if keyword != "" {
			searchTarget := strings.ToLower(fmt.Sprintf("%s %s %s %s", task.Name, task.Remark, sourceURL, targetPath))
			if !strings.Contains(searchTarget, keyword) {
				continue
			}
		}

		// 查询该仓库任务最近一次的执行状态
		var latestLog models.TaskLog
		lastStatus := "none"
		lastLogID := ""
		if dbRes := database.DB.Where("task_id = ?", task.ID).Order("created_at desc").Limit(1).Find(&latestLog); dbRes.Error == nil && dbRes.RowsAffected > 0 {
			lastStatus = latestLog.Status
			lastLogID = latestLog.ID
		}

		result = append(result, RepoDTO{
			ID:         task.ID,
			Name:       task.Name,
			Remark:     task.Remark,
			SourceURL:  sourceURL,
			Branch:     branch,
			TargetPath: targetPath,
			Schedule:   task.Schedule,
			Enabled:    utils.DerefBool(task.Enabled, true),
			LastRun:    task.LastRun,
			NextRun:    task.NextRun,
			LastStatus: lastStatus,
			LastLogID:  lastLogID,
		})
	}

	resData := map[string]interface{}{
		"total": len(result),
		"repos": result,
	}
	data, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 2. sync_repo
func registerSyncRepoTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("sync_repo",
		mcp.WithDescription("手动触发指定 Git 脚本仓库的即时拉取同步（执行 git pull / 依赖更新与任务解析）"),
		mcp.WithString("repo_id", mcp.Required(), mcp.Description("代码仓库任务 ID")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSyncRepo(ctx, deps, req)
	})
}

func handleSyncRepo(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoID, err := req.RequireString("repo_id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 repo_id"), nil
	}

	task := deps.TaskService.GetTaskByID(repoID)
	if task == nil {
		return mcp.NewToolResultError(fmt.Sprintf("未找到代码仓库任务: %s", repoID)), nil
	}

	if task.Type != constant.TaskTypeRepo {
		return mcp.NewToolResultError(fmt.Sprintf("指定任务 %s 不是代码仓库类型 (实际类型: %s)", repoID, task.Type)), nil
	}

	execResult := deps.ExecutorService.ExecuteTask(repoID, nil)
	if execResult == nil || !execResult.Success {
		errMsg := "执行失败"
		if execResult != nil && execResult.Error != "" {
			errMsg = execResult.Error
		}
		return mcp.NewToolResultError(fmt.Sprintf("触发仓库同步失败: %s", errMsg)), nil
	}

	res := map[string]any{
		"success":    true,
		"repo_id":    repoID,
		"repo_name":  task.Name,
		"status":     execResult.Status,
		"log_id":     execResult.LogID,
		"start_time": execResult.StartTime.Format("2006-01-02 15:04:05"),
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
