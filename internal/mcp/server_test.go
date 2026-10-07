package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
)

func setupTestDB(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "baihu-mcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	dbCfg := &database.Config{
		Type: "sqlite",
		Path: dbPath,
	}

	if err := database.Init(dbCfg); err != nil {
		t.Fatalf("Failed to init test db: %v", err)
	}

	if err := database.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test db: %v", err)
	}

	return tempDir
}

func makeCallToolReq(name string, args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Name = name
	req.Params.Arguments = args
	return req
}

// 1. 服务初始化与注册完整性测试
func TestNewBaihuMCPServer(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	scriptsDir := filepath.Join(tempDir, "scripts")
	_ = os.MkdirAll(scriptsDir, 0755)

	deps := InitDefaultDeps()
	deps.FileWorkDir = scriptsDir

	s := NewBaihuMCPServer(deps)
	if s == nil {
		t.Fatal("NewBaihuMCPServer returned nil")
	}

	tools := s.ListTools()
	expectedTools := []string{
		"get_system_status",
		"list_tasks",
		"get_task",
		"create_task",
		"update_task",
		"delete_task",
		"execute_task",
		"stop_task",
		"list_env_vars",
		"set_env_var",
		"delete_env_var",
		"list_task_logs",
		"get_log_detail",
		"clean_task_logs",
		"get_file_tree",
		"read_script",
		"save_script",
		"delete_script",
		"search_scripts",
		"list_store_apps",
		"get_store_app",
		"list_installed_apps",
		"get_installed_app",
		"install_app",
		"switch_app_scenario",
		"uninstall_app",
		"list_repos",
		"sync_repo",
		"list_notify_channels",
		"test_notification",
		"send_notification",
	}

	for _, toolName := range expectedTools {
		if _, ok := tools[toolName]; !ok {
			t.Errorf("Expected tool %s to be registered", toolName)
		}
	}

	prompts := s.ListPrompts()
	if _, ok := prompts["diagnose_task_failure"]; !ok {
		t.Error("Expected prompt diagnose_task_failure to be registered")
	}
	if _, ok := prompts["create_scheduled_task"]; !ok {
		t.Error("Expected prompt create_scheduled_task to be registered")
	}

	resources := s.ListResources()
	if _, ok := resources["baihu://status"]; !ok {
		t.Error("Expected resource baihu://status to be registered")
	}
}

// 2. 任务全生命周期工具测试 (create -> get -> update -> execute -> stop -> delete)
func TestTaskLifecycleTools(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()
	deps.FileWorkDir = filepath.Join(tempDir, "scripts")
	_ = os.MkdirAll(deps.FileWorkDir, 0755)
	s := NewBaihuMCPServer(deps)
	ctx := context.Background()

	// 2.1 创建任务
	createTool := s.GetTool("create_task")
	createRes, err := createTool.Handler(ctx, makeCallToolReq("create_task", map[string]any{
		"name":     "生命周期测试任务",
		"command":  "echo hello",
		"schedule": "0 0 10 * * *",
		"remark":   "测试任务说明",
	}))
	if err != nil || createRes.IsError {
		t.Fatalf("create_task failed: %v", err)
	}

	// 提取创建的任务
	taskList := deps.TaskService.GetTasks()
	if len(taskList) == 0 {
		t.Fatal("Task not found in db after create_task")
	}
	taskID := taskList[0].ID

	// 2.2 获取任务详情
	getTool := s.GetTool("get_task")
	getRes, err := getTool.Handler(ctx, makeCallToolReq("get_task", map[string]any{
		"id": taskID,
	}))
	if err != nil || getRes.IsError {
		t.Fatalf("get_task failed: %v", err)
	}

	// 2.3 更新任务
	updateTool := s.GetTool("update_task")
	updateRes, err := updateTool.Handler(ctx, makeCallToolReq("update_task", map[string]any{
		"id":       taskID,
		"name":     "已更名的测试任务",
		"command":  "echo updated",
		"schedule": "0 30 10 * * *",
	}))
	if err != nil || updateRes.IsError {
		t.Fatalf("update_task failed: %v", err)
	}

	// 2.4 触发执行任务
	execTool := s.GetTool("execute_task")
	execRes, err := execTool.Handler(ctx, makeCallToolReq("execute_task", map[string]any{
		"id": taskID,
	}))
	if err != nil || execRes.IsError {
		t.Fatalf("execute_task failed: %v", err)
	}

	// 2.5 停止任务
	stopTool := s.GetTool("stop_task")
	stopRes, err := stopTool.Handler(ctx, makeCallToolReq("stop_task", map[string]any{
		"task_id": taskID,
	}))
	if err != nil || stopRes.IsError {
		t.Fatalf("stop_task failed: %v", err)
	}

	// 2.6 删除任务
	deleteTool := s.GetTool("delete_task")
	delRes, err := deleteTool.Handler(ctx, makeCallToolReq("delete_task", map[string]any{
		"id": taskID,
	}))
	if err != nil || delRes.IsError {
		t.Fatalf("delete_task failed: %v", err)
	}
}

