package tasks

import (
	"baihu/internal/constant"
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SettingsService 接口定义（避免循环依赖）
type SettingsService interface {
	Get(section, key string) string
}

// EnvService 接口定义（避免循环依赖）
type EnvService interface {
	GetEnvVarsByIDs(ids string) []string
}

// ExecutionResult represents the result of a task execution
type ExecutionResult struct {
	TaskID  int
	Success bool
	Output  string
	Error   string
	Start   time.Time
	End     time.Time
}

// taskJob 任务队列项
type taskJob struct {
	taskID int
}

// ExecutorService handles task execution
type ExecutorService struct {
	taskService          *TaskService
	taskExecutionService *TaskExecutionService
	settingsService      SettingsService
	envService           EnvService
	results              []ExecutionResult
	runningTasks         map[int]bool
	mu                   sync.RWMutex
	resultsMu            sync.RWMutex

	// 任务队列和 worker pool
	taskQueue   chan taskJob
	workerCount int
	rateLimiter <-chan time.Time
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewExecutorService creates a new executor service
func NewExecutorService(taskService *TaskService, taskExecutionService *TaskExecutionService, settingsService SettingsService, envService EnvService) *ExecutorService {
	// 从设置中读取调度配置
	workerCount := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyWorkerCount, 4)
	queueSize := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyQueueSize, 100)
	rateInterval := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyRateInterval, 200)

	logger.Infof("[Executor] 配置: workers=%d, queue=%d, rate=%dms", workerCount, queueSize, rateInterval)

	es := &ExecutorService{
		taskService:          taskService,
		taskExecutionService: taskExecutionService,
		settingsService:      settingsService,
		envService:           envService,
		results:              make([]ExecutionResult, 0, 100),
		runningTasks:         make(map[int]bool),
		taskQueue:            make(chan taskJob, queueSize),
		workerCount:          workerCount,
		rateLimiter:          time.Tick(time.Duration(rateInterval) * time.Millisecond),
		stopCh:               make(chan struct{}),
	}

	// 启动 worker pool
	es.startWorkers()

	return es
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

// startWorkers 启动 worker pool
func (es *ExecutorService) startWorkers() {
	for i := 0; i < es.workerCount; i++ {
		es.wg.Add(1)
		go es.worker(i)
	}
}

// worker 从队列中取任务执行
func (es *ExecutorService) worker(id int) {
	defer es.wg.Done()
	for {
		select {
		case <-es.stopCh:
			return
		case job := <-es.taskQueue:
			// 速率限制
			<-es.rateLimiter
			es.executeTaskInternal(job.taskID)
		}
	}
}

// Stop 停止 executor service
func (es *ExecutorService) Stop() {
	close(es.stopCh)
	es.wg.Wait()
}

// Reload 重新加载配置并重建 worker pool
func (es *ExecutorService) Reload() {
	logger.Info("[Executor] 正在重载配置...")

	// 停止现有 workers
	close(es.stopCh)
	es.wg.Wait()
	logger.Info("[Executor] 已停止工作线程")

	// 从设置中读取新配置
	workerCount := getIntSetting(es.settingsService, constant.SectionScheduler, constant.KeyWorkerCount, 4)
	queueSize := getIntSetting(es.settingsService, constant.SectionScheduler, constant.KeyQueueSize, 100)
	rateInterval := getIntSetting(es.settingsService, constant.SectionScheduler, constant.KeyRateInterval, 200)

	// 重建 channel 和配置
	es.mu.Lock()
	es.taskQueue = make(chan taskJob, queueSize)
	es.workerCount = workerCount
	es.rateLimiter = time.Tick(time.Duration(rateInterval) * time.Millisecond)
	es.stopCh = make(chan struct{})
	es.mu.Unlock()

	// 启动新的 workers
	es.startWorkers()

	logger.Infof("[Executor] 配置已重载: workers=%d, queue=%d, rate=%dms", workerCount, queueSize, rateInterval)
}

