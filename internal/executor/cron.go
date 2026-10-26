package executor

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/systime"

	"github.com/robfig/cron/v3"
)

// 东八区时区（默认）
var defaultLocation = systime.CST

// CronManager 统一的任务调度管理器
type CronManager struct {
	cron      *cron.Cron
	scheduler *Scheduler
	entryMap  map[string]cron.EntryID // task ID -> cron entry ID
	mu        sync.RWMutex
	logger    SchedulerLogger
	OnTrigger func(task CronTask) *ExecutionRequest // 任务触发时的请求构造工厂
}

// NewCronManager 创建一个新的计划任务管理器
func NewCronManager(scheduler *Scheduler) *CronManager {
	// 使用秒级精度的 cron parser
	c := cron.New(cron.WithSeconds(), cron.WithLocation(defaultLocation))

	m := &CronManager{
		cron:      c,
		scheduler: scheduler,
		entryMap:  make(map[string]cron.EntryID),
		logger:    &DefaultLogger{},
	}

	if scheduler != nil && scheduler.logger != nil {
		m.logger = scheduler.logger
	}

	return m
}

// SetLogger 设置自定义日志实现
func (m *CronManager) SetLogger(logger SchedulerLogger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logger = logger
}

// SetScheduler 更新关联的调度器实例
func (m *CronManager) SetScheduler(scheduler *Scheduler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scheduler = scheduler
}

// Start 启动调度器
func (m *CronManager) Start() {
	m.cron.Start()
	m.logger.Infof("[CronManager] 调度管理服务已启动")
}

// Stop 停止调度器
func (m *CronManager) Stop() {
	ctx := m.cron.Stop()
	<-ctx.Done()
	m.logger.Infof("[CronManager] 调度管理服务已停止")
}

// AddTask 添加或更新计划任务
func (m *CronManager) AddTask(task CronTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskID := task.GetID()

	// 如果已存在，先移除旧的
	if entryID, exists := m.entryMap[taskID]; exists {
		m.cron.Remove(entryID)
		delete(m.entryMap, taskID)
	}

	// 准备任务执行函数
	cmd := task.GetCommand()
	name := task.GetName()
	timeout := task.GetTimeout()
	workDir := task.GetWorkDir()
	envs := task.GetEnvs()
	languages := task.GetLanguages()
	useMise := task.UseMise()
	secrets := task.GetSecrets()

	// 校验并规整调度表达式（允许应用主任务/手动触发任务为空）
	validRes, err := ParseAndValidateSchedule(task.GetSchedule(), true)
	if err != nil {
		m.logger.Errorf("[CronManager] 任务 #%s 调度表达式非法: %v", taskID, err)
		return err
	}
	if validRes.IsEmpty {
		// 空调度（如应用主任务的调和入口、手动触发型任务）不注册 Cron；
		// 旧调度已在上方移除，此处干净跳过而非报错
		m.logger.Infof("[CronManager] 任务 %s (#%s) 无调度配置，跳过 Cron 注册", name, taskID)
		return nil
	}
	schedule := validRes.Normalized
	entryID, err := m.cron.AddFunc(schedule, func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Errorf("[CronManager] 任务 #%s 执行过程中发生 Panic: %v", taskID, r)
			}
		}()

		// 构造执行请求的 Builder
		reqBuilder := func() *ExecutionRequest {
			if m.OnTrigger != nil {
				return m.OnTrigger(task)
			}
			return &ExecutionRequest{
				TaskID:      taskID,
				Name:        name,
				Command:     cmd,
				PreCommand:  task.GetPreCommand(),
				PostCommand: task.GetPostCommand(),
				Type:        TaskTypeCron,
				Timeout:     timeout,
				WorkDir:     workDir,
				Envs: func() []string {
					if vars := task.GetEnvVars(); len(vars) > 0 {
						return vars
					}
					return ParseEnvVars(envs)
				}(),
				Secrets:   secrets,
				Languages: languages,
				UseMise:   useMise,
			}
		}

		m.mu.RLock()
		sched := m.scheduler
		m.mu.RUnlock()

		randomRange := task.GetRandomRange()
		if randomRange > 0 && sched != nil {
			// 生成 0 到 randomRange 之间的随机秒数
			delaySeconds := rand.Intn(randomRange)
			delay := time.Duration(delaySeconds) * time.Second
			m.logger.Infof("[CronManager] 任务 %s (#%s) 将随机延迟 %v (范围: %ds) 后入队", name, taskID, delay, randomRange)

			// 使用调度器的延时投递功能，不阻塞当前 Cron 协程
			sched.EnqueueDelayed(delay, reqBuilder)
		} else {
			m.logger.Infof("[CronManager] 触发计划任务: %s (#%s)", name, taskID)
			if sched != nil {
				sched.EnqueueOrExecute(reqBuilder())
			}
		}

		// 触发下次运行时间更新事件
		m.triggerNextRunEvent(taskID, &ExecutionRequest{TaskID: taskID})
	})

	if err != nil {
		m.logger.Errorf("[CronManager] 添加任务失败 #%s: %v", taskID, err)
		return err
	}

	m.entryMap[taskID] = entryID
	m.logger.Infof("[CronManager] 已添加调度: %s (#%s) [类型: %s, Cron: %s]", name, taskID, task.GetType(), task.GetSchedule())

	// 初始触发一次下次运行时间通知
	go func() {
		req := &ExecutionRequest{
			TaskID:  taskID,
			Name:    name,
			Type:    TaskTypeCron,
			UseMise: task.UseMise(),
		}
		m.triggerNextRunEvent(taskID, req)
	}()

	return nil
}

