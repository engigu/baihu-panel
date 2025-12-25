package services

import (
	"sync"
	"time"

	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"

	"github.com/robfig/cron/v3"
)

// CronService manages scheduled tasks using robfig/cron
type CronService struct {
	cron                *cron.Cron
	taskService         *TaskService
	executorService     *ExecutorService
	syncTaskService     *SyncTaskService
	syncExecutorService *SyncExecutorService
	entryMap            map[uint]cron.EntryID // task ID -> cron entry ID
	syncEntryMap        map[uint]cron.EntryID // sync task ID -> cron entry ID
	mu                  sync.RWMutex
}

// NewCronService creates a new cron service
func NewCronService(taskService *TaskService, executorService *ExecutorService) *CronService {
	// 使用秒级精度的 cron parser，支持 6 位表达式（秒 分 时 日 月 周）
	c := cron.New(cron.WithSeconds())

	return &CronService{
		cron:            c,
		taskService:     taskService,
		executorService: executorService,
		entryMap:        make(map[uint]cron.EntryID),
		syncEntryMap:    make(map[uint]cron.EntryID),
	}
}

// SetSyncServices 设置同步服务（避免循环依赖）
func (cs *CronService) SetSyncServices(syncTaskService *SyncTaskService, syncExecutorService *SyncExecutorService) {
	cs.syncTaskService = syncTaskService
	cs.syncExecutorService = syncExecutorService
}

// Start starts the cron service and loads all enabled tasks
func (cs *CronService) Start() {
	cs.loadTasks()
	cs.loadSyncTasks()
	cs.cron.Start()
	logger.Info("Cron service started")
}

// Stop stops the cron service
func (cs *CronService) Stop() {
	ctx := cs.cron.Stop()
	<-ctx.Done()
	logger.Info("Cron service stopped")
}

// loadTasks loads all enabled tasks from database
func (cs *CronService) loadTasks() {
	tasks := cs.taskService.GetTasks()
	for _, task := range tasks {
		if task.Enabled {
			err := cs.AddTask(&task)
			if err != nil {
				return
			}
		}
	}
}

// AddTask adds a task to the cron scheduler
func (cs *CronService) AddTask(task *models.Task) error {
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
		logger.Errorf("Failed to add task %d: %v", task.ID, err)
		return err
	}

	cs.entryMap[task.ID] = entryID
	cs.mu.Unlock()

	logger.Infof("Task %d (%s) scheduled with cron: %s", task.ID, task.Name, task.Schedule)

	// 更新下次运行时间
	cs.updateNextRun(task.ID)
	return nil
}

// RemoveTask removes a task from the cron scheduler
func (cs *CronService) RemoveTask(taskID uint) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if entryID, exists := cs.entryMap[taskID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.entryMap, taskID)
		logger.Infof("Task %d removed from scheduler", taskID)
	}
}

// runTask executes a task and updates its status
func (cs *CronService) runTask(taskID uint) {
	logger.Infof("Running task %d", taskID)

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
	return len(cs.entryMap) + len(cs.syncEntryMap)
}

// loadSyncTasks loads all enabled sync tasks from database
func (cs *CronService) loadSyncTasks() {
	if cs.syncTaskService == nil {
		return
	}
	tasks := cs.syncTaskService.GetSyncTasks()
	for _, task := range tasks {
		if task.Enabled {
			err := cs.AddSyncTask(&task)
			if err != nil {
				logger.Errorf("Failed to add sync task %d: %v", task.ID, err)
			}
		}
	}
}

// AddSyncTask adds a sync task to the cron scheduler
func (cs *CronService) AddSyncTask(task *models.SyncTask) error {
	cs.mu.Lock()

	// 如果已存在，先移除
	if entryID, exists := cs.syncEntryMap[task.ID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.syncEntryMap, task.ID)
	}

	taskID := task.ID
	entryID, err := cs.cron.AddFunc(task.Schedule, func() {
		cs.runSyncTask(taskID)
	})
	if err != nil {
		cs.mu.Unlock()
		logger.Errorf("Failed to add sync task %d: %v", task.ID, err)
		return err
	}

	cs.syncEntryMap[task.ID] = entryID
	cs.mu.Unlock()

	logger.Infof("Sync task %d (%s) scheduled with cron: %s", task.ID, task.Name, task.Schedule)

	// 更新下次运行时间
	cs.updateSyncNextRun(task.ID)
	return nil
}

// RemoveSyncTask removes a sync task from the cron scheduler
func (cs *CronService) RemoveSyncTask(taskID uint) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if entryID, exists := cs.syncEntryMap[taskID]; exists {
		cs.cron.Remove(entryID)
		delete(cs.syncEntryMap, taskID)
		logger.Infof("Sync task %d removed from scheduler", taskID)
	}
}

// runSyncTask executes a sync task and updates its status
func (cs *CronService) runSyncTask(taskID uint) {
	logger.Infof("Running sync task %d", taskID)

	// 更新 last_sync
	now := time.Now()
	database.DB.Model(&models.SyncTask{}).Where("id = ?", taskID).Update("last_sync", now)

	// 执行同步任务
	if cs.syncExecutorService != nil {
		go cs.syncExecutorService.ExecuteSyncTask(int(taskID))
	}

	// 更新 next_sync
	cs.updateSyncNextRun(taskID)
}

// updateSyncNextRun updates the next run time for a sync task
func (cs *CronService) updateSyncNextRun(taskID uint) {
	cs.mu.RLock()
	entryID, exists := cs.syncEntryMap[taskID]
	cs.mu.RUnlock()

	if !exists {
		return
	}

	entry := cs.cron.Entry(entryID)
	if !entry.Next.IsZero() {
		database.DB.Model(&models.SyncTask{}).Where("id = ?", taskID).Update("next_sync", entry.Next)
	}
}
