package utils

import (
	"fmt"
)

// FormatDuration 将毫秒数格式化为紧凑现代的 DevOps 风格耗时字符串
// 规则：
// 1. < 1000ms: 直接输出毫秒 (如 450ms)
// 2. 1s ~ 59.9s: 秒级，带适度小数 (如 2.49s, 15.2s)
// 3. 1m ~ 59m59s: 分秒组合 (如 9m13s)
// 4. >= 1h: 时分秒组合 (如 1h2m3s)
func FormatDuration(ms int64) string {
	if ms <= 0 {
		return "0ms"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}

	totalSeconds := ms / 1000

	if totalSeconds < 10 {
		return fmt.Sprintf("%.2fs", float64(ms)/1000.0)
	}
	if totalSeconds < 60 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
