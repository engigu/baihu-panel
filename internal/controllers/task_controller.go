package controllers

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
	"os"
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
	if strings.HasPrefix(workDir, constant.ScriptsDirPlaceholder) {
		return workDir
	}
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

// isValidDirName 校验目录名是否合法
func isValidDirName(dirName string) bool {
	if dirName == "." || strings.Contains(dirName, "/") || strings.Contains(dirName, "\\") || strings.Contains(dirName, "..") {
		return false
	}
	for _, ch := range dirName {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

// getRepoPhysicalPath 计算仓库任务的最终物理绝对路径
func getRepoPhysicalPath(targetPath, dirName, sourceURL, branch string) string {
	if dirName == "." {
		return "" // 如果不追加目录，此逻辑不负责判断其根目录（共享的 scripts 目录）
	}
	finalDirName := dirName
	if finalDirName == "" {
		finalDirName = utils.GetRepoIdentifier(sourceURL, branch)
	}
	if finalDirName == "" {
		return ""
	}

	basePath := targetPath
	if basePath == "" || basePath == constant.ScriptsDirPlaceholder {
		basePath = constant.ScriptsWorkDir
	} else if strings.HasPrefix(basePath, constant.ScriptsDirPlaceholder) {
		basePath = filepath.Join(constant.ScriptsWorkDir, strings.TrimPrefix(basePath, constant.ScriptsDirPlaceholder))
	} else if !filepath.IsAbs(basePath) {
		basePath = filepath.Join(constant.ScriptsWorkDir, basePath)
	}

	fullPath := filepath.Join(basePath, finalDirName)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return ""
	}
	return absPath
}
// CreateTask 创建任务
// @Summary 创建任务
// @Description 创建一个新的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body vo.TaskCreateReq true "任务创建信息"
// @Success 200 {object} utils.Response{data=vo.TaskVO}
// @Failure 400 {object} utils.Response
// @Router /tasks [post]
func (tc *TaskController) CreateTask(c *gin.Context) {
	var req vo.TaskCreateReq

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

	var sourceID string
	// 如果是仓库同步任务，根据 URL 生成 SourceID 用于去重
	if req.Type == constant.TaskTypeRepo && req.Config != "" {
		var repoCfg struct {
			SourceURL   string `json:"source_url"`
			Branch      string `json:"branch"`
			RepoDirName string `json:"repo_dir_name"`
			TargetPath  string `json:"target_path"`
		}
		if err := json.Unmarshal([]byte(req.Config), &repoCfg); err == nil && repoCfg.SourceURL != "" {
			if repoCfg.RepoDirName != "" {
				if !isValidDirName(repoCfg.RepoDirName) {
					utils.BadRequest(c, "自定义目录名只能包含字母、数字、下划线、短划线和点，不能只有点，且不能包含路径逻辑")
					return
				}
			}

			// 如果配置了自定义名字，使用配置的名字。没有配置的话，使用以前的username_reponame
			if repoCfg.RepoDirName != "" {
				sourceID = "repo_" + repoCfg.RepoDirName
			} else {
				sourceID = "repo_" + utils.GetRepoIdentifier(repoCfg.SourceURL, repoCfg.Branch)
			}

			// 校验 SourceID 是否已存在（任务唯一性）
			existingTask := tc.taskService.GetTaskBySourceID(sourceID)
			if existingTask != nil {
				utils.BadRequest(c, "当前任务已存在，请检查或更换仓库目录名称")
				return
			}

			// 校验物理目录是否存在
			newAbsPath := getRepoPhysicalPath(repoCfg.TargetPath, repoCfg.RepoDirName, repoCfg.SourceURL, repoCfg.Branch)
			if newAbsPath != "" {
				if info, err := os.Stat(newAbsPath); err == nil && info.IsDir() {
					utils.BadRequest(c, "本地已存在同名仓库文件夹，请更换自定义目录名或清理残留文件")
					return
				}
			}
		}
	}

	param := tasks.TaskParam{
		Name:          req.Name,
		Remark:        req.Remark,
		Command:       req.Command,
		PreCommand:    req.PreCommand,
		PostCommand:   req.PostCommand,
		Tags:          req.Tags,
		Type:          req.Type,
		Config:        req.Config,
		Schedule:      req.Schedule,
		Timeout:       req.Timeout,
		WorkDir:       workDir,
		CleanConfig:   req.CleanConfig,
		Envs:          req.Envs,
		Languages:     req.Languages,
		AgentID:       req.AgentID,
		TriggerType:   req.TriggerType,
		RetryCount:    req.RetryCount,
		RetryInterval: req.RetryInterval,
		RandomRange:   req.RandomRange,
		SourceID:      sourceID,
		PinType:       req.PinType,
		Enabled:       true,
		SandboxProfileID: req.SandboxProfileID,
	}

	var task *models.Task
	// 去重逻辑：如果已存在相同 SourceID 的仓库任务，则改为更新
	if sourceID != "" {
		task = tc.taskService.GetTaskBySourceID(sourceID)
		if task != nil {
			task = tc.taskService.UpdateTask(task.ID, &param)
		}
	}

	if task == nil {
		task = tc.taskService.CreateTask(&param)
	}

	// 如果是 Agent 任务，通知 Agent；否则添加到本地 cron
	if task.AgentID != nil && *task.AgentID != "" {
		tc.agentWSManager.BroadcastTasks(*task.AgentID)
	} else {
		tc.executorService.AddCronTask(task)
	}

	utils.Success(c, vo.ToTaskVO(task))
}

// BulkSaveTask 批量保存/导入任务配置（用于主节点下发同步）
// @Summary 批量保存任务
// @Description 批量导入任务配置，如果ID或同名存在则更新，不存在则创建
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /tasks/bulk_save [post]
func (tc *TaskController) BulkSaveTask(c *gin.Context) {
	var reqs []vo.TaskVO

	if err := c.ShouldBindJSON(&reqs); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	for _, req := range reqs {
		param := tasks.TaskParam{
			Name:          req.Name,
			Remark:        req.Remark,
			Command:       req.Command,
			PreCommand:    req.PreCommand,
			PostCommand:   req.PostCommand,
			Tags:          req.Tags,
			Type:          req.Type,
			Config:        req.Config,
			Schedule:      req.Schedule,
			Timeout:       req.Timeout,
			WorkDir:       req.WorkDir,
			CleanConfig:   req.CleanConfig,
			Envs:          req.Envs,
			Languages:     req.Languages,
			AgentID:       req.AgentID,
			TriggerType:   req.TriggerType,
			RetryCount:    req.RetryCount,
			RetryInterval: req.RetryInterval,
			RandomRange:   req.RandomRange,
			PinType:       req.PinType,
			Enabled:       req.Enabled,
			SourceID:      "", // 不直接覆盖
		}

		var existingTask *models.Task
		// 优先按 ID 匹配
		if req.ID != "" {
			existingTask = tc.taskService.GetTaskByID(req.ID)
		}
		// 如果 ID 没找到，尝试按 Name 匹配
		if existingTask == nil {
			var t models.Task
			res := database.DB.Where("name = ?", req.Name).First(&t)
			if res.Error == nil {
				existingTask = &t
			}
		}

		var savedTask *models.Task
		if existingTask != nil {
			savedTask = tc.taskService.UpdateTask(existingTask.ID, &param)
		} else {
			savedTask = tc.taskService.CreateTask(&param)
			// 如果原始有 ID，强制覆盖更新 ID 保持强同步一致性
			if req.ID != "" && savedTask != nil {
				database.DB.Model(savedTask).Update("id", req.ID)
				savedTask.ID = req.ID
			}
		}

		// 如果是 Agent 任务，通知 Agent；否则添加到本地 cron
		if savedTask != nil {
			if savedTask.AgentID != nil && *savedTask.AgentID != "" {
				tc.agentWSManager.BroadcastTasks(*savedTask.AgentID)
			} else {
				tc.executorService.AddCronTask(savedTask)
			}
		}
	}

	utils.Success(c, nil)
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

	sortBy := c.DefaultQuery("sort_by", "")
	order := c.DefaultQuery("order", "")

	tasks, total := tc.taskService.GetTasksWithPagination(p.Page, p.PageSize, name, agentID, tags, taskType, sortBy, order)
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
// @Param body body vo.TaskUpdateReq true "任务更新信息"
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

	var req vo.TaskUpdateReq

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

	var sourceID string
	if req.Type == constant.TaskTypeRepo && req.Config != "" {
		var repoCfg struct {
			SourceURL   string `json:"source_url"`
			Branch      string `json:"branch"`
			RepoDirName string `json:"repo_dir_name"`
			TargetPath  string `json:"target_path"`
		}
		if err := json.Unmarshal([]byte(req.Config), &repoCfg); err == nil && repoCfg.SourceURL != "" {
			if repoCfg.RepoDirName != "" {
				if !isValidDirName(repoCfg.RepoDirName) {
					utils.BadRequest(c, "自定义目录名只能包含字母、数字、下划线、短划线和点，不能只有点，且不能包含路径逻辑")
					return
				}
			}

			// 如果配置了自定义名字，使用配置的名字。没有配置的话，使用以前的username_reponame
			if repoCfg.RepoDirName != "" {
				sourceID = "repo_" + repoCfg.RepoDirName
			} else {
				sourceID = "repo_" + utils.GetRepoIdentifier(repoCfg.SourceURL, repoCfg.Branch)
			}

			// 验证更新后的 SourceID 是否和别的任务冲突
			if sourceID != oldTask.SourceID {
				existingTask := tc.taskService.GetTaskBySourceID(sourceID)
				if existingTask != nil && existingTask.ID != oldTask.ID {
					utils.BadRequest(c, "当前任务已存在，请检查或更换仓库目录名称")
					return
				}
			}

			// 计算新的物理路径
			newAbsPath := getRepoPhysicalPath(repoCfg.TargetPath, repoCfg.RepoDirName, repoCfg.SourceURL, repoCfg.Branch)

			var oldAbsPath string
			if oldTask != nil && oldTask.Type == constant.TaskTypeRepo && oldTask.Config != "" {
				var oldCfg struct {
					SourceURL   string `json:"source_url"`
					Branch      string `json:"branch"`
					RepoDirName string `json:"repo_dir_name"`
					TargetPath  string `json:"target_path"`
				}
				if json.Unmarshal([]byte(oldTask.Config), &oldCfg) == nil {
					oldAbsPath = getRepoPhysicalPath(oldCfg.TargetPath, oldCfg.RepoDirName, oldCfg.SourceURL, oldCfg.Branch)
				}
			}

			// 如果路径发生了改变（或者是个全新计算的路径），并且新路径已存在，则报错拦截
			if newAbsPath != "" && newAbsPath != oldAbsPath {
				if info, err := os.Stat(newAbsPath); err == nil && info.IsDir() {
					utils.BadRequest(c, "目标目录在本地已存在同名文件夹，请更换目录名或清理残留文件")
					return
				}
			}
		}
	} else if oldTask != nil {
		sourceID = oldTask.SourceID
	}

	param := tasks.TaskParam{
		Name:          req.Name,
		Remark:        req.Remark,
		Command:       req.Command,
		PreCommand:    req.PreCommand,
		PostCommand:   req.PostCommand,
		Tags:          req.Tags,
		Type:          req.Type,
		Config:        req.Config,
		Schedule:      req.Schedule,
		Timeout:       req.Timeout,
		WorkDir:       workDir,
		CleanConfig:   req.CleanConfig,
		Envs:          req.Envs,
		Languages:     req.Languages,
		AgentID:       req.AgentID,
		TriggerType:   req.TriggerType,
		RetryCount:    req.RetryCount,
		RetryInterval: req.RetryInterval,
		RandomRange:   req.RandomRange,
		SourceID:      sourceID,
		PinType:       req.PinType,
		Enabled:       req.Enabled,
		SandboxProfileID: req.SandboxProfileID,
	}

	task := tc.taskService.UpdateTask(id, &param)
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
		if utils.DerefBool(task.Enabled, true) {
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

	// 获取任务信息（用于通知 agent 和物理删除校验）
	task := tc.taskService.GetTaskByID(id)
	if task == nil {
		utils.NotFound(c, "任务不存在")
		return
	}

	agentID := task.AgentID
	deleteFiles := c.Query("delete_files") == "true"

	// 如果需要删除物理文件且是仓库任务
	if deleteFiles && task.Type == constant.TaskTypeRepo {
		tc.deleteRepoPhysicalFiles(task)
	}

	tc.executorService.RemoveCronTask(id)
	tc.executorService.GetScheduler().StopTask(id)

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

// deleteRepoPhysicalFiles 删除仓库关联的物理文件
func (tc *TaskController) deleteRepoPhysicalFiles(task *models.Task) {
	if task.Type != constant.TaskTypeRepo {
		return
	}

	logger.Infof("[Controller] 开始尝试物理删除任务关联文件: %s", task.Name)
	var repoCfg models.RepoConfig
	if err := json.Unmarshal([]byte(task.Config), &repoCfg); err != nil {
		logger.Errorf("[Controller] 解析任务配置失败: %v", err)
		return
	}

	targetPath := repoCfg.TargetPath
	if targetPath == "" {
		// 如果 TargetPath 为空，调用系统的计算函数获取默认目录名
		repoId := utils.GetRepoIdentifier(repoCfg.SourceURL, repoCfg.Branch)
		if repoId != "" {
			targetPath = repoId
			logger.Infof("[Controller] TargetPath 为空，使用计算出的标识符: %s", targetPath)
		}
	}

	if targetPath == "" || targetPath == constant.ScriptsDirPlaceholder {
		logger.Warnf("[Controller] 任务 %s 无法确定有效的物理删除路径，跳过", task.Name)
		return
	}

	// 确定绝对路径
	scriptsDir, _ := filepath.Abs(constant.ScriptsWorkDir)
	fullPath := targetPath
	if strings.HasPrefix(targetPath, constant.ScriptsDirPlaceholder) {
		fullPath = filepath.Join(scriptsDir, strings.TrimPrefix(targetPath, constant.ScriptsDirPlaceholder))
	} else if !filepath.IsAbs(targetPath) {
		fullPath = filepath.Join(scriptsDir, targetPath)
	}

	absTargetPath, _ := filepath.Abs(fullPath)
	logger.Infof("[Controller] 最终计算的绝对路径: %s, Scripts目录: %s", absTargetPath, scriptsDir)
	scriptsDir, _ = filepath.Abs(constant.ScriptsWorkDir)

	// 安全检查：使用 Rel 判断路径关系
	rel, err := filepath.Rel(scriptsDir, absTargetPath)
	if err != nil {
		logger.Errorf("[Controller] 计算相对路径失败: %v", err)
		return
	}

	// 必须是在 scripts 目录下（不以 .. 开头）且不能是 scripts 目录本身 (.)
	if rel != "." && !strings.HasPrefix(rel, "..") {
		err := os.RemoveAll(absTargetPath)
		if err != nil {
			logger.Errorf("[Controller] 物理删除文件夹失败: %s, 路径: %s, 错误: %v", task.Name, absTargetPath, err)
		} else {
			logger.Infof("[Controller] 已成功物理删除文件夹: %s, 路径: %s", task.Name, absTargetPath)
		}
	} else {
		logger.Warnf("[Controller] 拒绝物理删除安全目录之外的路径: %s", absTargetPath)
	}
}

func (tc *TaskController) BatchDeleteTasks(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 收集涉及到的 AgentID
	agentIDs := make(map[string]struct{})
	for _, id := range req.IDs {
		// 获取任务信息
		task := tc.taskService.GetTaskByID(id)
		if task != nil {
			if task.AgentID != nil && *task.AgentID != "" {
				agentIDs[*task.AgentID] = struct{}{}
			}
		}

		// 移除 cron 调度
		tc.executorService.RemoveCronTask(id)
		tc.executorService.GetScheduler().StopTask(id)
	}

	// 执行批量删除
	count := tc.taskService.BatchDeleteTasks(req.IDs)

	// 通知受影响的 Agent
	for agentID := range agentIDs {
		tc.agentWSManager.BroadcastTasks(agentID)
	}

	utils.Success(c, gin.H{"count": count})
}

// BatchDeleteByQuery 根据查询条件批量删除任务
// @Summary 根据查询条件批量删除任务
// @Description 根据查询条件批量删除匹配的所有任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name query string false "任务名称关键词"
// @Param tags query string false "标签关键词"
// @Param type query string false "任务类型"
// @Param agent_id query string false "执行位置(节点ID)"
// @Success 200 {object} utils.Response{data=map[string]int}
// @Failure 401 {object} utils.Response "未授权"
// @Router /tasks/batch-by-query [delete]
func (tc *TaskController) BatchDeleteByQuery(c *gin.Context) {
	name := c.Query("name")
	agentIDStr := c.Query("agent_id")
	tags := c.Query("tags")
	taskType := c.Query("type")

	var agentID *string
	if agentIDStr != "" {
		agentID = &agentIDStr
	}

	tasks, _ := tc.taskService.GetTasksWithPagination(1, 999999, name, agentID, tags, taskType, "", "")
	if len(tasks) == 0 {
		utils.Success(c, gin.H{"count": 0})
		return
	}

	var ids []string
	agentIDs := make(map[string]struct{})
	for _, task := range tasks {
		ids = append(ids, task.ID)
		if task.AgentID != nil && *task.AgentID != "" {
			agentIDs[*task.AgentID] = struct{}{}
		}
		// 移除 cron 调度
		tc.executorService.RemoveCronTask(task.ID)
		tc.executorService.GetScheduler().StopTask(task.ID)
	}

	// 执行批量删除
	count := tc.taskService.BatchDeleteTasks(ids)

	// 通知受影响的 Agent
	for aID := range agentIDs {
		tc.agentWSManager.BroadcastTasks(aID)
	}

	utils.Success(c, gin.H{"count": count})
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

// GetTags 获取所有任务标签
// @Summary 获取所有任务标签
// @Description 获取系统中所有任务已使用的唯一标签列表
// @Tags 任务管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=[]string}
// @Router /tasks/tags [get]
func (tc *TaskController) GetTags(c *gin.Context) {
	tags, err := tc.taskService.GetAllTags()
	if err != nil {
		utils.ServerError(c, err.Error())
		return
	}
	utils.Success(c, tags)
}

// SyncRepoTasks 增量同步仓库任务状态（供本地 reposync 进程调用）
func (tc *TaskController) SyncRepoTasks(c *gin.Context) {
	var req struct {
		RepoID      string   `json:"repo_id"`
		UpsertedIDs []string `json:"upserted_ids"`
		DeletedIDs  []string `json:"deleted_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	tc.executorService.SyncRepoTasks(req.UpsertedIDs, req.DeletedIDs)
	utils.SuccessMsg(c, "增量同步成功")
}

// ToggleTask 切换任务启用/禁用状态
func (tc *TaskController) ToggleTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的任务ID")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	task := tc.taskService.GetTaskByID(id)
	if task == nil {
		utils.NotFound(c, "任务不存在")
		return
	}

	// 获取旧 AgentID
	var oldAgentID *string
	oldAgentID = task.AgentID

	// 构造更新参数，仅修改 Enabled
	param := tasks.TaskParam{
		Name:          task.Name,
		Remark:        task.Remark,
		Command:       string(task.Command),
		PreCommand:    string(task.PreCommand),
		PostCommand:   string(task.PostCommand),
		Tags:          task.Tags,
		Type:          task.Type,
		Config:        string(task.Config),
		Schedule:      task.Schedule,
		Timeout:       task.Timeout,
		WorkDir:       task.WorkDir,
		CleanConfig:   task.CleanConfig,
		Envs:          string(task.Envs),
		Languages:     task.Languages,
		AgentID:       task.AgentID,
		TriggerType:   task.TriggerType,
		RetryCount:    task.RetryCount,
		RetryInterval: task.RetryInterval,
		RandomRange:   task.RandomRange,
		SourceID:      task.SourceID,
		PinType:       task.PinType,
		Enabled:       req.Enabled,
	}

	updatedTask := tc.taskService.UpdateTask(id, &param)
	if updatedTask == nil {
		utils.NotFound(c, "任务不存在")
		return
	}

	// 处理调度器更新
	if updatedTask.AgentID != nil && *updatedTask.AgentID != "" {
		tc.executorService.RemoveCronTask(updatedTask.ID)
		tc.agentWSManager.BroadcastTasks(*updatedTask.AgentID)
	} else {
		if req.Enabled {
			tc.executorService.AddCronTask(updatedTask)
		} else {
			tc.executorService.RemoveCronTask(updatedTask.ID)
		}
		if oldAgentID != nil && *oldAgentID != "" {
			tc.agentWSManager.BroadcastTasks(*oldAgentID)
		}
	}

	utils.Success(c, vo.ToTaskVO(updatedTask))
}
