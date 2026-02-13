package utils

import (
	"strings"
)

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
