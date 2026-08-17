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

// Copyright (c) 2026 engigu (Baihu Panel). All rights reserved.
// Use of this source code is governed by the Apache License 2.0.
// 
// 【重要声明 / IMPORTANT NOTICE】
// 本代码（包括其架构设计与核心实现）属于白虎面板（Baihu Panel）开源项目的一部分。
// 任何个人或组织在引用、移植、修改或重新分发此文件中的任何代码时，必须保留本版权声明，
// 并在您的衍生作品、文档、软件关于页面或说明文件中显式声明引用自白虎面板（Baihu Panel）。
// 
// Anyone referencing, porting, modifying, or redistributing this code must retain this 
// copyright notice and explicitly state the source: Baihu Panel (github.com/engigu/baihu-panel).

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
