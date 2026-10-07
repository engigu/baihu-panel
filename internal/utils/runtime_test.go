package utils

import (
	"os"
	"testing"

	"github.com/shirou/gopsutil/v3/process"
)

func TestFreeMemory_ReclaimsRSS(t *testing.T) {
	getRSS := func() uint64 {
		if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
			if memInfo, err := p.MemoryInfo(); err == nil {
				return memInfo.RSS
			}
		}
		return 0
	}

	initialRSS := getRSS()

	// 模拟一次大文件同步：分配 40MB 的切片并写入数据
	allocateLargeSlice := func() []byte {
		buf := make([]byte, 40*1024*1024)
		for i := 0; i < len(buf); i += 4096 {
			buf[i] = 1
		}
		return buf
	}

	data := allocateLargeSlice()
	inflatedRSS := getRSS()
	if inflatedRSS <= initialRSS {
		t.Logf("RSS 未明显变化: initial=%d, inflated=%d", initialRSS, inflatedRSS)
	} else {
		t.Logf("分配 40MB 后的 RSS: %d 字节 (增加约 %.2f MB)", inflatedRSS, float64(inflatedRSS-initialRSS)/1024/1024)
	}

	// 丢弃大对象引用并调用 FreeMemory
	data = nil
	_ = data
	FreeMemory()

	afterRSS := getRSS()
	t.Logf("FreeMemory 后的 RSS: %d 字节 (较峰值下降约 %.2f MB)", afterRSS, float64(inflatedRSS-afterRSS)/1024/1024)

	if afterRSS >= inflatedRSS && inflatedRSS > initialRSS {
		t.Errorf("FreeMemory 未能释放物理常驻内存: inflated=%d, after=%d", inflatedRSS, afterRSS)
	}
}
