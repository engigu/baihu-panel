package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

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
	agentService    *services.AgentService
	wsManager       *services.AgentWSManager
	settingsService *services.SettingsService
}

// NewAgentController 创建 Agent 控制器
func NewAgentController(settingsService *services.SettingsService) *AgentController {
	return &AgentController{
		agentService:    services.NewAgentService(),
		wsManager:       services.GetAgentWSManager(),
		settingsService: settingsService,
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
	needUpdate := c.agentService.CheckNeedUpdate(req.Version, req.BuildTime)
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

	// 先尝试通过 token 查找 Agent
	agent := c.agentService.GetByToken(token)

	// 如果找不到，尝试验证令牌并通过 machine_id 查找
	if agent == nil {
		machineID := ctx.GetHeader("X-Machine-ID")
		if machineID != "" {
			// 验证令牌是否有效
			if _, err := c.agentService.ValidateToken(token); err == nil {
				// 令牌有效，尝试通过 machine_id 查找 Agent
				agent = c.agentService.GetByMachineID(machineID)
			}
		}
	}

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
	// 添加 panic 恢复
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[AgentWS] WSConnect panic: %v", r)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
		}
	}()

	ip := ctx.ClientIP()

	// 打印请求信息用于调试
	logger.Infof("[AgentWS] 收到连接请求: IP=%s, URL=%s", ip, ctx.Request.URL.String())

	// 检查 IP 限流
	if allowed, reason := c.wsManager.CheckRateLimit(ip); !allowed {
		logger.Warnf("[AgentWS] IP %s 被限流: %s", ip, reason)
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": reason})
		return
	}

	token := ctx.Query("token")
	if token == "" {
		c.wsManager.RecordConnectFail(ip)
		logger.Warnf("[AgentWS] 连接失败: 缺少 token, IP=%s", ip)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 token"})
		return
	}

	machineID := ctx.Query("machine_id")
	logger.Infof("[AgentWS] Token: %s..., MachineID: %s...", token[:8], machineID[:16])

	isNewAgent := false

	// 先尝试用 token 查找已有 Agent
	agent := c.agentService.GetByToken(token)
	logger.Infof("[AgentWS] GetByToken 结果: agent=%v", agent != nil)

	// 如果没找到，尝试用令牌注册（会检查 machine_id 是否已存在）
	if agent == nil {
		logger.Infof("[AgentWS] 尝试注册新 Agent")
		var err error
		agent, isNewAgent, err = c.agentService.RegisterByToken(token, machineID, ip)
		if err != nil {
			c.wsManager.RecordConnectFail(ip)
			logger.Warnf("[AgentWS] 注册失败: %v, IP=%s, token=%s", err, ip, token[:8]+"...")
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		logger.Infof("[AgentWS] 注册成功: Agent #%d, isNew=%v", agent.ID, isNewAgent)
	}

	if !agent.Enabled {
		c.wsManager.RecordConnectFail(ip)
		logger.Warnf("[AgentWS] Agent #%d 已禁用, IP=%s", agent.ID, ip)
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Agent 已禁用"})
		return
	}

	logger.Infof("[AgentWS] 准备升级连接: Agent #%d, IP=%s", agent.ID, ip)
	conn, err := agentUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logger.Errorf("[AgentWS] 升级连接失败: %v, Agent #%d, IP=%s", err, agent.ID, ip)
		return
	}

	// 连接成功，重置失败计数
	c.wsManager.RecordConnectSuccess(ip)

	// 注册连接
	ac := c.wsManager.Register(agent.ID, conn, ip)

	// 更新 Agent 状态
	c.agentService.Heartbeat(token, ip, "", "", "", "", "")

	// 获取调度配置
	workerCount := getIntSetting(c.settingsService, constant.SectionScheduler, constant.KeyWorkerCount, 4)
	queueSize := getIntSetting(c.settingsService, constant.SectionScheduler, constant.KeyQueueSize, 100)
	rateInterval := getIntSetting(c.settingsService, constant.SectionScheduler, constant.KeyRateInterval, 200)

	// 发送连接成功消息（包含注册状态和调度配置）
	c.wsManager.SendToAgent(agent.ID, services.WSTypeConnected, map[string]interface{}{
		"agent_id":     agent.ID,
		"name":         agent.Name,
		"is_new_agent": isNewAgent,
		"machine_id":   machineID,
		"scheduler_config": map[string]interface{}{
			"worker_count":  workerCount,
			"queue_size":    queueSize,
			"rate_interval": rateInterval,
		},
	})

	logger.Infof("[AgentWS] Agent #%d 连接成功 (配置: workers=%d, queue=%d, rate=%d)",
		agent.ID, workerCount, queueSize, rateInterval)

	// 启动读写协程
	go c.wsWritePump(ac)
	go c.wsReadPump(ac, agent)
}

