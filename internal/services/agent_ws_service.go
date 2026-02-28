package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"

	"github.com/gorilla/websocket"
)

// AgentWSManager WebSocket 连接管理器
type AgentWSManager struct {
	connections   map[string]*AgentConnection           // Agent UUID -> 连接对象
	ipConnections map[string]int                        // IP -> 连接数
	ipLastAttempt map[string]time.Time                  // IP -> 最后连接尝试时间
	ipFailCount   map[string]int                        // IP -> 连续失败次数
	remoteWaiters map[string]chan *models.AgentTaskResult // 日志 UUID -> 结果通道
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
	AgentUUID string
	IP       string
	Conn     *websocket.Conn
	Send     chan []byte
	LastPing time.Time
	closed   bool
	mu       sync.Mutex
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
		go agentWSManager.cleanupLoop()
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
func (m *AgentWSManager) Register(agentUUID string, conn *websocket.Conn, ip string) *AgentConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭旧连接
	if old, exists := m.connections[agentUUID]; exists {
		// 减少旧 IP 的连接计数
		if old.IP != "" {
			if count, ok := m.ipConnections[old.IP]; ok && count > 0 {
				m.ipConnections[old.IP] = count - 1
			}
		}
		old.Close()
	}

	ac := &AgentConnection{
		AgentUUID: agentUUID,
		IP:       ip,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		LastPing: time.Now(),
	}
	m.connections[agentUUID] = ac

	// 增加 IP 连接计数
	m.ipConnections[ip]++

	logger.Infof("[AgentWS] Agent %s 已连接 (%s)", agentUUID, ip)
	return ac
}

// Unregister 注销连接（只注销指定的连接实例）
func (m *AgentWSManager) Unregister(agentUUID string, ac *AgentConnection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 只有当前连接和 map 中的连接是同一个实例时才删除
	if conn, exists := m.connections[agentUUID]; exists && conn == ac {
		// 减少 IP 连接计数
		if conn.IP != "" {
			if count, ok := m.ipConnections[conn.IP]; ok && count > 0 {
				m.ipConnections[conn.IP] = count - 1
			}
		}
		conn.Close()
		delete(m.connections, agentID)
		logger.Infof("[AgentWS] Agent %s 已断开", agentUUID)
	}
}

// GetConnection 获取连接
func (m *AgentWSManager) GetConnection(agentUUID string) *AgentConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[agentUUID]
}

// SendToAgent 发送消息给指定 Agent
func (m *AgentWSManager) SendToAgent(agentUUID string, msgType string, data interface{}) error {
	conn := m.GetConnection(agentUUID)
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
func (m *AgentWSManager) BroadcastTasks(agentUUID string) {
	agentService := NewAgentService()
	tasks := agentService.GetTasks(agentUUID)
	m.SendToAgent(agentUUID, WSTypeTasks, map[string]interface{}{
		"tasks": tasks,
	})
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

// cleanupLoop 清理超时连接
func (m *AgentWSManager) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 启动时，先将所有 "online" 状态的 Agent 重置为 "offline"
	// 因为 WebSocket 连接在应用启动时是空的，所有 Agent 客观上都是离线状态
	// 等它们重新连接上来后，会变为 "online"
	NewAgentService().ResetAllAgentsToOffline()

	for range ticker.C {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("[AgentWS] cleanupLoop panic: %v", r)
				}
			}()
			m.mu.Lock()
			now := time.Now()

			// 清理超时连接
			for agentUUID, conn := range m.connections {
				if now.Sub(conn.LastPing) > 2*time.Minute {
					// 减少 IP 连接计数
					if conn.IP != "" {
						if count, ok := m.ipConnections[conn.IP]; ok && count > 0 {
							m.ipConnections[conn.IP] = count - 1
						}
					}
					conn.Close()
					delete(m.connections, agentUUID)
					// 更新数据库状态
					database.DB.Model(&models.Agent{}).Where("uuid = ?", agentUUID).Update("status", constant.AgentStatusOffline)
					logger.Infof("[AgentWS] Agent %s 心跳超时，已断开", agentUUID)
				}
			}

			// 定期清理数据库中的过期状态（处理服务重启或异常终止的情况）
			// 有些 Agent 虽然没有连接，但数据库状态可能是 "online"
			cutoff := now.Add(-2 * time.Minute)
			database.DB.Model(&models.Agent{}).
				Where("status = ? AND last_seen < ?", constant.AgentStatusOnline, cutoff).
				Update("status", constant.AgentStatusOffline)

			// 清理过期的限流记录（超过 10 分钟未活动）

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

			m.mu.Unlock()
		}()
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