// EnqueueTask 将任务加入队列（供 cron 调度器调用）
func (es *ExecutorService) EnqueueTask(taskID int) {
	select {
	case es.taskQueue <- taskJob{taskID: taskID}:
		// 成功入队
	default:
		// 队列满，直接执行（降级处理）
		logger.Warnf("[Executor] 任务队列已满，直接执行任务 #%d", taskID)
		go es.executeTaskInternal(taskID)
	}
}

// ExecuteTask executes a task by ID（同步执行，供 API 调用）
func (es *ExecutorService) ExecuteTask(taskID int) *ExecutionResult {
	return es.executeTaskInternal(taskID)
}

// executeTaskInternal 内部执行任务逻辑
func (es *ExecutorService) executeTaskInternal(taskID int) *ExecutionResult {
	task := es.taskService.GetTaskByID(taskID)
	if task == nil {
		return &ExecutionResult{
			TaskID:  taskID,
			Success: false,
			Error:   "Task not found",
			Start:   time.Now(),
			End:     time.Now(),
		}
	}

	// 标记任务开始运行
	es.mu.Lock()
	es.runningTasks[taskID] = true
	es.mu.Unlock()

	var result *ExecutionResult

	// 使用统一的任务执行服务
	req := &TaskExecutionRequest{
		TaskID: uint(taskID),
		Task:   task,
	}

	start := time.Now()
	err := es.taskExecutionService.ExecuteTask(req)
	end := time.Now()

	if err != nil {
		result = &ExecutionResult{
			TaskID:  taskID,
			Success: false,
			Error:   err.Error(),
			Start:   start,
			End:     end,
		}
	} else {
		result = &ExecutionResult{
			TaskID:  taskID,
			Success: true,
			Output:  "任务已提交执行",
			Start:   start,
			End:     end,
		}
	}

	// 标记任务结束
	es.mu.Lock()
	delete(es.runningTasks, taskID)
	es.mu.Unlock()

	return result
}

// executeNormalTask 执行普通任务
func (es *ExecutorService) executeNormalTask(task *models.Task) *ExecutionResult {
	// 演示模式下使用 echo 替换实际命令
	if constant.DemoMode {
		return es.ExecuteCommandWithOptions("echo '[演示模式] 任务执行已跳过，实际命令不会运行'", time.Minute, nil, "")
	}

	// 加载环境变量
	envVars := es.envService.GetEnvVarsByIDs(task.Envs)

	// 确定工作目录
	workDir := task.WorkDir
	if workDir == "" {
		workDir = constant.ScriptsWorkDir
	}

	// 使用任务配置的超时时间
	timeout := task.Timeout
	if timeout <= 0 {
		timeout = constant.DefaultTaskTimeout
	}
	return es.ExecuteCommandWithOptions(task.Command, time.Duration(timeout)*time.Minute, envVars, workDir)
}

