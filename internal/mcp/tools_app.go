package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/app"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerAppTools 注册声明式应用 (Declarative Apps) 与应用商店 (AppStore) 相关 MCP 工具
func registerAppTools(s *server.MCPServer, deps *Deps) {
	registerListStoreAppsTool(s, deps)
	registerGetStoreAppTool(s, deps)
	registerListInstalledAppsTool(s, deps)
	registerGetInstalledAppTool(s, deps)
	registerInstallAppTool(s, deps)
	registerSwitchAppScenarioTool(s, deps)
	registerUninstallAppTool(s, deps)
	registerCheckAppUpdatesTool(s, deps)
	registerUpdateAppTool(s, deps)
	registerRollbackAppTool(s, deps)
}

// 1. list_store_apps: 检索应用商店列表
func registerListStoreAppsTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("list_store_apps",
		mcp.WithDescription("从白虎官方应用商店中检索可用的声明式自动化应用列表，支持按关键字或分类过滤"),
		mcp.WithString("keyword", mcp.Description("搜索关键字（匹配应用名称、标识、作者、分类或功能描述）")),
		mcp.WithString("category", mcp.Description("所属分类过滤（如 '福利签到', '系统工具', '消息推送', '运维管理'）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListStoreApps(ctx, deps, req)
	})
}

func handleListStoreApps(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	keyword := strings.TrimSpace(strings.ToLower(req.GetString("keyword", "")))
	category := strings.TrimSpace(req.GetString("category", ""))

	market, err := deps.AppService.FetchMarketplace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("拉取应用商店失败: %v", err)), nil
	}

	var filtered []map[string]interface{}
	for _, it := range market.Apps {
		if category != "" {
			c, _ := it["category"].(string)
			if !strings.EqualFold(c, category) {
				continue
			}
		}

		if keyword != "" {
			id, _ := it["id"].(string)
			name, _ := it["name"].(string)
			desc, _ := it["description"].(string)
			author, _ := it["author"].(string)
			cat, _ := it["category"].(string)

			text := strings.ToLower(fmt.Sprintf("%s %s %s %s %s", id, name, desc, author, cat))
			if !strings.Contains(text, keyword) {
				continue
			}
		}

		item := map[string]interface{}{
			"id":          it["id"],
			"name":        it["name"],
			"version":     it["version"],
			"author":      it["author"],
			"category":    it["category"],
			"description": it["description"],
			"homepage":    it["homepage"],
			"last_commit": it["last_commit"],
			"template":    it["template"],
		}
		if scenarios, ok := it["scenarios"].([]interface{}); ok {
			item["scenarios_count"] = len(scenarios)
		}
		if tasks, ok := it["tasks"].([]interface{}); ok {
			item["tasks_count"] = len(tasks)
		}
		filtered = append(filtered, item)
	}

	resData := map[string]interface{}{
		"total": len(filtered),
		"apps":  filtered,
	}
	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 2. get_store_app: 获取应用商店应用完整规范与详情
func registerGetStoreAppTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("get_store_app",
		mcp.WithDescription("获取应用商店中指定应用的完整定义与元数据，包含依赖语言、预置任务清单、环境变量契约与可用场景"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("应用唯一标识符（如 'bilibili-tool-pro', 'jdpro', 'ark'）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetStoreApp(ctx, deps, req)
	})
}

func handleGetStoreApp(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	appID := strings.TrimSpace(req.GetString("app_id", ""))
	if appID == "" {
		return mcp.NewToolResultError("app_id 不能为空"), nil
	}

	manifest, rawMap, err := deps.AppService.GetMarketplaceApp(appID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取应用失败: %v", err)), nil
	}

	resData := map[string]interface{}{
		"id":            manifest.ID,
		"name":          manifest.Name,
		"version":       manifest.Version,
		"author":        manifest.Author,
		"category":      manifest.Category,
		"description":   manifest.Description,
		"homepage":      manifest.Homepage,
		"icon":          manifest.Icon,
		"template":      rawMap["template"],
		"schedule_opts": manifest.ScheduleOpts,
		"tasks":         manifest.GetTasks(),
		"env_schema":    manifest.EnvSchema,
		"scenarios":     manifest.Scenarios,
	}

	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 3. list_installed_apps: 获取已安装声明式应用列表
func registerListInstalledAppsTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("list_installed_apps",
		mcp.WithDescription("获取当前白虎面板中已安装的声明式应用清单及当前运行场景"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListInstalledApps(ctx, deps, req)
	})
}

