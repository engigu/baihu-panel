package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models/vo"

	"github.com/gorilla/websocket"
)

// SystemWSManager 前端系统事件 WebSocket 管理器 (单例)
type SystemWSManager struct {
	clients map[*ClientConnection]bool
	mu      sync.RWMutex
}

// ClientConnection 代表一个前端页面的 WebSocket 连接
type ClientConnection struct {
	Conn   *websocket.Conn
	Send   chan []byte
	closed bool
	mu     sync.Mutex
}

var systemWSManager *SystemWSManager
var systemWSOnce sync.Once

// GetSystemWSManager 获取系统 WebSocket 管理器单例
func GetSystemWSManager() *SystemWSManager {
	systemWSOnce.Do(func() {
		systemWSManager = &SystemWSManager{
			clients: make(map[*ClientConnection]bool),
		}
	})
	return systemWSManager
}

// Register 注册一个新的前端连接
func (m *SystemWSManager) Register(conn *websocket.Conn) *ClientConnection {
	m.mu.Lock()
	defer m.mu.Unlock()

	client := &ClientConnection{
		Conn: conn,
		Send: make(chan []byte, 256),
	}
	m.clients[client] = true
	return client
}

// Unregister 注销一个前端连接
func (m *SystemWSManager) Unregister(client *ClientConnection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.clients[client]; ok {
		delete(m.clients, client)
		client.Close()
	}
}

// Broadcast 广播消息给所有在线前端
func (m *SystemWSManager) Broadcast(msgType string, payload interface{}) {
	msg := vo.WSMessage{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Errorf("[SystemWS] 序列化消息失败: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		select {
		case client.Send <- data:
		default:
			// 缓冲区满，可能该客户端连接已死
			go m.Unregister(client)
		}
	}
}

// SubscribeEvents 订阅系统事件总线并分发给 WebSocket
func (m *SystemWSManager) SubscribeEvents(bus *eventbus.EventBus) {
	// 任务相关事件
	taskEvents := []string{
		constant.EventTaskSuccess,
		constant.EventTaskFailed,
		constant.EventTaskTimeout,
		constant.EventTaskRunning,
		constant.EventTaskQueued,
		constant.EventTaskCancelled,
	}

	for _, evt := range taskEvents {
		bus.Subscribe(evt, func(e eventbus.Event) {
			m.Broadcast(e.Type, e.Payload)
		})
	}

	// 系统通知事件
	bus.Subscribe(constant.EventSystemNotice, func(e eventbus.Event) {
		m.Broadcast("notice", e.Payload)
	})

	// 应用日志新增事件（驱动运行日志下属4大标签页实时流式刷新列表）
	bus.Subscribe(constant.EventAppLogAdded, func(e eventbus.Event) {
		m.Broadcast(e.Type, e.Payload)
	})
}

func (c *ClientConnection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.Conn.Close()
	close(c.Send)
}
