package tasks

import (
	"encoding/json"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
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
func (s *TaskLogService) CreateEmptyLog(taskID string, command string) (*models.TaskLog, error) {
	startTime := models.Now()
	taskLog := &models.TaskLog{
		ID:        utils.GenerateID(),
		TaskID:    taskID,
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
	if taskLog.ID != "" {
		// 先检查记录是否存在，如果不存在则创建，存在则更新
		var count int64
		database.DB.Model(&models.TaskLog{}).Where("id = ?", taskLog.ID).Count(&count)
		if count > 0 {
			err = database.DB.Model(taskLog).Where("id = ?", taskLog.ID).Updates(taskLog).Error
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

	taskLog := &models.TaskLog{
		ID:        logID,
		TaskID:    result.TaskID,
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

	taskLog := &models.TaskLog{
		ID:        utils.GenerateID(),
		TaskID:    taskID,
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
