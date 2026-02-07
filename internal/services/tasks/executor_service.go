package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AgentWSManager 接口定义（避免循环依赖）
type AgentWSManager interface {
	RegisterRemoteWaiter(logID uint) chan *models.AgentTaskResult
	UnregisterRemoteWaiter(logID uint)
	SendToAgent(agentID uint, msgType string, data interface{}) error
}

// SettingsService 接口定义（避免循环依赖）
type SettingsService interface {
	Get(section, key string) string
}

// EnvService 接口定义（避免循环依赖）
type EnvService interface {
	GetEnvVarsByIDs(ids string) []string
}

// ExecutorService handles task execution and scheduling
type ExecutorService struct {
	taskService     *TaskService
	taskLogService  *TaskLogService
	agentWSManager  AgentWSManager
	settingsService SettingsService
	envService      EnvService
	scheduler       *executor.Scheduler
	cronManager     *executor.CronManager
	results         []executor.ExecutionResult
	mu              sync.RWMutex
	resultsMu       sync.RWMutex
	stopCh          chan struct{}
}

func (es *ExecutorService) GetScheduler() *executor.Scheduler {
	return es.scheduler
}

// NewExecutorService creates a new executor service
func NewExecutorService(
	taskService *TaskService,
	taskLogService *TaskLogService,
	agentWSManager AgentWSManager,
	settingsService SettingsService,
	envService EnvService,
) *ExecutorService {
	es := &ExecutorService{
		taskService:     taskService,
		taskLogService:  taskLogService,
		agentWSManager:  agentWSManager,
		settingsService: settingsService,
		envService:      envService,
		results:         make([]executor.ExecutionResult, 0, 100),
		stopCh:          make(chan struct{}),
	}

	// 1. 初始化调度器
	es.initScheduler()

	// 2. 初始化计划任务管理器
	es.cronManager = executor.NewCronManager(es.scheduler)

	return es
}

func (es *ExecutorService) initScheduler() {
	workerCount := getIntSetting(es.settingsService, constant.SectionScheduler, constant.KeyWorkerCount, 4)
	queueSize := getIntSetting(es.settingsService, constant.SectionScheduler, constant.KeyQueueSize, 100)
	rateInterval := getIntSetting(es.settingsService, constant.SectionScheduler, constant.KeyRateInterval, 200)

	config := executor.SchedulerConfig{
		WorkerCount:  workerCount,
		QueueSize:    queueSize,
		RateInterval: time.Duration(rateInterval) * time.Millisecond,
	}

	handler := &ServerSchedulerHandler{es: es}
	es.scheduler = executor.NewScheduler(config, handler)
	es.scheduler.SetLogger(logger.NewSchedulerLogger())
	es.scheduler.SetExecutor(es.ExecuteDispatcher)
	es.scheduler.Start()

	logger.Infof("[Executor] 调度器已启动: workers=%d, queue=%d, rate=%dms", workerCount, queueSize, rateInterval)
}

// ServerSchedulerHandler 实现 executor.SchedulerEventHandler
type ServerSchedulerHandler struct {
	es *ExecutorService
}

func (h *ServerSchedulerHandler) OnTaskScheduled(req *executor.ExecutionRequest) {
	// 任务入队事件，可以在此处更新数据库状态为 "pending"
}

