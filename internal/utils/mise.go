package utils

import (
	"os/exec"
	"strings"
	"sync"
)

var nodePathCache sync.Map

// GetMiseNodePath 获取指定版本的 node 全局包路径，使用内存缓存避免重复获取
func GetMiseNodePath(version string) string {
	if version == "" {
		version = "latest"
	}

	if val, ok := nodePathCache.Load(version); ok {
		return val.(string)
	}

	cmd := exec.Command("mise", "where", "node@"+version)
	out, err := cmd.CombinedOutput()
	if err == nil {
		nodeDir := strings.TrimSpace(string(out))
		if nodeDir != "" {
			// 采用双路径策略：lib/node_modules 是标准路径，lib 是某些环境（如 mise Docker）下的特殊路径
			// 通过冒号分隔，让 Node.js 按顺序搜索，保证最大兼容性
			nodePath := nodeDir + "/lib/node_modules:" + nodeDir + "/lib"
			nodePathCache.Store(version, nodePath)
			return nodePath
		}
	}

	return ""
}

// InjectNodePath 检查语言环境中是否有 node，如果有则自动获取并注入 NODE_PATH 到环境变量切片中
func InjectNodePath(envs *[]string, languages []map[string]string) {
	for _, lang := range languages {
		if lang["name"] == "node" {
			if nodePath := GetMiseNodePath(lang["version"]); nodePath != "" {
				*envs = append(*envs, "NODE_PATH="+nodePath)
			}
			break
		}
	}
}

// BuildMiseCommand 构建多语言 mise 执行命令 (字符串形式)
func BuildMiseCommand(command string, languages []map[string]string) string {
	if len(languages) == 0 {
		return command
	}

	var builder strings.Builder
	builder.WriteString("mise exec")

	for _, lang := range languages {
		name := lang["name"]
		version := lang["version"]
		if name == "" {
			continue
		}
		if version == "" {
			version = "latest"
		}
		builder.WriteString(" " + name + "@" + version)
	}

	builder.WriteString(" -- " + command)
	return builder.String()
}

// BuildMiseCommandArgs 构建多语言 mise 执行命令 (参数列表形式)
func BuildMiseCommandArgs(cmdArgs []string, languages []map[string]string) []string {
	if len(languages) == 0 {
		return cmdArgs
	}

	args := []string{"mise", "exec"}
	for _, lang := range languages {
		name := lang["name"]
		version := lang["version"]
		if name == "" {
			continue
		}
		if version == "" {
			version = "latest"
		}
		args = append(args, name+"@"+version)
	}
	args = append(args, "--")
	args = append(args, cmdArgs...)
	return args
}

// BuildMiseCommandSimple 构建单个语言的 mise 执行命令
func BuildMiseCommandSimple(command string, language, version string) string {
	if language == "" {
		return command
	}
	spec := language
	if version != "" {
		spec += "@" + version
	}
	return "mise exec " + spec + " -- " + command
}

// BuildMiseCommandArgsSimple 构建单个语言的 mise 执行命令 (参数列表形式)
func BuildMiseCommandArgsSimple(cmdArgs []string, language, version string) []string {
	if language == "" {
		return cmdArgs
	}
	spec := language
	if version != "" {
		spec += "@" + version
	}
	return append([]string{"mise", "exec", spec, "--"}, cmdArgs...)
}
