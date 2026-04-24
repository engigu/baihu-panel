package tasks

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/common"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/engigu/baihu-panel/internal/utils/idgen"
	"strings"
)

type TaskService struct {
	relationService *common.RelationService
}

func NewTaskService() *TaskService {
	return &TaskService{
		relationService: common.NewRelationService(),
	}
}

func (ts *TaskService) GetTaskBySourceID(sourceID string) *models.Task {
	var task models.Task
	res := database.DB.Where("source_id = ?", sourceID).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}
	return &task
}

func (ts *TaskService) CreateTask(name, command, schedule string, timeout int, workDir, cleanConfig, envs, taskType, config string, agentID *string, languages models.TaskLanguages, triggerType string, tags string, retryCount int, retryInterval int, randomRange int, sourceID string) *models.Task {
	if taskType == "" {
		taskType = "task"
	}
	if triggerType == "" {
		triggerType = constant.TriggerTypeCron
	}
	task := &models.Task{
		ID:            idgen.GenerateID(),
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
		Enabled:       utils.BoolPtr(true),
		RetryCount:    retryCount,
		RetryInterval: retryInterval,
		RandomRange:   randomRange,
		SourceID:      sourceID,
		CreatedAt:     models.Now(),
		UpdatedAt:     models.Now(),
	}
	if triggerType != constant.TriggerTypeCron {
		task.NextRun = nil
	}
	database.DB.Select("*").Create(task)

	// 更新标签关联关系
	if tags != "" {
		tagList := strings.Split(tags, ",")
		ts.relationService.UpdateRelations(task.ID, "task_tag", "tag", tagList)
	}

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
	if tags != "" {
		// 使用子查询进行关联过滤，支持精确匹配标签
		subQuery := database.DB.Table(models.DataRelation{}.TableName()).
			Select("data_id").
			Joins("JOIN "+models.DataStorage{}.TableName()+" ds ON "+models.DataRelation{}.TableName()+".relate_id = ds.id").
			Where(models.DataRelation{}.TableName()+".type = ? AND ds.type = ? AND ds.key = ?", "task_tag", "tag", tags)
		query = query.Where("id IN (?)", subQuery)
	}
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if agentID != nil {
		query = query.Where("agent_id = ?", *agentID)
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	// 处理返回的任务标签（如果需要从关联表实时获取，可以取消下面注释，目前为了兼容性仍读取 Task.Tags 字段）
	/*
	for i := range tasks {
		if keys, err := ts.relationService.GetRelatedKeys(tasks[i].ID, "task_tag"); err == nil {
			tasks[i].Tags = strings.Join(keys, ",")
		}
	}
	*/

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

func (ts *TaskService) UpdateTask(id string, name, command, schedule string, timeout int, workDir, cleanConfig, envs string, enabled bool, taskType, config string, agentID *string, languages models.TaskLanguages, triggerType string, tags string, retryCount int, retryInterval int, randomRange int, sourceID string) *models.Task {
	var task models.Task
	res := database.DB.Where("id = ?", id).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
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
	task.Enabled = &enabled
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
	if sourceID != "" {
		task.SourceID = sourceID
	}

	database.DB.Model(&task).Select(
		"Name", "Command", "Tags", "Schedule", "Timeout", "WorkDir",
		"CleanConfig", "Envs", "Enabled", "AgentID", "Languages",
		"RetryCount", "RetryInterval", "RandomRange", "Type",
		"TriggerType", "Config", "SourceID",
	).Updates(&task)

	// 更新标签关联关系
	tagList := strings.Split(tags, ",")
	ts.relationService.UpdateRelations(task.ID, "task_tag", "tag", tagList)

	return &task
}

func (ts *TaskService) DeleteTask(id string) bool {
	// 同时删除关联的通知推送设置
	database.DB.Where("type = ? AND data_id = ?", constant.BindingTypeTask, id).Delete(&models.NotifyBinding{})
	
	// 删除标签关联关系
	database.DB.Where("type = ? AND data_id = ?", "task_tag", id).Delete(&models.DataRelation{})

	result := database.DB.Where("id = ?", id).Delete(&models.Task{})
	return result.RowsAffected > 0
}

func (ts *TaskService) BatchDeleteTasks(ids []string) int64 {
	// 同时删除关联的通知推送设置
	database.DB.Where("type = ? AND data_id IN ?", constant.BindingTypeTask, ids).Delete(&models.NotifyBinding{})
	
	// 批量删除标签关联关系
	database.DB.Where("type = ? AND data_id IN ?", "task_tag", ids).Delete(&models.DataRelation{})

	result := database.DB.Where("id IN ?", ids).Delete(&models.Task{})
	return result.RowsAffected
}

func (ts *TaskService) GetAllTags() []string {
	keys, _ := ts.relationService.GetDistinctKeys("tag")
	return keys
}
