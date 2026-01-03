package tasks

import (
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// AgentWSManager 接口定义（避免循环依赖）
type AgentWSManager interface {
	SendToAgent(agentID uint, msgType string, data interface{}) error
}

// TaskExecutionService 统一的任务执行服务
type TaskExecutionService struct {
	taskLogService *TaskLogService
	agentWSManager AgentWSManager
}

// NewTaskExecutionService 创建任务执行服务
func NewTaskExecutionService(agentWSManager AgentWSManager, sendStatsService SendStatsService) *TaskExecutionService {
	return &TaskExecutionService{
		taskLogService: NewTaskLogService(sendStatsService),
		agentWSManager: agentWSManager,
	}
}

// TaskExecutionRequest 任务执行请求
type TaskExecutionRequest struct {
	TaskID  uint
	Task    *models.Task
	AgentID *uint // nil 表示本地执行
}

// TaskExecutionResult 任务执行结果
type TaskExecutionResult struct {
	TaskID   uint
	AgentID  *uint
	Command  string
	Output   string
	Status   string // success, failed
	Duration int64  // milliseconds
	ExitCode int
	Start    time.Time
	End      time.Time
}

// ExecuteTask 执行任务（统一入口）
func (s *TaskExecutionService) ExecuteTask(req *TaskExecutionRequest) error {
	if req.Task.AgentID != nil && *req.Task.AgentID > 0 {
		// 远程执行：通过 Agent
		return s.executeRemote(req)
	}
	// 本地执行
	return s.executeLocal(req)
}

// executeLocal 本地执行任务
func (s *TaskExecutionService) executeLocal(req *TaskExecutionRequest) error {
	task := req.Task
	logger.Infof("[TaskExecution] 本地执行任务 #%d: %s", task.ID, task.Name)

	start := time.Now()

	// 准备命令
	ctx, cancel := s.createContext(task.Timeout)
	defer cancel()

	cmd, err := s.prepareCommand(ctx, task)
	if err != nil {
		return s.handleExecutionError(task.ID, task.Command, start, err)
	}

	// 执行命令
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	execErr := cmd.Run()
	end := time.Now()

	// 构建结果
	result := &TaskExecutionResult{
		TaskID:   task.ID,
		AgentID:  nil,
		Command:  task.Command,
		Output:   stdout.String(),
		Start:    start,
		End:      end,
		Duration: end.Sub(start).Milliseconds(),
	}

	if execErr != nil {
		result.Status = "failed"
		result.Output += "\n[ERROR]\n" + stderr.String() + "\n" + execErr.Error()
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.Status = "success"
		result.ExitCode = 0
	}

	// 处理执行结果
	return s.processExecutionResult(result)
}

// executeRemote 远程执行任务（通过 Agent）
func (s *TaskExecutionService) executeRemote(req *TaskExecutionRequest) error {
	task := req.Task
	agentID := *task.AgentID

	logger.Infof("[TaskExecution] 远程执行任务 #%d: %s (Agent #%d)", task.ID, task.Name, agentID)

	// 检查 Agent 是否在线
	var agent models.Agent
	if err := database.DB.First(&agent, agentID).Error; err != nil {
		return fmt.Errorf("Agent #%d 不存在", agentID)
	}

	if !agent.Enabled {
		return fmt.Errorf("Agent #%d 已禁用", agentID)
	}

	// 通过 WebSocket 发送立即执行命令给 Agent
	if s.agentWSManager == nil {
		return fmt.Errorf("AgentWSManager 未初始化")
	}
	err := s.agentWSManager.SendToAgent(agentID, "execute", map[string]interface{}{
		"task_id": task.ID,
	})
	if err != nil {
		return fmt.Errorf("发送执行命令失败: %v", err)
	}

	logger.Infof("[TaskExecution] 已发送立即执行命令给 Agent #%d，任务 #%d", agentID, task.ID)
	return nil
}

// prepareCommand 准备执行命令
func (s *TaskExecutionService) prepareCommand(ctx context.Context, task *models.Task) (*exec.Cmd, error) {
	command := task.Command

	// 处理工作目录
	if task.WorkDir != "" {
		// 验证工作目录
		if _, err := os.Stat(task.WorkDir); err != nil {
			return nil, fmt.Errorf("工作目录不存在或无法访问: %s", task.WorkDir)
		}
	}

	// 处理环境变量
	envVars := s.loadEnvVars(task.Envs)

	// 根据操作系统创建命令
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		// 如果有工作目录，在命令前加 cd
		if task.WorkDir != "" {
			command = fmt.Sprintf("cd %s && %s", task.WorkDir, command)
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	// 设置环境变量
	if len(envVars) > 0 {
		cmd.Env = append(os.Environ(), envVars...)
	}

	return cmd, nil
}

// createContext 创建带超时的上下文
func (s *TaskExecutionService) createContext(timeout int) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 // 默认 30 分钟
	}
	return context.WithTimeout(context.Background(), time.Duration(timeout)*time.Minute)
}

