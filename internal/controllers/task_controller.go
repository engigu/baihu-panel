package controllers

import (
	"path/filepath"

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
		AgentID       *string             `json:"agent_id"`
		TriggerType   string              `json:"trigger_type"`
		RetryCount    int                 `json:"retry_count"`
		RetryInterval int                 `json:"retry_interval"`
		RandomRange   int                 `json:"random_range"`
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
	if req.AgentID == nil || *req.AgentID == "" {
		workDir = resolveWorkDir(req.WorkDir)
	}

	task := tc.taskService.CreateTask(req.Name, req.Command, req.Schedule, req.Timeout, workDir, req.CleanConfig, req.Envs, req.Type, req.Config, req.AgentID, req.Languages, req.TriggerType, req.Tags, req.RetryCount, req.RetryInterval, req.RandomRange)

	// 如果是 Agent 任务，通知 Agent；否则添加到本地 cron
	if task.AgentID != nil && *task.AgentID != "" {
		tc.agentWSManager.BroadcastTasks(*task.AgentID)
	} else {
		tc.executorService.AddCronTask(task)
	}

	utils.Success(c, vo.ToTaskVO(task))
}

// GetTasks 获取任务列表
// @Summary 获取任务列表
// @Description 分页获取任务列表，支持按名称、Agent ID、标签、类型筛选
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name query string false "任务名称"
// @Param agent_id query string false "Agent ID"
// @Param tags query string false "标签"
// @Param type query string false "任务类型"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} utils.Response{data=utils.PaginationData{data=[]vo.TaskVO}}
// @Router /tasks [get]
func (tc *TaskController) GetTasks(c *gin.Context) {
	p := utils.ParsePagination(c)
	name := c.DefaultQuery("name", "")
	agentIDStr := c.DefaultQuery("agent_id", "")

	tags := c.DefaultQuery("tags", "")
	taskType := c.DefaultQuery("type", "")

	var agentID *string
	if agentIDStr != "" {
		agentID = &agentIDStr
	}

	tasks, total := tc.taskService.GetTasksWithPagination(p.Page, p.PageSize, name, agentID, tags, taskType)
	utils.PaginatedResponse(c, vo.ToTaskVOListFromModels(tasks), total, p)
}

// GetTask 获取任务详情
// @Summary 获取任务详情
// @Description 根据 ID 获取任务详情
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Success 200 {object} utils.Response{data=vo.TaskVO}
// @Failure 404 {object} utils.Response
// @Router /tasks/{id} [get]
func (tc *TaskController) GetTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
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

// UpdateTask 更新任务
// @Summary 更新任务
// @Description 根据 ID 更新任务信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Param body body object true "任务更新信息"
// @Success 200 {object} utils.Response{data=vo.TaskVO}
// @Failure 404 {object} utils.Response
// @Router /tasks/{id} [put]
func (tc *TaskController) UpdateTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	// 获取旧任务信息（用于判断 agent 变更）
	oldTask := tc.taskService.GetTaskByID(id)
	var oldAgentID *string
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
		AgentID       *string             `json:"agent_id"`
		TriggerType   string              `json:"trigger_type"`
		RetryCount    int                 `json:"retry_count"`
		RetryInterval int                 `json:"retry_interval"`
		RandomRange   int                 `json:"random_range"`
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
	if req.AgentID == nil || *req.AgentID == "" {
		workDir = resolveWorkDir(req.WorkDir)
	}

	task := tc.taskService.UpdateTask(id, req.Name, req.Command, req.Schedule, req.Timeout, workDir, req.CleanConfig, req.Envs, req.Enabled, req.Type, req.Config, req.AgentID, req.Languages, req.TriggerType, req.Tags, req.RetryCount, req.RetryInterval, req.RandomRange)
	if task == nil {
		utils.NotFound(c, "任务不存在")
		return
	}

	// 处理任务调度
	if task.AgentID != nil && *task.AgentID != "" {
		// Agent 任务：从本地 cron 移除，通知 Agent
		tc.executorService.RemoveCronTask(task.ID)
		tc.agentWSManager.BroadcastTasks(*task.AgentID)
		// 如果 agent 变更了，也通知旧 agent
		if oldAgentID != nil && *oldAgentID != "" && *oldAgentID != *task.AgentID {
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
		if oldAgentID != nil && *oldAgentID != "" {
			tc.agentWSManager.BroadcastTasks(*oldAgentID)
		}
	}

	utils.Success(c, vo.ToTaskVO(task))
}

// DeleteTask 删除任务
// @Summary 删除任务
// @Description 根据 ID 删除任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "任务ID"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /tasks/{id} [delete]
func (tc *TaskController) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	// 获取任务信息（用于通知 agent）
	task := tc.taskService.GetTaskByID(id)
	var agentID *string
	if task != nil {
		agentID = task.AgentID
	}

	tc.executorService.RemoveCronTask(id)

	success := tc.taskService.DeleteTask(id)
	if !success {
		utils.NotFound(c, "任务不存在")
		return
	}

	// 如果是 agent 任务，通知 agent
	if agentID != nil && *agentID != "" {
		tc.agentWSManager.BroadcastTasks(*agentID)
	}

	utils.SuccessMsg(c, "删除成功")
}

// StopTask 停止任务
// @Summary 停止任务
// @Description 根据运行日志 ID 停止正在执行的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param logID path string true "运行日志ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /tasks/stop/{logID} [post]
func (tc *TaskController) StopTask(c *gin.Context) {
	logID := c.Param("logID")
	if logID == "" {
		utils.BadRequest(c, "无效的日志ID")
		return
	}

	err := tc.executorService.StopTaskExecution(logID)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.SuccessMsg(c, "停止请求已发送")
}
