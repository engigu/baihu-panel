package services

import (
	"bytes"
	"context"
	"os/exec"
	"sync"
	"time"

	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/utils"
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

// ExecutorService handles task execution
type ExecutorService struct {
	taskService  *TaskService
	results      []ExecutionResult
	runningTasks map[int]bool
	callbacks    []ExecutionCallback
	mu           sync.RWMutex
}

// NewExecutorService creates a new executor service
func NewExecutorService(taskService *TaskService) *ExecutorService {
	es := &ExecutorService{
		taskService:  taskService,
		results:      make([]ExecutionResult, 0),
		runningTasks: make(map[int]bool),
		callbacks:    make([]ExecutionCallback, 0),
	}
	// 注册默认回调
	es.RegisterCallback(es.saveTaskLogCallback)
	es.RegisterCallback(es.updateStatsCallback)
	return es
}

// RegisterCallback 注册执行完成回调
func (es *ExecutorService) RegisterCallback(cb ExecutionCallback) {
	es.mu.Lock()
	es.callbacks = append(es.callbacks, cb)
	es.mu.Unlock()
}

// executeCallbacks 执行所有回调
func (es *ExecutorService) executeCallbacks(taskID uint, command string, result *ExecutionResult) {
	es.mu.RLock()
	callbacks := make([]ExecutionCallback, len(es.callbacks))
	copy(callbacks, es.callbacks)
	es.mu.RUnlock()

	for _, cb := range callbacks {
		cb(taskID, command, result)
	}
}

// saveTaskLogCallback 保存任务日志的回调
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

// ExecuteTask executes a task by ID
func (es *ExecutorService) ExecuteTask(taskID int) *ExecutionResult {
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

	// 使用任务配置的超时时间
	timeout := task.Timeout
	if timeout <= 0 {
		timeout = constant.DefaultTaskTimeout
	}
	result := es.ExecuteCommandWithTimeout(task.Command, time.Duration(timeout)*time.Minute)
	result.TaskID = taskID

	// 标记任务结束
	es.mu.Lock()
	delete(es.runningTasks, taskID)
	es.mu.Unlock()

	// 执行回调
	es.executeCallbacks(uint(taskID), task.Command, result)

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

	es.mu.Lock()
	es.results = append(es.results, *result)
	if len(es.results) > 100 {
		es.results = es.results[1:]
	}
	es.mu.Unlock()

	return result
}

// GetLastResults returns the last execution results
func (es *ExecutorService) GetLastResults(count int) []ExecutionResult {
	es.mu.RLock()
	defer es.mu.RUnlock()

	start := 0
	if len(es.results) > count {
		start = len(es.results) - count
	}

	results := make([]ExecutionResult, len(es.results[start:]))
	copy(results, es.results[start:])
	return results
}
