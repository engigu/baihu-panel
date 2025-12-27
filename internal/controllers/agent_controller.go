package controllers

import (
	"baihu/internal/models"
	"baihu/internal/services"
	"baihu/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AgentController Agent 控制器
type AgentController struct {
	agentService *services.AgentService
}

// NewAgentController 创建 Agent 控制器
func NewAgentController() *AgentController {
	return &AgentController{
		agentService: services.NewAgentService(),
	}
}

// List 获取已审核的 Agent 列表
func (c *AgentController) List(ctx *gin.Context) {
	agents := c.agentService.List()
	utils.Success(ctx, agents)
}

// ListPending 获取待审核的 Agent 列表
func (c *AgentController) ListPending(ctx *gin.Context) {
	agents := c.agentService.ListPending()
	utils.Success(ctx, agents)
}

// Approve 审核通过 Agent
func (c *AgentController) Approve(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	agent, err := c.agentService.Approve(uint(id))
	if err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	utils.Success(ctx, agent)
}

// Reject 拒绝 Agent
func (c *AgentController) Reject(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	if err := c.agentService.Reject(uint(id)); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "已拒绝")
}

// Update 更新 Agent
func (c *AgentController) Update(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	if err := c.agentService.Update(uint(id), req.Name, req.Description, req.Enabled); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "更新成功")
}

// Delete 删除 Agent
func (c *AgentController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	if err := c.agentService.Delete(uint(id)); err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "删除成功")
}

// RegenerateToken 重新生成 Token
func (c *AgentController) RegenerateToken(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	token, err := c.agentService.RegenerateToken(uint(id))
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"token": token})
}

// ========== Agent API（供 Agent 调用）==========

// Register Agent 注册（无需认证）
func (c *AgentController) Register(ctx *gin.Context) {
	var req models.AgentRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	if req.Name == "" {
		utils.BadRequest(ctx, "名称不能为空")
		return
	}

	ip := ctx.ClientIP()
	agent, err := c.agentService.Register(&req, ip)
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"agent_id": agent.ID,
		"status":   agent.Status,
		"message":  "注册成功，等待审核",
	})
}

// CheckStatus Agent 检查状态（用于轮询等待审核结果）
func (c *AgentController) CheckStatus(ctx *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	ip := ctx.ClientIP()
	agent, err := c.agentService.CheckPendingAgent(req.Name, ip)
	if err != nil {
		utils.NotFound(ctx, err.Error())
		return
	}

	response := gin.H{
		"agent_id": agent.ID,
		"status":   agent.Status,
	}

	// 如果已审核通过，返回 Token
	if agent.Status != "pending" && agent.Token != "" {
		response["token"] = agent.Token
	}

	utils.Success(ctx, response)
}

// Heartbeat Agent 心跳
func (c *AgentController) Heartbeat(ctx *gin.Context) {
	token := c.getAgentToken(ctx)
	if token == "" {
		utils.Unauthorized(ctx, "缺少认证 Token")
		return
	}

	var req struct {
		Version    string `json:"version"`
		BuildTime  string `json:"build_time"`
		Hostname   string `json:"hostname"`
		OS         string `json:"os"`
		Arch       string `json:"arch"`
		AutoUpdate bool   `json:"auto_update"`
	}
	ctx.ShouldBindJSON(&req)

	ip := ctx.ClientIP()
	agent, err := c.agentService.Heartbeat(token, ip, req.Version, req.BuildTime, req.Hostname, req.OS, req.Arch)
	if err != nil {
		utils.Unauthorized(ctx, err.Error())
		return
	}

	// 检查是否需要更新
	latestVersion := c.agentService.GetLatestVersion()
	needUpdate := latestVersion != "" && req.Version != "" && req.Version != latestVersion
	forceUpdate := agent.ForceUpdate

	// 如果强制更新已触发，重置标志
	if forceUpdate && needUpdate {
		c.agentService.ClearForceUpdate(agent.ID)
	}

	utils.Success(ctx, gin.H{
		"agent_id":       agent.ID,
		"name":           agent.Name,
		"need_update":    needUpdate,
		"force_update":   forceUpdate,
		"latest_version": latestVersion,
	})
}

// GetTasks Agent 获取任务列表
func (c *AgentController) GetTasks(ctx *gin.Context) {
	token := c.getAgentToken(ctx)
	if token == "" {
		utils.Unauthorized(ctx, "缺少认证 Token")
		return
	}

	agent := c.agentService.GetByToken(token)
	if agent == nil {
		utils.Unauthorized(ctx, "无效的 Token")
		return
	}

	if !agent.Enabled {
		utils.Forbidden(ctx, "Agent 已禁用")
		return
	}

	tasks := c.agentService.GetTasks(agent.ID)
	utils.Success(ctx, gin.H{
		"agent_id": agent.ID,
		"tasks":    tasks,
	})
}

// ReportResult Agent 上报执行结果
func (c *AgentController) ReportResult(ctx *gin.Context) {
	token := c.getAgentToken(ctx)
	if token == "" {
		utils.Unauthorized(ctx, "缺少认证 Token")
		return
	}

	agent := c.agentService.GetByToken(token)
	if agent == nil {
		utils.Unauthorized(ctx, "无效的 Token")
		return
	}

	if !agent.Enabled {
		utils.Forbidden(ctx, "Agent 已禁用")
		return
	}

	var result models.AgentTaskResult
	if err := ctx.ShouldBindJSON(&result); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	result.AgentID = agent.ID

	if err := c.agentService.ReportResult(&result); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "上报成功")
}

// getAgentToken 从请求头获取 Agent Token
func (c *AgentController) getAgentToken(ctx *gin.Context) string {
	auth := ctx.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	// Bearer <token>
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return auth
}

// Download 下载 Agent 程序
func (c *AgentController) Download(ctx *gin.Context) {
	osType := ctx.DefaultQuery("os", "linux")
	arch := ctx.DefaultQuery("arch", "amd64")

	data, filename, err := c.agentService.GetAgentBinary(osType, arch)
	if err != nil {
		utils.NotFound(ctx, err.Error())
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename="+filename)
	ctx.Header("Content-Type", "application/gzip")
	ctx.Header("Content-Length", strconv.Itoa(len(data)))
	ctx.Data(200, "application/gzip", data)
}

// GetVersion 获取 Agent 最新版本信息
func (c *AgentController) GetVersion(ctx *gin.Context) {
	version := c.agentService.GetLatestVersion()
	platforms := c.agentService.GetAvailablePlatforms()

	utils.Success(ctx, gin.H{
		"version":   version,
		"platforms": platforms,
	})
}

// ForceUpdate 强制更新指定 Agent
func (c *AgentController) ForceUpdate(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	if err := c.agentService.SetForceUpdate(uint(id)); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "已标记强制更新，Agent 下次心跳时将自动更新")
}
