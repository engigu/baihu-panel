package services

import (
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// AgentWSManager WebSocket 连接管理器
type AgentWSManager struct {
	connections   map[uint]*AgentConnection // agentID -> connection
	ipConnections map[string]int            // IP -> 连接数
	ipLastAttempt map[string]time.Time      // IP -> 最后连接尝试时间
	ipFailCount   map[string]int            // IP -> 连续失败次数
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
	AgentID   uint
	IP        string
	Conn      *websocket.Conn
	Send      chan []byte
	LastPing  time.Time
	mu        sync.Mutex
}

// WSMessage WebSocket 消息结构
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// 消息类型常量
const (
	WSTypeHeartbeat    = "heartbeat"
	WSTypeHeartbeatAck = "heartbeat_ack"
	WSTypeTasks        = "tasks"
	WSTypeTaskResult   = "task_result"
	WSTypeUpdate       = "update"
	WSTypeDisconnect   = "disconnect"
	WSTypeConnected    = "connected"   // 连接成功，包含注册状态
	WSTypeDisabled     = "disabled"    // Agent 被禁用
	WSTypeEnabled      = "enabled"     // Agent 被启用
	WSTypeFetchTasks   = "fetch_tasks" // Agent 请求任务列表
)

var agentWSManager *AgentWSManager
var agentWSOnce sync.Once

// GetAgentWSManager 获取单例
func GetAgentWSManager() *AgentWSManager {
	agentWSOnce.Do(func() {
		agentWSManager = &AgentWSManager{
			connections:   make(map[uint]*AgentConnection),
			ipConnections: make(map[string]int),
			ipLastAttempt: make(map[string]time.Time),
			ipFailCount:   make(map[string]int),
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
func (m *AgentWSManager) Register(agentID uint, conn *websocket.Conn, ip string) *AgentConnection {
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

	ac := &AgentConnection{
		AgentID:  agentID,
		IP:       ip,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		LastPing: time.Now(),
	}
	m.connections[agentID] = ac

	// 增加 IP 连接计数
	m.ipConnections[ip]++

	logger.Infof("[AgentWS] Agent #%d 已连接 (%s)", agentID, ip)
	return ac
}

// Unregister 注销连接
func (m *AgentWSManager) Unregister(agentID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[agentID]; exists {
		// 减少 IP 连接计数
		if conn.IP != "" {
			if count, ok := m.ipConnections[conn.IP]; ok && count > 0 {
				m.ipConnections[conn.IP] = count - 1
			}
		}
		conn.Close()
		delete(m.connections, agentID)
		logger.Infof("[AgentWS] Agent #%d 已断开", agentID)
	}
}

// GetConnection 获取连接
func (m *AgentWSManager) GetConnection(agentID uint) *AgentConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[agentID]
}

// SendToAgent 发送消息给指定 Agent
func (m *AgentWSManager) SendToAgent(agentID uint, msgType string, data interface{}) error {
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
func (m *AgentWSManager) BroadcastTasks(agentID uint) {
	agentService := NewAgentService()
	tasks := agentService.GetTasks(agentID)
	m.SendToAgent(agentID, WSTypeTasks, map[string]interface{}{
		"tasks": tasks,
	})
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

	for range ticker.C {
		m.mu.Lock()
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
				database.DB.Model(&models.Agent{}).Where("id = ?", agentID).Update("status", "offline")
				logger.Infof("[AgentWS] Agent #%d 心跳超时，已断开", agentID)
			}
		}

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
	}
}

// Close 关闭连接
func (c *AgentConnection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Conn != nil {
		c.Conn.Close()
		c.Conn = nil
	}
	if c.Send != nil {
		close(c.Send)
		c.Send = nil
	}
}

// WriteMessage 写入消息
func (c *AgentConnection) WriteMessage(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Conn == nil {
		return nil
	}
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

// UpdatePing 更新心跳时间
func (c *AgentConnection) UpdatePing() {
	c.LastPing = time.Now()
}
