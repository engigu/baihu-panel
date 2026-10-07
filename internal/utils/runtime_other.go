//go:build !windows

package utils

// trimWorkingSet 非 Windows 系统由内核自行回收与管理物理常驻内存
func trimWorkingSet() {
}
