//go:build linux

package memopt

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// DropCache 遍历目标文件或目录，通过 posix_fadvise(FADV_DONTNEED) 释放已被读入内存的 Page Cache
func DropCache(paths ...string) (int, error) {
	totalFiles := 0
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		if !fi.IsDir() {
			if dropFileCache(p) {
				totalFiles++
			}
			continue
		}

		_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if dropFileCache(path) {
				totalFiles++
			}
			return nil
		})
	}
	return totalFiles, nil
}

func dropFileCache(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	return unix.Fadvise(int(f.Fd()), 0, 0, unix.FADV_DONTNEED) == nil
}
