package tasks

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services/relation"
	"github.com/engigu/baihu-panel/internal/utils"
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
	ts.loadTagsAndEnvs([]models.Task{task})
	
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
	relation.DataRelation.SaveTags(task.ID, constant.RelationTypeTaskTag, p.Tags)
	task.Tags = p.Tags
	relation.DataRelation.SaveRelations(task.ID, constant.RelationTypeTaskEnv, p.Envs)
	task.Envs = models.BigText(p.Envs)

	return task
}

func (ts *TaskService) GetTasks() []models.Task {
	var tasks []models.Task
	database.DB.Find(&tasks)
	ts.loadTagsAndEnvs(tasks)
	return tasks
}

// GetTasksWithPagination 分页获取任务列表
func (ts *TaskService) GetTasksWithPagination(page, pageSize int, name string, agentID *string, tags string, taskType string, sortBy string, order string) ([]models.Task, int64) {
	var tasks []models.Task
	var total int64

	query := database.DB.Model(&models.Task{})
	if name != "" {
		query = query.Where("name LIKE ? OR remark LIKE ?", "%"+name+"%", "%"+name+"%")
	}

	// 标签筛选 (交集或并集均可，这里保留原本的逻辑为并集，但是利用数据关联表)
	if tags != "" {
		tagList := strings.Split(tags, ",")
		var validTags []string
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				validTags = append(validTags, tag)
			}
		}
		if len(validTags) > 0 {
			var storageIDs []string
			database.DB.Model(&models.DataStorage{}).Where("type = ? AND name IN ?", constant.RelationTypeTaskTag, validTags).Pluck("id", &storageIDs)

			var taskIDs []string
			if len(storageIDs) > 0 {
				database.DB.Model(&models.DataRelation{}).Where("type = ? AND relate_id IN ?", constant.RelationTypeTaskTag, storageIDs).Pluck("data_id", &taskIDs)
			}

			if len(taskIDs) > 0 {
				query = query.Where("id IN ?", taskIDs)
			} else {
				query = query.Where("1 = 0")
			}
		}
	}

	if taskType != "" && taskType != "all" {
		query = query.Where("type = ?", taskType)
	}
	if agentID != nil {
		query = query.Where("agent_id = ?", *agentID)
	}

	sortColumn := "created_at"
	if sortBy != "" {
		switch sortBy {
		case "name", "next_run", "last_run", "created_at", "enabled":
			sortColumn = sortBy
		}
	}
	sortOrder := "DESC"
	if strings.ToUpper(order) == "ASC" {
		sortOrder = "ASC"
	}

	query.Count(&total)
	query.Order("pin_type DESC, " + sortColumn + " " + sortOrder).Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)
	ts.loadTagsAndEnvs(tasks)

	return tasks, total
}

func (ts *TaskService) GetTaskByID(id string) *models.Task {
	var task models.Task
	res := database.DB.Where("id = ?", id).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}
	tasks := []models.Task{task}
	ts.loadTagsAndEnvs(tasks)
	
	return &tasks[0]
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
	task.Schedule = p.Schedule
	task.Timeout = p.Timeout
	task.WorkDir = p.WorkDir
	task.CleanConfig = p.CleanConfig
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
		"CleanConfig", "Enabled", "AgentID", "Languages",
		"RetryCount", "RetryInterval", "RandomRange", "Type",
		"TriggerType", "Config", "SourceID", "PinType",
		"PreCommand", "PostCommand",
	).Updates(&task)

	relation.DataRelation.SaveTags(task.ID, constant.RelationTypeTaskTag, p.Tags)
	task.Tags = p.Tags
	relation.DataRelation.SaveRelations(task.ID, constant.RelationTypeTaskEnv, p.Envs)
	task.Envs = models.BigText(p.Envs)

	return &task
}

func (ts *TaskService) DeleteTask(id string) bool {
	// 同时删除关联的通知推送设置
	database.DB.Where("type = ? AND data_id = ?", constant.BindingTypeTask, id).Delete(&models.NotifyBinding{})
	relation.DataRelation.CleanRelations(id, constant.RelationTypeTaskTag)
	relation.DataRelation.CleanRelations(id, constant.RelationTypeTaskEnv)

	result := database.DB.Where("id = ?", id).Delete(&models.Task{})
	return result.RowsAffected > 0
}

func (ts *TaskService) BatchDeleteTasks(ids []string) int64 {
	// 同时删除关联的通知推送设置
	database.DB.Where("type = ? AND data_id IN ?", constant.BindingTypeTask, ids).Delete(&models.NotifyBinding{})
	database.DB.Where("type = ? AND data_id IN ?", constant.RelationTypeTaskTag, ids).Delete(&models.DataRelation{})
	database.DB.Where("type = ? AND data_id IN ?", constant.RelationTypeTaskEnv, ids).Delete(&models.DataRelation{})

	result := database.DB.Where("id IN ?", ids).Delete(&models.Task{})
	return result.RowsAffected
}

