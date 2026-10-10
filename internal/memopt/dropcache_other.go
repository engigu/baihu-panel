//go:build !linux

package memopt

// DropCache 在非 Linux 平台下为空实现
func DropCache(paths ...string) (int, error) {
	return 0, nil
}
