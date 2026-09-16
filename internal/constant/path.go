package constant

import (
	"os"
	"path/filepath"
	"strings"
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
	// 将最终计算得出的全局根目录写入环境变量，确保所有子命令和任务调度共享
	_ = os.Setenv("BH_ROOT_DIR", rootDir)

	ConfigPath = filepath.Clean(filepath.Join(rootDir, "configs", "config.ini"))
	DataDir = filepath.Clean(filepath.Join(rootDir, "data"))
	DefaultDBPath = filepath.Clean(filepath.Join(rootDir, "data", "baihu.db"))
	WebDistDir = filepath.Clean(filepath.Join(rootDir, "web", "dist"))
	ScriptsWorkDir = filepath.Clean(filepath.Join(rootDir, "data", "scripts"))
	_ = os.Setenv("BH_SCRIPTS_DIR", ScriptsWorkDir)
}

// ResolveAppRootDir 获取应用程序的绝对根目录路径。
func ResolveAppRootDir() string {
	// 0. 优先检查显式传递的 BH_ROOT_DIR 环境变量
	if envRoot := os.Getenv("BH_ROOT_DIR"); envRoot != "" {
		if abs, err := filepath.Abs(envRoot); err == nil {
			return filepath.Clean(abs)
		}
		return filepath.Clean(envRoot)
	}
	// 1. 检查当前工作目录（CWD）及其上级目录
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			if _, err := os.Stat(filepath.Join(dir, "configs", "config.ini")); err == nil {
				if abs, err := filepath.Abs(dir); err == nil {
					return abs
				}
				return dir
			}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				if abs, err := filepath.Abs(dir); err == nil {
					return abs
				}
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 2. 检查当前可执行文件路径及其上级目录（如果是 go run 的临时编译目录则忽略）
	if exe, err := os.Executable(); err == nil {
		cleanExe := filepath.Clean(exe)
		if !strings.Contains(cleanExe, "go-build") && !strings.Contains(cleanExe, "Temp") && !strings.Contains(cleanExe, "tmp") {
			dir := filepath.Dir(cleanExe)
			for {
				if _, err := os.Stat(filepath.Join(dir, "configs", "config.ini")); err == nil {
					if abs, err := filepath.Abs(dir); err == nil {
						return abs
					}
					return dir
				}
				if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
					if abs, err := filepath.Abs(dir); err == nil {
						return abs
					}
					return dir
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
		}
	}

	// 3. 兜底回退到当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		if abs, err := filepath.Abs(cwd); err == nil {
			return abs
		}
		return cwd
	}
	return "."
}
