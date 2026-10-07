package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gorilla/websocket"
)

// AgentWSManager WebSocket 连接管理器
type AgentWSManager struct {
	connections   map[string]*AgentConnection             // Agent ID -> 连接对象
	ipConnections map[string]int                          // IP -> 连接数
	ipLastAttempt map[string]time.Time                    // IP -> 最后连接尝试时间
	ipFailCount   map[string]int                          // IP -> 连续失败次数
	remoteWaiters map[string]chan *models.AgentTaskResult // 日志 ID -> 结果通道
	mu            sync.RWMutex
}

// 限流配置
const (
	maxConnectionsPerIP = 10              // 每个 IP 最大连接数
	minConnectInterval  = 5 * time.Second // 同一 IP 最小连接间隔
	maxFailCount        = 5               // 最大连续失败次数
	failBlockDuration   = 5 * time.Minute // 失败后封禁时长
)

// AgentConnection Agent WebSocket 连接
type AgentConnection struct {
	AgentID      string
	IP           string
	Conn         *websocket.Conn
	Send         chan []byte
	LastPing     time.Time
	Capabilities []string
	closed       bool
	mu           sync.Mutex
}

// WSMessage WebSocket 消息结构
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// 消息类型常量
const (
	WSTypeHeartbeat     = constant.WSTypeHeartbeat
	WSTypeHeartbeatAck  = constant.WSTypeHeartbeatAck
	WSTypeTasks         = constant.WSTypeTasks
	WSTypeTaskResult    = constant.WSTypeTaskResult
	WSTypeUpdate        = constant.WSTypeUpdate
	WSTypeDisconnect    = constant.WSTypeDisconnect
	WSTypeConnected     = constant.WSTypeConnected
	WSTypeDisabled      = constant.WSTypeDisabled
	WSTypeEnabled       = constant.WSTypeEnabled
	WSTypeFetchTasks    = constant.WSTypeFetchTasks
	WSTypeTaskLog       = constant.WSTypeTaskLog
	WSTypeExecute       = constant.WSTypeExecute
	WSTypeTaskHeartbeat = constant.WSTypeTaskHeartbeat
	WSTypeSyncRequest   = constant.WSTypeSyncRequest
	WSTypeSyncResult    = constant.WSTypeSyncResult
)

var agentWSManager *AgentWSManager
var agentWSOnce sync.Once

// GetAgentWSManager 获取单例
func GetAgentWSManager() *AgentWSManager {
	agentWSOnce.Do(func() {
		agentWSManager = &AgentWSManager{
			connections:   make(map[string]*AgentConnection),
			ipConnections: make(map[string]int),
			ipLastAttempt: make(map[string]time.Time),
			ipFailCount:   make(map[string]int),
			remoteWaiters: make(map[string]chan *models.AgentTaskResult),
		}
		// 启动时，先将所有 "online" 状态的 Agent 重置为 "offline"
		NewAgentService().ResetAllAgentsToOffline()

		// 将清理任务注册到系统内部 Cron，每 30 秒执行一次
		executor.GetSysCron().AddJob("@every 30s", agentWSManager.cleanupLoop)
	})
	return agentWSManager
}

// CheckRateLimit 检查 IP 限流，返回是否允许连接
func (m *AgentWSManager) CheckRateLimit(ip string) (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// 检查是否被封禁（连续失败过多）
	if failCount, exists := m.ipFailCount[ip]; exists && failCount >= maxFailCount {
		if lastAttempt, ok := m.ipLastAttempt[ip]; ok {
			if now.Sub(lastAttempt) < failBlockDuration {
				remaining := failBlockDuration - now.Sub(lastAttempt)
				return false, "连接失败次数过多，请 " + remaining.Round(time.Second).String() + " 后重试"
			}
			// 封禁时间已过，重置计数
			delete(m.ipFailCount, ip)
		}
	}

	// 检查连接频率
	if lastAttempt, exists := m.ipLastAttempt[ip]; exists {
		if now.Sub(lastAttempt) < minConnectInterval {
			return false, "连接过于频繁，请稍后重试"
		}
	}

	// 检查 IP 连接数
	if count, exists := m.ipConnections[ip]; exists && count >= maxConnectionsPerIP {
		return false, "该 IP 连接数已达上限"
	}

	m.ipLastAttempt[ip] = now
	return true, ""
}

