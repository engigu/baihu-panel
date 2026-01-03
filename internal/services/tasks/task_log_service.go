package tasks

import (
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/utils"
	"encoding/json"
	"time"
)

// SendStatsService 接口定义（避免循环依赖）
type SendStatsService interface {
	IncrementStats(taskID uint, status string) error
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

// SaveTaskLog 保存任务日志（通用方法）
func (s *TaskLogService) SaveTaskLog(taskLog *models.TaskLog) error {
	if err := database.DB.Create(taskLog).Error; err != nil {
		return err
	}

	// 更新任务的 last_run
	database.DB.Model(&models.Task{}).Where("id = ?", taskLog.TaskID).Update("last_run", time.Now())

	return nil
}

// UpdateTaskStats 更新任务统计
func (s *TaskLogService) UpdateTaskStats(taskID uint, status string) {
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
func (s *TaskLogService) CleanTaskLogs(taskID uint) {
	var task models.Task
	if err := database.DB.First(&task, taskID).Error; err != nil {
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
		cutoff := time.Now().AddDate(0, 0, -config.Keep)
		result := database.DB.Where("task_id = ? AND created_at < ?", taskID, cutoff).Delete(&models.TaskLog{})
		deleted = result.RowsAffected
	case "count":
		var boundaryLog models.TaskLog
		err := database.DB.Where("task_id = ?", taskID).Order("id DESC").Offset(config.Keep - 1).Limit(1).First(&boundaryLog).Error
		if err == nil {
			result := database.DB.Where("task_id = ? AND id < ?", taskID, boundaryLog.ID).Delete(&models.TaskLog{})
			deleted = result.RowsAffected
		}
	}

	if deleted > 0 {
		logger.Infof("[TaskLog] 清理任务 #%d 的 %d 条日志", taskID, deleted)
	}
}

// ProcessTaskCompletion 处理任务完成后的所有操作（保存日志、更新统计、清理旧日志）
func (s *TaskLogService) ProcessTaskCompletion(taskLog *models.TaskLog) error {
	// 1. 保存日志
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
	// 压缩输出
	compressed, err := utils.CompressToBase64(result.Output)
	if err != nil {
		logger.Errorf("[TaskLog] 压缩日志失败: %v", err)
		compressed = ""
	}

	taskLog := &models.TaskLog{
		TaskID:   result.TaskID,
		AgentID:  &result.AgentID,
		Command:  result.Command,
		Output:   compressed,
		Status:   result.Status,
		Duration: result.Duration,
		ExitCode: result.ExitCode,
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
func (s *TaskLogService) CreateTaskLogFromLocalExecution(taskID uint, command, output, status string, duration int64, exitCode int, start, end time.Time) (*models.TaskLog, error) {
	// 压缩输出
	compressed, err := utils.CompressToBase64(output)
	if err != nil {
		logger.Errorf("[TaskLog] 压缩日志失败: %v", err)
		compressed = ""
	}

	startTime := models.LocalTime(start)
	endTime := models.LocalTime(end)

	taskLog := &models.TaskLog{
		TaskID:    taskID,
		Command:   command,
		Output:    compressed,
		Status:    status,
		Duration:  duration,
		ExitCode:  exitCode,
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	return taskLog, nil
}
