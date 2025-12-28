package controllers

import (
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/services"
	"baihu/internal/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var agentUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// AgentController Agent 控制器
type AgentController struct {
	agentService *services.AgentService
	wsManager    *services.AgentWSManager
}

// NewAgentController 创建 Agent 控制器
func NewAgentController() *AgentController {
	return &AgentController{
		agentService: services.NewAgentService(),
		wsManager:    services.GetAgentWSManager(),
	}
}

// List 获取 Agent 列表
func (c *AgentController) List(ctx *gin.Context) {
	agents := c.agentService.List()
	utils.Success(ctx, agents)
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

	// 获取旧状态
	oldAgent := c.agentService.GetByID(uint(id))
	if oldAgent == nil {
		utils.NotFound(ctx, "Agent 不存在")
		return
	}
	wasEnabled := oldAgent.Enabled

	if err := c.agentService.Update(uint(id), req.Name, req.Description, req.Enabled); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	// 如果启用状态发生变化，通知 Agent
	if wasEnabled != req.Enabled {
		if req.Enabled {
			// 启用：发送任务列表
			c.wsManager.SendToAgent(uint(id), services.WSTypeEnabled, map[string]interface{}{
				"message": "Agent 已启用",
			})
			// 发送任务列表
			c.wsManager.BroadcastTasks(uint(id))
		} else {
			// 禁用：发送禁用消息，Agent 收到后清空任务
			c.wsManager.SendToAgent(uint(id), services.WSTypeDisabled, map[string]interface{}{
				"message": "Agent 已禁用",
			})
		}
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
	agent, token, err := c.agentService.Register(&req, ip)
	if err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"agent_id": agent.ID,
		"token":    token,
		"message":  "注册成功",
	})
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


// ========== WebSocket ==========

// WSConnect Agent WebSocket 连接
func (c *AgentController) WSConnect(ctx *gin.Context) {
	ip := ctx.ClientIP()

	// 检查 IP 限流
	if allowed, reason := c.wsManager.CheckRateLimit(ip); !allowed {
		logger.Warnf("[AgentWS] IP %s 被限流: %s", ip, reason)
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": reason})
		return
	}

	token := ctx.Query("token")
	if token == "" {
		c.wsManager.RecordConnectFail(ip)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 token"})
		return
	}

	machineID := ctx.Query("machine_id")
	isNewAgent := false

	// 先尝试用 token 查找已有 Agent
	agent := c.agentService.GetByToken(token)

	// 如果没找到，尝试用令牌注册（会检查 machine_id 是否已存在）
	if agent == nil {
		var err error
		agent, isNewAgent, err = c.agentService.RegisterByToken(token, machineID, ip)
		if err != nil {
			c.wsManager.RecordConnectFail(ip)
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
	}

	if !agent.Enabled {
		c.wsManager.RecordConnectFail(ip)
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Agent 已禁用"})
		return
	}

	conn, err := agentUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logger.Errorf("[AgentWS] 升级连接失败: %v", err)
		return
	}

	// 连接成功，重置失败计数
	c.wsManager.RecordConnectSuccess(ip)

	// 注册连接
	ac := c.wsManager.Register(agent.ID, conn, ip)

	// 更新 Agent 状态
	c.agentService.Heartbeat(token, ip, "", "", "", "", "")

	// 发送连接成功消息（包含注册状态）
	c.wsManager.SendToAgent(agent.ID, services.WSTypeConnected, map[string]interface{}{
		"agent_id":     agent.ID,
		"name":         agent.Name,
		"is_new_agent": isNewAgent,
		"machine_id":   machineID,
	})

	// 启动读写协程
	go c.wsWritePump(ac)
	go c.wsReadPump(ac, agent)
}

// wsReadPump 读取消息
func (c *AgentController) wsReadPump(ac *services.AgentConnection, agent *models.Agent) {
	defer func() {
		c.wsManager.Unregister(agent.ID)
	}()

	ac.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	ac.Conn.SetPongHandler(func(string) error {
		ac.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		_, message, err := ac.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warnf("[AgentWS] Agent #%d 读取错误: %v", agent.ID, err)
			}
			break
		}

		var msg services.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleWSMessage(ac, agent, &msg)
	}
}

