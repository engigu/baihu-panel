package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
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
			var nodePath string
			if runtime.GOOS == "windows" {
				// Windows 下, Mise 安装的 Node.js 全局 node_modules 通常位于安装根目录下
				nodePath = filepath.Join(nodeDir, "node_modules")
			} else {
				// 采用双路径策略：lib/node_modules 是标准路径，lib 是某些环境（如 mise Docker）下的特殊路径
				// 通过冒号分隔，让 Node.js 按顺序搜索，保证最大兼容性
				nodePath = nodeDir + "/lib/node_modules:" + nodeDir + "/lib"
			}
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

// BuildAppTaskCommand 为应用受控任务构建多语言 mise 执行命令，自动剥离旧版硬编码前缀以保证 UI 语言配置生效
func BuildAppTaskCommand(command string, languages []map[string]string) string {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return ""
	}
	if strings.HasPrefix(cmd, "mise exec") && strings.Contains(cmd, " -- ") {
		idx := strings.Index(cmd, " -- ")
		cmd = strings.TrimSpace(cmd[idx+4:])
	}
	return BuildMiseCommand(cmd, languages)
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

type miseInstalledItem struct {
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
}

// ListMiseInstalledVersions 获取指定语言已安装的所有版本列表
func ListMiseInstalledVersions(language string) ([]string, error) {
	// 执行 mise ls <language> --json 命令结构化获取版本
	cmd := exec.Command("mise", "ls", language, "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var items []miseInstalledItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("解析 mise 输出失败: %w", err)
	}

	var versions []string
	for _, item := range items {
		if item.Version != "" {
			versions = append(versions, item.Version)
		}
	}
	return versions, nil
}

// ParseMiseLanguages 将空格分隔的 mise 语言规范字符串 (例如: "go@1.22.5 node@23.11.1" 或 "python") 解析为结构化的语言属性列表
func ParseMiseLanguages(miseLangStr string) []map[string]string {
	var result []map[string]string
	miseLangStr = strings.TrimSpace(miseLangStr)
	if miseLangStr == "" {
		return result
	}

	// 严格按空白分割
	parts := strings.Fields(miseLangStr)

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		sub := strings.Split(p, "@")
		name := strings.TrimSpace(sub[0])
		ver := ""
		if len(sub) > 1 {
			ver = strings.TrimSpace(sub[1])
		}
		if name != "" {
			result = append(result, map[string]string{
				"name":    name,
				"version": ver,
			})
		}
	}
	return result
}

