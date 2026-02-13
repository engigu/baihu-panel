package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
)

// safeBuffer 一个线程安全的字节缓冲区，用于合并 stdout 和 stderr
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	WorkerCount  int           // Worker 数量
	QueueSize    int           // 队列大小
	RateInterval time.Duration // 速率限制间隔
	Verbose      bool          // 是否开启详细日志
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
	TaskStatusPending   TaskStatus = TaskStatus(constant.TaskStatusPending)   // 等待中
	TaskStatusRunning   TaskStatus = TaskStatus(constant.TaskStatusRunning)   // 运行中
	TaskStatusSuccess   TaskStatus = TaskStatus(constant.TaskStatusSuccess)   // 成功
	TaskStatusFailed    TaskStatus = TaskStatus(constant.TaskStatusFailed)    // 失败
	TaskStatusTimeout   TaskStatus = TaskStatus(constant.TaskStatusTimeout)   // 超时
	TaskStatusCancelled TaskStatus = TaskStatus(constant.TaskStatusCancelled) // 已取消
)

// ExecutionRequest 执行请求（标准接口）
type ExecutionRequest struct {
	TaskID      string                 // 任务 ID
	LogID       uint                   // 日志 ID
	Name        string                 // 任务名称
	Type        TaskType               // 任务类型
	Command     string                 // 命令
	WorkDir     string                 // 工作目录
	Envs        []string               // 环境变量
	Timeout     int                    // 超时时间（分钟）
	Language    string                 // 语言环境
	LangVersion string                 // 语言版本
	UseMise     bool                   // 是否使用 mise
	Metadata    map[string]interface{} // 额外元数据
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
	runningTasks map[string]context.CancelFunc // 记录运行中的任务，用于停止 (TaskID -> CancelFunc)
	runningExecs map[uint]context.CancelFunc   // 记录运行中的执行，用于停止 (LogID -> CancelFunc)
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
				Command:     req.Command,
				WorkDir:     req.WorkDir,
				Envs:        req.Envs,
				Timeout:     req.Timeout,
				Language:    req.Language,
				LangVersion: req.LangVersion,
				UseMise:     req.UseMise,
			}, stdout, stderr, hooks)
		},
		taskQueue:    make(chan *ExecutionRequest, config.QueueSize),
		rateLimiter:  time.Tick(config.RateInterval),
		stopCh:       make(chan struct{}),
		logger:       &DefaultLogger{},
		runningTasks: make(map[string]context.CancelFunc),
		runningExecs: make(map[uint]context.CancelFunc),
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
			func() {
				defer func() {
					if r := recover(); r != nil {
						s.logger.Errorf("[Scheduler] Worker %d panic while processing task %s: %v", id, req.TaskID, r)
					}
				}()
				// 速率限制
				<-s.rateLimiter
				s.executeTask(req)
			}()
		}
	}
}

// executeTask 执行任务（本地执行）
func (s *Scheduler) executeTask(req *ExecutionRequest) (*ExecutionResult, error) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Errorf("[Scheduler] 任务 %s 执行过程中发生 Panic: %v", req.TaskID, r)
		}
	}()
	start := time.Now()

	s.logger.Infof("[Scheduler] 执行任务 %s (名称: %s, 类型: %s)", req.TaskID, req.Name, req.Type)

	finalCommand := req.Command
	if req.UseMise {
		finalCommand = BuildLanguageCommand(req.Command, req.Language, req.LangVersion)
	}
	s.logger.Infof("[Scheduler] 实际执行命令: %s", finalCommand)

	if s.config.Verbose {
		workDir := req.WorkDir
		if workDir == "" {
			workDir, _ = os.Getwd()
		}
		s.logger.Infof("[Scheduler] 任务 #%s 进程 UID: %d, GID: %d", req.TaskID, os.Getuid(), os.Getgid())
		s.logger.Infof("[Scheduler] 任务 #%s 工作目录: %s", req.TaskID, workDir)
	}

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
				Status:    constant.TaskStatusFailed,
				Error:     err.Error(),
				Duration:  0,
				ExitCode:  1,
				StartTime: start,
				EndTime:   time.Now(),
			}, err
		}
	}

	// 2. 准备输出缓冲区（使用合并缓冲区保证顺序）
	var combinedBuf safeBuffer
	var stdoutWriter, stderrWriter io.Writer

	if stdout != nil && stdout == stderr {
		// 如果 stdout 和 stderr 是同一个对象，合并成一个 MultiWriter
		// 这样后面 ExecuteWithHooks 才能识别出它们是同一个，从而开启 PTY 模式
		mw := io.MultiWriter(&combinedBuf, stdout)
		stdoutWriter = mw
		stderrWriter = mw
	} else {
		if stdout != nil {
			stdoutWriter = io.MultiWriter(&combinedBuf, stdout)
		} else {
			stdoutWriter = &combinedBuf
		}

		if stderr != nil {
			stderrWriter = io.MultiWriter(&combinedBuf, stderr)
		} else {
			stderrWriter = &combinedBuf
		}
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
	if req.LogID > 0 {
		s.runningExecs[req.LogID] = cancel
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningTasks, req.TaskID)
		if req.LogID > 0 {
			delete(s.runningExecs, req.LogID)
		}
		s.mu.Unlock()
	}()

	execResult, execErr := s.executor(ctx, req, stdoutWriter, stderrWriter)

	// 5. 构建结果
	result := &ExecutionResult{
		TaskID: req.TaskID,
		LogID:  req.LogID, // 传递 LogID
	}

	if execResult != nil {
		result.Success = execResult.Status == constant.TaskStatusSuccess
		result.Output = combinedBuf.String()
		result.Status = execResult.Status
		result.Duration = execResult.Duration
		result.ExitCode = execResult.ExitCode
		result.StartTime = execResult.StartTime
		result.EndTime = execResult.EndTime
	} else {
		result.Success = false
		result.Status = constant.TaskStatusFailed
		result.StartTime = start
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Milliseconds()
		result.Output = combinedBuf.String()
	}

	if execErr != nil {
		result.Error = execErr.Error()
		if ctx.Err() == context.Canceled {
			result.Status = constant.TaskStatusCancelled
		} else if ctx.Err() == context.DeadlineExceeded {
			result.Status = constant.TaskStatusTimeout
		}
	}

	// 6. 执行后事件
	if s.handler != nil {
		if execResult != nil {
			// 只要有执行结果（即使执行失败），都认为是任务完成了（包含输出）
			s.handler.OnTaskCompleted(req, result)
		} else if execErr != nil {
			// 只有在完全没有结果的情况下（如无法启动、Panic等），才认为是任务失败
			s.handler.OnTaskFailed(req, execErr)
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

// StopTask 停止正在运行的任务（通过 TaskID，可能会停止多个并发副本）
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

// StopLog 停止正在运行的任务（通过 LogID，精确停止单个执行副本）
func (s *Scheduler) StopLog(logID uint) bool {
	s.mu.RLock()
	cancel, exists := s.runningExecs[logID]
	s.mu.RUnlock()

	if exists && cancel != nil {
		cancel()
		s.logger.Infof("[Scheduler] 已尝试停止任务执行 #%d", logID)
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