// loadEnvVars 加载环境变量
func (s *TaskExecutionService) loadEnvVars(envIDs string) []string {
	if envIDs == "" {
		return nil
	}

	var envVars []models.EnvironmentVariable
	ids := strings.Split(envIDs, ",")
	database.DB.Where("id IN ?", ids).Find(&envVars)

	result := make([]string, 0, len(envVars))
	for _, env := range envVars {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return result
}

// handleExecutionError 处理执行错误
func (s *TaskExecutionService) handleExecutionError(taskID uint, command string, start time.Time, err error) error {
	end := time.Now()
	result := &TaskExecutionResult{
		TaskID:   taskID,
		Command:  command,
		Output:   fmt.Sprintf("[ERROR] 任务执行失败: %v", err),
		Status:   "failed",
		Duration: end.Sub(start).Milliseconds(),
		ExitCode: 1,
		Start:    start,
		End:      end,
	}
	return s.processExecutionResult(result)
}

// processExecutionResult 处理执行结果（统一的结果处理）
func (s *TaskExecutionService) processExecutionResult(result *TaskExecutionResult) error {
	// 创建任务日志
	taskLog, err := s.taskLogService.CreateTaskLogFromLocalExecution(
		result.TaskID,
		result.Command,
		result.Output,
		result.Status,
		result.Duration,
		result.ExitCode,
		result.Start,
		result.End,
	)
	if err != nil {
		logger.Errorf("[TaskExecution] 创建任务日志失败: %v", err)
		return err
	}

	// 如果是 Agent 执行的，设置 AgentID
	if result.AgentID != nil {
		taskLog.AgentID = result.AgentID
	}

	// 处理任务完成（保存日志、更新统计、清理旧日志）
	if err := s.taskLogService.ProcessTaskCompletion(taskLog); err != nil {
		logger.Errorf("[TaskExecution] 处理任务完成失败: %v", err)
		return err
	}

	logger.Infof("[TaskExecution] 任务 #%d 执行完成 (%s)", result.TaskID, result.Status)
	return nil
}

// ProcessAgentResult 处理 Agent 上报的结果（统一入口）
func (s *TaskExecutionService) ProcessAgentResult(agentResult *models.AgentTaskResult) error {
	logger.Infof("[TaskExecution] 处理 Agent #%d 上报的任务 #%d 结果", agentResult.AgentID, agentResult.TaskID)

	// 转换为统一的执行结果
	result := &TaskExecutionResult{
		TaskID:   agentResult.TaskID,
		AgentID:  &agentResult.AgentID,
		Command:  agentResult.Command,
		Output:   agentResult.Output,
		Status:   agentResult.Status,
		Duration: agentResult.Duration,
		ExitCode: agentResult.ExitCode,
		Start:    time.Unix(agentResult.StartTime, 0),
		End:      time.Unix(agentResult.EndTime, 0),
	}

	// 使用统一的结果处理流程
	return s.processExecutionResult(result)
}

// GetScriptPath 获取脚本路径
func (s *TaskExecutionService) GetScriptPath(scriptName string) string {
	return filepath.Join("data", "scripts", scriptName)
}
