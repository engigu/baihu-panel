package windows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
)

// FindPwsh 尝试查找系统中的 pwsh.exe。
// 优先在 PATH 中查找；若找不到，则自动扫描 Windows 上常用的 PowerShell 7 默认安装目录，
// 并在成功找到后将其所在目录动态追加至当前进程的 PATH 环境变量中，以确保后续调用顺利。
func FindPwsh() (string, bool) {
	if !IsWindows() {
		return "", false
	}
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path, true
	}

	candidates := []string{
		`C:\Program Files\PowerShell\7\pwsh.exe`,
		`C:\Program Files (x86)\PowerShell\7\pwsh.exe`,
		`C:\Program Files\PowerShell\7-preview\pwsh.exe`,
	}

	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		candidates = append(candidates,
			filepath.Join(userProfile, `scoop\apps\powershell\current\pwsh.exe`),
			filepath.Join(userProfile, `scoop\shims\pwsh.exe`),
			filepath.Join(userProfile, `.mise\shims\pwsh.exe`),
		)
	}

	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidates = append(candidates,
			filepath.Join(localAppData, `Microsoft\WindowsApps\pwsh.exe`),
		)
	}

	for _, cand := range candidates {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			dir := filepath.Dir(cand)
			pathEnv := os.Getenv("PATH")
			if pathEnv != "" {
				_ = os.Setenv("PATH", dir+";"+pathEnv)
			} else {
				_ = os.Setenv("PATH", dir)
			}
			return cand, true
		}
	}

	return "", false
}

// VerifyPwsh checks if pwsh.exe is installed on Windows.
// If it is not found, it calls logger.Fatalf and terminates the application.
func VerifyPwsh() {
	if IsWindows() {
		if _, ok := FindPwsh(); !ok {
			logger.Fatalf("Windows 系统必须依赖 PowerShell 7+ (pwsh.exe)，但在系统环境变量 PATH 及常用路径中均未找到，请先进行安装。")
		}
	}
}

// InterruptProcessGroup attempts to recursively kill child processes of the given parent PID on Windows.
// This is used to simulate Ctrl+C process interruption in standard input/output pipes.
func InterruptProcessGroup(parentPid int) {
	if !IsWindows() || parentPid <= 0 {
		return
	}
	// Query direct child processes and terminate their process trees recursively using taskkill
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("Get-CimInstance Win32_Process -Filter 'ParentProcessId = %d' | ForEach-Object { taskkill /F /T /PID $_.ProcessId }", parentPid))
	_ = cmd.Run()
}

// GetWindowsShellCmd returns the exec.Cmd configured for pwsh on Windows with proper flags.
func GetWindowsShellCmd(shell string) *exec.Cmd {
	return exec.Command(shell, "-NoLogo", "-NoProfile", "-NoExit", "-Command", "function Clear-Host { Write-Host -NoNewline \"$([char]27)[2J$([char]27)[H\" }")
}

// IsWindows returns true if the current OS is Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// FixPathEnv prepends C:\Windows\system32;C:\Windows to PATH environment variable on Windows
// to ensure system executables (like timeout.exe, find.exe) resolve correctly first.
func FixPathEnv(env []string) []string {
	if !IsWindows() {
		return env
	}
	var pathFound bool
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			parts := strings.SplitN(e, "=", 2)
			env[i] = parts[0] + "=C:\\Windows\\system32;C:\\Windows;" + parts[1]
			pathFound = true
			break
		}
	}
	if !pathFound {
		env = append(env, "PATH=C:\\Windows\\system32;C:\\Windows")
	}
	return env
}

// GetPathSeparator returns the path list separator (semicolon for Windows, colon for Unix)
func GetPathSeparator() string {
	if IsWindows() {
		return ";"
	}
	return ":"
}

// GetExeExtension returns ".exe" on Windows, empty string elsewhere
func GetExeExtension() string {
	if IsWindows() {
		return ".exe"
	}
	return ""
}
