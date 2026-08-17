package tunnel

import (
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
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

// wsConn 包装 gorilla websocket 以实现 net.Conn 接口
type wsConn struct {
	conn   *websocket.Conn
	reader io.Reader
}

// NetConn 将 WebSocket 连接转换为标准 net.Conn
func NetConn(conn *websocket.Conn) net.Conn {
	return &wsConn{
		conn: conn,
	}
}

func (c *wsConn) Read(b []byte) (n int, err error) {
	for {
		if c.reader == nil {
			_, r, err := c.conn.NextReader()
			if err != nil {
				return 0, err
			}
			c.reader = r
		}
		n, err = c.reader.Read(b)
		if err == io.EOF {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (c *wsConn) Write(b []byte) (n int, err error) {
	// Yamux 保证了对底层连接的序列化写入（不会出现并发写），
	// 因此我们这里不需要加互斥锁 (Mutex)。
	w, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	n, err = w.Write(b)

	// 必须关闭 writer 以触发 WebSocket 的消息帧封包发送
	closeErr := w.Close()
	if err == nil {
		err = closeErr
	}
	return n, err
}

func (c *wsConn) Close() error {
	return c.conn.Close()
}

func (c *wsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *wsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.conn.SetWriteDeadline(t)
}

func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
