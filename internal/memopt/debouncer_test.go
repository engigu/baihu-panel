package memopt

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDirDebouncer_TrailingEdge(t *testing.T) {
	d := NewDirDebouncer()

	callCount := 0
	var mu sync.Mutex
	done := make(chan string, 1)

	// 模拟任务 A 在 0ms 触发
	d.Debounce("/data/scripts/app1", 80*time.Millisecond, func(dir string) {
		mu.Lock()
		callCount++
		mu.Unlock()
		done <- dir
	})

	// 模拟任务 B 在 30ms 触发（在 80ms 窗口内，应该顺延取消前一个并重置）
	time.Sleep(30 * time.Millisecond)
	d.Debounce("/data/scripts/app1", 80*time.Millisecond, func(dir string) {
		mu.Lock()
		callCount++
		mu.Unlock()
		done <- dir
	})

	// 等待定时器触发
	select {
	case dir := <-done:
		if dir != filepath.Clean("/data/scripts/app1") {
			t.Errorf("期望回调路径为 %s, 得到 %s", filepath.Clean("/data/scripts/app1"), dir)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("防抖定时器超时未触发")
	}

	// 稍微等待确保不会有重复触发
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()
	if count != 1 {
		t.Errorf("期望防抖合并后仅执行 1 次, 实际执行了 %d 次", count)
	}

	// 验证 map 内存是否彻底清理
	d.mu.Lock()
	mapLen := len(d.timers)
	d.mu.Unlock()
	if mapLen != 0 {
		t.Errorf("期望定时器触发后 map 被清空, 实际残留 %d 个", mapLen)
	}
}

func TestDirDebouncer_Cancel(t *testing.T) {
	d := NewDirDebouncer()
	triggered := false

	d.Debounce("/data/scripts/app_cancel", 50*time.Millisecond, func(dir string) {
		triggered = true
	})

	// 立即取消
	d.Cancel("/data/scripts/app_cancel")

	time.Sleep(100 * time.Millisecond)

	if triggered {
		t.Error("期望已被 Cancel 的防抖任务不触发回调")
	}

	d.mu.Lock()
	mapLen := len(d.timers)
	d.mu.Unlock()
	if mapLen != 0 {
		t.Errorf("Cancel 后 map 应为空, 实际残留 %d 个", mapLen)
	}
}

func TestDirDebouncer_PanicRecover(t *testing.T) {
	d := NewDirDebouncer()

	// 触发一个故意抛出 panic 的任务
	d.Debounce("/data/scripts/app_panic", 20*time.Millisecond, func(dir string) {
		panic("测试故意抛出的模拟异常")
	})

	// 等待足够时间，确保 panic 被 recover 捕获且不导致主测试进程退出
	time.Sleep(60 * time.Millisecond)

	d.mu.Lock()
	mapLen := len(d.timers)
	d.mu.Unlock()
	if mapLen != 0 {
		t.Errorf("触发 panic 后 map 应完成清理, 实际残留 %d 个", mapLen)
	}
}
