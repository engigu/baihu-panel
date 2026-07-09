package tasks

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"unicode/utf8"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/utils"
)

const (
	// maxLogBufferLen 定义了没有换行符时的最大缓冲长度 (4KB)
	maxLogBufferLen = 4096
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
	remainder   []byte   // Leftover bytes from previous write (partial lines)
	masks       []string // Secrets to mask
	closed      bool
}

// NewTinyLog 创建一个新的 TinyLog 实例（基于临时文件存储）并注册它，支持将配置的 masks 替换为 ********
func NewTinyLog(logID string, masks []string) (*TinyLog, error) {
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
		masks:       masks,
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

	originalInputLen := len(p)
	var payload []byte
	if len(l.remainder) > 0 {
		// 为了防止 p 和 l.remainder 底层数组有重叠或不可预期的修改，这里分配新内存
		payload = make([]byte, len(l.remainder)+len(p))
		copy(payload, l.remainder)
		copy(payload[len(l.remainder):], p)
		l.remainder = nil
	} else {
		payload = p
	}

	// 1. 寻找最后一个换行符 (\n 或 \r)
	lastLineBreak := bytes.LastIndexAny(payload, "\n\r")
	var completeBytes []byte
	var remainder []byte

	if lastLineBreak != -1 {
		// 2. 提取出完整的行
		completeBytes = payload[:lastLineBreak+1]
		remainder = payload[lastLineBreak+1:]
	} else {
		// 3. 没有换行符，且如果长度超过最大缓冲，强制截断并输出，防止内存无限制增长
		if len(payload) > maxLogBufferLen {
			// 寻找最后一个完整的 UTF-8 字符边界，避免乱码
			lastSafe := maxLogBufferLen
			for i := maxLogBufferLen; i > 0 && i > maxLogBufferLen-4; i-- {
				if utf8.RuneStart(payload[i-1]) {
					if !utf8.FullRune(payload[i-1 : maxLogBufferLen]) {
						lastSafe = i - 1
					}
					break
				}
			}
			completeBytes = payload[:lastSafe]
			remainder = payload[lastSafe:]
		} else {
			// 保留当前所有内容到下一轮 (必须 copy，因为 payload 底层可能是 io.Copy 的复用 buf)
			l.remainder = make([]byte, len(payload))
			copy(l.remainder, payload)
			return originalInputLen, nil
		}
	}

	// 4. 将剩余部分保存 (必须 copy，防止后续 Read 覆盖底层数组)
	if len(remainder) > 0 {
		l.remainder = make([]byte, len(remainder))
		copy(l.remainder, remainder)
	} else {
		l.remainder = nil
	}

	// 5. 将完整行转换为 UTF-8 并脱敏
	text := utils.MaskSecrets(utils.ToUTF8(completeBytes), l.masks)
	outData := []byte(text)

	// 6. 输出安全部分
	_, err = l.writer.Write(outData)
	if err != nil {
		return 0, err
	}

	// 6. 广播给所有订阅者
	if len(l.subscribers) > 0 {
		for _, ch := range l.subscribers {
			select {
			case ch <- outData:
			default:
				// 如果订阅者处理太慢，丢弃消息以避免阻塞写入
			}
		}
	}

	return originalInputLen, nil
}

// WriteString 方便地写入字符串
func (l *TinyLog) WriteString(s string) (n int, err error) {
	return l.Write([]byte(s))
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
		text := utils.MaskSecrets(utils.ToUTF8(l.remainder), l.masks)
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

	// 获取文件大小
	stat, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := stat.Size()

	// 如果日志极短，免去压缩和 Base64 编码，直接以 raw: 明文形式返回
	if size <= 128 {
		content, err := io.ReadAll(f)
		if err != nil {
			return "", err
		}
		return "raw:" + string(content), nil
	}

	// 创建压缩输出缓冲区
	var buf bytes.Buffer
	b64Writer := base64.NewEncoder(base64.StdEncoding, &buf)

	// 使用 Pool 优化压缩
	zw := utils.GetZstdWriter(b64Writer)
	defer utils.PutZstdWriter(zw)

	maxSize := int64(constant.MaxLogSize)
	if maxSize < 1024*1024 {
		maxSize = 1024 * 1024
	}

	var readStart int64 = 0
	if size > maxSize {
		readStart = size - maxSize
		// 写入一条截断提示
		truncatedMsg := fmt.Sprintf("\n\n[System] 日志过长，已自动截断，仅保留末尾 %d MB...\n\n", maxSize/1024/1024)
		if _, err := zw.Write([]byte(truncatedMsg)); err != nil {
			return "", err
		}
	}

	if readStart > 0 {
		if _, err := f.Seek(readStart, io.SeekStart); err != nil {
			return "", err
		}
	}

	// 流处理: 文件 -> Zstd -> Base64 -> 缓冲区
	if _, err := io.Copy(zw, f); err != nil {
		return "", err
	}

	// 关闭写入器以刷新数据
	if err := zw.Close(); err != nil {
		return "", err
	}
	if err := b64Writer.Close(); err != nil {
		return "", err
	}

	return "zstd:" + buf.String(), nil
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

// CleanupOrphanedTinyLogs 启动时清理残留的临时日志文件
func CleanupOrphanedTinyLogs() {
	tmpDir := os.TempDir()
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 9 && file.Name()[:9] == "task_log_" && filepath.Ext(file.Name()) == ".log" {
			os.Remove(filepath.Join(tmpDir, file.Name()))
			count++
		}
	}
	if count > 0 {
		logger.Infof("[System] 清理了 %d 个残留的任务日志临时文件", count)
	}
}
