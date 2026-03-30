package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"gorm.io/gorm"
)

// AgentWSManager 接口定义（避免循环依赖）
type AgentWSManager interface {
	RegisterRemoteWaiter(logID string) chan *models.AgentTaskResult
	UnregisterRemoteWaiter(logID string)
	SendToAgent(agentID string, msgType string, data interface{}) error
}

// SettingsService 接口定义（避免循环依赖）
type SettingsService interface {
	Get(section, key string) string
}

// EnvService 接口定义（避免循环依赖）
type EnvService interface {
	GetEnvVarsByIDs(ids string) []string
	GetAllEnvVars() []string
	GetEnvVarsAndSecretsByIDs(ids string) ([]string, []string)
	GetAllEnvVarsAndSecrets() ([]string, []string)
}

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
	taskID := req.TaskID

	task := h.es.taskService.GetTaskByID(taskID)
	// 系统任务（无 taskID）不记录数据库日志，直接返回空写入器
	if task == nil {
		return nil, nil, nil
	}

	var workflowID *string
	var workflowRunID string

	for _, env := range req.Envs {
		if strings.HasPrefix(env, "BAIHU_WF_ID=") {
			id := strings.TrimPrefix(env, "BAIHU_WF_ID=")
			workflowID = &id
		}
		if strings.HasPrefix(env, "BAIHU_WF_RUN_ID=") {
			workflowRunID = strings.TrimPrefix(env, "BAIHU_WF_RUN_ID=")
		}
	}

	// 1. 创建初始日志记录
	taskLog, err := h.es.taskLogService.CreateEmptyLog(task.ID, req.Command, workflowID, workflowRunID)
	if err != nil {
		return nil, nil, fmt.Errorf("创建初始日志失败: %v", err)
	}
	req.LogID = taskLog.ID // 设置 LogID 供后续环节使用

	// 2. 检查并记录运行状态（并发控制）
	goid, err := h.es.AddRunningGo(task.ID)
	if err != nil {
		// 并发限制，更新日志状态为失败
		taskLog.Status = constant.TaskStatusFailed
		comp, _ := utils.CompressToBase64("任务并发数限制，拒绝执行")
		taskLog.Output = models.BigText(comp)
		h.es.taskLogService.SaveTaskLog(taskLog)
		return nil, nil, fmt.Errorf("任务并发限制: %v", err)
	}

	req.Metadata.GoID = goid

	// 3. 创建 TinyLog 实时日志收集器
	tl, err := NewTinyLog(taskLog.ID, req.Secrets)
	if err != nil {
		h.es.RemoveRunningGo(task.ID, goid) // 回滚运行状态
		return nil, nil, fmt.Errorf("创建日志收集器失败: %v", err)
	}

	// 记录到内存缓冲
	h.es.UpdateResult(executor.ExecutionResult{
		TaskID:    req.TaskID,
		LogID:     req.LogID,
		Status:    constant.TaskStatusRunning,
		StartTime: time.Now(),
	})

	if req.Metadata.RetryIndex > 0 {
		tl.Write([]byte(fmt.Sprintf("\n[System] 此为任务失败后的第 %d 次重试执行...\n\n", req.Metadata.RetryIndex)))
	}

	// 对于本地任务，Scheduler 会通过返回的 Writer 写入日志
	// 对于远程任务，Scheduler 不会写入任何内容（由 Agent 推送至此 TL）
	return tl, tl, nil
}

func (h *ServerSchedulerHandler) OnTaskHeartbeat(req *executor.ExecutionRequest, duration int64) {
	if req.LogID != "" {
		h.es.taskLogService.UpdateTaskDuration(req.LogID, duration)
	}

	// 每分钟打印一次任务还在运行的日志
	if duration >= 60000 && (duration/60000 > (duration-3000)/60000) {
		logger.Infof("[Scheduler] 任务 #%s 仍在运行中... (已耗时: %v)",
			req.TaskID, (time.Duration(duration) * time.Millisecond).Round(time.Second))
	}
}

