package tasks

import (
	"sync"
	"time"

	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"

	"github.com/robfig/cron/v3"
)

// 东八区时区
var cstZone = time.FixedZone("CST", 8*3600)

// CronService manages scheduled tasks using robfig/cron
type CronService struct {
	cron            *cron.Cron
	taskService     *TaskService
	executorService *ExecutorService
	entryMap        map[uint]cron.EntryID // task ID -> cron entry ID
	mu              sync.RWMutex
}

// NewCronService creates a new cron service
func NewCronService(taskService *TaskService, executorService *ExecutorService) *CronService {
	// 使用秒级精度的 cron parser，支持 6 位表达式（秒 分 时 日 月 周），使用东八区时区
	c := cron.New(cron.WithSeconds(), cron.WithLocation(cstZone))

	return &CronService{
		cron:            c,
		taskService:     taskService,
		executorService: executorService,
		entryMap:        make(map[uint]cron.EntryID),
	}
}

// Start starts the cron service and loads all enabled tasks
func (cs *CronService) Start() {
	cs.loadTasks()
	cs.cron.Start()
	logger.Info("[Cron] 调度服务已启动")
}

// Stop stops the cron service
func (cs *CronService) Stop() {
	ctx := cs.cron.Stop()
	<-ctx.Done()
	logger.Info("[Cron] 调度服务已停止")
}

// loadTasks loads all enabled tasks from database
func (cs *CronService) loadTasks() {
	tasks := cs.taskService.GetTasks()
	count := 0
	for _, task := range tasks {
		// 只调度本地任务（agent_id 为空）
		if task.Enabled && task.AgentID == nil {
			err := cs.addTask(&task, false)
			if err != nil {
				return
			}
			count++
		}
	}
	logger.Infof("[Cron] 启动调度已加载 %d 个定时任务", count)
}

// addTask 内部添加任务方法，silent 控制是否打印日志
func (cs *CronService) addTask(task *models.Task, logEnabled bool) error {
	cs.mu.Lock()

	// 如果已存在，先移除
	if entryID, exists := cs.entryMap[task.ID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entryMap, task.ID)
	}

	taskID := task.ID
	entryID, err := cs.cron.AddFunc(task.Schedule, func() {
		cs.runTask(taskID)
	})
	if err != nil {
		cs.mu.Unlock()
		logger.Errorf("[Cron] 添加任务失败 #%d: %v", task.ID, err)
		return err
	}

	cs.entryMap[task.ID] = entryID
	cs.mu.Unlock()

	if logEnabled {
		logger.Infof("[Cron] 任务已调度 #%d %s (%s)", task.ID, task.Name, task.Schedule)
	}

	// 更新下次运行时间
	cs.updateNextRun(task.ID)
	return nil
}

// AddTask adds a task to the cron scheduler
func (cs *CronService) AddTask(task *models.Task) error {
	return cs.addTask(task, true)
}

// RemoveTask removes a task from the cron scheduler
func (cs *CronService) RemoveTask(taskID uint) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if entryID, exists := cs.entryMap[taskID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entryMap, taskID)
		logger.Infof("[Cron] 任务已移除 #%d", taskID)
	}
}

// runTask executes a task and updates its status
func (cs *CronService) runTask(taskID uint) {
	// 获取任务信息用于日志
	task := cs.taskService.GetTaskByID(int(taskID))
	if task != nil {
		logger.Infof("[Cron] 执行任务 #%d %s", taskID, task.Name)
	} else {
		logger.Infof("[Cron] 执行任务 #%d", taskID)
	}

	// 更新 last_run
	now := time.Now()
	database.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("last_run", now)

	// 将任务加入队列执行（通过 worker pool 控制并发）
	cs.executorService.EnqueueTask(int(taskID))

	// 更新 next_run
	cs.updateNextRun(taskID)
}

// updateNextRun updates the next run time for a task
func (cs *CronService) updateNextRun(taskID uint) {
	cs.mu.RLock()
	entryID, exists := cs.entryMap[taskID]
	cs.mu.RUnlock()

	if !exists {
		return
	}

	entry := cs.cron.Entry(entryID)
	if !entry.Next.IsZero() {
		database.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("next_run", entry.Next)
	}
}

// ValidateCron validates a cron expression (6 fields: second minute hour day month weekday)
func (cs *CronService) ValidateCron(expression string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(expression)
	return err
}

// GetScheduledCount returns the number of scheduled tasks
func (cs *CronService) GetScheduledCount() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.entryMap)
}
