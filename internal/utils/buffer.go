package utils

import (
	"fmt"
	"sync"
)

// TailBuffer 是一个只保留最后 N 字节数据的缓冲区
type TailBuffer struct {
	mu    sync.Mutex
	limit int
	data  []byte
	size  int // 当前实际存储的大小
	pos   int // 下一个写入位置 (针对环形缓冲区逻辑，但这里为了简单使用切片重排)
}

// NewTailBuffer 创建一个限制大小为 limit 的尾部缓冲区
func NewTailBuffer(limit int) *TailBuffer {
	return &TailBuffer{
		limit: limit,
		data:  make([]byte, 0, limit),
	}
}

// Write 实现 io.Writer 接口
func (b *TailBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n = len(p)
	if n >= b.limit {
		// 如果单次写入就超过了限制，直接取最后 limit 字节
		b.data = append(b.data[:0], p[n-b.limit:]...)
		return
	}

	available := b.limit - len(b.data)
	if n <= available {
		// 空间足够，直接追加
		b.data = append(b.data, p...)
	} else {
		// 空间不足，需要移除旧数据
		toRemove := n - available
		b.data = append(b.data[toRemove:], p...)
	}
	return
}

// Bytes 返回缓冲区内的所有数据
func (b *TailBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	res := make([]byte, len(b.data))
	copy(res, b.data)
	return res
}

// String 返回缓冲区内的字符串表示
func (b *TailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}

// Len 返回当前存储的数据长度
func (b *TailBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.data)
}

// TrimLog 裁剪日志，保留末尾指定大小
func TrimLog(content string, limit int) string {
	if len(content) <= limit {
		return content
	}
	// 简单裁剪，不考虑字符完整性，因为这是针对大文本的保护
	return fmt.Sprintf("\n\n[System] 日志过长，已自动截断，仅保留末尾 %d MB...\n\n", limit/1024/1024) + content[len(content)-limit:]
}
