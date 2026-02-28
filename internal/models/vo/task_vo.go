package vo

import (
	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/models"
)

// TaskVO 任务视图对象
type TaskVO struct {
	ID          uint                `json:"id"`
	Name        string              `json:"name"`
	Command     string              `json:"command"`
	Tags        string              `json:"tags"`
	Type        string              `json:"type"`
	TriggerType string              `json:"trigger_type"`
	Config      string              `json:"config"`
	Schedule    string              `json:"schedule"`
	Timeout     int                 `json:"timeout"`
	WorkDir     string              `json:"work_dir"`
	CleanConfig string              `json:"clean_config"`
	Envs        string              `json:"envs"`
	Languages   []map[string]string `json:"languages"`
	AgentID     *uint               `json:"agent_id"`
	Enabled       bool                `json:"enabled"`
	RetryCount    int                 `json:"retry_count"`
	RetryInterval int                 `json:"retry_interval"`
	LastRun       *models.LocalTime   `json:"last_run"`
	NextRun     *models.LocalTime   `json:"next_run"`
	CreatedAt   models.LocalTime    `json:"created_at"`
	UpdatedAt   models.LocalTime    `json:"updated_at"`
}

// ToTaskVO 将 Task 模型转换为 TaskVO
func ToTaskVO(task *models.Task) *TaskVO {
	if task == nil {
		return nil
	}
	return &TaskVO{
		ID:          task.ID,
		Name:        task.Name,
		Command:     task.Command,
		Tags:        task.Tags,
		Type:        task.Type,
		TriggerType: task.TriggerType,
		Config:      task.Config,
		Schedule:    task.Schedule,
		Timeout:     task.Timeout,
		WorkDir:     task.WorkDir,
		CleanConfig: task.CleanConfig,
		Envs:        task.Envs,
		Languages:   task.Languages,
		AgentID:       task.AgentID,
		Enabled:       task.Enabled,
		RetryCount:    task.RetryCount,
		RetryInterval: task.RetryInterval,
		LastRun:       task.LastRun,
		NextRun:     task.NextRun,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

// ToTaskVOList 将 Task 模型列表转换为 TaskVO 列表
func ToTaskVOList(tasks []*models.Task) []*TaskVO {
	if tasks == nil {
		return nil
	}
	vos := make([]*TaskVO, len(tasks))
	for i, t := range tasks {
		vos[i] = ToTaskVO(t)
	}
	return vos
}

// ToTaskVOListFromModels 将 Task 模型列表转换为 TaskVO 列表
func ToTaskVOListFromModels(tasks []models.Task) []*TaskVO {
	vos := make([]*TaskVO, len(tasks))
	for i := range tasks {
		vos[i] = ToTaskVO(&tasks[i])
	}
	return vos
}

// TaskLogVO 任务历史视图对象
type TaskLogVO struct {
	ID        uint              `json:"id"`
	TaskID    uint              `json:"task_id"`
	TaskName  string            `json:"task_name"`
	TaskType  string            `json:"task_type"`
	AgentID   *uint             `json:"agent_id"`
	Command   string            `json:"command"`
	Error     string            `json:"error"`
	Status    string            `json:"status"`
	Duration  int64             `json:"duration"`
	ExitCode  int               `json:"exit_code"`
	StartTime     *models.LocalTime `json:"start_time"`
	EndTime       *models.LocalTime `json:"end_time"`
	CreatedAt     models.LocalTime  `json:"created_at"`
	WorkflowID    *string           `json:"workflow_id"`
	WorkflowRunID string            `json:"workflow_run_id"`
	Output        string            `json:"output,omitempty"`
}

// ToTaskLogVO 将 TaskLog 模型转换为 TaskLogVO
// Note: This function assumes the Task field within models.TaskLog is preloaded
// or that taskName and taskType are provided from an external source.
func ToTaskLogVO(log *models.TaskLog) *TaskLogVO {
	if log == nil {
		return nil
	}
	return &TaskLogVO{
		ID:        log.ID,
		TaskID:    log.TaskID,
		AgentID:   log.AgentID,
		Command:   log.Command,
		Error:     log.Error,
		Status:    log.Status,
		Duration:  log.Duration,
		ExitCode:  log.ExitCode,
		StartTime:     log.StartTime,
		EndTime:       log.EndTime,
		CreatedAt:     log.CreatedAt,
		WorkflowID:    log.WorkflowID,
		WorkflowRunID: log.WorkflowRunID,
		Output:        log.Output,
	}
}

// ToTaskLogVOList 将 TaskLog 模型列表转换为 TaskLogVO 列表
func ToTaskLogVOList(logs []*models.TaskLog) []*TaskLogVO {
	if logs == nil {
		return nil
	}
	vos := make([]*TaskLogVO, len(logs))
	for i, l := range logs {
		vos[i] = ToTaskLogVO(l)
	}
	return vos
}

// ToTaskLogVOListFromModels 将 TaskLog 模型列表转换为 TaskLogVO 列表
func ToTaskLogVOListFromModels(logs []models.TaskLog) []*TaskLogVO {
	vos := make([]*TaskLogVO, len(logs))
	for i := range logs {
		vos[i] = ToTaskLogVO(&logs[i])
	}
	return vos
}

// ExecutionResultVO 任务执行结果视图对象
type ExecutionResultVO struct {
	TaskID    string `json:"task_id"`
	LogID     uint   `json:"log_id,omitempty"`
	Success   bool   `json:"success"`
	Status    string `json:"status"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	Duration  int64  `json:"duration,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

// ToExecutionResultVO 将 ExecutionResult 转换为 ExecutionResultVO
func ToExecutionResultVO(res *executor.ExecutionResult) *ExecutionResultVO {
	if res == nil {
		return nil
	}
	vo := &ExecutionResultVO{
		TaskID:   res.TaskID,
		LogID:    res.LogID,
		Success:  res.Success,
		Status:   res.Status,
		Output:   res.Output,
		Error:    res.Error,
		Duration: res.Duration,
		ExitCode: res.ExitCode,
	}
	if !res.StartTime.IsZero() {
		vo.StartTime = res.StartTime.Format("2006-01-02 15:04:05")
	}
	if !res.EndTime.IsZero() {
		vo.EndTime = res.EndTime.Format("2006-01-02 15:04:05")
	}
	return vo
}

// ToExecutionResultVOList 将 ExecutionResult 列表转换为 ExecutionResultVO 列表
func ToExecutionResultVOList(results []executor.ExecutionResult) []*ExecutionResultVO {
	if results == nil {
		return nil
	}
	vos := make([]*ExecutionResultVO, len(results))
	for i := range results {
		vos[i] = ToExecutionResultVO(&results[i])
	}
	return vos
}
