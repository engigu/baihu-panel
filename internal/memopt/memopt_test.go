package memopt

import (
	"testing"
)

func TestMemOpt_FreeAndMetrics(t *testing.T) {
	Init()

	initialRSS := GetRSS()

	// 模拟一次大内存占用
	buf := make([]byte, 30*1024*1024)
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = 1
	}

	inflatedRSS := GetRSS()
	t.Logf("内存分配后 RSS: %d, 较初始增长约 %.2f MB", inflatedRSS, float64(inflatedRSS-initialRSS)/1024/1024)

	// 释放引用并执行 Checkpoint
	buf = nil
	_ = buf
	Checkpoint("test_phase")

	afterRSS := GetRSS()
	t.Logf("Checkpoint 释放后 RSS: %d, 较峰值下降约 %.2f MB", afterRSS, float64(inflatedRSS-afterRSS)/1024/1024)

	metrics := GetMetrics()
	if metrics.NumGoroutine <= 0 {
		t.Errorf("NumGoroutine 异常: %d", metrics.NumGoroutine)
	}

	formatted := FormatBytes(1024 * 1024 * 5)
	if formatted != "5.0 MB" {
		t.Errorf("FormatBytes 结果不符预期: %s", formatted)
	}
}