// 3. 环境变量全生命周期工具测试 (set新建 -> set更新 -> list -> delete)
func TestEnvVarLifecycleTools(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()
	s := NewBaihuMCPServer(deps)
	ctx := context.Background()

	// 3.1 新建普通变量
	setTool := s.GetTool("set_env_var")
	setRes, err := setTool.Handler(ctx, makeCallToolReq("set_env_var", map[string]any{
		"name":   "APP_KEY",
		"value":  "initial_value",
		"remark": "初次写入",
		"tags":   "test,mcp",
	}))
	if err != nil || setRes.IsError {
		t.Fatalf("set_env_var initial failed: %v", err)
	}

	// 3.2 更新同名变量
	setRes, err = setTool.Handler(ctx, makeCallToolReq("set_env_var", map[string]any{
		"name":   "APP_KEY",
		"value":  "updated_value",
		"remark": "更新后",
		"tags":   "test,prod",
	}))
	if err != nil || setRes.IsError {
		t.Fatalf("set_env_var update failed: %v", err)
	}

	// 3.3 列表筛选查询
	listTool := s.GetTool("list_env_vars")
	listRes, err := listTool.Handler(ctx, makeCallToolReq("list_env_vars", map[string]any{
		"name": "APP_KEY",
	}))
	if err != nil || listRes.IsError {
		t.Fatalf("list_env_vars failed: %v", err)
	}

	adminID := deps.EnvService.GetDefaultAdminUserID()
	envs, _ := deps.EnvService.GetEnvVarsWithPagination(adminID, "APP_KEY", "", "", 1, 10)
	if len(envs) == 0 {
		t.Fatal("Expected env var to exist in db")
	}
	envID := envs[0].ID

	// 3.4 删除环境变量
	delTool := s.GetTool("delete_env_var")
	delRes, err := delTool.Handler(ctx, makeCallToolReq("delete_env_var", map[string]any{
		"id":    envID,
		"force": true,
	}))
	if err != nil || delRes.IsError {
		t.Fatalf("delete_env_var failed: %v", err)
	}
}

// 4. 文件与脚本管理工具测试 (save -> read -> tree -> search -> delete)
func TestFileManagementTools(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	scriptsDir := filepath.Join(tempDir, "scripts")
	_ = os.MkdirAll(scriptsDir, 0755)

	deps := InitDefaultDeps()
	deps.FileWorkDir = scriptsDir
	s := NewBaihuMCPServer(deps)
	ctx := context.Background()

	// 4.1 保存脚本
	saveTool := s.GetTool("save_script")
	saveRes, err := saveTool.Handler(ctx, makeCallToolReq("save_script", map[string]any{
		"path":    "demo/hello.py",
		"content": "print('hello python')",
	}))
	if err != nil || saveRes.IsError {
		t.Fatalf("save_script failed: %v", err)
	}

	// 4.2 读取脚本
	readTool := s.GetTool("read_script")
	readRes, err := readTool.Handler(ctx, makeCallToolReq("read_script", map[string]any{
		"path": "demo/hello.py",
	}))
	if err != nil || readRes.IsError {
		t.Fatalf("read_script failed: %v", err)
	}

	// 4.3 获取目录树
	treeTool := s.GetTool("get_file_tree")
	treeRes, err := treeTool.Handler(ctx, makeCallToolReq("get_file_tree", map[string]any{
		"path": "",
	}))
	if err != nil || treeRes.IsError {
		t.Fatalf("get_file_tree failed: %v", err)
	}

	// 4.4 模糊检索文件
	searchTool := s.GetTool("search_scripts")
	searchRes, err := searchTool.Handler(ctx, makeCallToolReq("search_scripts", map[string]any{
		"keyword": "hello",
	}))
	if err != nil || searchRes.IsError {
		t.Fatalf("search_scripts failed: %v", err)
	}

	// 4.5 删除脚本
	delTool := s.GetTool("delete_script")
	delRes, err := delTool.Handler(ctx, makeCallToolReq("delete_script", map[string]any{
		"path": "demo/hello.py",
	}))
	if err != nil || delRes.IsError {
		t.Fatalf("delete_script failed: %v", err)
	}
}