// RecordConnectFail 记录连接失败
func (m *AgentWSManager) RecordConnectFail(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ipFailCount[ip]++
	m.ipLastAttempt[ip] = time.Now()
	if m.ipFailCount[ip] >= maxFailCount {
		logger.Warnf("[AgentWS] IP %s 连续失败 %d 次，已封禁 %v", ip, m.ipFailCount[ip], failBlockDuration)
	}
}

// RecordConnectSuccess 记录连接成功，重置失败计数
func (m *AgentWSManager) RecordConnectSuccess(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.ipFailCount, ip)
}

// Register 注册连接
func (m *AgentWSManager) Register(agentID string, conn *websocket.Conn, ip string, rawCapabilities ...string) *AgentConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭旧连接
	if old, exists := m.connections[agentID]; exists {
		// 减少旧 IP 的连接计数
		if old.IP != "" {
			if count, ok := m.ipConnections[old.IP]; ok && count > 0 {
				m.ipConnections[old.IP] = count - 1
			}
		}
		old.Close()
	}

	var caps []string
	if len(rawCapabilities) > 0 && rawCapabilities[0] != "" {
		parts := strings.Split(rawCapabilities[0], ",")
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				caps = append(caps, trimmed)
			}
		}
	}

	ac := &AgentConnection{
		AgentID:      agentID,
		IP:           ip,
		Conn:         conn,
		Send:         make(chan []byte, 256),
		LastPing:     time.Now(),
		Capabilities: caps,
	}
	m.connections[agentID] = ac

	// 增加 IP 连接计数
	m.ipConnections[ip]++

	logger.Infof("[AgentWS] Agent #%s 已连接 (%s, 能力: %v)", agentID, ip, caps)
	return ac
}

// HasCapability 检查 Agent 连接是否具有指定能力
func (c *AgentConnection) HasCapability(capName string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range c.Capabilities {
		if item == capName {
			return true
		}
	}
	return false
}

