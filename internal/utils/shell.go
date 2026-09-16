package utils

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/engigu/baihu-panel/internal/windows"
)

var (
	defaultShell string
	defaultArgs  []string
	shellOnce    sync.Once
)

// GetShell 返回当前操作系统的 shell 和参数
func GetShell() (shell string, args []string) {
	shellOnce.Do(func() {
		if windows.IsWindows() {
			if path, ok := windows.FindPwsh(); ok {
				defaultShell = path
				defaultArgs = []string{}
				return
			}
			panic("Windows 系统上需要安装 PowerShell 7+ (pwsh.exe)，但在环境变量 PATH 中未找到，请先安装它。")
		}

		// 1. 优先在 PATH 中查找 bash
		if path, err := exec.LookPath("bash"); err == nil {
			defaultShell = path
			defaultArgs = []string{}
			return
		}

		// 2. 其次使用环境变量中的 SHELL
		if envShell := os.Getenv("SHELL"); envShell != "" {
			if _, err := os.Stat(envShell); err == nil {
				defaultShell = envShell
				defaultArgs = []string{}
				return
			}
		}

		// 3. 尝试在 PATH 中查找 zsh 或 sh
		for _, s := range []string{"zsh", "sh"} {
			if path, err := exec.LookPath(s); err == nil {
				defaultShell = path
				defaultArgs = []string{}
				return
			}
		}

		// 4. 最后回退到硬编码路径
		shells := []string{"/usr/bin/bash", "/bin/bash", "/usr/bin/sh", "/bin/sh"}
		for _, sh := range shells {
			if _, err := os.Stat(sh); err == nil {
				defaultShell = sh
				defaultArgs = []string{}
				return
			}
		}

		defaultShell = "sh"
		defaultArgs = []string{}
	})

	return defaultShell, defaultArgs
}

// GetShellCommand 返回执行命令的 shell 和参数
func GetShellCommand(command string) (shell string, args []string) {
	shell, _ = GetShell()
	if windows.IsWindows() {
		return shell, []string{"-NoProfile", "-NonInteractive", "-Command", command}
	}
	return shell, []string{"-c", command}
}

// NewShellCmd 创建一个交互式 shell 命令
func NewShellCmd() *exec.Cmd {
	shell, _ := GetShell()
	if windows.IsWindows() {
		return windows.GetWindowsShellCmd(shell)
	}
	// Unix 系统使用 -i 启用交互模式
	return exec.Command(shell, "-i")
}

// NewShellCommandCmd 创建一个执行指定命令的 shell 命令
func NewShellCommandCmd(command string) *exec.Cmd {
	shell, args := GetShellCommand(command)
	return exec.Command(shell, args...)
}

// WrapFailFastScript 为脚本注入严格的过程错误即刻终止（Fail-Fast）逻辑：
// - Windows (PowerShell 7+ / pwsh): 设置 $ErrorActionPreference = 'Stop' 和 $PSNativeCommandUseErrorActionPreference = $true
//   使得任何命令（包括外部原生 exe 如 dotnet/git/mise）退出码非 0 时立即中止后续执行并退出非 0 状态码。
// - Unix (bash/sh): 设置 set -eo pipefail，任何中间命令或管道失败立即中止。
func WrapFailFastScript(script string) string {
	script = strings.TrimSpace(script)
	if script == "" {
		return ""
	}
	if windows.IsWindows() {
		return "$ErrorActionPreference = 'Stop'\n$PSNativeCommandUseErrorActionPreference = $true\n" + script
	}
	return "set -eo pipefail\n" + script
}

// NewFailFastShellCommandCmd 创建一个带有过程错误即刻终止特性的 Shell 执行命令
func NewFailFastShellCommandCmd(command string) *exec.Cmd {
	return NewShellCommandCmd(WrapFailFastScript(command))
}


// QuotePath 转义并包裹路径，防止 Shell 注入
func QuotePath(path string) string {
	if path == "" {
		return "''"
	}
	if runtime.GOOS == "windows" {
		// 在 PowerShell / pwsh 中，使用单引号包裹路径。
		// 单引号内的单引号通过连续两个单引号转义，例如: ' -> ''
		return "'" + strings.ReplaceAll(path, "'", "''") + "'"
	}
	// 在 Unix-like 系统中，单引号包裹是最安全的
	// 需要将路径中的 ' 替换为 '\'' (结束当前引号，转义一个单引号，重新开启引号)
	return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
}
