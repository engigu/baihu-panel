package windows

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/engigu/baihu-panel/internal/logger"
)

// VerifyPwsh checks if pwsh.exe is installed on Windows.
// If it is not found, it calls logger.Fatalf and terminates the application.
func VerifyPwsh() {
	if IsWindows() {
		if _, err := exec.LookPath("pwsh"); err != nil {
			logger.Fatalf("PowerShell 7+ (pwsh.exe) is required on Windows, but was not found in PATH. Please install it first.")
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