// wsReadPump 读取消息
func (c *AgentController) wsReadPump(ac *services.AgentConnection, agent *models.Agent) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[AgentWS] Agent #%d wsReadPump panic: %v", agent.ID, r)
		}
		logger.Infof("[AgentWS] Agent #%d wsReadPump 退出", agent.ID)
		c.wsManager.Unregister(agent.ID, ac)
	}()

	// 检查连接是否有效（可能是旧连接被新连接替换）
	if ac == nil || ac.IsClosed() {
		return
	}

	ac.SetReadDeadline(time.Now().Add(90 * time.Second))
	// 注意：SetPongHandler 需要直接访问 Conn，但这里我们在连接建立后立即设置
	// 所以是安全的，因为此时连接还没有被其他 goroutine 关闭
	ac.Conn.SetPongHandler(func(string) error {
		ac.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		_, message, err := ac.ReadMessage()
		if err != nil {
			logger.Warnf("[AgentWS] Agent #%d 读取错误: %v", agent.ID, err)
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
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[AgentWS] Agent #%d wsWritePump panic: %v", ac.AgentID, r)
		}
		logger.Infof("[AgentWS] Agent #%d wsWritePump 退出", ac.AgentID)
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-ac.Send:
			if !ok {
				logger.Warnf("[AgentWS] Agent #%d Send channel 已关闭", ac.AgentID)
				return
			}
			if ac.IsClosed() {
				logger.Warnf("[AgentWS] Agent #%d 连接已关闭(write)", ac.AgentID)
				return
			}
			if err := ac.WriteMessage(message); err != nil {
				logger.Warnf("[AgentWS] Agent #%d 写入消息失败: %v", ac.AgentID, err)
				return
			}
		case <-ticker.C:
			if ac.IsClosed() {
				return
			}
			if err := ac.WritePing(); err != nil {
				logger.Warnf("[AgentWS] Agent #%d 发送 Ping 失败: %v", ac.AgentID, err)
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

	case services.WSTypeTaskLog:
		c.handleTaskLog(agent, msg.Data)

	case services.WSTypeFetchTasks:
		c.handleFetchTasks(agent)

	case services.WSTypeTaskHeartbeat: // 任务心跳
		c.handleTaskHeartbeat(agent, msg.Data)
	}
}

// handleTaskHeartbeat 处理任务心跳
func (c *AgentController) handleTaskHeartbeat(agent *models.Agent, data json.RawMessage) {
	var req struct {
		LogID    uint  `json:"log_id"`
		Duration int64 `json:"duration"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		logger.Errorf("[AgentWS] 解析心跳消息失败: %v", err)
		return
	}
	if req.LogID > 0 {
		logger.Infof("[AgentWS] 收到任务心跳: LogID=%d, Duration=%dms", req.LogID, req.Duration)
		c.agentService.UpdateTaskDuration(req.LogID, req.Duration)
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
	needUpdate := c.agentService.CheckNeedUpdate(req.Version, req.BuildTime)
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

// handleTaskLog 处理 Agent 发送的实时日志
func (c *AgentController) handleTaskLog(agent *models.Agent, data json.RawMessage) {
	var logMsg struct {
		LogID   uint   `json:"log_id"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &logMsg); err != nil {
		logger.Errorf("[AgentWS] 解析日志消息失败: %v", err)
		return
	}

	tl := tasks.GetActiveLog(logMsg.LogID)
	if tl != nil {
		tl.Write([]byte(logMsg.Content))
	} else {
		logger.Warnf("[AgentWS] 收到任务日志但未找到活跃 TinyLog: LogID=%d, ContentSize=%d", logMsg.LogID, len(logMsg.Content))
	}
}

// NotifyTaskUpdate 通知 Agent 任务更新
func (c *AgentController) NotifyTaskUpdate(agentID uint) {
	c.wsManager.BroadcastTasks(agentID)
}

// ========== 令牌管理 ==========

// ListTokens 获取令牌列表
func (c *AgentController) ListTokens(ctx *gin.Context) {
	tokens := c.agentService.ListTokens()
	utils.Success(ctx, tokens)
}

// CreateToken 创建令牌
func (c *AgentController) CreateToken(ctx *gin.Context) {
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

	token, err := c.agentService.CreateToken(req.Remark, req.MaxUses, expiresAt)
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, token)
}

// DeleteToken 删除令牌
func (c *AgentController) DeleteToken(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	if err := c.agentService.DeleteToken(uint(id)); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "删除成功")
}

// getIntSetting 辅助方法
func getIntSetting(s *services.SettingsService, section, key string, defaultVal int) int {
	val := s.Get(section, key)

	if val == "" {
		return defaultVal
	}
	if result, err := strconv.Atoi(val); err == nil {
		return result
	}
	return defaultVal
}