func (h *ServerSchedulerHandler) OnTaskStarted(req *executor.ExecutionRequest) {
	// Logic moved to OnTaskExecuting
}

func (h *ServerSchedulerHandler) OnTaskCompleted(req *executor.ExecutionRequest, result *executor.ExecutionResult) {
	if req.LogID == "" {
		return
	}

	taskID := req.TaskID

	task := h.es.taskService.GetTaskByID(taskID)
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
			logger.Errorf("[Executor] 压缩任务 #%s 日志失败: %v", task.ID, err)
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
		Command:   models.BigText(req.Command),
		Output:    models.BigText(output),
		Error:     models.BigText(result.Error),
		Status:    result.Status,
		Duration:  result.Duration,
		ExitCode:  result.ExitCode,
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	// 如果有 AgentID，也记录下来
	if task.AgentID != nil && *task.AgentID != "" {
		agentID := *task.AgentID
		taskLog.AgentID = &agentID
	}

	// 移除运行记录
	if req.Metadata.GoID != 0 {
		h.es.RemoveRunningGo(task.ID, req.Metadata.GoID)
	}

	// 处理任务完成（更新统计、清理旧日志等）
	h.es.taskLogService.ProcessTaskCompletion(taskLog)

	// 后置：检查引发 Workflow 的下游节点并发起
	h.es.TriggerWorkflowNextTasks(taskLog)


	// 更新内存缓冲
	h.es.UpdateResult(*result)

	// ======= 重试逻辑 =======
	h.es.HandleTaskRetry(task, req, result.Success, result.Status, result.ExitCode)

	// ======= 通知触发 =======
	// ======= 通知触发 =======
	go func() {
		var eventType string
		switch result.Status {
		case constant.TaskStatusSuccess:
			eventType = constant.EventTaskSuccess
		case constant.TaskStatusFailed:
			eventType = constant.EventTaskFailed
		case constant.TaskStatusTimeout:
			eventType = constant.EventTaskTimeout
		}
		if eventType != "" {
			eventbus.DefaultBus.Publish(eventbus.Event{
				Type: eventType,
				Payload: map[string]interface{}{
					"task_id":    task.ID,
					"task_name":  task.Name,
					"status":     result.Status,
					"start_time": result.StartTime.Format("2006-01-02 15:04:05"),
					"duration":   result.Duration,
					"output":     result.Output,
					"error":      result.Error,
				},
			})
		}
	}()
}

func (h *ServerSchedulerHandler) OnTaskFailed(req *executor.ExecutionRequest, err error) {
	if req.LogID == "" {
		return
	}

	taskID := req.TaskID

	// 移除运行记录
	if req.Metadata.GoID != 0 {
		h.es.RemoveRunningGo(taskID, req.Metadata.GoID)
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
		ID:        req.LogID,
		TaskID:    taskID,
		Command:   models.BigText(req.Command),
		Output:    models.BigText(output),
		Error:     models.BigText(err.Error()),
		Status:    constant.TaskStatusFailed,
		Duration:  0,
		ExitCode:  1,
		StartTime: &now,
		EndTime:   &now,
	}

	// 补充 AgentID
	task := h.es.taskService.GetTaskByID(taskID)
	if task != nil && task.AgentID != nil && *task.AgentID != "" {
		agentID := *task.AgentID
		taskLog.AgentID = &agentID
	}

	h.es.taskLogService.ProcessTaskCompletion(taskLog)

	// 失败时也检查是否存在 fallback 动作的连线
	h.es.TriggerWorkflowNextTasks(taskLog)

	// 更新内存缓冲
	h.es.UpdateResult(executor.ExecutionResult{
		TaskID:    req.TaskID,
		LogID:     req.LogID,
		Status:    constant.TaskStatusFailed,
		Error:     err.Error(),
		StartTime: time.Now(),
		EndTime:   time.Now(),
	})

	// ======= 重试逻辑 =======
	h.es.HandleTaskRetry(task, req, false, constant.TaskStatusFailed, 1)

	// ======= 通知触发 =======
	// ======= 通知触发 =======
	go func() {
		taskName := "未知任务"
		if task != nil {
			taskName = task.Name
		}
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type: constant.EventTaskFailed,
			Payload: map[string]interface{}{
				"task_id":   taskID,
				"task_name": taskName,
				"error":     err.Error(),
				"output":    output,
			},
		})
	}()
}

