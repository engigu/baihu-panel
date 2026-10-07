package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/systime"
	"github.com/engigu/baihu-panel/internal/utils"
)

// SendStatsService 接口定义（避免循环依赖）
type SendStatsService interface {
	IncrementStats(taskID string, status string) error
}

// TaskLogService 任务日志服务
type TaskLogService struct {
	sendStatsService SendStatsService
}

// NewTaskLogService 创建任务日志服务
func NewTaskLogService(sendStatsService SendStatsService) *TaskLogService {
	return &TaskLogService{
		sendStatsService: sendStatsService,
	}
}

// CleanConfig 清理配置
type CleanConfig struct {
	Type string `json:"type"` // day 或 count
	Keep int    `json:"keep"` // 保留天数或条数
}

// CreateEmptyLog 创建一个空的日志记录（任务开始时调用）
func (s *TaskLogService) CreateEmptyLog(taskID string, command string, taskName ...string) (*models.TaskLog, error) {
	name := ""
	if len(taskName) > 0 && taskName[0] != "" {
		name = taskName[0]
	} else if taskID != "" {
		database.DB.Model(&models.Task{}).Where("id = ?", taskID).Pluck("name", &name)
	}

	startTime := models.Now()
	taskLog := &models.TaskLog{
		ID:        utils.GenerateID(),
		TaskID:    taskID,
		TaskName:  name,
		Command:   models.BigText(command),
		Status:    "running",
		StartTime: &startTime,
		CreatedAt: models.Now(),
	}
	if err := database.DB.Create(taskLog).Error; err != nil {
		return nil, err
	}

	// 任务开始时即更新任务的 last_run 为启动时间
	database.DB.Model(&models.Task{}).Where("id = ?", taskID).Update("last_run", startTime)

	return taskLog, nil
}

// SaveTaskLog 保存或更新任务日志
func (s *TaskLogService) SaveTaskLog(taskLog *models.TaskLog) error {
	var err error
	if taskLog.TaskName == "" && taskLog.TaskID != "" {
		database.DB.Model(&models.Task{}).Where("id = ?", taskLog.TaskID).Pluck("name", &taskLog.TaskName)
	}

	if taskLog.ID != "" {
		// 先检查记录是否存在，如果不存在则创建，存在则更新
		var count int64
		database.DB.Model(&models.TaskLog{}).Where("id = ?", taskLog.ID).Count(&count)
		if count > 0 {
			updateFields := []string{"Status", "Duration", "ExitCode", "StartTime", "EndTime", "Output", "Error", "AgentID"}
			if taskLog.TaskName != "" {
				updateFields = append(updateFields, "TaskName")
			}
			err = database.DB.Model(taskLog).Where("id = ?", taskLog.ID).Select(updateFields).Updates(taskLog).Error
		} else {
			err = database.DB.Create(taskLog).Error
		}
	} else {
		taskLog.ID = utils.GenerateID()
		if taskLog.CreatedAt.Time().IsZero() {
			taskLog.CreatedAt = models.Now()
		}
		err = database.DB.Create(taskLog).Error
	}

	if err != nil {
		return err
	}

	// 更新任务的 last_run
	// 更新任务的 last_run，优先使用日志记录的启动时间
	lastRun := models.Now()
	if taskLog.StartTime != nil {
		lastRun = *taskLog.StartTime
	}
	database.DB.Model(&models.Task{}).Where("id = ?", taskLog.TaskID).Update("last_run", lastRun)

	return nil
}

// UpdateTaskDuration 更新任务耗时（心跳）
func (s *TaskLogService) UpdateTaskDuration(logID string, duration int64) error {
	return database.DB.Model(&models.TaskLog{}).Where("id = ?", logID).Update("duration", duration).Error
}

// UpdateLogCommand 更新日志中的命令内容（用于动态生成的命令脱敏）
func (s *TaskLogService) UpdateLogCommand(logID string, command string) error {
	return database.DB.Model(&models.TaskLog{}).Where("id = ?", logID).Update("command", models.BigText(command)).Error
}

// UpdateTaskStats 更新任务统计
func (s *TaskLogService) UpdateTaskStats(taskID string, status string) {
	if s.sendStatsService == nil {
		logger.Error("[TaskLog] SendStatsService 未初始化")
		return
	}
	err := s.sendStatsService.IncrementStats(taskID, status)
	if err != nil {
		logger.Errorf("UpdateTaskStats err: %v", err)
		return
	}
}