// BatchUpdateTasks 批量更新任务配置及环境版本
func (ts *TaskService) BatchUpdateTasks(req vo.TaskBatchUpdateReq) (int64, []string, error) {
	var tasks []models.Task

	if len(req.IDs) > 0 {
		database.DB.Where("id IN ?", req.IDs).Find(&tasks)
	} else if req.Filter != nil {
		query := database.DB.Model(&models.Task{})
		if req.Filter.Type != "" {
			query = query.Where("type = ?", req.Filter.Type)
		}

		var matchedTasks []models.Task
		query.Find(&matchedTasks)

		for _, t := range matchedTasks {
			// 过滤标签
			if req.Filter.Tag != "" {
				taskTags := relation.DataRelation.LoadTags([]string{t.ID}, constant.RelationTypeTaskTag)[t.ID]
				hasTag := false
				for _, tag := range taskTags {
					if strings.EqualFold(tag, req.Filter.Tag) {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}

			// 过滤语言版本
			if req.Filter.HasLanguage != "" {
				hasLang := false
				for _, l := range t.Languages {
					nameMatch := strings.EqualFold(l["name"], req.Filter.HasLanguage) || strings.EqualFold(l["language"], req.Filter.HasLanguage)
					if nameMatch {
						if req.Filter.LanguageVersion == "" || l["version"] == req.Filter.LanguageVersion {
							hasLang = true
							break
						}
					}
				}
				if !hasLang {
					continue
				}
			}

			tasks = append(tasks, t)
		}
	}

	if len(tasks) == 0 {
		return 0, nil, nil
	}

	var updatedIDs []string
	fields := req.UpdateFields

	for _, task := range tasks {
		selectCols := []string{"UpdatedAt"}

		// 1. 更新环境语言
		if len(fields.Languages) > 0 {
			if fields.LanguageUpdateMode == "overwrite" {
				task.Languages = fields.Languages
				selectCols = append(selectCols, "Languages")
			} else {
				// 精准替换模式 (replace)
				newLangs := models.TaskLanguages{}
				// 将要更新的语言存入 map
				updateMap := make(map[string]string)
				for _, l := range fields.Languages {
					name := l["name"]
					if name == "" {
						name = l["language"]
					}
					if name != "" {
						updateMap[strings.ToLower(name)] = l["version"]
					}
				}

				processed := make(map[string]bool)
				for _, existing := range task.Languages {
					name := existing["name"]
					if name == "" {
						name = existing["language"]
					}
					lowerName := strings.ToLower(name)
					if newVer, exists := updateMap[lowerName]; exists {
						existing["version"] = newVer
						processed[lowerName] = true
					}
					newLangs = append(newLangs, existing)
				}

				// 追加原任务没有的新语言
				for _, l := range fields.Languages {
					name := l["name"]
					if name == "" {
						name = l["language"]
					}
					lowerName := strings.ToLower(name)
					if !processed[lowerName] {
						newLangs = append(newLangs, l)
					}
				}

				task.Languages = newLangs
				selectCols = append(selectCols, "Languages")
			}
		}

		// 2. 更新超时时间
		if fields.Timeout != nil {
			task.Timeout = *fields.Timeout
			selectCols = append(selectCols, "Timeout")
		}

		// 3. 更新前置命令
		if fields.PreCommand != nil {
			task.PreCommand = models.BigText(*fields.PreCommand)
			selectCols = append(selectCols, "PreCommand")
		}

		// 4. 更新后置命令
		if fields.PostCommand != nil {
			task.PostCommand = models.BigText(*fields.PostCommand)
			selectCols = append(selectCols, "PostCommand")
		}

		// 5. 更新 AgentID
		if fields.AgentID != nil {
			task.AgentID = fields.AgentID
			selectCols = append(selectCols, "AgentID")
		}

		// 6. 更新标签
		if fields.Tags != nil {
			relation.DataRelation.SaveTags(task.ID, constant.RelationTypeTaskTag, *fields.Tags)
			task.Tags = *fields.Tags
			selectCols = append(selectCols, "Tags")
		}

		// 7. 更新重试与随机延迟配置
		if fields.RetryCount != nil {
			task.RetryCount = *fields.RetryCount
			selectCols = append(selectCols, "RetryCount")
		}
		if fields.RetryInterval != nil {
			task.RetryInterval = *fields.RetryInterval
			selectCols = append(selectCols, "RetryInterval")
		}
		if fields.RandomRange != nil {
			task.RandomRange = *fields.RandomRange
			selectCols = append(selectCols, "RandomRange")
		}
		if fields.CleanConfig != nil {
			task.CleanConfig = *fields.CleanConfig
			selectCols = append(selectCols, "CleanConfig")
		}

		// 8. 更新环境变量关联
		if fields.Envs != nil {
			relation.DataRelation.SaveRelations(task.ID, constant.RelationTypeTaskEnv, *fields.Envs)
			task.Envs = models.BigText(*fields.Envs)
		}

		database.DB.Model(&task).Select(selectCols).Updates(&task)
		updatedIDs = append(updatedIDs, task.ID)
	}

	return int64(len(updatedIDs)), updatedIDs, nil
}

// GetAllTags 获取所有任务标签
func (ts *TaskService) GetAllTags() ([]string, error) {
	return relation.DataRelation.GetAllTags(constant.RelationTypeTaskTag)
}

func (ts *TaskService) loadTagsAndEnvs(tasks []models.Task) {
	if len(tasks) == 0 {
		return
	}
	taskIDs := make([]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}
	
	tagsMap := relation.DataRelation.LoadTags(taskIDs, constant.RelationTypeTaskTag)
	envsMap := relation.DataRelation.LoadRelations(taskIDs, constant.RelationTypeTaskEnv)
	
	for i, t := range tasks {
		if tags, ok := tagsMap[t.ID]; ok {
			tasks[i].Tags = strings.Join(tags, ",")
		} else {
			tasks[i].Tags = ""
		}
		
		if envs, ok := envsMap[t.ID]; ok {
			tasks[i].Envs = models.BigText(strings.Join(envs, ","))
		} else {
			tasks[i].Envs = models.BigText("")
		}
	}
}