// executeRepoTask 执行仓库同步任务（调用 sync.py）
func (es *ExecutorService) executeRepoTask(task *models.Task) *ExecutionResult {
	// 演示模式下使用 echo 替换实际命令
	if constant.DemoMode {
		return es.ExecuteCommandWithOptions("echo '[演示模式] 仓库同步已跳过，实际命令不会运行'", time.Minute, nil, "")
	}

	result := &ExecutionResult{
		Success: false,
		Start:   time.Now(),
	}

	// 解析仓库配置
	var config models.RepoConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		result.End = time.Now()
		result.Error = "解析仓库配置失败: " + err.Error()
		return result
	}

	// 处理目标路径：为空则使用 scripts 目录，相对路径则基于 scripts 目录
	targetPath := config.TargetPath
	if targetPath == "" {
		targetPath = constant.ScriptsWorkDir
	} else if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(constant.ScriptsWorkDir, targetPath)
	}
	// 转换为绝对路径
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		absTargetPath = targetPath
	}

	// 构建 sync.py 命令参数
	args := []string{
		"/opt/sync.py",
		"--source-type", config.SourceType,
		"--source-url", config.SourceURL,
		"--target-path", absTargetPath,
	}

	// Git 分支
	if config.Branch != "" {
		args = append(args, "--branch", config.Branch)
	}

	// 稀疏路径
	if config.SparsePath != "" {
		args = append(args, "--path", config.SparsePath)
	}

	// 单文件模式
	if config.SingleFile {
		args = append(args, "--single-file")
	}

	// 代理设置
	if config.Proxy != "" && config.Proxy != "none" {
		args = append(args, "--proxy", config.Proxy)
		if config.Proxy == "custom" && config.ProxyURL != "" {
			args = append(args, "--proxy-url", config.ProxyURL)
		}
	}

	// 认证 Token
	if config.AuthToken != "" {
		args = append(args, "--auth-token", config.AuthToken)
	}

	// 构建命令
	command := "python3 " + strings.Join(args, " ")

	// 使用任务配置的超时时间
	timeout := task.Timeout
	if timeout <= 0 {
		timeout = constant.DefaultTaskTimeout
	}

	// 执行命令
	execResult := es.ExecuteCommandWithOptions(command, time.Duration(timeout)*time.Minute, nil, "/opt")

	result.End = time.Now()
	result.Output = execResult.Output
	result.Success = execResult.Success
	result.Error = execResult.Error

	return result
}

// GetRunningCount 获取正在运行的任务数量
func (es *ExecutorService) GetRunningCount() int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.runningTasks)
}

// ExecuteCommand executes a shell command with default timeout
func (es *ExecutorService) ExecuteCommand(command string) *ExecutionResult {
	return es.ExecuteCommandWithTimeout(command, time.Duration(constant.DefaultTaskTimeout)*time.Minute)
}

// ExecuteCommandWithTimeout executes a shell command with specified timeout
func (es *ExecutorService) ExecuteCommandWithTimeout(command string, timeout time.Duration) *ExecutionResult {
	return es.ExecuteCommandWithEnv(command, timeout, nil)
}

// ExecuteCommandWithEnv executes a shell command with specified timeout and environment variables
func (es *ExecutorService) ExecuteCommandWithEnv(command string, timeout time.Duration, envVars []string) *ExecutionResult {
	return es.ExecuteCommandWithOptions(command, timeout, envVars, "")
}

// ExecuteCommandWithOptions executes a shell command with specified timeout, environment variables and working directory
func (es *ExecutorService) ExecuteCommandWithOptions(command string, timeout time.Duration, envVars []string, workDir string) *ExecutionResult {
	result := &ExecutionResult{
		Success: false,
		Start:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	shell, args := utils.GetShellCommand(command)
	cmd := exec.CommandContext(ctx, shell, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置工作目录
	if workDir != "" {
		cmd.Dir = workDir
	}

	// 设置环境变量：继承系统环境变量 + 自定义环境变量
	if len(envVars) > 0 {
		cmd.Env = append(os.Environ(), envVars...)
	}

	err := cmd.Run()
	result.End = time.Now()

	result.Output = stdout.String()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Error = "执行超时\n" + stderr.String()
		} else {
			result.Error = err.Error() + "\n" + stderr.String()
		}
	} else {
		result.Success = true
	}

	// 使用独立锁保存结果
	es.resultsMu.Lock()
	es.results = append(es.results, *result)
	if len(es.results) > 100 {
		es.results = es.results[1:]
	}
	es.resultsMu.Unlock()

	return result
}

// GetLastResults returns the last execution results
func (es *ExecutorService) GetLastResults(count int) []ExecutionResult {
	es.resultsMu.RLock()
	defer es.resultsMu.RUnlock()

	start := 0
	if len(es.results) > count {
		start = len(es.results) - count
	}

	results := make([]ExecutionResult, len(es.results[start:]))
	copy(results, es.results[start:])
	return results
}
