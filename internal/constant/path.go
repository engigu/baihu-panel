package constant

import (
	"os"
	"path/filepath"
)

var (
	// ConfigPath 配置文件路径
	ConfigPath string

	// DataDir 数据目录
	DataDir string

	// DefaultDBPath 默认数据库路径
	DefaultDBPath string

	// WebDistDir 前端构建目录
	WebDistDir string

	// ScriptsWorkDir 脚本工作目录
	ScriptsWorkDir string
)

func init() {
	rootDir := ResolveAppRootDir()
	ConfigPath = filepath.Clean(filepath.Join(rootDir, "configs", "config.ini"))
	DataDir = filepath.Clean(filepath.Join(rootDir, "data"))
	DefaultDBPath = filepath.Clean(filepath.Join(rootDir, "data", "baihu.db"))
	WebDistDir = filepath.Clean(filepath.Join(rootDir, "web", "dist"))
	ScriptsWorkDir = filepath.Clean(filepath.Join(rootDir, "data", "scripts"))
}

// ResolveAppRootDir 获取应用程序的绝对根目录路径。
func ResolveAppRootDir() string {
	// 1. 检查当前工作目录（CWD）及其上级目录
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			if _, err := os.Stat(filepath.Join(dir, "configs", "config.ini")); err == nil {
				return dir
			}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 2. 检查当前可执行文件路径及其上级目录
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for {
			if _, err := os.Stat(filepath.Join(dir, "configs", "config.ini")); err == nil {
				return dir
			}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 3. 兜底回退到当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}
