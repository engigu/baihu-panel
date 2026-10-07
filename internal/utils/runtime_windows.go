//go:build windows

package utils

import (
	"syscall"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procSetProcessWSSize  = kernel32.NewProc("SetProcessWorkingSetSize")
	procGetCurrentProcess = kernel32.NewProc("GetCurrentProcess")
)

// trimWorkingSet 强制 Windows 操作系统清空并收缩当前进程的物理常驻工作集 (Working Set / RSS)
func trimWorkingSet() {
	if procSetProcessWSSize.Find() == nil && procGetCurrentProcess.Find() == nil {
		hProcess, _, _ := procGetCurrentProcess.Call()
		if hProcess != 0 {
			// 传入 (HANDLE, -1, -1) 即通知系统裁减进程工作集，将已释放的物理页立即归还 OS
			_, _, _ = procSetProcessWSSize.Call(hProcess, ^uintptr(0), ^uintptr(0))
		}
	}
}