// CleanTaskLogs 清理任务日志
func (s *TaskLogService) CleanTaskLogs(taskID string) {
	var task models.Task
	res := database.DB.Where("id = ?", taskID).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		return
	}

	if task.CleanConfig == "" {
		return
	}

	var config CleanConfig
	if err := json.Unmarshal([]byte(task.CleanConfig), &config); err != nil {
		logger.Errorf("[TaskLog] 解析清理配置失败: %v", err)
		return
	}

	if config.Keep <= 0 {
		return
	}

	var deleted int64
	switch config.Type {
	case "day":
		cutoff := systime.InCST(time.Now()).AddDate(0, 0, -config.Keep)
		result := database.DB.Where("task_id = ? AND created_at < ?", taskID, cutoff).Delete(&models.TaskLog{})
		deleted = result.RowsAffected
	case "count":
		var boundaryLog models.TaskLog
		res := database.DB.Where("task_id = ?", taskID).Order("id DESC").Offset(config.Keep - 1).Limit(1).Find(&boundaryLog)
		if res.Error == nil && res.RowsAffected > 0 {
			result := database.DB.Where("task_id = ? AND id < ?", taskID, boundaryLog.ID).Delete(&models.TaskLog{})
			deleted = result.RowsAffected
		}
	}

	if deleted > 0 {
		logger.Infof("[TaskLog] 清理旧日志: #%s 共 %d 条", taskID, deleted)
	}
}

// ClearLogs 按任务ID或天数清空历史日志 (days <= 0 表示不限时间全量清空)
func (s *TaskLogService) ClearLogs(taskID string, days int) (int64, error) {
	query := database.DB.Model(&models.TaskLog{})
	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}
	if days > 0 {
		cutoff := systime.InCST(time.Now()).AddDate(0, 0, -days)
		query = query.Where("created_at < ?", cutoff)
	}
	if taskID == "" && days <= 0 {
		query = query.Where("1 = 1")
	}
	res := query.Delete(&models.TaskLog{})
	return res.RowsAffected, res.Error
}

// ProcessTaskCompletion 处理任务完成后的所有操作（保存日志、更新统计、清理旧日志）
func (s *TaskLogService) ProcessTaskCompletion(taskLog *models.TaskLog) error {
	// 1. 保存/更新日志
	if err := s.SaveTaskLog(taskLog); err != nil {
		return err
	}

	// 2. 更新统计
	s.UpdateTaskStats(taskLog.TaskID, taskLog.Status)

	// 3. 异步清理旧日志
	go s.CleanTaskLogs(taskLog.TaskID)

	return nil
}

// CreateTaskLogFromAgentResult 从 Agent 结果创建任务日志
func (s *TaskLogService) CreateTaskLogFromAgentResult(result *models.AgentTaskResult) (*models.TaskLog, error) {
	// 裁剪并压缩输出
	trimmedOutput := utils.TrimLog(result.Output, constant.MaxLogSize)
	compressed, err := utils.CompressToBase64(trimmedOutput)
	if err != nil {
		logger.Errorf("[TaskLog] 压缩日志失败: %v", err)
		compressed = ""
	}

	logID := result.LogID
	if logID == "" {
		logID = utils.GenerateID()
	}

	var name string
	if result.TaskID != "" {
		database.DB.Model(&models.Task{}).Where("id = ?", result.TaskID).Pluck("name", &name)
	}

	taskLog := &models.TaskLog{
		ID:        logID,
		TaskID:    result.TaskID,
		TaskName:  name,
		AgentID:   &result.AgentID,
		Command:   models.BigText(result.Command),
		Output:    models.BigText(compressed),
		Error:     models.BigText(result.Error),
		Status:    result.Status,
		Duration:  result.Duration,
		ExitCode:  result.ExitCode,
		CreatedAt: models.Now(),
	}

	// 处理开始和结束时间
	if result.StartTime > 0 {
		startTime := models.LocalTime(time.Unix(result.StartTime, 0))
		taskLog.StartTime = &startTime
	}
	if result.EndTime > 0 {
		endTime := models.LocalTime(time.Unix(result.EndTime, 0))
		taskLog.EndTime = &endTime
	}

	return taskLog, nil
}

// CreateTaskLogFromLocalExecution 从本地执行结果创建任务日志
func (s *TaskLogService) CreateTaskLogFromLocalExecution(taskID string, command, output, systemErr, status string, duration int64, exitCode int, start, end time.Time, isCompressed bool) (*models.TaskLog, error) {
	var compressed string
	var err error

	if isCompressed {
		compressed = output
	} else {
		// 裁剪并压缩输出
		trimmedOutput := utils.TrimLog(output, constant.MaxLogSize)
		compressed, err = utils.CompressToBase64(trimmedOutput)
		if err != nil {
			logger.Errorf("[TaskLog] 压缩日志失败: %v", err)
			compressed = ""
		}
	}

	startTime := models.LocalTime(start)
	endTime := models.LocalTime(end)

	var name string
	if taskID != "" {
		database.DB.Model(&models.Task{}).Where("id = ?", taskID).Pluck("name", &name)
	}

	taskLog := &models.TaskLog{
		ID:        utils.GenerateID(),
		TaskID:    taskID,
		TaskName:  name,
		Command:   models.BigText(command),
		Output:    models.BigText(compressed),
		Error:     models.BigText(systemErr),
		Status:    status,
		Duration:  duration,
		ExitCode:  exitCode,
		StartTime: &startTime,
		EndTime:   &endTime,
		CreatedAt: models.Now(),
	}

	return taskLog, nil
}

const (
	DeletedTaskPrefix      = "[已删除] "
	DeletedTaskPlaceholder = "[已删除任务]"
)

