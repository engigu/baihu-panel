package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	WorkerCount  int           // Worker 数量
	QueueSize    int           // 队列大小
	RateInterval time.Duration // 速率限制间隔
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeCron   TaskType = "cron"   // 计划任务
	TaskTypeManual TaskType = "manual" // 手动任务
	TaskTypeSystem TaskType = "system" // 系统任务
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 等待中
	TaskStatusRunning   TaskStatus = "running"   // 运行中
	TaskStatusSuccess   TaskStatus = "success"   // 成功
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusTimeout   TaskStatus = "timeout"   // 超时
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// ExecutionRequest 执行请求（标准接口）
type ExecutionRequest struct {
	TaskID   string                 // 任务 ID
	LogID    uint                   // 日志 ID
	Name     string                 // 任务名称
	Type     TaskType               // 任务类型
	Command  string                 // 命令
	WorkDir  string                 // 工作目录
	Envs     []string               // 环境变量
	Timeout  int                    // 超时时间（分钟）
	Metadata map[string]interface{} // 额外元数据
}

// ExecutionResult 执行结果（标准接口）
type ExecutionResult struct {
	TaskID    string    // 任务 ID
	LogID     uint      // 日志 ID
	Success   bool      // 是否成功
	Output    string    // 输出内容
	Error     string    // 错误信息
	Status    string    // 状态: success, failed, timeout, cancelled
	Duration  int64     // 执行时长（毫秒）
	ExitCode  int       // 退出码
	StartTime time.Time // 开始时间
	EndTime   time.Time // 结束时间
}

// SchedulerEventHandler 调度器事件处理器（标准接口）
// 主服务端和 Agent 端通过实现不同的 Handler 来处理事件
type SchedulerEventHandler interface {
	// OnTaskScheduled 任务被调度（加入队列）时触发
	OnTaskScheduled(req *ExecutionRequest)

	// OnTaskExecuting 任务准备开始执行时触发
	// 返回 stdout/stderr 写入器用于实时日志推送
	// 主服务端：返回 TinyLog 写入器（写入本地文件）
	// Agent 端：返回 WebSocket 写入器（实时推送到主服务）
	OnTaskExecuting(req *ExecutionRequest) (stdout, stderr io.Writer, err error)

	// OnTaskStarted 任务实际开始运行（已经过了队列等待和速率限制）
	OnTaskStarted(req *ExecutionRequest)

	// OnTaskCompleted 任务执行完成时触发
	// 主服务端：压缩日志、更新数据库、清理旧日志
	// Agent 端：通过 WebSocket 发送执行结果到主服务
	OnTaskCompleted(req *ExecutionRequest, result *ExecutionResult)

	// OnTaskFailed 任务执行失败时触发
	OnTaskFailed(req *ExecutionRequest, err error)

	// OnCronNextRun 计划任务下次运行时间更新时触发
	OnCronNextRun(req *ExecutionRequest, nextRun time.Time)

	// OnTaskHeartbeat 任务执行心跳（用于更新实时耗时等）
	OnTaskHeartbeat(req *ExecutionRequest, duration int64)
}

// SchedulerLogger 日志接口（允许自定义日志实现）
type SchedulerLogger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// DefaultLogger 默认日志实现（使用 fmt）
type DefaultLogger struct{}

func (l *DefaultLogger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}
func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}
func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// schedulerHooksAdapter 适配器：将 executor.Hooks 映射到 SchedulerEventHandler
type schedulerHooksAdapter struct {
	handler SchedulerEventHandler
	req     *ExecutionRequest
}

func (h *schedulerHooksAdapter) PreExecute(ctx context.Context, req Request) (uint, error) {
	return h.req.LogID, nil
}

func (h *schedulerHooksAdapter) PostExecute(ctx context.Context, logID uint, result *Result) error {
	return nil
}

func (h *schedulerHooksAdapter) OnHeartbeat(ctx context.Context, logID uint, duration int64) error {
	if h.handler != nil {
		h.handler.OnTaskHeartbeat(h.req, duration)
	}
	return nil
}

// TaskExecutor 定义任务执行函数签名
type TaskExecutor func(ctx context.Context, req *ExecutionRequest, stdout, stderr io.Writer) (*Result, error)

