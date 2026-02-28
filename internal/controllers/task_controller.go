package controllers

import (
	"path/filepath"
	"strconv"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type TaskController struct {
	taskService     *tasks.TaskService
	executorService *tasks.ExecutorService
	agentWSManager  *services.AgentWSManager
}

func NewTaskController(taskService *tasks.TaskService, executorService *tasks.ExecutorService) *TaskController {
	return &TaskController{
		taskService:     taskService,
		executorService: executorService,
		agentWSManager:  services.GetAgentWSManager(),
	}
}

// resolveWorkDir 将相对路径转换为绝对路径
func resolveWorkDir(workDir string) string {
	if workDir == "" {
		// 空则使用默认 scripts 目录
		absPath, err := filepath.Abs(constant.ScriptsWorkDir)
		if err != nil {
			return constant.ScriptsWorkDir
		}
		return absPath
	}
	// 如果已经是绝对路径，直接返回
	if filepath.IsAbs(workDir) {
		return workDir
	}
	// 相对路径，基于 scripts 目录
	fullPath := filepath.Join(constant.ScriptsWorkDir, workDir)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fullPath
	}
	return absPath
}

func (tc *TaskController) CreateTask(c *gin.Context) {
	var req struct {
		Name        string              `json:"name" binding:"required"`
		Command     string              `json:"command"`
		Tags        string              `json:"tags"`
		Type        string              `json:"type"`
		Config      string              `json:"config"`
		Schedule    string              `json:"schedule"`
		Timeout     int                 `json:"timeout"`
		WorkDir     string              `json:"work_dir"`
		CleanConfig string              `json:"clean_config"`
		Envs        string              `json:"envs"`
		Languages   []map[string]string `json:"languages"`
		AgentID       *uint               `json:"agent_id"`
		TriggerType   string              `json:"trigger_type"`
		RetryCount    int                 `json:"retry_count"`
		RetryInterval int                 `json:"retry_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 普通任务需要命令
	if req.Type != constant.TaskTypeRepo && req.Command == "" {
		utils.BadRequest(c, "命令不能为空")
		return
	}

	if req.Schedule != "" {
		if err := tc.executorService.ValidateCron(req.Schedule); err != nil {
			utils.BadRequest(c, "无效的cron表达式: "+err.Error())
			return
		}
	}

	// 转换为绝对路径（Agent 任务保持原样）
	workDir := req.WorkDir
	if req.AgentID == nil || *req.AgentID == 0 {
		workDir = resolveWorkDir(req.WorkDir)
	}

	task := tc.taskService.CreateTask(req.Name, req.Command, req.Schedule, req.Timeout, workDir, req.CleanConfig, req.Envs, req.Type, req.Config, req.AgentID, req.Languages, req.TriggerType, req.Tags, req.RetryCount, req.RetryInterval)

	// 如果是 Agent 任务，通知 Agent；否则添加到本地 cron
	if task.AgentID != nil && *task.AgentID > 0 {
		tc.agentWSManager.BroadcastTasks(*task.AgentID)
	} else {
		tc.executorService.AddCronTask(task)
	}

	utils.Success(c, vo.ToTaskVO(task))
}

func (tc *TaskController) GetTasks(c *gin.Context) {
	p := utils.ParsePagination(c)
	name := c.DefaultQuery("name", "")
	agentIDStr := c.DefaultQuery("agent_id", "")

	tags := c.DefaultQuery("tags", "")
	taskType := c.DefaultQuery("type", "")

	var agentID *uint
	if agentIDStr != "" {
		if id, err := strconv.ParseUint(agentIDStr, 10, 32); err == nil {
			uid := uint(id)
			agentID = &uid
		}
	}

	tasks, total := tc.taskService.GetTasksWithPagination(p.Page, p.PageSize, name, agentID, tags, taskType)
	utils.PaginatedResponse(c, vo.ToTaskVOListFromModels(tasks), total, p)
}

func (tc *TaskController) GetTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	task := tc.taskService.GetTaskByID(id)
	if task == nil {
		utils.NotFound(c, "任务不存在")
		return
	}

	utils.Success(c, vo.ToTaskVO(task))
}

func (tc *TaskController) UpdateTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	// 获取旧任务信息（用于判断 agent 变更）
	oldTask := tc.taskService.GetTaskByID(id)
	var oldAgentID *uint
	if oldTask != nil {
		oldAgentID = oldTask.AgentID
	}

	var req struct {
		Name        string              `json:"name"`
		Command     string              `json:"command"`
		Tags        string              `json:"tags"`
		Type        string              `json:"type"`
		Config      string              `json:"config"`
		Schedule    string              `json:"schedule"`
		Timeout     int                 `json:"timeout"`
		WorkDir     string              `json:"work_dir"`
		CleanConfig string              `json:"clean_config"`
		Envs        string              `json:"envs"`
		Enabled     bool                `json:"enabled"`
		Languages   []map[string]string `json:"languages"`
		AgentID       *uint               `json:"agent_id"`
		TriggerType   string              `json:"trigger_type"`
		RetryCount    int                 `json:"retry_count"`
		RetryInterval int                 `json:"retry_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if req.Schedule != "" {
		if err := tc.executorService.ValidateCron(req.Schedule); err != nil {
			utils.BadRequest(c, "无效的cron表达式: "+err.Error())
			return
		}
	}

	// 转换为绝对路径（Agent 任务保持原样）
	workDir := req.WorkDir
	if req.AgentID == nil || *req.AgentID == 0 {
		workDir = resolveWorkDir(req.WorkDir)
	}

	task := tc.taskService.UpdateTask(id, req.Name, req.Command, req.Schedule, req.Timeout, workDir, req.CleanConfig, req.Envs, req.Enabled, req.Type, req.Config, req.AgentID, req.Languages, req.TriggerType, req.Tags, req.RetryCount, req.RetryInterval)
	if task == nil {
		utils.NotFound(c, "任务不存在")
		return
	}

	// 处理任务调度
	if task.AgentID != nil && *task.AgentID > 0 {
		// Agent 任务：从本地 cron 移除，通知 Agent
		tc.executorService.RemoveCronTask(task.ID)
		tc.agentWSManager.BroadcastTasks(*task.AgentID)
		// 如果 agent 变更了，也通知旧 agent
		if oldAgentID != nil && *oldAgentID > 0 && *oldAgentID != *task.AgentID {
			tc.agentWSManager.BroadcastTasks(*oldAgentID)
		}
	} else {
		// 本地任务
		if task.Enabled {
			tc.executorService.AddCronTask(task)
		} else {
			tc.executorService.RemoveCronTask(task.ID)
		}
		// 如果之前是 agent 任务，通知旧 agent 移除
		if oldAgentID != nil && *oldAgentID > 0 {
			tc.agentWSManager.BroadcastTasks(*oldAgentID)
		}
	}

	utils.Success(c, vo.ToTaskVO(task))
}

func (tc *TaskController) DeleteTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	// 获取任务信息（用于通知 agent）
	task := tc.taskService.GetTaskByID(id)
	var agentID *uint
	if task != nil {
		agentID = task.AgentID
	}

	tc.executorService.RemoveCronTask(uint(id))

	success := tc.taskService.DeleteTask(id)
	if !success {
		utils.NotFound(c, "任务不存在")
		return
	}

	// 如果是 agent 任务，通知 agent
	if agentID != nil && *agentID > 0 {
		tc.agentWSManager.BroadcastTasks(*agentID)
	}

	utils.SuccessMsg(c, "删除成功")
}

func (tc *TaskController) StopTask(c *gin.Context) {
	logID, err := strconv.ParseUint(c.Param("logID"), 10, 32)
	if err != nil {
		utils.BadRequest(c, "无效的日志ID")
		return
	}

	err = tc.executorService.StopTaskExecution(uint(logID))
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessMsg(c, "停止请求已发送")
}
