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

		// 目录受控遍历：深度覆盖至 6 层以完整纳入 mise 等环境目录 (如 installs/node/23.11.1/bin/node 为第 4 层)，
		// 并自动跳过 node_modules / .git / include / share / doc 等臃肿/无关目录
		walkDirLimited(p, 0, 6, func(filePath string) {
			if dropFileCache(filePath) {
				totalFiles++
			}
		})
	}
	return totalFiles, nil
}

func walkDirLimited(dir string, currentDepth, maxDepth int, onFile func(string)) {
	if currentDepth > maxDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			// 跳过已知巨型依赖、头文件、文档或无意义缓存子目录，避免磁盘 IO 与 Slab 膨胀
			if name == "node_modules" || name == ".git" || name == ".cache" || name == "tmp" ||
				name == "include" || name == "share" || name == "doc" || name == "man" {
				continue
			}
			walkDirLimited(filepath.Join(dir, name), currentDepth+1, maxDepth, onFile)
		} else if entry.Type().IsRegular() {
			onFile(filepath.Join(dir, name))
		}
	}
}

func dropFileCache(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	return unix.Fadvise(int(f.Fd()), 0, 0, unix.FADV_DONTNEED) == nil
}
