package tasks

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

type TaskService struct{}

func NewTaskService() *TaskService {
	return &TaskService{}
}

func (ts *TaskService) CreateTask(name, command, schedule string, timeout int, workDir, cleanConfig, envs, taskType, config string, agentID *string, languages []map[string]string, triggerType string, tags string, retryCount int, retryInterval int, randomRange int) *models.Task {
	if taskType == "" {
		taskType = "task"
	}
	if triggerType == "" {
		triggerType = constant.TriggerTypeCron
	}
	task := &models.Task{
		ID:            utils.GenerateID(),
		Name:          name,
		Command:       models.BigText(command),
		Tags:          tags,
		Type:          taskType,
		TriggerType:   triggerType,
		Config:        models.BigText(config),
		Schedule:      schedule,
		Timeout:       timeout,
		WorkDir:       workDir,
		CleanConfig:   cleanConfig,
		Envs:          models.BigText(envs),
		Languages:     languages,
		AgentID:       agentID,
		Enabled:       true,
		RetryCount:    retryCount,
		RetryInterval: retryInterval,
		RandomRange:   randomRange,
		CreatedAt:     models.Now(),
		UpdatedAt:     models.Now(),
	}
	if triggerType != constant.TriggerTypeCron {
		task.NextRun = nil
	}
	database.DB.Create(task)
	return task
}

func (ts *TaskService) GetTasks() []models.Task {
	var tasks []models.Task
	database.DB.Find(&tasks)
	return tasks
}

// GetTasksWithPagination 分页获取任务列表
func (ts *TaskService) GetTasksWithPagination(page, pageSize int, name string, agentID *string, tags string, taskType string) ([]models.Task, int64) {
	var tasks []models.Task
	var total int64

	query := database.DB.Model(&models.Task{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if tags != "" {
		query = query.Where("tags LIKE ?", "%"+tags+"%")
	}
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if agentID != nil {
		query = query.Where("agent_id = ?", *agentID)
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	return tasks, total
}

func (ts *TaskService) GetTaskByID(id string) *models.Task {
	var task models.Task
	if err := database.DB.Where("id = ?", id).First(&task).Error; err != nil {
		return nil
	}
	return &task
}

func (ts *TaskService) UpdateTask(id string, name, command, schedule string, timeout int, workDir, cleanConfig, envs string, enabled bool, taskType, config string, agentID *string, languages []map[string]string, triggerType string, tags string, retryCount int, retryInterval int, randomRange int) *models.Task {
	var task models.Task
	if err := database.DB.Where("id = ?", id).First(&task).Error; err != nil {
		return nil
	}
	task.Name = name
	task.Command = models.BigText(command)
	task.Tags = tags
	task.Schedule = schedule
	task.Timeout = timeout
	task.WorkDir = workDir
	task.CleanConfig = cleanConfig
	task.Envs = models.BigText(envs)
	task.Enabled = enabled
	task.AgentID = agentID
	task.Languages = languages
	task.Config = models.BigText(config)
	task.RetryCount = retryCount
	task.RetryInterval = retryInterval
	task.RandomRange = randomRange
	if taskType != "" {
		task.Type = taskType
	}
	if triggerType != "" {
		task.TriggerType = triggerType
	}

	database.DB.Model(&task).Select(
		"Name", "Command", "Tags", "Schedule", "Timeout", "WorkDir",
		"CleanConfig", "Envs", "Enabled", "AgentID", "Languages",
		"RetryCount", "RetryInterval", "RandomRange", "Type",
		"TriggerType", "Config",
	).Updates(&task)
	return &task
}

func (ts *TaskService) DeleteTask(id string) bool {
	// 同时删除关联的通知推送设置
	database.DB.Where("type = ? AND data_id = ?", constant.BindingTypeTask, id).Delete(&models.NotifyBinding{})
	
	result := database.DB.Unscoped().Where("id = ?", id).Delete(&models.Task{})
	return result.RowsAffected > 0
}
