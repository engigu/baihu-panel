package tunnel

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/hashicorp/yamux"
)

// TunnelSession 表示一个基于 WebSocket 的 Yamux 活跃会话
type TunnelSession struct {
	NodeID    string
	Token     string
	Session   *yamux.Session
	Transport *http.Transport
}

var (
	sessions   = make(map[string]*TunnelSession)
	sessionsMu sync.RWMutex
)

func AddSession(nodeID string, token string, session *yamux.Session) *TunnelSession {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	// 关闭已经存在的旧会话
	if old, exists := sessions[nodeID]; exists {
		old.Session.Close()
		old.Transport.CloseIdleConnections()
	}

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return session.Open()
		},
		DisableKeepAlives: true,
	}

	sess := &TunnelSession{
		NodeID:    nodeID,
		Token:     token,
		Session:   session,
		Transport: tr,
	}
	sessions[nodeID] = sess
	return sess
}

func RemoveSession(nodeID string, session *yamux.Session) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	if sess, exists := sessions[nodeID]; exists && sess.Session == session {
		sess.Session.Close()
		sess.Transport.CloseIdleConnections()
		delete(sessions, nodeID)
		
		// 节点下线时实时更新数据库状态
		database.DB.Model(&models.InterconnectNode{}).
			Where("id = ?", nodeID).
			Update("status", "offline")
	}
}

// CloseAllSessions 强制关闭所有现存的子节点会话（用于主控角色被取消时）
func CloseAllSessions() {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	for _, sess := range sessions {
		if sess.Session != nil {
			sess.Session.Close()
		}
		if sess.Transport != nil {
			sess.Transport.CloseIdleConnections()
		}
	}
	// 清空所有的会话
	sessions = make(map[string]*TunnelSession)
}

func GetSession(nodeID string) *TunnelSession {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	return sessions[nodeID]
}
