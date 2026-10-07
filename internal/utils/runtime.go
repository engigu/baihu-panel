package utils

import (
	"github.com/engigu/baihu-panel/internal/memopt"
)

// InitRuntime 设置运行时内存和性能优化参数（委托给 memopt 统一处理）
func InitRuntime() {
	memopt.Init()
}

// FreeMemory 显式触发内存回收，释放物理资源给 OS（委托给 memopt 统一处理）
// 仅建议在执行了超大批量任务或处理了大型文件后调用
func FreeMemory() {
	memopt.Free()
}