func (h *ServerSchedulerHandler) OnTaskExecuting(req *executor.ExecutionRequest) (io.Writer, io.Writer, error) {
	var taskID uint
	fmt.Sscanf(req.TaskID, "%d", &taskID)

	task := h.es.taskService.GetTaskByID(int(taskID))
	// 系统任务（无 taskID）不记录数据库日志，直接返回空写入器
	if task == nil {
		return nil, nil, nil
	}

	// 1. 创建初始日志记录
	taskLog, err := h.es.taskLogService.CreateEmptyLog(task.ID, task.Command)
	if err != nil {
		return nil, nil, fmt.Errorf("创建初始日志失败: %v", err)
	}
	req.LogID = taskLog.ID // 设置 LogID 供后续环节使用

	// 2. 检查并记录运行状态（并发控制）
	goid, err := h.es.AddRunningGo(task.ID)
	if err != nil {
		// 并发限制，更新日志状态为失败
		taskLog.Status = "failed"
		taskLog.Output, _ = utils.CompressToBase64("任务并发数限制，拒绝执行")
		h.es.taskLogService.SaveTaskLog(taskLog)
		return nil, nil, fmt.Errorf("任务并发限制: %v", err)
	}

	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	req.Metadata["goid"] = goid

	// 3. 创建 TinyLog 实时日志收集器
	tl, err := NewTinyLog(taskLog.ID)
	if err != nil {
		h.es.RemoveRunningGo(task.ID, goid) // 回滚运行状态
		return nil, nil, fmt.Errorf("创建日志收集器失败: %v", err)
	}

	// 对于本地任务，Scheduler 会通过返回的 Writer 写入日志
	// 对于远程任务，Scheduler 不会写入任何内容（由 Agent 推送至此 TL）
	return tl, tl, nil
}

func (h *ServerSchedulerHandler) OnTaskHeartbeat(req *executor.ExecutionRequest, duration int64) {
	if req.LogID > 0 {
		h.es.taskLogService.UpdateTaskDuration(req.LogID, duration)
	}
}

func (h *ServerSchedulerHandler) OnTaskStarted(req *executor.ExecutionRequest) {
	// Logic moved to OnTaskExecuting
}

