package tasks

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/engigu/baihu-panel/internal/utils"
)

var (
	// globalTinyLogManager 跟踪所有活跃的 TinyLog 实例
	globalTinyLogManager = &TinyLogManager{
		logs: make(map[string]*TinyLog),
	}
)

type TinyLogManager struct {
	mu   sync.RWMutex
	logs map[string]*TinyLog
}

func (m *TinyLogManager) Register(log *TinyLog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs[log.LogID] = log
}

func (m *TinyLogManager) Unregister(logID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.logs, logID)
}

func (m *TinyLogManager) Get(logID string) *TinyLog {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logs[logID]
}

// GetActiveLog 通过 ID 获取活跃的 TinyLog 实例
func GetActiveLog(logID string) *TinyLog {
	return globalTinyLogManager.Get(logID)
}

// TinyLog 是一个高性能、低内存占用的日志收集器
type TinyLog struct {
	LogID       string
	mu          sync.RWMutex
	file        *os.File
	path        string
	writer      *bufio.Writer
	subscribers []chan []byte
	remainder   []byte // Leftover bytes from previous write (partial multi-byte characters)
	closed      bool
}

// NewTinyLog 创建一个新的 TinyLog 实例（基于临时文件存储）并注册它
func NewTinyLog(logID string) (*TinyLog, error) {
	f, err := os.CreateTemp("", "task_log_*.log")
	if err != nil {
		return nil, err
	}

	tl := &TinyLog{
		LogID:       logID,
		file:        f,
		path:        f.Name(),
		writer:      bufio.NewWriter(f),
		subscribers: make([]chan []byte, 0),
	}
	globalTinyLogManager.Register(tl)
	return tl, nil
}

// Write 实现 io.Writer 接口
func (l *TinyLog) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return 0, os.ErrClosed
	}

	// 1. 合并上次调用剩余的字节（可能是半个 UTF-8 字符）
	originalInputLen := len(p)
	payload := p
	if len(l.remainder) > 0 {
		payload = append(l.remainder, p...)
		l.remainder = nil
	}

	// 2. 识别结尾不完整的 UTF-8 序列
	lastSafe := len(payload)
	// UTF-8 字符最多 4 字节，检查最后几个字节
	for i := len(payload) - 1; i >= 0 && i >= len(payload)-4; i-- {
		if utf8.RuneStart(payload[i]) {
			if !utf8.FullRune(payload[i:]) {
				// 发现末尾存在不完整字符
				lastSafe = i
				l.remainder = make([]byte, len(payload)-i)
				copy(l.remainder, payload[i:])
			}
			break
		}
	}

	// 如果整个负载都不完整且不超过一个 UTF-8 字符的最大长度，
	// 则全部保留到下次调用
	if lastSafe == 0 && len(l.remainder) > 0 {
		return originalInputLen, nil
	}

	// 3. 仅将完整的部分转换为 UTF-8
	text := utils.ToUTF8(payload[:lastSafe])
	data := []byte(text)

	// 4. 写入文件缓冲区
	_, err = l.writer.Write(data)
	if err != nil {
		return 0, err
	}

	// 5. 广播给所有订阅者
	if len(l.subscribers) > 0 {
		for _, ch := range l.subscribers {
			select {
			case ch <- data:
			default:
				// 如果订阅者处理太慢，丢弃消息以避免阻塞写入
			}
		}
	}

	return originalInputLen, nil
}

// Subscribe 返回一个实时接收日志块的通道
func (l *TinyLog) Subscribe() chan []byte {
	l.mu.Lock()
	defer l.mu.Unlock()

	ch := make(chan []byte, 100) // Buffer to handle bursts
	l.subscribers = append(l.subscribers, ch)
	return ch
}

// Unsubscribe 移除订阅者
func (l *TinyLog) Unsubscribe(ch chan []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, sub := range l.subscribers {
		if sub == ch {
			l.subscribers = append(l.subscribers[:i], l.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// Close 完成写入，关闭文件并注销实例
func (l *TinyLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	// 处理剩余的字节
	if len(l.remainder) > 0 {
		text := utils.ToUTF8(l.remainder)
		data := []byte(text)
		_, _ = l.writer.Write(data)

		// 通知订阅者最后一部分内容
		for _, ch := range l.subscribers {
			select {
			case ch <- data:
			default:
			}
		}
		l.remainder = nil
	}

	// 将缓冲区刷新到文件
	if err := l.writer.Flush(); err != nil {
		return err
	}

	// 关闭所有订阅者通道
	for _, ch := range l.subscribers {
		close(ch)
	}
	l.subscribers = nil

	l.closed = true
	globalTinyLogManager.Unregister(l.LogID)
	return l.file.Close()
}

// CompressAndCleanup 读取临时文件，进行压缩处理，返回结果并删除临时文件
func (l *TinyLog) CompressAndCleanup() (string, error) {
	// Ensure closed
	if !l.closed {
		l.Close()
	}

	// 打开临时文件进行读取
	f, err := os.Open(l.path)
	if err != nil {
		return "", err
	}
	defer func() {
		f.Close()
		os.Remove(l.path) // Cleanup
	}()

	// 创建压缩输出缓冲区
	var buf bytes.Buffer
	b64Writer := base64.NewEncoder(base64.StdEncoding, &buf)
	zlibWriter := zlib.NewWriter(b64Writer)

	// 流处理: 文件 -> Zlib -> Base64 -> 缓冲区
	if _, err := io.Copy(zlibWriter, f); err != nil {
		return "", err
	}

	// 关闭写入器以刷新数据
	if err := zlibWriter.Close(); err != nil {
		return "", err
	}
	if err := b64Writer.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ReadLastLines 返回日志的最后 n 行
func (l *TinyLog) ReadLastLines(n int) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 刷新写入器以确保磁盘上的文件是最新的
	_ = l.writer.Flush()

	stat, err := os.Stat(l.path)
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	var limit int64 = 65536 // 预览限制：最大 64KB
	if size < limit {
		limit = size
	}
	offset := size - limit

	data := make([]byte, limit)
	f, err := os.Open(l.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return nil, err
	}

	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) > n+1 {
		return bytes.Join(lines[len(lines)-n-1:], []byte{'\n'}), nil
	}
	return data, nil
}

// GetPath 返回临时文件路径
func (l *TinyLog) GetPath() string {
	return l.path
}
