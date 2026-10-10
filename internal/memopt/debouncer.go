package memopt

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/logger"
)

// DirDebouncer 目录任务结束清理防抖器（Trailing Edge 尾部合并防抖）
// 用于处理同目录下多任务高频/连续结束场景，消除重复遍历开销，由最后结束的任务在静默期触发唯一一次收尾
type DirDebouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
}

// NewDirDebouncer 创建新的目录防抖器
func NewDirDebouncer() *DirDebouncer {
	return &DirDebouncer{
		timers: make(map[string]*time.Timer),
	}
}

var defaultDirDebouncer = NewDirDebouncer()

// Cancel 显式取消指定目录正在挂起的防抖任务
func (d *DirDebouncer) Cancel(dir string) {
	if dir == "" {
		return
	}
	cleanDir := filepath.Clean(dir)

	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, ok := d.timers[cleanDir]; ok {
		timer.Stop()
		delete(d.timers, cleanDir)
	}
}

// Debounce 在指定静默延迟后执行清理；若在延迟期内有新任务结束，自动顺延重置计时器
func (d *DirDebouncer) Debounce(dir string, delay time.Duration, fn func(dir string)) {
	if dir == "" {
		return
	}
	cleanDir := filepath.Clean(dir)

	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. 若该目录已有挂起的计时器，直接取消并顺延重置
	if timer, ok := d.timers[cleanDir]; ok {
		timer.Stop()
	}

	// 2. 注册延迟执行：静默期到达后由最后一个任务执行，并自动从 map 中释放计时器句柄
	var timer *time.Timer
	timer = time.AfterFunc(delay, func() {
		// 隔离协程 Panic，防止底层异常拖垮主进程
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("[MemOpt] 目录清理防抖回调发生异常: %v", r)
			}
		}()

		d.mu.Lock()
		if cur, ok := d.timers[cleanDir]; ok && cur == timer {
			delete(d.timers, cleanDir)
			d.mu.Unlock()
			fn(cleanDir)
			return
		}
		d.mu.Unlock()
	})
	d.timers[cleanDir] = timer
}