// HandleTaskRetry 处理任务失败重试逻辑
func (es *ExecutorService) HandleTaskRetry(task *models.Task, req *executor.ExecutionRequest, isSuccess bool, status string, exitCode int) {
	if task == nil {
		return
	}
	
	if !isSuccess || status == constant.TaskStatusFailed || status == constant.TaskStatusTimeout || exitCode != 0 {
		retryIndex := req.Metadata.RetryIndex

		if retryIndex < task.RetryCount {
			retryIndex++
			logger.Infof("[Executor] 任务 #%s 执行失败/出错，将在 %d 秒后进行第 %d/%d 次重试...", task.ID, task.RetryInterval, retryIndex, task.RetryCount)
			
			es.scheduler.EnqueueDelayed(time.Duration(task.RetryInterval)*time.Second, func() *executor.ExecutionRequest {
				latestTask := es.taskService.GetTaskByID(task.ID)
				if latestTask == nil || !latestTask.Enabled {
					return nil
				}

				newEnvs, newSecrets := es.loadEnvVars(latestTask.ID, string(latestTask.Envs))
				return &executor.ExecutionRequest{
					TaskID:    req.TaskID,
					Name:      latestTask.Name,
					Command:   string(latestTask.Command),
					WorkDir:   latestTask.WorkDir,
					Envs:      newEnvs,
					Secrets:   newSecrets,
					Timeout:   latestTask.Timeout,
					Languages: []map[string]string(latestTask.Languages),
					UseMise:   latestTask.UseMise(),
					Type:      executor.TaskTypeManual,
					Metadata: executor.ExecutionMetadata{
						RetryIndex: retryIndex,
					},
				}
			})
		}
	}
}

func (h *ServerSchedulerHandler) OnCronNextRun(req *executor.ExecutionRequest, nextRun time.Time) {
	taskID := req.TaskID
	// 更新数据库中的下次运行时间
	database.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("next_run", nextRun)
}

// LocalTaskHooks 本地任务钩子适配器
type LocalTaskHooks struct {
	es    *ExecutorService
	logID string
}

func (h *LocalTaskHooks) PreExecute(ctx context.Context, req executor.Request) (string, error) {
	return h.logID, nil
}

func (h *LocalTaskHooks) PostExecute(ctx context.Context, logID string, result *executor.Result) error {
	return nil
}

func (h *LocalTaskHooks) OnHeartbeat(ctx context.Context, logID string, duration int64) error {
	if logID != "" {
		return h.es.taskLogService.UpdateTaskDuration(logID, duration)
	}
	return nil
}