// 5. 日志查询工具测试 (list_task_logs -> get_log_detail)
func TestLogInspectionTools(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()
	s := NewBaihuMCPServer(deps)
	ctx := context.Background()

	// 模拟写入一条失败日志
	now := models.Now()
	testLog := &models.TaskLog{
		ID:        "log_test_001",
		TaskID:    "task_test_001",
		TaskName:  "模拟执行任务",
		Command:   models.BigText("node not_found.js"),
		Output:    models.BigText("raw:Error: Cannot find module"),
		Error:     models.BigText("exit status 1"),
		Status:    constant.TaskStatusFailed,
		Duration:  3,
		ExitCode:  1,
		StartTime: &now,
		EndTime:   &now,
	}
	database.DB.Create(testLog)

	// 5.1 分页查询日志
	listTool := s.GetTool("list_task_logs")
	listRes, err := listTool.Handler(ctx, makeCallToolReq("list_task_logs", map[string]any{
		"task_id": "task_test_001",
		"status":  constant.TaskStatusFailed,
	}))
	if err != nil || listRes.IsError {
		t.Fatalf("list_task_logs failed: %v", err)
	}

	// 5.2 获取日志详情
	getTool := s.GetTool("get_log_detail")
	getRes, err := getTool.Handler(ctx, makeCallToolReq("get_log_detail", map[string]any{
		"log_id": "log_test_001",
	}))
	if err != nil || getRes.IsError {
		t.Fatalf("get_log_detail failed: %v", err)
	}
}

// 6. 静态资源与动态模板资源读取测试
func TestResourcesReading(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	scriptsDir := filepath.Join(tempDir, "scripts")
	_ = os.MkdirAll(scriptsDir, 0755)

	deps := InitDefaultDeps()
	deps.FileWorkDir = scriptsDir
	s := NewBaihuMCPServer(deps)

	// 写入测试文件与测试数据
	_ = deps.FileService.SaveFileContent("config.json", `{"test":true}`)

	task := deps.TaskService.CreateTask(&tasks.TaskParam{
		Name:    "资源测试任务",
		Command: "node app.js",
	})

	now := models.Now()
	testLog := &models.TaskLog{
		ID:        "log_res_001",
		TaskID:    task.ID,
		TaskName:  task.Name,
		Command:   models.BigText("node app.js"),
		Output:    models.BigText("raw:Execution finished successfully"),
		Status:    constant.TaskStatusSuccess,
		StartTime: &now,
		EndTime:   &now,
	}
	database.DB.Create(testLog)

	repoTask := &models.Task{
		ID:       "repo_res_001",
		Name:     "资源测试仓库",
		Type:     constant.TaskTypeRepo,
		Command:  "echo repo",
		Schedule: "0 0 4 * * *",
	}
	database.DB.Create(repoTask)

	// 测试标准 JSON-RPC resources/read
	testURIs := []string{
		"baihu://status",
		"notify://channels",
		"task://" + task.ID,
		"repo://" + repoTask.ID,
		"log://log_res_001",
		"file://config.json",
	}

	for _, uri := range testURIs {
		reqJSON, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "resources/read",
			"params": map[string]any{
				"uri": uri,
			},
		})

		resp := s.HandleMessage(context.Background(), reqJSON)
		respBytes, _ := json.Marshal(resp)
		if strings.Contains(string(respBytes), `"error"`) {
			t.Errorf("resources/read failed for URI %s, resp: %s", uri, string(respBytes))
		}
	}
}

