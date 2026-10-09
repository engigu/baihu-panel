//go:build linux

package memopt

import (
	"os"
	"path/filepath"

	"github.com/engigu/baihu-panel/internal/utils"
	"golang.org/x/sys/unix"
)

// DropCache 遍历目标文件或目录，通过 posix_fadvise(FADV_DONTNEED) 释放已被读入内存的 Page Cache
// 严格限制：仅在 Docker 容器内部运行时才会真正执行，宿主机/物理机直接安全跳过
func DropCache(paths ...string) (int, error) {
	if !utils.IsRunningInDocker() {
		return 0, nil
	}

	totalFiles := 0
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		if !fi.IsDir() {
			if fi.Mode().IsRegular() && dropFileCache(p) {
				totalFiles++
			}
			continue
		}

		_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || !info.Mode().IsRegular() {
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