func handleListInstalledApps(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	apps, err := deps.AppService.ListApps()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取已安装应用列表失败: %v", err)), nil
	}

	resJSON, _ := json.MarshalIndent(apps, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 4. get_installed_app: 获取已安装应用的配置详情与下属受控子任务
func registerGetInstalledAppTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("get_installed_app",
		mcp.WithDescription("获取已安装应用的配置详情、当前场景以及该应用拆解生成的所有受控子任务列表"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("已安装应用的主任务全局 ID (TaskID) 或 Manifest 标识符")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetInstalledApp(ctx, deps, req)
	})
}

func handleGetInstalledApp(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	appID := strings.TrimSpace(req.GetString("app_id", ""))
	if appID == "" {
		return mcp.NewToolResultError("app_id 不能为空"), nil
	}

	appDTO, err := deps.AppService.GetApp(appID)
	if err != nil {
		var masterTask models.Task
		if dbErr := database.DB.Where("source_id = ? AND type = ?", "app:"+appID, constant.TaskTypeApp).First(&masterTask).Error; dbErr == nil {
			appDTO, err = deps.AppService.GetApp(masterTask.ID)
		}
	}
	if err != nil || appDTO == nil {
		return mcp.NewToolResultError(fmt.Sprintf("未找到已安装应用: %s", appID)), nil
	}

	var childTasks []models.Task
	database.DB.Where("source_id = ? AND type = ?", appDTO.ID, constant.TaskTypeNormal).Order("created_at asc").Find(&childTasks)

	resData := map[string]interface{}{
		"app":         appDTO,
		"child_tasks": childTasks,
	}

	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 5. install_app: 一键部署/安装应用
func registerInstallAppTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("install_app",
		mcp.WithDescription("安装应用商店中的声明式应用，或通过远程/本地 Manifest YAML 一键部署。自动完成代码拉取、环境构建、任务编排与定时注册"),
		mcp.WithString("app_id", mcp.Description("应用商店中的应用唯一标识（如 'bilibili-tool-pro'）")),
		mcp.WithString("path_or_url", mcp.Description("自定义应用 Manifest 的公网 HTTP(S) URL 或本地绝对路径（若提供则优先于 app_id）")),
		mcp.WithString("scenario_id", mcp.Description("初始启用的运行场景 ID（如 'standard', 'minimal'，缺省时使用 Manifest 默认场景）")),
		mcp.WithString("schedule", mcp.Description("覆盖主应用默认 Cron 定时规则表达式（6位标准 Cron）")),
		mcp.WithBoolean("force_setup", mcp.Description("是否强制重新编译/安装环境依赖（默认 false）")),
		mcp.WithBoolean("skip_setup", mcp.Description("是否跳过环境与依赖安装阶段（默认 false）")),
		mcp.WithBoolean("overwrite_env", mcp.Description("是否覆盖同名已有环境变量（默认 false）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleInstallApp(ctx, deps, req)
	})
}

func handleInstallApp(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	appID := strings.TrimSpace(req.GetString("app_id", ""))
	pathOrURL := strings.TrimSpace(req.GetString("path_or_url", ""))

	if appID == "" && pathOrURL == "" {
		return mcp.NewToolResultError("必须指定 app_id 或 path_or_url 之一"), nil
	}

	var buf bytes.Buffer
	opts := app.ApplyOptions{
		ScenarioID:   req.GetString("scenario_id", ""),
		Schedule:     req.GetString("schedule", ""),
		ForceSetup:   req.GetBool("force_setup", false),
		SkipSetup:    req.GetBool("skip_setup", false),
		OverwriteEnv: req.GetBool("overwrite_env", false),
		LogWriter:    &buf,
	}

	var res *app.ApplyResult
	var err error

	if pathOrURL != "" {
		res, err = deps.AppService.ApplyApp(pathOrURL, opts)
	} else {
		res, err = deps.AppService.ApplyMarketplaceApp(appID, opts)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("安装应用失败: %v\n安装日志:\n%s", err, buf.String())), nil
	}

	if deps.ExecutorService != nil && res != nil && res.ID != "" {
		deps.ExecutorService.SyncAppTasks(res.ID)
	}

	resData := map[string]interface{}{
		"success": true,
		"result":  res,
		"log":     buf.String(),
	}
	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 6. switch_app_scenario: 切换应用场景预设
func registerSwitchAppScenarioTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("switch_app_scenario",
		mcp.WithDescription("切换已安装应用的运行场景预设（批量调整子任务的启停状态与覆盖 Cron 定时规则）"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("应用 TaskID 或 Manifest ID")),
		mcp.WithString("scenario_id", mcp.Required(), mcp.Description("目标场景 ID（如 'standard', 'minimal'）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSwitchAppScenario(ctx, deps, req)
	})
}

func handleSwitchAppScenario(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	appID := strings.TrimSpace(req.GetString("app_id", ""))
	scenarioID := strings.TrimSpace(req.GetString("scenario_id", ""))
	if appID == "" || scenarioID == "" {
		return mcp.NewToolResultError("app_id 和 scenario_id 均不能为空"), nil
	}

	targetTaskID := appID
	var masterTask models.Task
	if dbErr := database.DB.Where("id = ? AND type = ?", appID, constant.TaskTypeApp).First(&masterTask).Error; dbErr != nil {
		if dbErr2 := database.DB.Where("source_id = ? AND type = ?", "app:"+appID, constant.TaskTypeApp).First(&masterTask).Error; dbErr2 == nil {
			targetTaskID = masterTask.ID
		}
	}

	var buf bytes.Buffer
	err := deps.AppService.SwitchScenario(targetTaskID, scenarioID, &buf)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("切换场景失败: %v\n执行日志:\n%s", err, buf.String())), nil
	}

	if deps.ExecutorService != nil {
		deps.ExecutorService.SyncAppTasks(targetTaskID)
	}

	resData := map[string]interface{}{
		"success":     true,
		"app_id":      targetTaskID,
		"scenario_id": scenarioID,
		"log":         buf.String(),
	}
	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 7. uninstall_app: 卸载应用
func registerUninstallAppTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("uninstall_app",
		mcp.WithDescription("卸载已安装的声明式应用，级联安全清理其全部受控子任务、关联环境变量、无引用孤儿标签及本地代码产物目录"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("待卸载应用的任务 TaskID 或 Manifest ID")),
		mcp.WithBoolean("clean_data", mcp.Description("是否物理删除本地源码与编译产物目录（默认 true）")),
		mcp.WithBoolean("clean_envs", mcp.Description("是否同时清理应用关联的环境变量（默认 true）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUninstallApp(ctx, deps, req)
	})
}

func handleUninstallApp(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	appID := strings.TrimSpace(req.GetString("app_id", ""))
	if appID == "" {
		return mcp.NewToolResultError("app_id 不能为空"), nil
	}

	cleanData := req.GetBool("clean_data", true)
	cleanEnvs := req.GetBool("clean_envs", true)

	var buf bytes.Buffer
	deletedTaskIDs, err := deps.AppService.RemoveAppWithOptions(appID, app.AppRemoveOptions{
		CleanData: cleanData,
		CleanEnvs: cleanEnvs,
	}, &buf)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("卸载应用失败: %v\n执行日志:\n%s", err, buf.String())), nil
	}

	if deps.ExecutorService != nil {
		for _, tid := range deletedTaskIDs {
			deps.ExecutorService.RemoveCronTask(tid)
			deps.ExecutorService.GetScheduler().StopTask(tid)
		}
	}

	resData := map[string]interface{}{
		"success":          true,
		"deleted_task_ids": deletedTaskIDs,
		"log":              buf.String(),
	}
	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 8. check_app_updates: 检测已装应用的可用更新
func registerCheckAppUpdatesTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("check_app_updates",
		mcp.WithDescription("检测已安装声明式应用的可用更新：比对市场索引或远程清单源的版本与定义差异，返回每个应用的可更新状态与原因"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCheckAppUpdates(ctx, deps, req)
	})
}

func handleCheckAppUpdates(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	infos, err := deps.AppService.CheckUpdates()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("检测应用更新失败: %v", err)), nil
	}

	updatable := 0
	for _, info := range infos {
		if info.Updatable {
			updatable++
		}
	}

	resData := map[string]interface{}{
		"updates":         infos,
		"updatable_count": updatable,
	}
	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}

