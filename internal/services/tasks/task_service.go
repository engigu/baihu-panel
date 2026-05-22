package tasks

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
	"strings"
)

// TaskParam 任务创建与更新参数传输对象
type TaskParam struct {
	Name          string
	Remark        string
	Command       string
	PreCommand    string
	PostCommand   string
	Tags          string
	Type          string
	Config        string
	Schedule      string
	Timeout       int
	WorkDir       string
	CleanConfig   string
	Envs          string
	Languages     models.TaskLanguages
	AgentID       *string
	TriggerType   string
	RetryCount    int
	RetryInterval int
	RandomRange   int
	SourceID      string
	PinType       string
	Enabled       bool
}

type TaskService struct {
}

func NewTaskService() *TaskService {
	return &TaskService{}
}

func (ts *TaskService) GetTaskBySourceID(sourceID string) *models.Task {
	var task models.Task
	res := database.DB.Where("source_id = ?", sourceID).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}
	return &task
}

func (ts *TaskService) CreateTask(p *TaskParam) *models.Task {
	if p.Type == "" {
		p.Type = "task"
	}
	if p.TriggerType == "" {
		p.TriggerType = constant.TriggerTypeCron
	}
	if p.PinType == "" {
		p.PinType = constant.PinTypeNone
	}
	task := &models.Task{
		ID:            utils.GenerateID(),
		Name:          p.Name,
		Remark:        p.Remark,
		Command:       models.BigText(p.Command),
		PreCommand:    models.BigText(p.PreCommand),
		PostCommand:   models.BigText(p.PostCommand),
		PinType:       p.PinType,
		Tags:          p.Tags,
		Type:          p.Type,
		TriggerType:   p.TriggerType,
		Config:        models.BigText(p.Config),
		Schedule:      p.Schedule,
		Timeout:       p.Timeout,
		WorkDir:       p.WorkDir,
		CleanConfig:   p.CleanConfig,
		Envs:          models.BigText(p.Envs),
		Languages:     p.Languages,
		AgentID:       p.AgentID,
		Enabled:       utils.BoolPtr(true),
		RetryCount:    p.RetryCount,
		RetryInterval: p.RetryInterval,
		RandomRange:   p.RandomRange,
		SourceID:      p.SourceID,
		CreatedAt:     models.Now(),
		UpdatedAt:     models.Now(),
	}
	if p.TriggerType != constant.TriggerTypeCron {
		task.NextRun = nil
	}
	database.DB.Select("*").Create(task)

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
		query = query.Where("name LIKE ? OR remark LIKE ?", "%"+name+"%", "%"+name+"%")
	}

	// 标签筛选 (并集)
	if tags != "" {
		tagList := strings.Split(tags, ",")
		var orConditions []string
		var orValues []interface{}
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				orConditions = append(orConditions, "tags LIKE ?")
				orValues = append(orValues, "%"+tag+"%")
			}
		}
		if len(orConditions) > 0 {
			query = query.Where(strings.Join(orConditions, " OR "), orValues...)
		}
	}

	if taskType != "" && taskType != "all" {
		query = query.Where("type = ?", taskType)
	}
	if agentID != nil {
		query = query.Where("agent_id = ?", *agentID)
	}

	query.Count(&total)
	query.Order("pin_type DESC, created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	return tasks, total
}

func (ts *TaskService) GetTaskByID(id string) *models.Task {
	var task models.Task
	res := database.DB.Where("id = ?", id).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}
	return &task
}

func (ts *TaskService) UpdateTask(id string, p *TaskParam) *models.Task {
	var task models.Task
	res := database.DB.Where("id = ?", id).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}
	task.Name = p.Name
	task.Remark = p.Remark
	task.Command = models.BigText(p.Command)
	task.PreCommand = models.BigText(p.PreCommand)
	task.PostCommand = models.BigText(p.PostCommand)
	task.PinType = p.PinType
	task.Tags = p.Tags
	task.Schedule = p.Schedule
	task.Timeout = p.Timeout
	task.WorkDir = p.WorkDir
	task.CleanConfig = p.CleanConfig
	task.Envs = models.BigText(p.Envs)
	task.Enabled = &p.Enabled
	task.AgentID = p.AgentID
	task.Languages = p.Languages
	task.Config = models.BigText(p.Config)
	task.RetryCount = p.RetryCount
	task.RetryInterval = p.RetryInterval
	task.RandomRange = p.RandomRange
	if p.Type != "" {
		task.Type = p.Type
	}
	if p.TriggerType != "" {
		task.TriggerType = p.TriggerType
	}
	if p.SourceID != "" {
		task.SourceID = p.SourceID
	}

	database.DB.Model(&task).Select(
		"Name", "Remark", "Command", "Tags", "Schedule", "Timeout", "WorkDir",
		"CleanConfig", "Envs", "Enabled", "AgentID", "Languages",
		"RetryCount", "RetryInterval", "RandomRange", "Type",
		"TriggerType", "Config", "SourceID", "PinType",
		"PreCommand", "PostCommand",
	).Updates(&task)

	return &task
}

func (ts *TaskService) DeleteTask(id string) bool {
	// 同时删除关联的通知推送设置
	database.DB.Where("type = ? AND data_id = ?", constant.BindingTypeTask, id).Delete(&models.NotifyBinding{})

	result := database.DB.Where("id = ?", id).Delete(&models.Task{})
	return result.RowsAffected > 0
}

func (ts *TaskService) BatchDeleteTasks(ids []string) int64 {
	// 同时删除关联的通知推送设置
	database.DB.Where("type = ? AND data_id IN ?", constant.BindingTypeTask, ids).Delete(&models.NotifyBinding{})

	result := database.DB.Where("id IN ?", ids).Delete(&models.Task{})
	return result.RowsAffected
}

// GetAllTags 获取所有任务标签
func (ts *TaskService) GetAllTags() ([]string, error) {
	var tasks []models.Task
	database.DB.Select("tags").Where("tags != ?", "").Find(&tasks)

	tagMap := make(map[string]bool)
	for _, task := range tasks {
		tags := strings.Split(task.Tags, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagMap[tag] = true
			}
		}
	}

	result := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		result = append(result, tag)
	}
	return result, nil
}
