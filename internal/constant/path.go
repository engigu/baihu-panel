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
	ScriptsWorkDir = ResolveScriptsDir(rootDir)
	_ = os.Setenv("BH_SCRIPTS_DIR", ScriptsWorkDir)
}

// ResolveScriptsDir 解析脚本工作目录（优先读取 BH_SCRIPTS_DIR 环境变量，若无则基于 rootDir/data/scripts 兜底）
func ResolveScriptsDir(rootDir string) string {
	if scriptsDir := os.Getenv("BH_SCRIPTS_DIR"); scriptsDir != "" {
		if !strings.Contains(scriptsDir, "go-build") && !strings.Contains(scriptsDir, "Temp") {
			if absScriptsDir, err := filepath.Abs(scriptsDir); err == nil {
				return filepath.Clean(absScriptsDir)
			}
			return filepath.Clean(scriptsDir)
		}
	}

	if rootDir == "" {
		rootDir = ResolveAppRootDir()
	}
	defaultScriptsDir := filepath.Join(rootDir, "data", "scripts")
	if abs, err := filepath.Abs(defaultScriptsDir); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(defaultScriptsDir)
}

// NormalizeScriptPath 将任意路径归一化为以 $SCRIPTS_DIR$ 开头的逻辑路径
// 逻辑：如果路径在当前系统的 ScriptsWorkDir (data/scripts) 目录下，归一化为 $SCRIPTS_DIR$/xxx；
// 如果是相对路径（如 apps/xxx/bin），自动补全 $SCRIPTS_DIR$ 占位符前缀归一化存库；
// 如果是非 scriptsDir 目录下的外部绝对路径，原样保留。
func NormalizeScriptPath(rawPath string) string {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return ScriptsDirPlaceholder
	}

	// 如果本身已经是 $SCRIPTS_DIR$ 开头，统一斜杠后返回
	if strings.HasPrefix(rawPath, ScriptsDirPlaceholder) {
		return filepath.ToSlash(filepath.Clean(rawPath))
	}

	cleanRaw := filepath.Clean(rawPath)
	cleanScripts := filepath.Clean(ScriptsWorkDir)

	// 不区分大小写判断前缀（兼顾 Windows）
	if strings.HasPrefix(strings.ToLower(cleanRaw), strings.ToLower(cleanScripts)) {
		rel, err := filepath.Rel(cleanScripts, cleanRaw)
		if err == nil && rel != "." && rel != "" {
			return filepath.ToSlash(filepath.Join(ScriptsDirPlaceholder, rel))
		}
		return ScriptsDirPlaceholder
	}

	// 如果传入的是相对路径（如 apps/xxx/bin），自动补全 $SCRIPTS_DIR$ 占位符前缀归一化存库
	if !filepath.IsAbs(cleanRaw) {
		return filepath.ToSlash(filepath.Join(ScriptsDirPlaceholder, cleanRaw))
	}

	// 不在脚本目录下的外部绝对路径，原样保留（转为标准斜杠）
	return filepath.ToSlash(cleanRaw)
}

// ResolveScriptPath 将以 $SCRIPTS_DIR$ 开头的逻辑路径还原为当前系统的真实物理绝对路径
func ResolveScriptPath(logicPath string) string {
	logicPath = strings.TrimSpace(logicPath)
	if logicPath == "" {
		return ""
	}
	if logicPath == ScriptsDirPlaceholder {
		return ScriptsWorkDir
	}

	if strings.HasPrefix(logicPath, ScriptsDirPlaceholder) {
		rel := strings.TrimPrefix(logicPath, ScriptsDirPlaceholder)
		rel = strings.TrimPrefix(rel, "/")
		rel = strings.TrimPrefix(rel, "\\")
		if rel == "" {
			return ScriptsWorkDir
		}
		return filepath.Clean(filepath.Join(ScriptsWorkDir, rel))
	}

	// 若不含 $SCRIPTS_DIR$ 占位符且为相对路径（非绝对路径），自动基于脚本根目录拼接还原
	if !filepath.IsAbs(logicPath) {
		return filepath.Clean(filepath.Join(ScriptsWorkDir, logicPath))
	}

	// 若不含 $SCRIPTS_DIR$ 占位符，原样返回
	return logicPath
}

// ResolveCommand 将命令字符串中的 $SCRIPTS_DIR$ 占位符纯文本替换为当前系统的真实物理绝对路径
func ResolveCommand(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	if strings.Contains(command, ScriptsDirPlaceholder) {
		return strings.ReplaceAll(command, ScriptsDirPlaceholder, ScriptsWorkDir)
	}
	return command
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