// 9. update_app: 一键更新已装应用到最新版本
func registerUpdateAppTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("update_app",
		mcp.WithDescription("一键更新已安装的声明式应用：按市场/远程清单最新版本重跑部署流水线，自动保留环境变量值、定时规则、场景与用户定制项。新版本存在缺值的必填环境变量时会被安全阀拦截，需传入 env_values 补值后重放"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("应用 TaskID 或 Manifest ID")),
		mcp.WithObject("env_values", mcp.Description("409 安全阀补值（key 为变量名）：安全阀拦截后按缺失清单补齐再重放")),
		mcp.WithBoolean("skip_setup", mcp.Description("是否跳过环境与依赖安装（默认继承应用已保存构建选项）")),
		mcp.WithBoolean("force_setup", mcp.Description("是否强制重新编译/安装环境依赖（默认继承应用已保存构建选项）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleUpdateApp(ctx, deps, req)
	})
}

func handleUpdateApp(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return runAppUpdateOperation(ctx, deps, req, false)
}

// 10. rollback_app: 回滚到更新前快照版本
func registerRollbackAppTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("rollback_app",
		mcp.WithDescription("将已安装的声明式应用回滚到上一成功版本（更新前自动生成的快照），同样保留全部用户配置与定制项。快照必填变量缺值时会被安全阀拦截，需传入 env_values 补值后重放"),
		mcp.WithString("app_id", mcp.Required(), mcp.Description("应用 TaskID 或 Manifest ID")),
		mcp.WithObject("env_values", mcp.Description("409 安全阀补值（key 为变量名）：安全阀拦截后按缺失清单补齐再重放")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRollbackApp(ctx, deps, req)
	})
}

func handleRollbackApp(ctx context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return runAppUpdateOperation(ctx, deps, req, true)
}

// runAppUpdateOperation 更新/回滚共用编排：解析参数 → 调用服务 → 安全阀错误转结构化提示 → 同步调度
func runAppUpdateOperation(ctx context.Context, deps *Deps, req mcp.CallToolRequest, rollback bool) (*mcp.CallToolResult, error) {
	if deps.AppService == nil {
		return mcp.NewToolResultError("AppService 未初始化"), nil
	}

	appID := strings.TrimSpace(req.GetString("app_id", ""))
	if appID == "" {
		return mcp.NewToolResultError("app_id 不能为空"), nil
	}

	envValues := make(map[string]string)
	if raw, ok := req.GetArguments()["env_values"].(map[string]interface{}); ok {
		for k, v := range raw {
			if s, _ := v.(string); strings.TrimSpace(s) != "" {
				envValues[k] = s
			}
		}
	}

	opts := app.AppUpdateOptions{
		EnvValues:  envValues,
		SkipSetup:  req.GetBool("skip_setup", false),
		ForceSetup: req.GetBool("force_setup", false),
	}

	var buf bytes.Buffer
	opName := "更新"
	if rollback {
		opName = "回滚"
	}

	var res *app.ApplyResult
	var err error
	if rollback {
		res, err = deps.AppService.RollbackApp(appID, opts, &buf)
	} else {
		res, err = deps.AppService.UpdateApp(appID, opts, &buf)
	}

	if err != nil {
		var conflict *app.AppUpdateConflictError
		if errors.As(err, &conflict) {
			// 安全阀拦截：回传缺失清单，引导补值后重放（破坏性变更永不静默执行）
			resData := map[string]interface{}{
				"success":      false,
				"blocked":      "missing_required_envs",
				"error":        err.Error(),
				"missing_envs": conflict.MissingEnvs,
				"log":          buf.String(),
			}
			resJSON, _ := json.MarshalIndent(resData, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("%s被安全阀拦截，请按 missing_envs 补齐 env_values 后重放：\n%s", opName, string(resJSON))), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("%s应用失败: %v\n执行日志:\n%s", opName, err, buf.String())), nil
	}

	if deps.ExecutorService != nil && res != nil && res.ID != "" {
		deps.ExecutorService.SyncAppTasks(res.ID)
	}

	resData := map[string]interface{}{
		"success": true,
		"result":  res,
		"log":     buf.String(),
	}
	resJSON, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(resJSON)), nil
}
