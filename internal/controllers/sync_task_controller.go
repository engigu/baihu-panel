package controllers

import (
	"path/filepath"
	"strconv"

	"baihu/internal/constant"
	"baihu/internal/services"
	"baihu/internal/utils"

	"github.com/gin-gonic/gin"
)

type SyncTaskController struct {
	syncTaskService     *services.SyncTaskService
	syncExecutorService *services.SyncExecutorService
	cronService         *services.CronService
}

func NewSyncTaskController(syncTaskService *services.SyncTaskService, syncExecutorService *services.SyncExecutorService, cronService *services.CronService) *SyncTaskController {
	return &SyncTaskController{
		syncTaskService:     syncTaskService,
		syncExecutorService: syncExecutorService,
		cronService:         cronService,
	}
}

// resolveTargetPath 将相对路径转换为绝对路径
func resolveTargetPath(targetPath string) string {
	// 如果为空，使用 scripts 根目录
	if targetPath == "" {
		absPath, err := filepath.Abs(constant.ScriptsWorkDir)
		if err != nil {
			return constant.ScriptsWorkDir
		}
		return absPath
	}
	// 如果已经是绝对路径，直接返回
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	// 相对路径，基于 scripts 目录
	fullPath := filepath.Join(constant.ScriptsWorkDir, targetPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fullPath
	}
	return absPath
}

// CreateSyncTask 创建同步任务
func (c *SyncTaskController) CreateSyncTask(ctx *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		SourceType  string `json:"source_type" binding:"required"`
		SourceURL   string `json:"source_url" binding:"required"`
		Branch      string `json:"branch"`
		TargetPath  string `json:"target_path"`
		Schedule    string `json:"schedule" binding:"required"`
		Proxy       string `json:"proxy"`
		ProxyURL    string `json:"proxy_url"`
		AuthToken   string `json:"auth_token"`
		CleanConfig string `json:"clean_config"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	// 验证源类型
	if req.SourceType != "url" && req.SourceType != "git" {
		utils.BadRequest(ctx, "源类型必须是 url 或 git")
		return
	}

	// 验证 cron 表达式
	if err := c.cronService.ValidateCron(req.Schedule); err != nil {
		utils.BadRequest(ctx, "无效的cron表达式: "+err.Error())
		return
	}

	// 转换目标路径
	targetPath := resolveTargetPath(req.TargetPath)

	// 验证目标路径
	if !c.syncExecutorService.ValidateTargetPath(targetPath) {
		utils.BadRequest(ctx, "目标路径必须在脚本目录内")
		return
	}

	task := c.syncTaskService.CreateSyncTask(
		req.Name, req.SourceType, req.SourceURL, req.Branch,
		targetPath, req.Schedule, req.Proxy, req.ProxyURL,
		req.AuthToken, req.CleanConfig,
	)

	// 添加到调度器
	c.cronService.AddSyncTask(task)

	utils.Success(ctx, task)
}

// GetSyncTasks 获取同步任务列表
func (c *SyncTaskController) GetSyncTasks(ctx *gin.Context) {
	p := utils.ParsePagination(ctx)
	name := ctx.DefaultQuery("name", "")

	tasks, total := c.syncTaskService.GetSyncTasksWithPagination(p.Page, p.PageSize, name)
	utils.PaginatedResponse(ctx, tasks, total, p)
}

// GetSyncTask 获取单个同步任务
func (c *SyncTaskController) GetSyncTask(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.BadRequest(ctx, "无效的任务ID")
		return
	}

	task := c.syncTaskService.GetSyncTaskByID(id)
	if task == nil {
		utils.NotFound(ctx, "任务不存在")
		return
	}

	utils.Success(ctx, task)
}

// UpdateSyncTask 更新同步任务
func (c *SyncTaskController) UpdateSyncTask(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.BadRequest(ctx, "无效的任务ID")
		return
	}

	var req struct {
		Name        string `json:"name"`
		SourceType  string `json:"source_type"`
		SourceURL   string `json:"source_url"`
		Branch      string `json:"branch"`
		TargetPath  string `json:"target_path"`
		Schedule    string `json:"schedule"`
		Proxy       string `json:"proxy"`
		ProxyURL    string `json:"proxy_url"`
		AuthToken   string `json:"auth_token"`
		CleanConfig string `json:"clean_config"`
		Enabled     bool   `json:"enabled"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	// 验证源类型
	if req.SourceType != "" && req.SourceType != "url" && req.SourceType != "git" {
		utils.BadRequest(ctx, "源类型必须是 url 或 git")
		return
	}

	// 验证 cron 表达式
	if req.Schedule != "" {
		if err := c.cronService.ValidateCron(req.Schedule); err != nil {
			utils.BadRequest(ctx, "无效的cron表达式: "+err.Error())
			return
		}
	}

	// 转换目标路径
	targetPath := resolveTargetPath(req.TargetPath)

	// 验证目标路径
	if targetPath != "" && !c.syncExecutorService.ValidateTargetPath(targetPath) {
		utils.BadRequest(ctx, "目标路径必须在脚本目录内")
		return
	}

	task := c.syncTaskService.UpdateSyncTask(
		id, req.Name, req.SourceType, req.SourceURL, req.Branch,
		targetPath, req.Schedule, req.Proxy, req.ProxyURL,
		req.AuthToken, req.CleanConfig, req.Enabled,
	)

	if task == nil {
		utils.NotFound(ctx, "任务不存在")
		return
	}

	// 更新调度器
	if task.Enabled {
		c.cronService.AddSyncTask(task)
	} else {
		c.cronService.RemoveSyncTask(task.ID)
	}

	utils.Success(ctx, task)
}

// DeleteSyncTask 删除同步任务
func (c *SyncTaskController) DeleteSyncTask(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.BadRequest(ctx, "无效的任务ID")
		return
	}

	// 从调度器移除
	c.cronService.RemoveSyncTask(uint(id))

	success := c.syncTaskService.DeleteSyncTask(id)
	if !success {
		utils.NotFound(ctx, "任务不存在")
		return
	}

	utils.SuccessMsg(ctx, "删除成功")
}

// ExecuteSyncTask 手动执行同步任务
func (c *SyncTaskController) ExecuteSyncTask(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.BadRequest(ctx, "无效的任务ID")
		return
	}

	result := c.syncExecutorService.ExecuteSyncTask(id)
	if !result.Success {
		utils.Error(ctx, 500, result.Error)
		return
	}

	utils.SuccessMsg(ctx, "同步完成")
}