// ResolveTaskLogInfo 解析任务日志的显示名称、类型与已删除状态
func ResolveTaskLogInfo(task *models.Task, log *models.TaskLog) (displayName string, taskType string, taskDeleted bool) {
	taskType = "task"
	if task != nil && task.ID != "" {
		if task.Type != "" {
			taskType = task.Type
		}
		if task.Name != "" {
			return task.Name, taskType, false
		}
		if log != nil && log.TaskName != "" {
			return log.TaskName, taskType, false
		}
		return "", taskType, false
	}

	// 任务记录不存在，判定为已删除
	taskDeleted = true
	if log != nil && log.TaskName != "" {
		displayName = DeletedTaskPrefix + log.TaskName
	} else {
		displayName = DeletedTaskPlaceholder
	}
	return displayName, taskType, taskDeleted
}

// GetLogsWithPagination 分页查询任务执行历史记录列表
func (s *TaskLogService) GetLogsWithPagination(page, pageSize int, taskID, taskName, status, date string) ([]vo.TaskLogVO, int64) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := database.DB.Model(&models.TaskLog{})
	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if date == "today" {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)
		query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
	} else if date != "" {
		if t, err := time.ParseInLocation("2006-01-02", date, time.Local); err == nil {
			startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
		}
	}

	// 按任务名称过滤
	if taskName != "" {
		trimmedName := strings.TrimSpace(taskName)
		if trimmedName == "[已删除]" || trimmedName == "已删除" || trimmedName == ":deleted" {
			var activeIDs []string
			database.DB.Model(&models.Task{}).Pluck("id", &activeIDs)
			if len(activeIDs) > 0 {
				query = query.Where("task_id NOT IN ?", activeIDs)
			}
		} else if strings.HasPrefix(trimmedName, "[已删除]") {
			keyword := strings.TrimSpace(strings.TrimPrefix(trimmedName, "[已删除]"))
			var activeIDs []string
			database.DB.Model(&models.Task{}).Pluck("id", &activeIDs)
			if len(activeIDs) > 0 {
				query = query.Where("task_id NOT IN ?", activeIDs)
			}
			if keyword != "" {
				query = query.Where("task_name LIKE ?", "%"+keyword+"%")
			}
		} else {
			var taskIDs []string
			database.DB.Model(&models.Task{}).Where("name LIKE ?", "%"+trimmedName+"%").Pluck("id", &taskIDs)
			if len(taskIDs) > 0 {
				query = query.Where("(task_id IN ? OR task_name LIKE ?)", taskIDs, "%"+trimmedName+"%")
			} else {
				query = query.Where("task_name LIKE ?", "%"+trimmedName+"%")
			}
		}
	}

	var total int64
	query.Count(&total)

	var logs []models.TaskLog
	offset := (page - 1) * pageSize
	query.Omit("output", "error").Order("id DESC").Offset(offset).Limit(pageSize).Find(&logs)

	taskIDList := make([]string, 0, len(logs))
	for _, log := range logs {
		taskIDList = append(taskIDList, log.TaskID)
	}

	var tasksList []models.Task
	if len(taskIDList) > 0 {
		database.DB.Select("id", "name", "type").Where("id IN ?", taskIDList).Find(&tasksList)
	}
	taskMap := make(map[string]models.Task)
	for _, t := range tasksList {
		taskMap[t.ID] = t
	}

	result := make([]vo.TaskLogVO, len(logs))
	for i, log := range logs {
		var taskPtr *models.Task
		if t, exists := taskMap[log.TaskID]; exists {
			taskPtr = &t
		}
		displayName, taskType, taskDeleted := ResolveTaskLogInfo(taskPtr, &log)

		result[i] = vo.TaskLogVO{
			ID:          log.ID,
			TaskID:      log.TaskID,
			TaskName:    displayName,
			TaskDeleted: taskDeleted,
			TaskType:    taskType,
			AgentID:     log.AgentID,
			Command:     string(log.Command),
			Status:      log.Status,
			Duration:    log.Duration,
			StartTime:   log.StartTime,
			EndTime:     log.EndTime,
			CreatedAt:   log.CreatedAt,
		}
	}

	return result, total
}

// GetLogDetailByID 根据 ID 获取日志详情
func (s *TaskLogService) GetLogDetailByID(id string) (*vo.TaskLogVO, error) {
	var log models.TaskLog
	res := database.DB.Where("id = ?", id).Limit(1).Find(&log)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, fmt.Errorf("日志不存在")
	}

	logVO := vo.ToTaskLogVO(&log)
	if logVO != nil {
		var task models.Task
		var taskPtr *models.Task
		if taskRes := database.DB.Where("id = ?", log.TaskID).Limit(1).Find(&task); taskRes.Error == nil && taskRes.RowsAffected > 0 {
			taskPtr = &task
		}
		displayName, taskType, taskDeleted := ResolveTaskLogInfo(taskPtr, &log)
		logVO.TaskName = displayName
		logVO.TaskType = taskType
		logVO.TaskDeleted = taskDeleted
	}

	return logVO, nil
}

