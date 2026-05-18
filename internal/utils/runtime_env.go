package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
)

// BuildRuntimeProcessEnv 构造 Baihu 内部可信子进程需要继承的运行时环境变量。
// 仅包含 Baihu 自己的路径/数据库配置，不包含用户任务环境变量。
func BuildRuntimeProcessEnv() []string {
	envs := make([]string, 0, 11)

	configPath := constant.ConfigPath
	if absConfig, err := filepath.Abs(constant.ConfigPath); err == nil {
		configPath = absConfig
	}
	envs = append(envs, formatEnvVar("BH_CONFIG_PATH", configPath))

	if scriptsDir := ResolveAbsScriptsDir(); strings.TrimSpace(scriptsDir) != "" {
		envs = append(envs, formatEnvVar("BH_SCRIPTS_DIR", scriptsDir))
	}

	appendEnvIfSet(&envs, "BH_DB_TYPE", constant.RuntimeDBType)
	appendEnvIfSet(&envs, "BH_DB_HOST", constant.RuntimeDBHost)
	if constant.RuntimeDBPort > 0 {
		envs = append(envs, formatEnvVar("BH_DB_PORT", fmt.Sprintf("%d", constant.RuntimeDBPort)))
	}
	appendEnvIfSet(&envs, "BH_DB_USER", constant.RuntimeDBUser)
	appendEnvIfSet(&envs, "BH_DB_PASSWORD", constant.RuntimeDBPassword)
	appendEnvIfSet(&envs, "BH_DB_NAME", constant.RuntimeDBName)
	appendEnvIfSet(&envs, "BH_DB_PATH", constant.RuntimeDBPath)
	appendEnvIfSet(&envs, "BH_DB_DSN", constant.RuntimeDBDSN)
	appendEnvIfSet(&envs, "BH_DB_TABLE_PREFIX", constant.RuntimeDBTablePrefix)

	return envs
}

// GetSystemSecrets 返回当前运行时的所有系统级敏感机密（如数据库账号密码等）
func GetSystemSecrets() []string {
	secrets := make([]string, 0, 8)
	addIfNotEmpty := func(s string) {
		if s != "" {
			secrets = append(secrets, s)
		}
	}

	addIfNotEmpty(constant.RuntimeDBPassword)
	addIfNotEmpty(constant.RuntimeDBUser)
	addIfNotEmpty(constant.RuntimeDBHost)
	addIfNotEmpty(constant.RuntimeDBName)
	addIfNotEmpty(constant.RuntimeDBPath)
	addIfNotEmpty(constant.RuntimeDBDSN)

	return secrets
}

// BuildShellEnvPrefix 将 KEY=VALUE 环境变量切片转换为 shell 前缀。
func BuildShellEnvPrefix(envs []string) string {
	parts := make([]string, 0, len(envs))
	for _, env := range envs {
		key, value, ok := strings.Cut(env, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		parts = append(parts, ShellEnvAssignment(key, value))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ") + " "
}

// ShellEnvAssignment 生成 shell 可安全使用的 KEY='VALUE' 赋值片段。
func ShellEnvAssignment(key, value string) string {
	return key + "='" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// ResolveAbsScriptsDir 解析 Baihu 运行时脚本目录的绝对路径。
func ResolveAbsScriptsDir() string {
	if scriptsDir := os.Getenv("BH_SCRIPTS_DIR"); scriptsDir != "" {
		if filepath.IsAbs(scriptsDir) {
			return filepath.Clean(scriptsDir)
		}
		if absScriptsDir, err := filepath.Abs(scriptsDir); err == nil {
			return absScriptsDir
		}
		return filepath.Clean(scriptsDir)
	}

	return constant.ScriptsWorkDir
}

func appendEnvIfSet(envs *[]string, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	*envs = append(*envs, formatEnvVar(key, value))
}

func formatEnvVar(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}