// ExecuteDispatcher 实现任务分发逻辑
func (es *ExecutorService) ExecuteDispatcher(ctx context.Context, req *executor.ExecutionRequest, stdout, stderr io.Writer) (*executor.Result, error) {
	taskID := req.TaskID
	
	// 解析路径变量 (如 $SCRIPTS_DIR$)
	req.Command = es.ResolvePath(req.Command)
	req.WorkDir = es.ResolvePath(req.WorkDir)

	task := es.taskService.GetTaskByID(taskID)
	// 系统任务（无 taskID）直接本地执行
	if task == nil {
		return executor.Execute(ctx, executor.Request{
			Command: req.Command,
			WorkDir: req.WorkDir,
			Envs:    req.Envs,
			Timeout: req.Timeout,
			UseMise: false, // 系统任务不使用 mise
		}, stdout, stderr)
	}

	// 重新加载最新的环境变量，满足即时生效的需求
	es.refreshExecutionRequestEnvs(req, task)

	// 特殊处理仓库同步任务
	if task.Type == constant.TaskTypeRepo {
		cmd, workDir := es.BuildRepoCommand(task)
		if cmd != "" {
			req.Command = cmd
			req.WorkDir = workDir
		}
	}

	// 远程任务
	if task.AgentID != nil && *task.AgentID != "" {
		// 将请求中已包含的环境变量（已合并）传递给 Agent
		return es.ExecuteRemoteForScheduler(task, req.LogID, executor.FormatEnvVars(req.Envs), req.Secrets)
	}

	// 本地任务
	hooks := &LocalTaskHooks{es: es, logID: req.LogID}
	return executor.ExecuteWithHooks(ctx, executor.Request{
		Command:   req.Command,
		WorkDir:   req.WorkDir,
		Envs:      req.Envs,
		Timeout:   req.Timeout,
		Languages: []map[string]string(task.Languages),
		UseMise:   req.UseMise, // 使用请求中的 UseMise 标志 (由调度器统一处理过)
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
	go es.loadCronTasks()
	es.cronManager.Start()
	// logger.Info("[Executor] 计划任务管理器已启动")
}

// StopCron 停止计划任务
func (es *ExecutorService) StopCron() {
	es.cronManager.Stop()
	// logger.Info("[Executor] 计划任务管理器已停止")
}

// AddCronTask 添加计划任务
func (es *ExecutorService) AddCronTask(task *models.Task) error {
	if task.TriggerType != constant.TriggerTypeCron {
		es.RemoveCronTask(task.ID) // 如果不是cron类型，确保从调度器移除
		return nil
	}
	// 在加入调度器前，预先加载好环境信息
	task.RuntimeEnvs, task.RuntimeSecrets = es.loadEnvVars(task.ID, string(task.Envs))

	return es.cronManager.AddTask(task)
}

// RemoveCronTask 移除计划任务
func (es *ExecutorService) RemoveCronTask(taskID string) {
	es.cronManager.RemoveTask(taskID)
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
		if !task.Enabled {
			continue
		}

		if task.TriggerType == constant.TriggerTypeBaihuStartup {
			go func(t models.Task) {
				// 延迟一点时间再触发，确保系统完全启动
				time.Sleep(3 * time.Second)
				logger.Infof("[Executor] 触发开机服务启动任务 #%s: %s", t.ID, t.Name)
				es.ExecuteTask(t.ID, nil)
			}(task)
		} else if task.TriggerType == constant.TriggerTypeCron && task.Schedule != "" && (task.AgentID == nil || *task.AgentID == "") {
			// 只调度本地任务（agent_id 为空或 0）的定时任务
			err := es.AddCronTask(&task)
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
func (es *ExecutorService) ExecuteTask(taskID string, extraEnvs []string) *executor.ExecutionResult {
	task := es.taskService.GetTaskByID(taskID)
	if task == nil {
		return &executor.ExecutionResult{
			TaskID:    taskID,
			Success:   false,
			Error:     "任务不存在",
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	// 1. 检查并发
	if err := es.CheckConcurrency(taskID); err != nil {
		return &executor.ExecutionResult{
			TaskID:    taskID,
			Success:   false,
			Error:     err.Error(), // 这里会返回 "任务正在运行中，拒绝并行执行"
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
	}

	envs, secrets := es.loadEnvVars(task.ID, string(task.Envs))
	if len(extraEnvs) > 0 {
		envs = append(envs, extraEnvs...)
	}

	req := &executor.ExecutionRequest{
		TaskID:    task.ID,
		Name:      task.Name,
		Command:   string(task.Command),
		WorkDir:   task.WorkDir,
		Envs:      envs,
		Secrets:   secrets,
		Timeout:   task.Timeout,
		Languages: []map[string]string(task.Languages),
		UseMise:   task.UseMise(),
		Type:      executor.TaskTypeManual,
	}

	es.scheduler.EnqueueOrExecute(req)

	return &executor.ExecutionResult{
		TaskID:    task.ID,
		Success:   true,
		Status:    constant.TaskStatusQueued,
		StartTime: time.Now(),
	}
}

// StopTaskExecution stops a running task execution by LogID
func (es *ExecutorService) StopTaskExecution(logID string) error {
	var taskLog models.TaskLog
	res := database.DB.Where("id = ?", logID).Limit(1).Find(&taskLog)
	if res.Error != nil || res.RowsAffected == 0 {
		return fmt.Errorf("日志不存在")
	}

	if taskLog.Status != constant.TaskStatusRunning {
		return fmt.Errorf("任务已结束")
	}

	task := es.taskService.GetTaskByID(taskLog.TaskID)
	if task == nil {
		return fmt.Errorf("任务不存在")
	}

	// 远程任务：发送停止指令到 Agent
	if task.AgentID != nil && *task.AgentID != "" {
		logger.Infof("[Executor] 请求停止远程任务 #%s (Agent #%s, LogID: %s)", task.ID, *task.AgentID, logID)
		return es.agentWSManager.SendToAgent(*task.AgentID, constant.WSTypeStop, map[string]interface{}{
			"log_id": logID,
		})
	}

	// 本地任务：直接停止调度器中的执行实例
	logger.Infof("[Executor] 请求停止本地任务 #%s (LogID: %s)", task.ID, logID)
	if es.scheduler.StopLog(logID) {
		return nil
	}

	return fmt.Errorf("任务当前不在运行队列中或已完成")
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

// UpdateResult 更新内存中的执行结果缓冲
func (es *ExecutorService) UpdateResult(res executor.ExecutionResult) {
	es.resultsMu.Lock()
	defer es.resultsMu.Unlock()

	// 按照用户要求，任务结束后清空 Output 以节省内存。
	// 结束状态的任务如果需要查看完整日志，会自动从数据库/文件中读取。
	isFinished := res.Status == constant.TaskStatusSuccess ||
		res.Status == constant.TaskStatusFailed ||
		res.Status == constant.TaskStatusTimeout ||
		res.Status == constant.TaskStatusCancelled

	if isFinished {
		res.Output = ""
	}

	// 查找是否已存在（通过 LogID）
	for i := range es.results {
		if es.results[i].LogID == res.LogID && res.LogID != "" {
			es.results[i] = res
			return
		}
	}

	// 不存在则追加到末尾
	if len(es.results) >= 100 {
		// 移除最旧的一个
		es.results = es.results[1:]
	}
	es.results = append(es.results, res)
}

// GetLastResults returns the last execution results
func (es *ExecutorService) GetLastResults(count int) []executor.ExecutionResult {
	es.resultsMu.RLock()
	defer es.resultsMu.RUnlock()

	total := len(es.results)
	if count > total {
		count = total
	}

	if count <= 0 {
		return []executor.ExecutionResult{}
	}

	// 返回副本，按时间倒序（最新的在前）
	res := make([]executor.ExecutionResult, 0, count)
	for i := 0; i < count; i++ {
		res = append(res, es.results[total-1-i])
	}
	return res
}

// --- 以下内容从 TaskExecutionService 合并 ---

// CleanupRunningTasks 清理所有任务的运行状态（在重启时调用）
func (es *ExecutorService) CleanupRunningTasks() error {
	logger.Info("[Executor] 正在清理残留的任务运行状态...")
	return database.DB.Model(&models.Task{}).Where("1=1").Update("running_go", "[]").Error
}

// CheckConcurrency 检查任务并发限制（只读检查）
func (es *ExecutorService) CheckConcurrency(taskID string) error {
	var task models.Task
	res := database.DB.Select("config, running_go").Where("id = ?", taskID).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		if res.Error != nil {
			return res.Error
		}
		return gorm.ErrRecordNotFound
	}
	var goids []int64
	if string(task.RunningGo) != "" {
		_ = json.Unmarshal([]byte(string(task.RunningGo)), &goids)
	}
 
 	var config models.TaskConfig
	if string(task.Config) != "" {
		_ = json.Unmarshal([]byte(string(task.Config)), &config)
	}

	if config.Concurrency == 0 && len(goids) > 0 {
		return fmt.Errorf("任务正在运行中，拒绝并行执行，请前往日志查看")
	}
	return nil
}

// AddRunningGo 添加当前 goroutine ID 到任务的 running_go 字段
func (es *ExecutorService) AddRunningGo(taskID string) (int64, error) {
	goid := utils.GetGoroutineID()
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = database.DB.Transaction(func(tx *gorm.DB) error {
			var task models.Task
			res := tx.Where("id = ?", taskID).Limit(1).Find(&task)
			if res.Error != nil || res.RowsAffected == 0 {
				if res.Error != nil {
					return res.Error
				}
				return gorm.ErrRecordNotFound
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
			return tx.Model(&task).Update("running_go", models.BigText(data)).Error
		})
		if lastErr == nil {
			return goid, nil
		}
		// 如果是业务错误（任务正在运行），不重试
		if lastErr.Error() == "task is running" {
			return goid, lastErr
		}
		// 数据库锁错误，等待后重试
		time.Sleep(100 * time.Millisecond)
	}
	return goid, fmt.Errorf("任务并发限制: %v", lastErr)
}

// RemoveRunningGo 从任务的 running_go 字段移除指定 goroutine ID
func (es *ExecutorService) RemoveRunningGo(taskID string, goid int64) {
	for attempt := 0; attempt < 3; attempt++ {
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			var task models.Task
			res := tx.Where("id = ?", taskID).Limit(1).Find(&task)
			if res.Error != nil || res.RowsAffected == 0 {
				if res.Error != nil {
					return res.Error
				}
				return gorm.ErrRecordNotFound
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
		if err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// ExecuteRemoteForScheduler 供 Scheduler 调用，执行远程任务并等待结果
func (es *ExecutorService) ExecuteRemoteForScheduler(task *models.Task, logID string, envs string, secrets []string) (*executor.Result, error) {
	agentID := *task.AgentID
	logger.Infof("[Executor] 远程执行任务 #%s: %s (Agent #%s, LogID: %s)", task.ID, task.Name, agentID, logID)

	// 1. 检查 Agent 状态
	var agent models.Agent
	res := database.DB.Where("id = ?", agentID).Limit(1).Find(&agent)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, fmt.Errorf("Agent #%s 不存在", agentID)
	}
	if !agent.Enabled {
		return nil, fmt.Errorf("Agent #%s 已禁用", agentID)
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
		"envs":    envs,
		"secrets": secrets,
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
			Error:     agentResult.Error,
			Status:    agentResult.Status,
			Duration:  agentResult.Duration,
			ExitCode:  agentResult.ExitCode,
			StartTime: time.Unix(agentResult.StartTime, 0),
			EndTime:   time.Unix(agentResult.EndTime, 0),
		}, nil
	case <-time.After(time.Duration(timeout) * time.Minute):
		end := time.Now()
		return &executor.Result{
			Status:    constant.TaskStatusFailed,
			Error:     "等待 Agent 结果超时",
			Duration:  end.Sub(start).Milliseconds(),
			ExitCode:  -1,
			StartTime: start,
			EndTime:   end,
		}, fmt.Errorf("等待 Agent 结果超时")
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

	exePath, err := os.Executable()
	if err != nil {
		exePath = "baihu" // Fallback if executable path can't be found
	}

	args := []string{
		"reposync",
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
	if config.WhitelistPaths != "" {
		args = append(args, "--whitelist-paths", config.WhitelistPaths)
	}
	if config.Blacklist != "" {
		args = append(args, "--blacklist", config.Blacklist)
	}
	if config.Dependence != "" {
		args = append(args, "--dependence", config.Dependence)
	}
	if config.Extensions != "" {
		args = append(args, "--extensions", config.Extensions)
	}
	
	// 传递任务 ID，以便 reposync 内部直接处理脚本注册并输出日志
	args = append(args, "--task-id", task.ID)
	args = append(args, "--task-timeout", fmt.Sprintf("%d", task.Timeout))
	if langData, err := json.Marshal(task.Languages); err == nil {
		args = append(args, "--task-langs", string(langData))
	}

	// 为了防止 shell 解释特殊字符（如 |），对每个参数进行转义/加引号
	quotedArgs := make([]string, len(args))
	for i, arg := range args {
		quotedArgs[i] = utils.QuotePath(arg)
	}

	cmdStr := utils.QuotePath(exePath) + " " + strings.Join(quotedArgs, " ")
	return buildRepoCommandEnvPrefix()+cmdStr, filepath.Dir(exePath)
}

// loadEnvVars 加载环境变量和掩码信息，支持全局注入及重名合并
func (es *ExecutorService) loadEnvVars(taskID string, envIDs string) ([]string, []string) {
	// 1. 检查是否开启了注入全部环境变量
	if taskID != "" && es.taskService != nil {
		task := es.taskService.GetTaskByID(taskID)
		if task != nil && task.Config != "" {
			var config models.TaskConfig
			if err := json.Unmarshal([]byte(task.Config), &config); err == nil {
				if config.AllEnvs {
					if es.envService != nil {
						return es.envService.GetAllEnvVarsAndSecrets()
					}
				}
			}
		}
	}

	// 2. 否则按 ID 列表进行加载（支持合并逻辑在 envService 中处理）
	if envIDs == "" {
		return nil, nil
	}

	if es.envService != nil {
		return es.envService.GetEnvVarsAndSecretsByIDs(envIDs)
	}

	return nil, nil
}

// refreshExecutionRequestEnvs 重新加载最新的环境变量，并与原请求中的变量合并（保留额外变量）
func (es *ExecutorService) refreshExecutionRequestEnvs(req *executor.ExecutionRequest, task *models.Task) {
	if task == nil || (req.Type != executor.TaskTypeCron && req.Type != executor.TaskTypeManual) {
		return
	}

	// 1. 备份原请求中的环境变量（用于后续保留手动指定的额外变量）
	currentEnvs := req.Envs

	// 2. 从数据库加载最新的环境变量设置
	envs, secrets := es.loadEnvVars(task.ID, string(task.Envs))
	req.Envs = envs
	req.Secrets = secrets

	// 3. 将原请求中存在但数据库中不存在的“额外变量”合并回来（如 API 注入、手动执行参数等）
	for _, ce := range currentEnvs {
		idx := strings.Index(ce, "=")
		if idx == -1 {
			continue
		}
		name := ce[:idx]
		found := false
		for _, ne := range envs {
			if strings.HasPrefix(ne, name+"=") {
				found = true
				break
			}
		}
		if !found {
			req.Envs = append(req.Envs, ce)
		}
	}
}

func (es *ExecutorService) ResolvePath(path string) string {
	absScriptsDir := resolveAbsScriptsDir()
	return strings.ReplaceAll(path, "$SCRIPTS_DIR$", absScriptsDir)
}

func buildRepoCommandEnvPrefix() string {
	absConfig, err := filepath.Abs(constant.ConfigPath)
	if err != nil {
		absConfig = constant.ConfigPath
	}

	absScriptsDir := resolveAbsScriptsDir()
	return "BH_CONFIG_PATH='" + strings.ReplaceAll(absConfig, "'", "'\\''") + "' BH_SCRIPTS_DIR='" + strings.ReplaceAll(absScriptsDir, "'", "'\\''") + "' "
}

func resolveAbsScriptsDir() string {
	if scriptsDir := os.Getenv("BH_SCRIPTS_DIR"); scriptsDir != "" {
		if filepath.IsAbs(scriptsDir) {
			return filepath.Clean(scriptsDir)
		}
		if absScriptsDir, err := filepath.Abs(scriptsDir); err == nil {
			return absScriptsDir
		}
		return filepath.Clean(scriptsDir)
	}

	if configPath := os.Getenv("BH_CONFIG_PATH"); configPath != "" {
		if !filepath.IsAbs(configPath) {
			if absConfigPath, err := filepath.Abs(configPath); err == nil {
				configPath = absConfigPath
			}
		}

		projectRoot := filepath.Dir(filepath.Dir(configPath))
		return filepath.Clean(filepath.Join(projectRoot, constant.ScriptsWorkDir))
	}

	if absScriptsDir, err := filepath.Abs(constant.ScriptsWorkDir); err == nil {
		return absScriptsDir
	}

	return filepath.Clean(constant.ScriptsWorkDir)
}
