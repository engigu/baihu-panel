package tunnel

import (
	"net"
	"sync/atomic"
)

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