// 7. 智能提示词模板测试 (prompts/get)
func TestPromptsGeneration(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()
	s := NewBaihuMCPServer(deps)

	// 准备一条排查日志
	now := models.Now()
	testLog := &models.TaskLog{
		ID:        "log_prompt_001",
		TaskID:    "task_prompt_001",
		TaskName:  "异常任务排查",
		Command:   models.BigText("python -m invalid_module"),
		Output:    models.BigText("raw:ModuleNotFoundError: No module named 'invalid_module'"),
		Error:     models.BigText("ModuleNotFoundError"),
		Status:    constant.TaskStatusFailed,
		StartTime: &now,
		EndTime:   &now,
	}
	database.DB.Create(testLog)

	// 7.1 测试 diagnose_task_failure prompt
	reqJSON, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "prompts/get",
		"params": map[string]any{
			"name": "diagnose_task_failure",
			"arguments": map[string]string{
				"log_id": "log_prompt_001",
			},
		},
	})
	resp := s.HandleMessage(context.Background(), reqJSON)
	respBytes, _ := json.Marshal(resp)
	if strings.Contains(string(respBytes), `"error"`) || !strings.Contains(string(respBytes), "ModuleNotFoundError") {
		t.Errorf("diagnose_task_failure prompt failed: %s", string(respBytes))
	}

	// 7.2 测试 create_scheduled_task prompt
	reqJSON2, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "prompts/get",
		"params": map[string]any{
			"name": "create_scheduled_task",
			"arguments": map[string]string{
				"task_description": "每天凌晨2点拉取最新代码并构建",
			},
		},
	})
	resp2 := s.HandleMessage(context.Background(), reqJSON2)
	respBytes2, _ := json.Marshal(resp2)
	if strings.Contains(string(respBytes2), `"error"`) || !strings.Contains(string(respBytes2), "每天凌晨2点") {
		t.Errorf("create_scheduled_task prompt failed: %s", string(respBytes2))
	}
}

// 8. SSE HTTP 端点与 OpenAPI Token 鉴权集成测试
func TestSSETransportIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()

	// 初始化一个管理员用户与 OpenAPI Token
	adminUser := models.User{
		ID:       "admin_001",
		Username: "admin",
		Role:     constant.AdminRole,
	}
	database.DB.Create(&adminUser)

	validToken := "sk-baihu-mcp-test-token"
	settingsSvc := services.NewSettingsService()
	tokenCfg := vo.TokenConfig{
		Enabled: true,
		Token:   validToken,
	}
	tokenBytes, _ := json.Marshal(tokenCfg)
	settingsSvc.Set(constant.SectionSite, constant.KeyOpenapiToken, string(tokenBytes))

	// 构造测试路由
	r := gin.New()
	v1 := r.Group("/open2api/v1")
	v1.Use(middleware.OpenapiRequired())
	RegisterOpenAPIMCPRoutes(v1, deps)

	// 8.1 未携带 Token 访问 SSE -> 401 Unauthorized
	reqUnauthorized, _ := http.NewRequest(http.MethodGet, "/open2api/v1/mcp/sse", nil)
	wUnauthorized := httptest.NewRecorder()
	r.ServeHTTP(wUnauthorized, reqUnauthorized)
	if wUnauthorized.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for unauthorized SSE request, got %d", wUnauthorized.Code)
	}

	// 8.2 携带有效 URL Query ?token=xxx 访问 SSE -> 成功建立 SSE 连接 (返回 200 与 text/event-stream)
	reqAuthorized, _ := http.NewRequest(http.MethodGet, "/open2api/v1/mcp/sse?token="+validToken, nil)
	wAuthorized := httptest.NewRecorder()

	// 异步监听以防长时间保持挂起
	go r.ServeHTTP(wAuthorized, reqAuthorized)
	time.Sleep(100 * time.Millisecond)

	contentType := wAuthorized.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") && wAuthorized.Code != http.StatusOK {
		t.Errorf("Expected SSE content-type text/event-stream, got code=%d header=%s", wAuthorized.Code, contentType)
	}
}