// Scheduler 统一调度器（独立组件，可在主服务和 Agent 中复用）
// 调度器本身只负责队列管理和任务调度，具体的执行逻辑和事件处理由 Handler 实现
type Scheduler struct {
	config       SchedulerConfig
	handler      SchedulerEventHandler
	executor     TaskExecutor
	taskQueue    chan *ExecutionRequest
	rateLimiter  <-chan time.Time
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	logger       SchedulerLogger
	runningTasks map[string]context.CancelFunc // 记录运行中的任务，用于停止
}

// NewScheduler 创建调度器
func NewScheduler(config SchedulerConfig, handler SchedulerEventHandler) *Scheduler {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 4
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100
	}
	if config.RateInterval <= 0 {
		config.RateInterval = 200 * time.Millisecond
	}

	s := &Scheduler{
		config:  config,
		handler: handler,
		executor: func(ctx context.Context, req *ExecutionRequest, stdout, stderr io.Writer) (*Result, error) {
			hooks := &schedulerHooksAdapter{handler: handler, req: req}
			return ExecuteWithHooks(ctx, Request{
				Command: req.Command,
				WorkDir: req.WorkDir,
				Envs:    req.Envs,
				Timeout: req.Timeout,
			}, stdout, stderr, hooks)
		},
		taskQueue:    make(chan *ExecutionRequest, config.QueueSize),
		rateLimiter:  time.Tick(config.RateInterval),
		stopCh:       make(chan struct{}),
		logger:       &DefaultLogger{},
		runningTasks: make(map[string]context.CancelFunc),
	}

	return s
}

// SetLogger 设置自定义日志实现
func (s *Scheduler) SetLogger(logger SchedulerLogger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = logger
}

// SetExecutor 设置任务执行器
func (s *Scheduler) SetExecutor(executor TaskExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executor = executor
}

// Start 启动调度器
func (s *Scheduler) Start() {
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	s.logger.Infof("[Scheduler] 已启动")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	s.logger.Infof("[Scheduler] 已停止")
}

// Enqueue 将任务加入队列
func (s *Scheduler) Enqueue(req *ExecutionRequest) error {
	select {
	case s.taskQueue <- req:
		if s.handler != nil {
			s.handler.OnTaskScheduled(req)
		}
		return nil
	default:
		// 队列满，返回错误
		return fmt.Errorf("任务队列已满")
	}
}

// EnqueueOrExecute 将任务加入队列，如果队列满则直接执行
func (s *Scheduler) EnqueueOrExecute(req *ExecutionRequest) {
	select {
	case s.taskQueue <- req:
		// 成功入队
		if s.handler != nil {
			s.handler.OnTaskScheduled(req)
		}
	default:
		// 队列满，直接执行（降级处理）
		s.logger.Warnf("[Scheduler] 任务队列已满，直接执行任务 %s", req.TaskID)
		go s.executeTask(req)
	}
}

// ExecuteSync 同步执行任务（不经过队列）
func (s *Scheduler) ExecuteSync(req *ExecutionRequest) (*ExecutionResult, error) {
	return s.executeTask(req)
}

// worker 工作协程
func (s *Scheduler) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			return
		case req := <-s.taskQueue:
			// 速率限制
			<-s.rateLimiter
			s.executeTask(req)
		}
	}
}

