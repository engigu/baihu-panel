package executor

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// 东八区时区（默认）
var defaultLocation = time.FixedZone("CST", 8*3600)

// CronManager 统一的任务调度管理器
type CronManager struct {
	cron      *cron.Cron
	scheduler *Scheduler
	entryMap  map[string]cron.EntryID // task ID -> cron entry ID
	mu        sync.RWMutex
	logger    SchedulerLogger
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

	entryID, err := m.cron.AddFunc(task.GetSchedule(), func() {
		m.logger.Infof("[CronManager] 触发计划任务 #%s (%s)", taskID, name)

		req := &ExecutionRequest{
			TaskID:  taskID,
			Name:    name,
			Command: cmd,
			Type:    TaskTypeCron,
			Timeout: timeout,
		}

		// 如果有关联的 Scheduler，加入队列执行
		if m.scheduler != nil {
			m.scheduler.EnqueueOrExecute(req)
		}

		// 触发下次运行时间更新事件
		m.triggerNextRunEvent(taskID, req)
	})

	if err != nil {
		m.logger.Errorf("[CronManager] 添加任务失败 #%s: %v", taskID, err)
		return err
	}

	m.entryMap[taskID] = entryID
	m.logger.Infof("[CronManager] 任务已调度 #%s %s (%s)", taskID, name, task.GetSchedule())

	// 初始触发一次下次运行时间通知
	go func() {
		req := &ExecutionRequest{TaskID: taskID, Name: name, Type: TaskTypeCron}
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

// ValidateCron 校验 Cron 表达式
func (m *CronManager) ValidateCron(expression string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(expression)
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