// 9. 安全沙箱路径穿透防护测试
func TestSafePath(t *testing.T) {
	baseDir := filepath.Join(os.TempDir(), "baihu-safe-test")
	_ = os.MkdirAll(baseDir, 0755)
	defer os.RemoveAll(baseDir)

	fileSvc := services.NewFileService(baseDir)

	clean, safe := fileSvc.CheckPath("test.js", false)
	if !safe || !filepath.IsAbs(clean) {
		t.Errorf("Expected test.js to be safe, got safe=%v, clean=%s", safe, clean)
	}

	_, safe = fileSvc.CheckPath("../../etc/passwd", false)
	if safe {
		t.Error("Expected traversal path to be unsafe")
	}

	_, safe = fileSvc.CheckPath(".", false)
	if safe {
		t.Error("Expected root path to be unsafe when allowRoot=false")
	}

	_, safe = fileSvc.CheckPath(".", true)
	if !safe {
		t.Error("Expected root path to be safe when allowRoot=true")
	}
}

// 10. 路径规范化测试
func TestTaskParamNormalization(t *testing.T) {
	normalized := constant.NormalizeScriptPath("apps/test-app")
	if normalized == "" {
		t.Error("Normalized path should not be empty")
	}
}

// 11. 验证 SSE 会话最大并发连接协程限制 (超过 3 个活跃连接自动 429 熔断保护系统)
func TestSSESessionLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()
	r := gin.New()
	RegisterOpenAPIMCPRoutes(r.Group("/open2api/v1"), deps)

	// 并发模拟占用 MaxActiveSSESessions 个连接
	for i := 0; i < MaxActiveSSESessions; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/open2api/v1/mcp/sse", nil)
		w := httptest.NewRecorder()
		go r.ServeHTTP(w, req)
	}
	time.Sleep(50 * time.Millisecond)

	// 第 4 个连接发起请求，验证被 429 拦截
	reqExceeded, _ := http.NewRequest(http.MethodGet, "/open2api/v1/mcp/sse", nil)
	wExceeded := httptest.NewRecorder()
	r.ServeHTTP(wExceeded, reqExceeded)

	if wExceeded.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 Too Many Requests when exceeding session limit, got %d", wExceeded.Code)
	}
}

// 12. 声明式应用与应用商店生命周期工具测试
func TestAppLifecycleTools(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()
	s := NewBaihuMCPServer(deps)
	ctx := context.Background()

	// 12.1 测试 list_installed_apps
	listTool := s.GetTool("list_installed_apps")
	listRes, err := listTool.Handler(ctx, makeCallToolReq("list_installed_apps", nil))
	if err != nil || listRes.IsError {
		t.Fatalf("list_installed_apps failed: %v", err)
	}

	// 12.2 创建模拟声明式应用
	manifestRaw := `spec_version: "v1"
id: "test-mcp-app"
name: "MCP测试应用"
version: "1.0.0"
author: "TestAuthor"
category: "运维管理"
tasks:
  - id: "worker"
    name: "工作者子任务"
    command: "echo worker"
    default_cron: "0 0 12 * * *"
    enabled: true
scenarios:
  - id: "standard"
    name: "标准模式"
    default: true
    task_presets:
      worker:
        enabled: true
  - id: "minimal"
    name: "精简模式"
    task_presets:
      worker:
        enabled: false
`
	enabledVal := true
	masterApp := &models.Task{
		ID:       "app_mcp_test_001",
		Name:     "MCP测试应用",
		Type:     constant.TaskTypeApp,
		SourceID: "app:test-mcp-app",
		Schedule: "0 0 8 * * *",
		WorkDir:  filepath.Join(tempDir, "data", "scripts", "apps", "testauthor-test-mcp-app"),
		Enabled:  &enabledVal,
	}
	appCfg := &models.AppTaskConfig{
		ID:              "test-mcp-app",
		Name:            "MCP测试应用",
		Version:         "1.0.0",
		Author:          "TestAuthor",
		Category:        "运维管理",
		ManifestRaw:     manifestRaw,
		CurrentScenario: "standard",
		Status:          "installed",
	}
	u := models.UnifiedTaskConfig{App: appCfg}
	masterApp.UnifiedConfig = models.BigText(u.ToJSON())
	database.DB.Create(masterApp)

	childTask := &models.Task{
		ID:       "task_mcp_child_001",
		Name:     "工作者子任务",
		Type:     constant.TaskTypeNormal,
		SourceID: masterApp.ID,
		Schedule: "0 0 12 * * *",
		Command:  "echo worker",
		Enabled:  &enabledVal,
	}
	database.DB.Create(childTask)

	// 12.3 获取已安装应用详情 get_installed_app
	getTool := s.GetTool("get_installed_app")
	getRes, err := getTool.Handler(ctx, makeCallToolReq("get_installed_app", map[string]any{
		"app_id": masterApp.ID,
	}))
	if err != nil || getRes.IsError {
		t.Fatalf("get_installed_app failed: %v", err)
	}

	// 12.4 读取 app://{id} 资源
	reqJSON, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/read",
		"params": map[string]any{
			"uri": "app://" + masterApp.ID,
		},
	})
	resp := s.HandleMessage(ctx, reqJSON)
	respBytes, _ := json.Marshal(resp)
	if strings.Contains(string(respBytes), `"error"`) || !strings.Contains(string(respBytes), "MCP测试应用") {
		t.Errorf("resources/read failed for app resource: %s", string(respBytes))
	}

	// 12.5 切换场景 switch_app_scenario
	switchTool := s.GetTool("switch_app_scenario")
	switchRes, err := switchTool.Handler(ctx, makeCallToolReq("switch_app_scenario", map[string]any{
		"app_id":      masterApp.ID,
		"scenario_id": "minimal",
	}))
	if err != nil || switchRes.IsError {
		t.Fatalf("switch_app_scenario failed: %v", err)
	}

	// 12.6 卸载应用 uninstall_app
	delTool := s.GetTool("uninstall_app")
	delRes, err := delTool.Handler(ctx, makeCallToolReq("uninstall_app", map[string]any{
		"app_id": masterApp.ID,
	}))
	if err != nil || delRes.IsError {
		t.Fatalf("uninstall_app failed: %v", err)
	}
}