func (h *ServerSchedulerHandler) OnTaskCompleted(req *executor.ExecutionRequest, result *executor.ExecutionResult) {
	if req.LogID == 0 {
		return
	}

	var taskID uint
	fmt.Sscanf(req.TaskID, "%d", &taskID)

	task := h.es.taskService.GetTaskByID(int(taskID))
	if task == nil {
		return
	}

	// 无论本地还是远程，都在此处处理日志压缩和落库
	tl := GetActiveLog(req.LogID)
	var output string
	if tl != nil {
		// 压缩并清理实时日志
		var err error
		output, err = tl.CompressAndCleanup()
		if err != nil {
			logger.Errorf("[Executor] 压缩任务 #%d 日志失败: %v", task.ID, err)
			output = "[System Error] 日志处理失败: " + err.Error()
		}
	} else {
		// 如果 TinyLog 已经丢失，尝试从 result.Output 中恢复一次（主要针对本地任务）
		output, _ = utils.CompressToBase64(result.Output)
	}

	// 构造待保存的日志模型
	startTime := models.LocalTime(result.StartTime)
	endTime := models.LocalTime(result.EndTime)

	taskLog := &models.TaskLog{
		ID:        req.LogID,
		TaskID:    task.ID,
		Command:   req.Command,
		Output:    output,
		Status:    result.Status,
		Duration:  result.Duration,
		ExitCode:  result.ExitCode,
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	// 如果有 AgentID，也记录下来
	if task.AgentID != nil && *task.AgentID > 0 {
		agentID := *task.AgentID
		taskLog.AgentID = &agentID
	}

	// 移除运行记录
	if req.Metadata != nil {
		if goid, ok := req.Metadata["goid"].(int64); ok {
			h.es.RemoveRunningGo(task.ID, goid)
		}
	}

	// 处理任务完成（更新统计、清理旧日志等）
	h.es.taskLogService.ProcessTaskCompletion(taskLog)
}

func (h *ServerSchedulerHandler) OnTaskFailed(req *executor.ExecutionRequest, err error) {
	if req.LogID == 0 {
		return
	}

	var taskID uint
	fmt.Sscanf(req.TaskID, "%d", &taskID)

	// 移除运行记录
	if req.Metadata != nil {
		if goid, ok := req.Metadata["goid"].(int64); ok {
			h.es.RemoveRunningGo(taskID, goid)
		}
	}

	// 构造错误日志
	tl := GetActiveLog(req.LogID)
	var output string
	if tl != nil {
		tl.Write([]byte(fmt.Sprintf("\n[System Error] %v", err)))
		output, _ = tl.CompressAndCleanup()
	} else {
		output, _ = utils.CompressToBase64(fmt.Sprintf("任务执行失败: %v", err))
	}

	now := models.LocalTime(time.Now())
	taskLog := &models.TaskLog{
		ID:       req.LogID,
		TaskID:   taskID,
		Output:   output,
		Status:   "failed",
		Duration: 0,
		ExitCode: 1,
		EndTime:  &now,
	}

	// 补充 AgentID
	task := h.es.taskService.GetTaskByID(int(taskID))
	if task != nil && task.AgentID != nil && *task.AgentID > 0 {
		agentID := *task.AgentID
		taskLog.AgentID = &agentID
	}

	h.es.taskLogService.ProcessTaskCompletion(taskLog)
}

func (h *ServerSchedulerHandler) OnCronNextRun(req *executor.ExecutionRequest, nextRun time.Time) {
	var taskID uint
	fmt.Sscanf(req.TaskID, "%d", &taskID)
	// 更新数据库中的下次运行时间
	database.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("next_run", nextRun)
}

// LocalTaskHooks 本地任务钩子适配器
type LocalTaskHooks struct {
	es    *ExecutorService
	logID uint
}

func (h *LocalTaskHooks) PreExecute(ctx context.Context, req executor.Request) (uint, error) {
	return h.logID, nil
}

func (h *LocalTaskHooks) PostExecute(ctx context.Context, logID uint, result *executor.Result) error {
	return nil
}

func (h *LocalTaskHooks) OnHeartbeat(ctx context.Context, logID uint, duration int64) error {
	if logID > 0 {
		return h.es.taskLogService.UpdateTaskDuration(logID, duration)
	}
	return nil
}

// ExecuteDispatcher 实现任务分发逻辑
func (es *ExecutorService) ExecuteDispatcher(ctx context.Context, req *executor.ExecutionRequest, stdout, stderr io.Writer) (*executor.Result, error) {
	var taskID uint
	fmt.Sscanf(req.TaskID, "%d", &taskID)

	task := es.taskService.GetTaskByID(int(taskID))
	// 系统任务（无 taskID）直接本地执行
	if task == nil {
		return executor.Execute(ctx, executor.Request{
			Command: req.Command,
			WorkDir: req.WorkDir,
			Envs:    req.Envs,
			Timeout: req.Timeout,
		}, stdout, stderr)
	}

	// 特殊处理仓库同步任务
	if task.Type == "repo" {
		cmd, workDir := es.BuildRepoCommand(task)
		if cmd != "" {
			req.Command = cmd
			req.WorkDir = workDir
		}
	}

	// 加载环境变量
	if task.Envs != "" {
		req.Envs = append(req.Envs, es.loadEnvVars(task.Envs)...)
	}

	// 远程任务
	if task.AgentID != nil && *task.AgentID > 0 {
		return es.ExecuteRemoteForScheduler(task, req.LogID)
	}

	// 本地任务
	hooks := &LocalTaskHooks{es: es, logID: req.LogID}
	return executor.ExecuteWithHooks(ctx, executor.Request{
		Command: req.Command,
		WorkDir: req.WorkDir,
		Envs:    req.Envs,
		Timeout: req.Timeout,
	}, stdout, stderr, hooks)
}

// getIntSetting 从设置中获取整数值
func getIntSetting(s SettingsService, section, key string, defaultVal int) int {
	val := s.Get(section, key)
	if val == "" {
		return defaultVal
	}
	var result int
	if _, err := fmt.Sscanf(val, "%d", &result); err != nil {
		return defaultVal
	}
	return result
}

// Stop 停止 executor service
func (es *ExecutorService) Stop() {
	es.StopCron()
	es.scheduler.Stop()
}

// StartCron 启动计划任务
func (es *ExecutorService) StartCron() {
	es.loadCronTasks()
	es.cronManager.Start()
	logger.Info("[Executor] 计划任务管理器已启动")
}

// StopCron 停止计划任务
func (es *ExecutorService) StopCron() {
	es.cronManager.Stop()
	logger.Info("[Executor] 计划任务管理器已停止")
}

// AddCronTask 添加计划任务
func (es *ExecutorService) AddCronTask(task *models.Task) error {
	return es.cronManager.AddTask(task)
}

// RemoveCronTask 移除计划任务
func (es *ExecutorService) RemoveCronTask(taskID uint) {
	es.cronManager.RemoveTask(fmt.Sprintf("%d", taskID))
}

// ValidateCron 验证 Cron 表达式
func (es *ExecutorService) ValidateCron(expression string) error {
	return es.cronManager.ValidateCron(expression)
}

// GetScheduledCount 获取已加载的计划任务数量
func (es *ExecutorService) GetScheduledCount() int {
	return es.cronManager.GetScheduledCount()
}

// loadCronTasks 加载所有已启用的本地计划任务
func (es *ExecutorService) loadCronTasks() {
	tasks := es.taskService.GetTasks()
	count := 0
	for _, task := range tasks {
		// 只调度本地任务（agent_id 为空或 0）
		if task.Enabled && (task.AgentID == nil || *task.AgentID == 0) {
			err := es.cronManager.AddTask(&task)
			if err != nil {
				continue
			}
			count++
		}
	}
	logger.Infof("[Executor] 启动调度已加载 %d 个定时任务", count)
}

// Reload 重新加载配置并重建调度器
func (es *ExecutorService) Reload() {
	logger.Info("[Executor] 正在重载配置...")
	es.scheduler.Stop()

	// 从设置中读取新配置
	es.initScheduler()
}

// ExecuteTask executes a task by ID（同步执行，供 API 调用）
func (es *ExecutorService) ExecuteTask(taskID int) *executor.ExecutionResult {
	task := es.taskService.GetTaskByID(taskID)
	if task == nil {
		return &executor.ExecutionResult{
			TaskID:    fmt.Sprintf("%d", taskID),
			Success:   false,
			Error:     "任务不存在",
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	// 1. 检查并发
	if err := es.CheckConcurrency(uint(taskID)); err != nil {
		return &executor.ExecutionResult{
			TaskID:    fmt.Sprintf("%d", taskID),
			Success:   false,
			Error:     err.Error(), // 这里会返回 "任务正在运行中，拒绝并行执行"
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	req := &executor.ExecutionRequest{
		TaskID:  fmt.Sprintf("%d", task.ID),
		Name:    task.Name,
		Command: task.Command,
		WorkDir: task.WorkDir,
		Envs:    es.loadEnvVars(task.Envs),
		Timeout: task.Timeout,
		Type:    executor.TaskTypeManual,
	}

	es.scheduler.EnqueueOrExecute(req)

	return &executor.ExecutionResult{
		TaskID:    fmt.Sprintf("%d", task.ID),
		Success:   true,
		Status:    "queued",
		StartTime: time.Now(),
	}
}

// GetRunningCount 获取正在运行任务数量
func (es *ExecutorService) GetRunningCount() int {
	return es.scheduler.GetRunningTaskCount()
}

// ExecuteCommand executes a shell command with default timeout
func (es *ExecutorService) ExecuteCommand(command string) *executor.ExecutionResult {
	return es.ExecuteCommandWithTimeout(command, time.Duration(constant.DefaultTaskTimeout)*time.Minute)
}

// ExecuteCommandWithTimeout executes a shell command with specified timeout
func (es *ExecutorService) ExecuteCommandWithTimeout(command string, timeout time.Duration) *executor.ExecutionResult {
	return es.ExecuteCommandWithEnv(command, timeout, nil)
}

// ExecuteCommandWithEnv executes a shell command with specified timeout and environment variables
func (es *ExecutorService) ExecuteCommandWithEnv(command string, timeout time.Duration, envVars []string) *executor.ExecutionResult {
	return es.ExecuteCommandWithOptions(command, timeout, envVars, "")
}

// ExecuteCommandWithOptions executes a shell command with specified timeout, environment variables and working directory
func (es *ExecutorService) ExecuteCommandWithOptions(command string, timeout time.Duration, envVars []string, workDir string) *executor.ExecutionResult {
	req := &executor.ExecutionRequest{
		Command: command,
		Timeout: int(timeout.Minutes()),
		Envs:    envVars,
		WorkDir: workDir,
		Type:    executor.TaskTypeSystem,
	}

	res, _ := es.scheduler.ExecuteSync(req)

	// 使用独立锁保存结果
	// TODO: 适配 ExecutionResult 的转换并保存结果

	return res
}

// GetLastResults returns the last execution results
func (es *ExecutorService) GetLastResults(count int) []executor.ExecutionResult {
	es.resultsMu.RLock()
	defer es.resultsMu.RUnlock()
	return nil
}

// --- 以下内容从 TaskExecutionService 合并 ---

// CleanupRunningTasks 清理所有任务的运行状态（在重启时调用）
func (es *ExecutorService) CleanupRunningTasks() error {
	logger.Info("[Executor] 正在清理残留的任务运行状态...")
	return database.DB.Model(&models.Task{}).Where("1=1").Update("running_go", "[]").Error
}

// CheckConcurrency 检查任务并发限制（只读检查）
func (es *ExecutorService) CheckConcurrency(taskID uint) error {
	var task models.Task
	if err := database.DB.Select("config, running_go").First(&task, taskID).Error; err != nil {
		return err
	}
	var goids []int64
	if task.RunningGo != "" {
		_ = json.Unmarshal([]byte(task.RunningGo), &goids)
	}

	var config models.TaskConfig
	if task.Config != "" {
		_ = json.Unmarshal([]byte(task.Config), &config)
	}

	if config.Concurrency == 0 && len(goids) > 0 {
		return fmt.Errorf("任务正在运行中，拒绝并行执行，请前往日志查看")
	}
	return nil
}

// AddRunningGo 添加当前 goroutine ID 到任务的 running_go 字段
func (es *ExecutorService) AddRunningGo(taskID uint) (int64, error) {
	goid := utils.GetGoroutineID()
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var task models.Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			return err
		}
		var goids []int64
		if task.RunningGo != "" {
			_ = json.Unmarshal([]byte(task.RunningGo), &goids)
		}

		// 解析配置以获取并发设置
		var config models.TaskConfig
		if task.Config != "" {
			_ = json.Unmarshal([]byte(task.Config), &config)
		}

		// 如果并发为0(禁用)且已有执行中的任务，返回错误
		if config.Concurrency == 0 && len(goids) > 0 {
			return fmt.Errorf("task is running")
		}

		goids = append(goids, goid)
		data, _ := json.Marshal(goids)
		return tx.Model(&task).Update("running_go", string(data)).Error
	})
	return goid, err
}

// RemoveRunningGo 从任务的 running_go 字段移除指定 goroutine ID
func (es *ExecutorService) RemoveRunningGo(taskID uint, goid int64) {
	database.DB.Transaction(func(tx *gorm.DB) error {
		var task models.Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			return err
		}
		var goids []int64
		if task.RunningGo != "" {
			_ = json.Unmarshal([]byte(task.RunningGo), &goids)
		}
		newGoids := make([]int64, 0)
		for _, id := range goids {
			if id != goid {
				newGoids = append(newGoids, id)
			}
		}
		data, _ := json.Marshal(newGoids)
		return tx.Model(&task).Update("running_go", string(data)).Error
	})
}

// ExecuteRemoteForScheduler 供 Scheduler 调用，执行远程任务并等待结果
func (es *ExecutorService) ExecuteRemoteForScheduler(task *models.Task, logID uint) (*executor.Result, error) {
	agentID := *task.AgentID
	logger.Infof("[Executor] 远程执行任务 #%d: %s (Agent #%d, LogID: %d)", task.ID, task.Name, agentID, logID)

	// 1. 检查 Agent 状态
	var agent models.Agent
	if err := database.DB.First(&agent, agentID).Error; err != nil {
		return nil, fmt.Errorf("Agent #%d 不存在", agentID)
	}
	if !agent.Enabled {
		return nil, fmt.Errorf("Agent #%d 已禁用", agentID)
	}
	if es.agentWSManager == nil {
		return nil, fmt.Errorf("AgentWSManager 未初始化")
	}

	// 2. 注册结果等待者
	resultChan := es.agentWSManager.RegisterRemoteWaiter(logID)
	defer es.agentWSManager.UnregisterRemoteWaiter(logID)

	// 3. 发送指令
	err := es.agentWSManager.SendToAgent(agentID, constant.WSTypeExecute, map[string]interface{}{
		"task_id": task.ID,
		"log_id":  logID,
	})
	if err != nil {
		return nil, fmt.Errorf("发送执行命令失败: %v", err)
	}

	// 4. 等待结果或超时
	timeout := task.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	start := time.Now()
	select {
	case agentResult := <-resultChan:
		return &executor.Result{
			Output:    agentResult.Output,
			Status:    agentResult.Status,
			Duration:  agentResult.Duration,
			ExitCode:  agentResult.ExitCode,
			StartTime: time.Unix(agentResult.StartTime, 0),
			EndTime:   time.Unix(agentResult.EndTime, 0),
		}, nil
	case <-time.After(time.Duration(timeout) * time.Minute):
		end := time.Now()
		return &executor.Result{
			Status:    "failed",
			Duration:  end.Sub(start).Milliseconds(),
			ExitCode:  -1,
			StartTime: start,
			EndTime:   end,
		}, fmt.Errorf("远程执行超时")
	}
}

// HandleAgentResult 处理来自 Agent 的异步结果
func (es *ExecutorService) HandleAgentResult(result *models.AgentTaskResult) error {
	taskLog, err := es.taskLogService.CreateTaskLogFromAgentResult(result)
	if err != nil {
		return err
	}
	return es.taskLogService.ProcessTaskCompletion(taskLog)
}

// BuildRepoCommand 构建仓库同步任务的命令
func (es *ExecutorService) BuildRepoCommand(task *models.Task) (string, string) {
	var config models.RepoConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return "", ""
	}

	targetPath := config.TargetPath
	if targetPath == "" {
		targetPath = constant.ScriptsWorkDir
	} else if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(constant.ScriptsWorkDir, targetPath)
	}
	absTargetPath, _ := filepath.Abs(targetPath)

	args := []string{
		"/opt/sync.py",
		"--source-type", config.SourceType,
		"--source-url", config.SourceURL,
		"--target-path", absTargetPath,
	}
	if config.Branch != "" {
		args = append(args, "--branch", config.Branch)
	}
	if config.SparsePath != "" {
		args = append(args, "--path", config.SparsePath)
	}
	if config.SingleFile {
		args = append(args, "--single-file")
	}
	if config.Proxy != "" && config.Proxy != "none" {
		args = append(args, "--proxy", config.Proxy)
		if config.Proxy == "custom" && config.ProxyURL != "" {
			args = append(args, "--proxy-url", config.ProxyURL)
		}
	}
	if config.AuthToken != "" {
		args = append(args, "--auth-token", config.AuthToken)
	}

	return "python3 " + strings.Join(args, " "), "/opt"
}

// loadEnvVars 加载环境变量
func (es *ExecutorService) loadEnvVars(envIDs string) []string {
	if envIDs == "" {
		return nil
	}

	var envVars []models.EnvironmentVariable
	ids := strings.Split(envIDs, ",")
	database.DB.Where("id IN ?", ids).Find(&envVars)

	result := make([]string, 0, len(envVars))
	for _, env := range envVars {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return result
}
