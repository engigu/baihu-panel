package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
)

// ========== PID 文件管理 ==========

func getPidFile() string {
	return filepath.Join(dataDir, "agent.pid")
}

func writePidFile() {
	os.MkdirAll(dataDir, 0755)
	pidFile := getPidFile()
	os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func readPidFile() int {
	pidFile := getPidFile()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(string(data))
	return pid
}

func removePidFile() {
	os.Remove(getPidFile())
}

// ========== 命令实现 ==========

func cmdStop() {
	pid := readPidFile()
	if pid == 0 {
		fmt.Println("Agent 未运行")
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("找不到进程 %d\n", pid)
		removePidFile()
		return
	}

	if runtime.GOOS == "windows" {
		err = process.Kill()
	} else {
		err = process.Signal(syscall.SIGTERM)
	}

	if err != nil {
		fmt.Printf("停止失败: %v\n", err)
		return
	}

	fmt.Println("Agent 已停止")
	removePidFile()
}

func cmdStatus() {
	pid := readPidFile()
	if pid == 0 {
		fmt.Println("状态: 未运行")
		return
	}

	// 检查进程是否存在
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("状态: 未运行")
		removePidFile()
		return
	}

	// Unix 系统发送信号 0 检查进程
	if runtime.GOOS != "windows" {
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			fmt.Println("状态: 未运行")
			removePidFile()
			return
		}
	}

	fmt.Printf("状态: 运行中 (PID: %d)\n", pid)
}

func cmdInstall() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	if runtime.GOOS == "windows" {
		installWindows(exePath, exeDir)
	} else {
		installLinux(exePath, exeDir)
	}
}

func cmdUninstall() {
	if runtime.GOOS == "windows" {
		uninstallWindows()
	} else {
		uninstallLinux()
	}
}

// ========== Linux systemd ==========

func installLinux(exePath, exeDir string) {
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s start
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, ServiceDesc, exeDir, exePath)

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", ServiceName)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		fmt.Printf("创建服务文件失败: %v\n", err)
		fmt.Println("请使用 sudo 运行")
		return
	}

	// 重载 systemd
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", ServiceName).Run()

	fmt.Printf("服务已安装: %s\n", servicePath)
	fmt.Println("使用以下命令管理服务:")
	fmt.Printf("  启动: sudo systemctl start %s\n", ServiceName)
	fmt.Printf("  停止: sudo systemctl stop %s\n", ServiceName)
	fmt.Printf("  状态: sudo systemctl status %s\n", ServiceName)
}

func uninstallLinux() {
	// 停止服务
	exec.Command("systemctl", "stop", ServiceName).Run()
	exec.Command("systemctl", "disable", ServiceName).Run()

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", ServiceName)
	if err := os.Remove(servicePath); err != nil {
		fmt.Printf("删除服务文件失败: %v\n", err)
		fmt.Println("请使用 sudo 运行")
		return
	}

	exec.Command("systemctl", "daemon-reload").Run()
	fmt.Println("服务已卸载")
}

// ========== Windows 服务 ==========

func installWindows(exePath, exeDir string) {
	// 使用 sc.exe 创建服务
	cmd := exec.Command("sc", "create", ServiceName,
		"binPath=", fmt.Sprintf(`"%s" start`, exePath),
		"start=", "auto",
		"DisplayName=", ServiceDesc)

	if err := cmd.Run(); err != nil {
		fmt.Printf("创建服务失败: %v\n", err)
		fmt.Println("请以管理员身份运行")
		return
	}

	// 设置服务描述
	exec.Command("sc", "description", ServiceName, ServiceDesc).Run()

	fmt.Println("服务已安装")
	fmt.Println("使用以下命令管理服务:")
	fmt.Printf("  启动: sc start %s\n", ServiceName)
	fmt.Printf("  停止: sc stop %s\n", ServiceName)
	fmt.Printf("  状态: sc query %s\n", ServiceName)
}

func uninstallWindows() {
	// 停止服务
	exec.Command("sc", "stop", ServiceName).Run()

	// 删除服务
	cmd := exec.Command("sc", "delete", ServiceName)
	if err := cmd.Run(); err != nil {
		fmt.Printf("删除服务失败: %v\n", err)
		fmt.Println("请以管理员身份运行")
		return
	}

	fmt.Println("服务已卸载")
}