// 13. 代码仓库同步、消息通知与日志维护测试
func TestRepoAndNotifyLifecycleTools(t *testing.T) {
	tempDir := setupTestDB(t)
	defer os.RemoveAll(tempDir)

	deps := InitDefaultDeps()
	s := NewBaihuMCPServer(deps)
	ctx := context.Background()

	// 13.1 创建模拟代码仓库任务 (TaskTypeRepo)
	repoTask := &models.Task{
		ID:       "repo_mcp_001",
		Name:     "测试代码仓库",
		Type:     constant.TaskTypeRepo,
		Command:  "echo repo sync",
		Schedule: "0 0 4 * * *",
	}
	repoCfg := &models.RepoConfig{
		SourceType: "git",
		SourceURL:  "https://github.com/engigu/baihu-test.git",
		Branch:     "main",
		TargetPath: "test_repo",
	}
	u := models.UnifiedTaskConfig{Repo: repoCfg}
	repoTask.UnifiedConfig = models.BigText(u.ToJSON())
	database.DB.Create(repoTask)

	// 13.2 测试 list_repos
	listRepoTool := s.GetTool("list_repos")
	listRepoRes, err := listRepoTool.Handler(ctx, makeCallToolReq("list_repos", nil))
	if err != nil || listRepoRes.IsError {
		t.Fatalf("list_repos failed: %v", err)
	}

	// 13.3 测试 sync_repo
	syncTool := s.GetTool("sync_repo")
	syncRes, err := syncTool.Handler(ctx, makeCallToolReq("sync_repo", map[string]any{
		"repo_id": "repo_mcp_001",
	}))
	if err != nil || syncRes.IsError {
		t.Fatalf("sync_repo failed: %v", err)
	}

	// 13.4 测试通知渠道 list_notify_channels
	listNotifyTool := s.GetTool("list_notify_channels")
	listNotifyRes, err := listNotifyTool.Handler(ctx, makeCallToolReq("list_notify_channels", nil))
	if err != nil || listNotifyRes.IsError {
		t.Fatalf("list_notify_channels failed: %v", err)
	}

	// 13.5 保存一个虚拟通知渠道用于测试
	_ = deps.NotifyService.SaveChannel(services.NotifyChannel{
		Name:    "MCP测试Server酱",
		Type:    "serverchan",
		Enabled: false,
		Config: map[string]string{
			"sendkey": "sctp_test_key_123456",
		},
	})

	// 13.6 测试 clean_task_logs
	cleanTool := s.GetTool("clean_task_logs")
	cleanRes, err := cleanTool.Handler(ctx, makeCallToolReq("clean_task_logs", map[string]any{
		"task_id": "repo_mcp_001",
		"days":    0,
	}))
	if err != nil || cleanRes.IsError {
		t.Fatalf("clean_task_logs failed: %v", err)
	}
}


