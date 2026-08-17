package tunnel

import (
	"net"
	"sync/atomic"
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

var (
	txBytes uint64
	rxBytes uint64
)

// trafficCounterConn 包装了原生的 net.Conn，用于统计底层收发的真实物理字节数
type trafficCounterConn struct {
	net.Conn
}

func (c *trafficCounterConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		atomic.AddUint64(&rxBytes, uint64(n))
	}
	return
}

func (c *trafficCounterConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		atomic.AddUint64(&txBytes, uint64(n))
	}
	return
}

// GetTxBytes 返回隧道累计发送的字节数
func GetTxBytes() uint64 {
	return atomic.LoadUint64(&txBytes)
}

// GetRxBytes 返回隧道累计接收的字节数
func GetRxBytes() uint64 {
	return atomic.LoadUint64(&rxBytes)
}

// ResetTrafficBytes 重置流量统计计数器
func ResetTrafficBytes() {
	atomic.StoreUint64(&txBytes, 0)
	atomic.StoreUint64(&rxBytes, 0)
}