// RemoveTask 移除计划任务
func (m *CronManager) RemoveTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entryID, exists := m.entryMap[taskID]; exists {
		m.cron.Remove(entryID)
		delete(m.entryMap, taskID)
		m.logger.Infof("[CronManager] 任务已移除 #%s", taskID)
	}
}

// ClearTasks 清空所有计划任务
func (m *CronManager) ClearTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for taskID, entryID := range m.entryMap {
		m.cron.Remove(entryID)
		delete(m.entryMap, taskID)
	}
	m.logger.Infof("[CronManager] 已清空所有计划任务调度")
}

// triggerNextRunEvent 触发下次运行时间更新事件
func (m *CronManager) triggerNextRunEvent(taskID string, req *ExecutionRequest) {
	m.mu.RLock()
	entryID, exists := m.entryMap[taskID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	entry := m.cron.Entry(entryID)
	if !entry.Next.IsZero() && m.scheduler != nil && m.scheduler.handler != nil {
		m.scheduler.handler.OnCronNextRun(req, entry.Next)
	}
}

// ScheduleValidationResult 调度表达式校验与归一化结果
type ScheduleValidationResult struct {
	Original   string // 原始输入字符串
	Normalized string // 归一化后的标准 6 位 Cron 表达式（或 @ 宏指令）
	IsEmpty    bool   // 是否为空调度
	Fields     int    // 字段位数 (5, 6 或 0)
}

// ParseAndValidateSchedule 专门用于任务调度表达式的校验、5/6 位自适应与归一化
// allowEmpty: 是否允许空调度（应用主任务/手动任务传 true，定时脚本任务传 false）
func ParseAndValidateSchedule(schedule string, allowEmpty bool) (*ScheduleValidationResult, error) {
	raw := strings.TrimSpace(schedule)
	if raw == "" {
		if allowEmpty {
			return &ScheduleValidationResult{Original: raw, IsEmpty: true}, nil
		}
		return nil, fmt.Errorf("定时调度表达式不能为空")
	}

	result := &ScheduleValidationResult{
		Original: raw,
		IsEmpty:  false,
	}

	// 1. 处理 @ 宏指令（如 @every 1h, @daily, @hourly）
	if strings.HasPrefix(raw, "@") {
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(raw); err != nil {
			return nil, fmt.Errorf("无效的调度宏指令 [%s]: %w", raw, err)
		}
		result.Normalized = raw
		return result, nil
	}

	// 2. 字段位数分析与自适应补齐
	fields := strings.Fields(raw)
	result.Fields = len(fields)

	switch len(fields) {
	case 5:
		// 5 位标准 Linux Cron (分 时 日 月 周) 自动在秒位补 0，规整为底层支持的 6 位秒级 Cron
		result.Normalized = "0 " + raw
	case 6:
		// 6 位标准秒级 Cron 原样保留
		result.Normalized = raw
	default:
		return nil, fmt.Errorf("无效的 Cron 表达式 [%s]: 仅支持 5 位 (分 时 日 月 周) 或 6 位 (秒 分 时 日 月 周)，当前为 %d 位", raw, len(fields))
	}

	// 3. 底层标准秒级解析器深度语法校验
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(result.Normalized); err != nil {
		return nil, fmt.Errorf("Cron 语法校验失败 [%s]: %w", raw, err)
	}

	return result, nil
}

// ValidateCron 校验 Cron 表达式（支持 5 位 Linux Cron 与 6 位秒级 Cron 自适应）
func (m *CronManager) ValidateCron(expression string) error {
	_, err := ParseAndValidateSchedule(expression, false)
	return err
}

// GetEntry 获取任务详情
func (m *CronManager) GetEntry(taskID string) (cron.Entry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entryID, exists := m.entryMap[taskID]
	if !exists {
		return cron.Entry{}, false
	}

	return m.cron.Entry(entryID), true
}

// GetScheduledCount 获取已调度任务总数
func (m *CronManager) GetScheduledCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entryMap)
}
