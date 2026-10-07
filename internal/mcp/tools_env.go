package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerEnvTools 注册所有环境变量相关工具
func registerEnvTools(s *server.MCPServer, deps *Deps) {
	registerListEnvVarsTool(s, deps)
	registerSetEnvVarTool(s, deps)
	registerDeleteEnvVarTool(s, deps)
}

// 1. list_env_vars
func registerListEnvVarsTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("list_env_vars",
		mcp.WithDescription("获取环境变量列表，支持按变量名称、备注、类型和标签筛选"),
		mcp.WithString("name", mcp.Description("变量名称或备注搜索关键字")),
		mcp.WithString("type", mcp.Description("变量类型：normal (普通) 或 secret (机密)")),
		mcp.WithString("tags", mcp.Description("按标签筛选（逗号分隔）")),
		mcp.WithInteger("page", mcp.Description("页码，默认为 1")),
		mcp.WithInteger("page_size", mcp.Description("每页大小，默认为 50")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListEnvVars(ctx, deps, req)
	})
}

func handleListEnvVars(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	envType := req.GetString("type", "")
	tags := req.GetString("tags", "")
	page := req.GetInt("page", 1)
	pageSize := req.GetInt("page_size", 50)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	userID := deps.EnvService.GetDefaultAdminUserID()
	envs, total := deps.EnvService.GetEnvVarsWithPagination(userID, name, envType, tags, page, pageSize)
	voList := vo.ToEnvVOListFromModels(envs)

	res := map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"envs":      voList,
	}
	data, _ := json.MarshalIndent(res, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 2. set_env_var
func registerSetEnvVarTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("set_env_var",
		mcp.WithDescription("新增或更新环境变量。如果存在同名变量则更新其值，不存在则自动新建，并自动触发 Agent 配置广播"),
		mcp.WithString("name", mcp.Required(), mcp.Description("环境变量名（如 JD_COOKIE, BILI_COOKIE 等）")),
		mcp.WithString("value", mcp.Required(), mcp.Description("环境变量值")),
		mcp.WithString("remark", mcp.Description("备注说明")),
		mcp.WithString("type", mcp.Description("类型：normal (默认普通文本) 或 secret (加密存储)")),
		mcp.WithString("tags", mcp.Description("关联标签（逗号分隔）")),
		mcp.WithBoolean("enabled", mcp.Description("是否启用，默认为 true")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSetEnvVar(ctx, deps, req)
	})
}

func handleSetEnvVar(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 name"), nil
	}
	value, err := req.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 value"), nil
	}

	remark := req.GetString("remark", "")
	envType := req.GetString("type", constant.EnvTypeNormal)
	if envType != constant.EnvTypeSecret {
		envType = constant.EnvTypeNormal
	}
	tags := req.GetString("tags", "")
	enabled := true
	if val, err := req.RequireBool("enabled"); err == nil {
		enabled = val
	}

	userID := deps.EnvService.GetDefaultAdminUserID()
	envVar, isNew := deps.EnvService.SetOrUpdateEnvVar(name, value, remark, envType, false, enabled, userID, tags)
	if envVar == nil {
		return mcp.NewToolResultError("保存环境变量失败"), nil
	}

	action := "更新"
	if isNew {
		action = "创建"
	}
	return mcp.NewToolResultText(fmt.Sprintf("成功%s环境变量: %s (ID: %s)", action, name, envVar.ID)), nil
}

// 3. delete_env_var
func registerDeleteEnvVarTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("delete_env_var",
		mcp.WithDescription("删除指定的环境变量"),
		mcp.WithString("id", mcp.Required(), mcp.Description("环境变量主键 ID")),
		mcp.WithBoolean("force", mcp.Description("是否强制删除（即便被任务关联引用）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteEnvVar(ctx, deps, req)
	})
}

func handleDeleteEnvVar(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 id"), nil
	}
	force := false
	if val, err := req.RequireBool("force"); err == nil {
		force = val
	}

	success, associatedTasks := deps.EnvService.DeleteEnvVarWithBroadcast(id, force)
	if !success {
		if len(associatedTasks) > 0 {
			taskNames := make([]string, 0, len(associatedTasks))
			for _, t := range associatedTasks {
				taskNames = append(taskNames, t.Name)
			}
			return mcp.NewToolResultError(fmt.Sprintf("该环境变量被以下任务关联引用，无法直接删除: %v。若仍要删除，请传入 force: true", taskNames)), nil
		}
		return mcp.NewToolResultError("删除环境变量失败，变量可能不存在"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("成功删除环境变量 (ID: %s)", id)), nil
}
