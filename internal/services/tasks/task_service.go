package tasks

import (
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
)

type TaskService struct{}

func NewTaskService() *TaskService {
	return &TaskService{}
}

func (ts *TaskService) CreateTask(name, command, schedule string, timeout int, workDir, cleanConfig, envs, taskType, config string, agentID *uint, language, langVersion string) *models.Task {
	if taskType == "" {
		taskType = "task"
	}
	task := &models.Task{
		Name:        name,
		Command:     command,
		Type:        taskType,
		Config:      config,
		Schedule:    schedule,
		Timeout:     timeout,
		WorkDir:     workDir,
		CleanConfig: cleanConfig,
		Envs:        envs,
		Language:    language,
		LangVersion: langVersion,
		AgentID:     agentID,
		Enabled:     true,
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
func (ts *TaskService) GetTasksWithPagination(page, pageSize int, name string, agentID *uint) ([]models.Task, int64) {
	var tasks []models.Task
	var total int64

	query := database.DB.Model(&models.Task{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if agentID != nil {
		query = query.Where("agent_id = ?", *agentID)
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	return tasks, total
}

func (ts *TaskService) GetTaskByID(id int) *models.Task {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil
	}
	return &task
}

func (ts *TaskService) UpdateTask(id int, name, command, schedule string, timeout int, workDir, cleanConfig, envs string, enabled bool, taskType, config string, agentID *uint, language, langVersion string) *models.Task {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil
	}
	task.Name = name
	task.Command = command
	task.Schedule = schedule
	task.Timeout = timeout
	task.WorkDir = workDir
	task.CleanConfig = cleanConfig
	task.Envs = envs
	task.Enabled = enabled
	task.AgentID = agentID
	task.Language = language
	task.LangVersion = langVersion
	if taskType != "" {
		task.Type = taskType
	}
	task.Config = config
	database.DB.Save(&task)
	return &task
}

func (ts *TaskService) DeleteTask(id int) bool {
	result := database.DB.Delete(&models.Task{}, id)
	return result.RowsAffected > 0
}