// PushAgentSync 向指定 Agent 推送目录或脚本同步任务
func (m *AgentWSManager) PushAgentSync(agentID string, task *models.Task, logID string) (*executor.Result, error) {
	conn := m.GetConnection(agentID)
	if conn == nil {
		return nil, fmt.Errorf("目标 Agent (#%s) 离线，无法推送脚本", agentID)
	}

	tl := tasks.GetActiveLog(logID)
	var logBuf bytes.Buffer
	writeLog := func(format string, a ...interface{}) {
		msg := fmt.Sprintf(format, a...)
		if !strings.HasPrefix(msg, "=") && !strings.HasPrefix(msg, "-") {
			timestamp := time.Now().Format("2006-01-02 15:04:05")
			msg = fmt.Sprintf("[%s] %s", timestamp, msg)
		}
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		logBuf.WriteString(msg)
		if tl != nil {
			_, _ = tl.WriteString(msg)
		}
	}

	syncConfig := task.GetAgentSync()
	if syncConfig == nil || len(syncConfig.DirMappings) == 0 {
		errMsg := "任务未配置 AgentSync 映射目录"
		writeLog("[AgentSync 错误] %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	writeLog("====================================================================================================")
	writeLog("  Agent 脚本与目录同步任务开始")
	writeLog("====================================================================================================")
	writeLog(">> 任务信息: %s (ID: %s)", task.Name, task.ID)
	writeLog(">> 目标节点: Agent #%s (IP: %s)", agentID, conn.IP)
	writeLog(">> 清理策略: 清空目标目录 = %t", syncConfig.CleanTarget)
	if syncConfig.SyncMode != "" {
		writeLog(">> 同步模式: %s (防抖延迟: %d秒)", syncConfig.SyncMode, syncConfig.DebounceDelay)
	}
	if len(syncConfig.IgnoreRules) > 0 {
		writeLog(">> 排除过滤: %s", strings.Join(syncConfig.IgnoreRules, ", "))
	}
	writeLog(">> 映射编排 (%d 条):", len(syncConfig.DirMappings))
	for idx, mapping := range syncConfig.DirMappings {
		remark := ""
		if mapping.Remark != "" {
			remark = fmt.Sprintf(" (%s)", mapping.Remark)
		}
		writeLog("   [%d] 本地源: %s => Agent目标: %s%s", idx+1, mapping.SourcePath, mapping.TargetPath, remark)
	}
	writeLog("----------------------------------------------------------------------------------------------------")

	// 1. 转换并打包映射目录
	var tarMappings []utils.TarMapping
	for _, mapping := range syncConfig.DirMappings {
		srcAbs := constant.ResolveScriptPath(mapping.SourcePath)
		tarMappings = append(tarMappings, utils.TarMapping{
			SourcePath: srcAbs,
			TargetPath: mapping.TargetPath,
		})
	}

	// 函数执行完毕后，延迟触发一次物理内存回收，确保大对象占用彻底归还给 OS
	defer func() {
		go func() {
			time.Sleep(600 * time.Millisecond)
			utils.FreeMemory()
		}()
	}()

	writeLog("[AgentSync] 正在扫描源目录并生成增量归档压缩包...")
	start := time.Now()
	var buf bytes.Buffer
	if err := utils.CreateTarGzWithMappings(&buf, tarMappings, syncConfig.IgnoreRules); err != nil {
		errMsg := fmt.Sprintf("打包同步目录失败: %v", err)
		writeLog("[AgentSync 错误] %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	archiveSize := buf.Len()
	archiveBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	// 立即清空并释放二进制压缩缓冲区内存
	buf.Reset()

	packDuration := time.Since(start).Round(time.Millisecond)
	writeLog("[AgentSync] 本地归档打包完成 (大小: %d 字节, 耗时: %v), 正在通过 WebSocket 推送至 Agent...", archiveSize, packDuration)
	logger.Infof("[AgentSync] 任务 #%s (LogID: %s) 打包完毕: 归档大小 %d 字节, 过滤规则 %d 条, 开始下发至 Agent #%s",
		task.ID, logID, archiveSize, len(syncConfig.IgnoreRules), agentID)

	// 2. 注册等待远程结果
	waiterID := logID
	if waiterID == "" {
		waiterID = fmt.Sprintf("sync_%d", time.Now().UnixNano())
	}
	resultChan := m.RegisterRemoteWaiter(waiterID)
	defer m.UnregisterRemoteWaiter(waiterID)

	// 3. 发送同步请求消息
	reqData := map[string]interface{}{
		"task_id":      task.ID,
		"log_id":       waiterID,
		"clean_target": syncConfig.CleanTarget,
		"mappings":     syncConfig.DirMappings,
		"archive_data": archiveBase64,
	}

	if err := m.SendToAgent(agentID, constant.WSTypeSyncRequest, reqData); err != nil {
		errMsg := fmt.Sprintf("向 Agent 下发同步请求失败: %v", err)
		writeLog("[AgentSync 错误] %s", errMsg)
		return nil, fmt.Errorf("%s", errMsg)
	}

	// 下发完成后立即解除对 Base64 编码大对象的引用，并立即触发一次主动内存回收与物理工作集裁剪
	reqData["archive_data"] = nil
	reqData = nil
	archiveBase64 = ""
	go func() {
		time.Sleep(100 * time.Millisecond)
		utils.FreeMemory()
	}()

	writeLog("[AgentSync] 归档数据包下发完毕，正在等待目标 Agent 节点解包部署与落盘校验...")

	// 4. 等待结果或超时
	timeoutMinutes := task.Timeout
	if timeoutMinutes <= 0 {
		timeoutMinutes = 10
	}
	timeoutChan := time.After(time.Duration(timeoutMinutes) * time.Minute)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case agentResult := <-resultChan:
			end := time.Now()
			duration := agentResult.Duration
			if duration <= 0 {
				duration = end.Sub(start).Milliseconds()
			}
			if agentResult.Status == constant.TaskStatusSuccess {
				writeLog("----------------------------------------------------------------------------------------------------")
				writeLog("[AgentSync 成功] 目标 Agent 已完成解压与写入！总耗时: %v", (time.Duration(duration) * time.Millisecond).Round(time.Millisecond))
				writeLog("====================================================================================================")
			} else {
				writeLog("----------------------------------------------------------------------------------------------------")
				writeLog("[AgentSync 失败] Agent 报错: %s", agentResult.Error)
				writeLog("====================================================================================================")
			}
			finalOutput := agentResult.Output
			if finalOutput == "" {
				finalOutput = logBuf.String()
			}
			return &executor.Result{
				Output:    finalOutput,
				Error:     agentResult.Error,
				Status:    agentResult.Status,
				Duration:  duration,
				ExitCode:  agentResult.ExitCode,
				StartTime: start,
				EndTime:   end,
			}, nil

		case <-timeoutChan:
			end := time.Now()
			errMsg := "等待 Agent 同步结果超时"
			writeLog("[AgentSync 超时] %s", errMsg)
			return &executor.Result{
				Output:    logBuf.String(),
				Status:    constant.TaskStatusFailed,
				Error:     errMsg,
				Duration:  end.Sub(start).Milliseconds(),
				ExitCode:  -1,
				StartTime: start,
				EndTime:   end,
			}, fmt.Errorf("%s", errMsg)

		case <-ticker.C:
			if !m.IsAgentOnline(agentID) {
				end := time.Now()
				errMsg := "Agent 离线，同步被迫终止"
				writeLog("[AgentSync 中断] %s", errMsg)
				return &executor.Result{
					Output:    logBuf.String(),
					Status:    constant.TaskStatusFailed,
					Error:     errMsg,
					Duration:  end.Sub(start).Milliseconds(),
					ExitCode:  -1,
					StartTime: start,
					EndTime:   end,
				}, fmt.Errorf("%s", errMsg)
			}
		}
	}
}

// Unregister 注销连接（只注销指定的连接实例）
func (m *AgentWSManager) Unregister(agentID string, ac *AgentConnection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 只有当前连接和 map 中的连接是同一个实例时才删除
	if conn, exists := m.connections[agentID]; exists && conn == ac {
		// 减少 IP 连接计数
		if conn.IP != "" {
			if count, ok := m.ipConnections[conn.IP]; ok && count > 0 {
				m.ipConnections[conn.IP] = count - 1
			}
		}
		conn.Close()
		delete(m.connections, agentID)
		logger.Infof("[AgentWS] Agent #%s 已断开", agentID)
	}
}

// GetConnection 获取连接
func (m *AgentWSManager) GetConnection(agentID string) *AgentConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[agentID]
}

// IsAgentOnline 检查指定 Agent 是否在线
func (m *AgentWSManager) IsAgentOnline(agentID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.connections[agentID]
	return exists
}

// SendToAgent 发送消息给指定 Agent
func (m *AgentWSManager) SendToAgent(agentID string, msgType string, data interface{}) error {
	conn := m.GetConnection(agentID)
	if conn == nil {
		return nil // Agent 不在线
	}

	dataBytes, _ := json.Marshal(data)
	msg := WSMessage{Type: msgType, Data: dataBytes}
	msgBytes, _ := json.Marshal(msg)

	select {
	case conn.Send <- msgBytes:
		return nil
	default:
		return nil // 缓冲区满，丢弃
	}
}

// BroadcastTasks 广播任务更新给指定 Agent
func (m *AgentWSManager) BroadcastTasks(agentID string) {
	agentService := NewAgentService()
	tasks := agentService.GetTasks(agentID)
	m.SendToAgent(agentID, WSTypeTasks, map[string]interface{}{
		"tasks": tasks,
	})
}

// BroadcastTasksToAll 广播任务更新给所有在线 Agent
func (m *AgentWSManager) BroadcastTasksToAll() {
	m.mu.RLock()
	var agentIDs []string
	for agentID := range m.connections {
		agentIDs = append(agentIDs, agentID)
	}
	m.mu.RUnlock()

	for _, agentID := range agentIDs {
		m.BroadcastTasks(agentID)
	}
}

// RegisterRemoteWaiter 注册远程任务结果等待者
func (m *AgentWSManager) RegisterRemoteWaiter(logID string) chan *models.AgentTaskResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan *models.AgentTaskResult, 1)
	m.remoteWaiters[logID] = ch
	return ch
}

