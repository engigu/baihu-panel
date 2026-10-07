//go:build !windows

package memopt

import (
	"runtime"
	"runtime/debug"
)

// trimPlatformWorkingSet 非 Windows 系统（Linux/macOS）通过双轮 GC 与显式 FreeOSMemory 强制退还物理内存页
func trimPlatformWorkingSet() {
	// 执行第二轮 GC：第一轮 GC 清理未引用对象并唤醒 finalizer，第二轮彻底回收残留的元数据与关联内存
	runtime.GC()
	// 显式通知操作系统内核释放未占用的堆空间物理内存页 (Decommit / MADV_DONTNEED)
	debug.FreeOSMemory()
}