// executeTask 执行任务（本地执行）
func (s *Scheduler) executeTask(req *ExecutionRequest) (*ExecutionResult, error) {
	start := time.Now()

	s.logger.Infof("[Scheduler] 执行任务 %s (名称: %s, 类型: %s)", req.TaskID, req.Name, req.Type)

	// 1. 执行前事件：获取 stdout/stderr 写入器
	var stdout, stderr io.Writer
	var err error
	if s.handler != nil {
		stdout, stderr, err = s.handler.OnTaskExecuting(req)
		if err != nil {
			s.logger.Errorf("[Scheduler] 任务 %s 执行前事件失败: %v", req.TaskID, err)
			if s.handler != nil {
				s.handler.OnTaskFailed(req, err)
			}
			return &ExecutionResult{
				TaskID:    req.TaskID,
				Success:   false,
				Status:    "failed",
				Error:     err.Error(),
				Duration:  0,
				ExitCode:  1,
				StartTime: start,
				EndTime:   time.Now(),
			}, err
		}
	}

	// 2. 准备输出缓冲区
	var stdoutBuf, stderrBuf bytes.Buffer
	var stdoutWriter, stderrWriter io.Writer

	if stdout != nil {
		stdoutWriter = io.MultiWriter(&stdoutBuf, stdout)
	} else {
		stdoutWriter = &stdoutBuf
	}

	if stderr != nil {
		stderrWriter = io.MultiWriter(&stderrBuf, stderr)
	} else {
		stderrWriter = &stderrBuf
	}

	// 3. 实际开始执行事件 (经过队列和速率限制之后)
	if s.handler != nil {
		s.handler.OnTaskStarted(req)
	}

	// 4. 执行命令（使用 executor.Execute）
	// 创建带取消功能的上下文
	ctx, cancel := context.WithCancel(context.Background())
	if req.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Minute)
	}
	defer cancel()

	// 注册到运行中任务
	s.mu.Lock()
	s.runningTasks[req.TaskID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningTasks, req.TaskID)
		s.mu.Unlock()
	}()

	execResult, execErr := s.executor(ctx, req, stdoutWriter, stderrWriter)

	// 5. 构建结果
	result := &ExecutionResult{
		TaskID:    req.TaskID,
		LogID:     req.LogID, // 传递 LogID
		Success:   execResult.Status == "success",
		Output:    stdoutBuf.String(),
		Status:    execResult.Status,
		Duration:  execResult.Duration,
		ExitCode:  execResult.ExitCode,
		StartTime: execResult.StartTime,
		EndTime:   execResult.EndTime,
	}

	if execErr != nil {
		result.Error = execErr.Error()
		errOutput := stderrBuf.String()
		if errOutput != "" {
			result.Output += "\n[ERROR]\n" + errOutput
		}
		if ctx.Err() == context.Canceled {
			result.Status = "cancelled"
		} else if ctx.Err() == context.DeadlineExceeded {
			result.Status = "timeout"
		}
	}

	// 6. 执行后事件
	if s.handler != nil {
		if execErr != nil {
			s.handler.OnTaskFailed(req, execErr)
		} else {
			s.handler.OnTaskCompleted(req, result)
		}
	}

	if execErr != nil {
		s.logger.Errorf("[Scheduler] 任务 %s 执行失败: %v", req.TaskID, execErr)
	} else {
		s.logger.Infof("[Scheduler] 任务 %s 执行完成 (状态: %s, 耗时: %dms)",
			req.TaskID, result.Status, result.Duration)
	}

	return result, execErr
}

// StopTask 停止正在运行的任务
func (s *Scheduler) StopTask(taskID string) bool {
	s.mu.RLock()
	cancel, exists := s.runningTasks[taskID]
	s.mu.RUnlock()

	if exists && cancel != nil {
		cancel()
		s.logger.Infof("[Scheduler] 已尝试停止任务 %s", taskID)
		return true
	}
	return false
}

// GetRunningTaskCount 获取正在运行的任务数量
func (s *Scheduler) GetRunningTaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.runningTasks)
}

// GetRunningTasks 获取所有正在运行的任务 ID
func (s *Scheduler) GetRunningTasks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.runningTasks))
	for id := range s.runningTasks {
		ids = append(ids, id)
	}
	return ids
}

// Reload 重新加载配置
func (s *Scheduler) Reload(config SchedulerConfig) {
	s.logger.Infof("[Scheduler] 正在重载配置...")

	// 停止现有 workers
	close(s.stopCh)
	s.wg.Wait()

	// 更新配置
	s.mu.Lock()
	s.config = config
	s.taskQueue = make(chan *ExecutionRequest, config.QueueSize)
	s.rateLimiter = time.Tick(config.RateInterval)
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	// 重启 workers
	s.Start()

	s.logger.Infof("[Scheduler] 配置已重载: workers=%d, queue=%d, rate=%v",
		config.WorkerCount, config.QueueSize, config.RateInterval)
}

// GetQueueSize 获取当前队列大小
func (s *Scheduler) GetQueueSize() int {
	return len(s.taskQueue)
}

// GetConfig 获取配置
func (s *Scheduler) GetConfig() SchedulerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}