// wsWritePump 写入消息
func (c *AgentController) wsWritePump(ac *services.AgentConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-ac.Send:
			if !ok {
				return
			}
			if err := ac.WriteMessage(message); err != nil {
				return
			}
		case <-ticker.C:
			ac.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := ac.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWSMessage 处理 WebSocket 消息
func (c *AgentController) handleWSMessage(ac *services.AgentConnection, agent *models.Agent, msg *services.WSMessage) {
	switch msg.Type {
	case services.WSTypeHeartbeat:
		c.handleHeartbeat(ac, agent, msg.Data)

	case services.WSTypeTaskResult:
		c.handleTaskResult(agent, msg.Data)

	case services.WSTypeFetchTasks:
		c.handleFetchTasks(agent)
	}
}

// handleFetchTasks 处理 Agent 请求任务列表
func (c *AgentController) handleFetchTasks(agent *models.Agent) {
	tasks := c.agentService.GetTasks(agent.ID)
	c.wsManager.SendToAgent(agent.ID, services.WSTypeTasks, map[string]interface{}{
		"tasks": tasks,
	})
	logger.Infof("[AgentWS] Agent #%d 请求任务列表，返回 %d 个任务", agent.ID, len(tasks))
}

// handleHeartbeat 处理心跳
func (c *AgentController) handleHeartbeat(ac *services.AgentConnection, agent *models.Agent, data json.RawMessage) {
	var req struct {
		Version    string `json:"version"`
		BuildTime  string `json:"build_time"`
		Hostname   string `json:"hostname"`
		OS         string `json:"os"`
		Arch       string `json:"arch"`
		AutoUpdate bool   `json:"auto_update"`
	}
	json.Unmarshal(data, &req)

	ac.UpdatePing()

	// 更新 Agent 信息（使用连接时保存的 IP）
	c.agentService.Heartbeat(agent.Token, ac.IP, req.Version, req.BuildTime, req.Hostname, req.OS, req.Arch)

	// 检查是否需要更新
	latestVersion := c.agentService.GetLatestVersion()
	needUpdate := latestVersion != "" && req.Version != "" && req.Version != latestVersion
	forceUpdate := agent.ForceUpdate

	if forceUpdate && needUpdate {
		c.agentService.ClearForceUpdate(agent.ID)
	}

	// 发送心跳响应
	response := map[string]interface{}{
		"agent_id":       agent.ID,
		"name":           agent.Name,
		"need_update":    needUpdate,
		"force_update":   forceUpdate,
		"latest_version": latestVersion,
	}
	c.wsManager.SendToAgent(agent.ID, services.WSTypeHeartbeatAck, response)
}

// handleTaskResult 处理任务结果
func (c *AgentController) handleTaskResult(agent *models.Agent, data json.RawMessage) {
	var result models.AgentTaskResult
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}

	result.AgentID = agent.ID
	c.agentService.ReportResult(&result)
}

// NotifyTaskUpdate 通知 Agent 任务更新
func (c *AgentController) NotifyTaskUpdate(agentID uint) {
	c.wsManager.BroadcastTasks(agentID)
}

// ========== 注册码管理 ==========

// ListRegCodes 获取注册码列表
func (c *AgentController) ListRegCodes(ctx *gin.Context) {
	codes := c.agentService.ListRegCodes()
	utils.Success(ctx, codes)
}

// CreateRegCode 创建注册码
func (c *AgentController) CreateRegCode(ctx *gin.Context) {
	var req struct {
		Remark    string `json:"remark"`
		MaxUses   int    `json:"max_uses"`
		ExpiresAt string `json:"expires_at"` // 格式: 2006-01-02 15:04:05
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", req.ExpiresAt, time.Local)
		if err != nil {
			utils.BadRequest(ctx, "过期时间格式错误")
			return
		}
		expiresAt = &t
	}

	code, err := c.agentService.CreateRegCode(req.Remark, req.MaxUses, expiresAt)
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, code)
}

// DeleteRegCode 删除注册码
func (c *AgentController) DeleteRegCode(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	if err := c.agentService.DeleteRegCode(uint(id)); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "删除成功")
}
