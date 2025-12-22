package services

import (
	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ExecutionResult represents the result of a task execution
type ExecutionResult struct {
	TaskID  int
	Success bool
	Output  string
	Error   string
	Start   time.Time
	End     time.Time
}

// ExecutionCallback 任务执行完成后的回调函数类型
type ExecutionCallback func(taskID uint, command string, result *ExecutionResult)

// taskJob 任务队列项
type taskJob struct {
	taskID int
}

// ExecutorService handles task execution
type ExecutorService struct {
	taskService  *TaskService
	results      []ExecutionResult
	runningTasks map[int]bool
	callbacks    []ExecutionCallback
	mu           sync.RWMutex
	resultsMu    sync.RWMutex

	// 任务队列和 worker pool
	taskQueue   chan taskJob
	workerCount int
	rateLimiter <-chan time.Time
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewExecutorService creates a new executor service
func NewExecutorService(taskService *TaskService) *ExecutorService {
	// 从设置中读取调度配置
	settingsService := NewSettingsService()
	workerCount := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyWorkerCount, 4)
	queueSize := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyQueueSize, 100)
	rateInterval := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyRateInterval, 200)

	logger.Infof("Executor service config: workers=%d, queue=%d, rate=%dms", workerCount, queueSize, rateInterval)

	es := &ExecutorService{
		taskService:  taskService,
		results:      make([]ExecutionResult, 0, 100),
		runningTasks: make(map[int]bool),
		callbacks:    make([]ExecutionCallback, 0),
		taskQueue:    make(chan taskJob, queueSize),
		workerCount:  workerCount,
		rateLimiter:  time.Tick(time.Duration(rateInterval) * time.Millisecond),
		stopCh:       make(chan struct{}),
	}

	// 注册默认回调
	es.RegisterCallback(es.saveTaskLogCallback)
	es.RegisterCallback(es.updateStatsCallback)
	es.RegisterCallback(es.cleanLogsCallback)

	// 启动 worker pool
	es.startWorkers()

	return es
}

// getIntSetting 从设置中获取整数值
func getIntSetting(s *SettingsService, section, key string, defaultVal int) int {
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
	logger.Info("Reloading executor service...")

	// 停止现有 workers
	close(es.stopCh)
	es.wg.Wait()
	logger.Info("Stopped executor service...")

	// 从设置中读取新配置
	settingsService := NewSettingsService()
	workerCount := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyWorkerCount, 4)
	queueSize := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyQueueSize, 100)
	rateInterval := getIntSetting(settingsService, constant.SectionScheduler, constant.KeyRateInterval, 200)

	// 重建 channel 和配置
	es.mu.Lock()
	es.taskQueue = make(chan taskJob, queueSize)
	es.workerCount = workerCount
	es.rateLimiter = time.Tick(time.Duration(rateInterval) * time.Millisecond)
	es.stopCh = make(chan struct{})
	es.mu.Unlock()

	// 启动新的 workers
	es.startWorkers()

	logger.Infof("Executor service reloaded: workers=%d, queue=%d, rate=%dms", workerCount, queueSize, rateInterval)
}

// RegisterCallback 注册执行完成回调
func (es *ExecutorService) RegisterCallback(cb ExecutionCallback) {
	es.mu.Lock()
	es.callbacks = append(es.callbacks, cb)
	es.mu.Unlock()
}

// executeCallbacksAsync 异步执行所有回调
func (es *ExecutorService) executeCallbacksAsync(taskID uint, command string, result *ExecutionResult) {
	es.mu.RLock()
	callbacks := make([]ExecutionCallback, len(es.callbacks))
	copy(callbacks, es.callbacks)
	es.mu.RUnlock()

	go func() {
		for _, cb := range callbacks {
			cb(taskID, command, result)
		}
	}()
}

// saveTaskLogCallback 保存任务日志的回调（异步执行，压缩在此处进行）
func (es *ExecutorService) saveTaskLogCallback(taskID uint, command string, result *ExecutionResult) {
	output := result.Output
	if result.Error != "" {
		output += "\n[ERROR]\n" + result.Error
	}

	compressed, err := utils.CompressToBase64(output)
	if err != nil {
		logger.Errorf("Failed to compress log: %v", err)
		compressed = ""
	}

	status := "success"
	if !result.Success {
		status = "failed"
	}

	taskLog := &models.TaskLog{
		TaskID:   taskID,
		Command:  command,
		Output:   compressed,
		Status:   status,
		Duration: result.End.Sub(result.Start).Milliseconds(),
	}

	if err := database.DB.Create(taskLog).Error; err != nil {
		logger.Errorf("Failed to save task log: %v", err)
	}
}

// updateStatsCallback 更新统计数据的回调
func (es *ExecutorService) updateStatsCallback(taskID uint, _ string, result *ExecutionResult) {
	status := "success"
	if !result.Success {
		status = "failed"
	}
	sendStatsService := NewSendStatsService()
	if err := sendStatsService.IncrementStats(taskID, status); err != nil {
		logger.Errorf("Failed to update stats: %v", err)
	}
}

// CleanConfig 清理配置结构
type CleanConfig struct {
	Type string `json:"type"` // "day" 或 "count"
	Keep int    `json:"keep"` // 保留天数或条数
}

// cleanLogsCallback 清理日志的回调
func (es *ExecutorService) cleanLogsCallback(taskID uint, _ string, _ *ExecutionResult) {
	task := es.taskService.GetTaskByID(int(taskID))
	if task == nil || task.CleanConfig == "" {
		return
	}

	var config CleanConfig
	if err := json.Unmarshal([]byte(task.CleanConfig), &config); err != nil {
		logger.Errorf("Failed to parse clean config: %v", err)
		return
	}

	if config.Keep <= 0 {
		return
	}

	var deleted int64
	switch config.Type {
	case "day":
		cutoff := time.Now().AddDate(0, 0, -config.Keep)
		result := database.DB.Where("task_id = ? AND created_at < ?", taskID, cutoff).Delete(&models.TaskLog{})
		deleted = result.RowsAffected
	case "count":
		var boundaryLog models.TaskLog
		err := database.DB.Where("task_id = ?", taskID).Order("id DESC").Offset(config.Keep - 1).Limit(1).First(&boundaryLog).Error
		if err == nil {
			result := database.DB.Where("task_id = ? AND id < ?", taskID, boundaryLog.ID).Delete(&models.TaskLog{})
			deleted = result.RowsAffected
		}
	}

	if deleted > 0 {
		logger.Infof("Cleaned %d logs for task %d", deleted, taskID)
	}
}

// EnqueueTask 将任务加入队列（供 cron 调度器调用）
func (es *ExecutorService) EnqueueTask(taskID int) {
	select {
	case es.taskQueue <- taskJob{taskID: taskID}:
		// 成功入队
	default:
		// 队列满，直接执行（降级处理）
		logger.Warnf("Task queue full, executing task %d directly", taskID)
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

	// 加载环境变量
	envService := NewEnvService()
	envVars := envService.GetEnvVarsByIDs(task.Envs)

	// 使用任务配置的超时时间
	timeout := task.Timeout
	if timeout <= 0 {
		timeout = constant.DefaultTaskTimeout
	}
	result := es.ExecuteCommandWithEnv(task.Command, time.Duration(timeout)*time.Minute, envVars)
	result.TaskID = taskID

	// 标记任务结束
	es.mu.Lock()
	delete(es.runningTasks, taskID)
	es.mu.Unlock()

	// 异步执行回调（日志压缩、统计更新、日志清理）
	es.executeCallbacksAsync(uint(taskID), task.Command, result)

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
