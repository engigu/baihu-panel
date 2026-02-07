package tasks

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
	"os"
	"sync"

	"github.com/engigu/baihu-panel/internal/utils"
)

var (
	// globalTinyLogManager keeps track of all active TinyLog instances
	globalTinyLogManager = &TinyLogManager{
		logs: make(map[uint]*TinyLog),
	}
)

type TinyLogManager struct {
	mu   sync.RWMutex
	logs map[uint]*TinyLog
}

func (m *TinyLogManager) Register(log *TinyLog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs[log.LogID] = log
}

func (m *TinyLogManager) Unregister(logID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.logs, logID)
}

func (m *TinyLogManager) Get(logID uint) *TinyLog {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logs[logID]
}

// GetActiveLog returns an active TinyLog by its ID
func GetActiveLog(logID uint) *TinyLog {
	return globalTinyLogManager.Get(logID)
}

// TinyLog is a high-performance, low-memory log collector
type TinyLog struct {
	LogID       uint
	mu          sync.RWMutex
	file        *os.File
	path        string
	writer      *bufio.Writer
	subscribers []chan []byte
	closed      bool
}

// NewTinyLog creates a new TinyLog instance backed by a temporary file and registers it
func NewTinyLog(logID uint) (*TinyLog, error) {
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

// Write implements io.Writer
func (l *TinyLog) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return 0, os.ErrClosed
	}

	// 1. Convert to UTF-8 if necessary (common on Windows)
	text := utils.ToUTF8(p)
	data := []byte(text)

	// 2. Write to file buffer
	_, err = l.writer.Write(data)
	if err != nil {
		return 0, err
	}

	// 3. Broadcast to subscribers
	if len(l.subscribers) > 0 {
		for _, ch := range l.subscribers {
			select {
			case ch <- data:
			default:
				// Drop message if subscriber is too slow to avoid blocking writer
			}
		}
	}

	return len(p), nil
}

// Subscribe returns a channel that receives log chunks in real-time
func (l *TinyLog) Subscribe() chan []byte {
	l.mu.Lock()
	defer l.mu.Unlock()

	ch := make(chan []byte, 100) // Buffer to handle bursts
	l.subscribers = append(l.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber
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

// Close finishes writing and closes the file, and unregisters itself
func (l *TinyLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	// Flush buffer to file
	if err := l.writer.Flush(); err != nil {
		return err
	}

	// Close all subscribers
	for _, ch := range l.subscribers {
		close(ch)
	}
	l.subscribers = nil

	l.closed = true
	globalTinyLogManager.Unregister(l.LogID)
	return l.file.Close()
}

// CompressAndCleanup reads the temporary file, compresses it, returns the result, and removes the file
func (l *TinyLog) CompressAndCleanup() (string, error) {
	// Ensure closed
	if !l.closed {
		l.Close()
	}

	// Open temp file for reading
	f, err := os.Open(l.path)
	if err != nil {
		return "", err
	}
	defer func() {
		f.Close()
		os.Remove(l.path) // Cleanup
	}()

	// Create buffer for compressed output
	var buf bytes.Buffer
	b64Writer := base64.NewEncoder(base64.StdEncoding, &buf)
	zlibWriter := zlib.NewWriter(b64Writer)

	// Stream: File -> Zlib -> Base64 -> Buffer
	if _, err := io.Copy(zlibWriter, f); err != nil {
		return "", err
	}

	// Close generic writers to flush data
	if err := zlibWriter.Close(); err != nil {
		return "", err
	}
	if err := b64Writer.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ReadLastLines returns the last n lines of the log
func (l *TinyLog) ReadLastLines(n int) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Flush writer to ensure file on disk is up to date
	_ = l.writer.Flush()

	stat, err := os.Stat(l.path)
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	var limit int64 = 65536 // Max 64KB for "last 100 lines" preview
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

// GetPath returns the temporary file path
func (l *TinyLog) GetPath() string {
	return l.path
}