// UnregisterRemoteWaiter 注销远程任务结果等待者
func (m *AgentWSManager) UnregisterRemoteWaiter(logID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.remoteWaiters, logID)
}

// NotifyRemoteResult 通知远程任务结果
func (m *AgentWSManager) NotifyRemoteResult(result *models.AgentTaskResult) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ch, ok := m.remoteWaiters[result.LogID]; ok {
		select {
		case ch <- result:
			return true
		default:
			return false
		}
	}
	return false
}

// OnlineCount 在线 Agent 数量
func (m *AgentWSManager) OnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// cleanupLoop 清理超时连接 (由 SysCron 每 30 秒调用一次)
func (m *AgentWSManager) cleanupLoop() {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[AgentWS] cleanupLoop panic: %v", r)
		}
	}()

	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()

	// 清理超时连接
	for agentID, conn := range m.connections {
		if now.Sub(conn.LastPing) > 2*time.Minute {
			// 减少 IP 连接计数
			if conn.IP != "" {
				if count, ok := m.ipConnections[conn.IP]; ok && count > 0 {
					m.ipConnections[conn.IP] = count - 1
				}
			}
			conn.Close()
			delete(m.connections, agentID)
			// 更新数据库状态
			database.DB.Model(&models.Agent{}).Where("id = ?", agentID).Update("status", constant.AgentStatusOffline)
			logger.Infof("[AgentWS] Agent #%s 心跳超时，已断开", agentID)
		}
	}

	// 定期清理数据库中的过期状态（处理服务重启或异常终止的情况）
	cutoff := now.Add(-2 * time.Minute)
	database.DB.Model(&models.Agent{}).
		Where("status = ? AND last_seen < ?", constant.AgentStatusOnline, cutoff).
		Update("status", constant.AgentStatusOffline)

	// 清理过期的限流记录（超过 10 分钟未活动）
	for ip, lastAttempt := range m.ipLastAttempt {
		if now.Sub(lastAttempt) > 10*time.Minute {
			delete(m.ipLastAttempt, ip)
			delete(m.ipFailCount, ip)
			// 只清理没有活跃连接的 IP 计数
			if m.ipConnections[ip] == 0 {
				delete(m.ipConnections, ip)
			}
		}
	}
}

// Close 关闭连接
func (c *AgentConnection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	if c.Conn != nil {
		c.Conn.Close()
	}
	if c.Send != nil {
		close(c.Send)
	}
}

// IsClosed 检查连接是否已关闭
func (c *AgentConnection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// WriteMessage 写入消息
func (c *AgentConnection) WriteMessage(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.Conn == nil {
		return nil
	}
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

// SetReadDeadline 设置读取超时
func (c *AgentConnection) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.Conn == nil {
		return nil
	}
	return c.Conn.SetReadDeadline(t)
}

// ReadMessage 读取消息
func (c *AgentConnection) ReadMessage() (int, []byte, error) {
	// 不加锁，因为 ReadMessage 是阻塞的
	// 但需要先检查连接状态
	c.mu.Lock()
	if c.closed || c.Conn == nil {
		c.mu.Unlock()
		return 0, nil, websocket.ErrCloseSent
	}
	conn := c.Conn
	c.mu.Unlock()
	return conn.ReadMessage()
}

// WritePing 发送 ping 消息
func (c *AgentConnection) WritePing() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.Conn == nil {
		return nil
	}
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

// UpdatePing 更新心跳时间
func (c *AgentConnection) UpdatePing() {
	c.LastPing = time.Now()
}
