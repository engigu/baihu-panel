package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/engigu/baihu-panel/internal/utils"
)

const ServiceName = "baihu-agent"
const ServiceDesc = "Baihu Agent Service"

// 版本信息（通过 ldflags 注入）
var (
	Version   = "dev"
	BuildTime = ""
)

// 全局配置
var (
	configFile = "config.ini"
	logFile    = "logs/agent.log"
	dataDir    = "data"
)

func main() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	os.Chdir(exeDir)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// 解析额外参数
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-c", "--config":
			if i+1 < len(os.Args) {
				configFile = os.Args[i+1]
				i++
			}
		case "-l", "--log":
			if i+1 < len(os.Args) {
				logFile = os.Args[i+1]
				i++
			}
		case "-d", "--daemon":
			isDaemon = true
		case "--restart":
			isRestart = true
		}
	}

	switch cmd {
	case "start":
		cmdStart()
	case "run":
		cmdRun()
	case "stop":
		cmdStop()
	case "status":
		cmdStatus()
	case "tasks":
		cmdTasks()
	case "logs":
		cmdLogs()
	case "install":
		cmdInstall()
	case "uninstall":
		cmdUninstall()
	case "version", "-v", "--version":
		fmt.Printf("Baihu Agent v%s\n", Version)
		if BuildTime != "" {
			fmt.Printf("Build Time: %s\n", BuildTime)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("未知命令: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`Baihu Agent v%s

用法: baihu-agent <命令> [选项]

命令:
  start       启动 Agent（后台运行）
  run         前台运行 Agent
  stop        停止 Agent
  status      查看运行状态
  tasks       查看已下发的任务列表
  logs        查看日志（实时跟踪）
  install     安装为系统服务（开机自启）
  uninstall   卸载系统服务
  version     显示版本信息
  help        显示帮助信息

选项:
  -c, --config <file>   配置文件路径 (默认: config.ini)
  -l, --log <file>      日志文件路径 (默认: logs/agent.log)

示例:
  baihu-agent start
  baihu-agent run
  baihu-agent logs
  baihu-agent start -c /etc/baihu/config.ini
  baihu-agent install
  baihu-agent status
  baihu-agent tasks
`, Version)
}

// daemon 模式标记
var isDaemon = false

// 是否从 daemon 重启（用于自动更新后重启）
var isRestart = false

func cmdStart() {
	// 检查是否已经在运行（使用文件锁）
	pid := readPidFile()
	if pid != 0 && isProcessRunning(pid) {
		fmt.Printf("Agent 已在运行 (PID: %d)\n", pid)
		return
	}

	// 如果不是 daemon 子进程，则启动 daemon
	if !isDaemon {
		startDaemon()
		return
	}

	// 以下是 daemon 子进程的逻辑
	// 尝试获取文件锁
	if !tryLock() {
		fmt.Println("Agent 已在运行（无法获取锁）")
		return
	}
	defer unlock()

	initLogger(logFile, true)

	config := &Config{Interval: 30}
	if err := loadConfigFile(configFile, config); err != nil {
		if !os.IsNotExist(err) {
			log.Warnf("加载配置文件失败: %v", err)
		}
	}

	// 从环境变量加载
	if v := os.Getenv("AGENT_SERVER"); v != "" {
		config.ServerURL = v
	}
	if v := os.Getenv("AGENT_NAME"); v != "" {
		config.Name = v
	}

	if config.ServerURL == "" {
		log.Fatal("请在配置文件中设置 server_url")
	}
	if config.Name == "" {
		hostname, _ := os.Hostname()
		config.Name = hostname
	}

	log.Infof("Baihu Agent Version: %s", Version)
	if BuildTime != "" {
		log.Infof("构建时间: %s", BuildTime)
	}
	log.Infof("服务器: %s", config.ServerURL)
	log.Infof("名称: %s", config.Name)

	writePidFile()

	agent := NewAgent(config, configFile)
	if err := agent.Start(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在停止...")
	agent.Stop()
	removePidFile()
}

// cmdRun 前台运行
func cmdRun() {
	// 检查是否已经在运行（重启模式下跳过检查）
	if !isRestart {
		pid := readPidFile()
		if pid != 0 && isProcessRunning(pid) {
			fmt.Printf("Agent 已在运行 (PID: %d)\n", pid)
			return
		}
	}

	// 尝试获取文件锁
	if !tryLock() {
		fmt.Println("Agent 已在运行（无法获取锁）")
		return
	}
	defer unlock()

	// 重启模式下只输出到文件（因为是从 daemon 进程 exec 过来的）
	initLogger(logFile, isRestart)

	config := &Config{Interval: 30}
	if err := loadConfigFile(configFile, config); err != nil {
		if !os.IsNotExist(err) {
			log.Warnf("加载配置文件失败: %v", err)
		}
	}

	if v := os.Getenv("AGENT_SERVER"); v != "" {
		config.ServerURL = v
	}
	if v := os.Getenv("AGENT_NAME"); v != "" {
		config.Name = v
	}

	if config.ServerURL == "" {
		log.Fatal("请在配置文件中设置 server_url")
	}
	if config.Name == "" {
		hostname, _ := os.Hostname()
		config.Name = hostname
	}

	log.Infof("Baihu Agent Version: %s", Version)
	if BuildTime != "" {
		log.Infof("构建时间: %s", BuildTime)
	}
	log.Infof("服务器: %s", config.ServerURL)
	log.Infof("名称: %s", config.Name)

	writePidFile()

	agent := NewAgent(config, configFile)
	if err := agent.Start(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在停止...")
	agent.Stop()
	removePidFile()
}

func startDaemon() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("获取可执行文件路径失败: %v\n", err)
		return
	}

	// 构建子进程参数，添加 --daemon 标记
	args := []string{"start", "--daemon"}
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] != "--daemon" && os.Args[i] != "-d" {
			args = append(args, os.Args[i])
		}
	}

	// 打开 /dev/null 用于丢弃输出（日志由 logger 写入文件）
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		fmt.Printf("打开 /dev/null 失败: %v\n", err)
		return
	}

	// 启动子进程
	cmd := &exec.Cmd{
		Path:   exePath,
		Args:   append([]string{exePath}, args...),
		Dir:    filepath.Dir(exePath),
		Stdout: devNull,
		Stderr: devNull,
	}

	// 设置进程组，使子进程独立运行
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("启动失败: %v\n", err)
		devNull.Close()
		return
	}

	devNull.Close()
	fmt.Printf("Agent 已启动 (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("日志文件: %s\n", logFile)
}

func cmdTasks() {
	config := &Config{Interval: 30}
	if err := loadConfigFile(configFile, config); err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		return
	}

	if config.ServerURL == "" {
		fmt.Println("错误: 缺少服务器地址，请在配置文件中设置 server_url")
		return
	}

	if config.Token == "" {
		fmt.Println("错误: 缺少令牌，请在配置文件中设置 token")
		return
	}

	agent := &Agent{
		config:    config,
		machineID: utils.GenerateMachineID(),
		client:    &http.Client{Timeout: 30 * time.Second},
	}

	resp, err := agent.doRequest("GET", "/api/agent/tasks", nil)
	if err != nil {
		fmt.Printf("获取任务列表失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("获取任务列表失败 (HTTP %d): %s\n", resp.StatusCode, string(body))
		return
	}

	// 解析服务端响应（包含 code/msg/data 包装）
	var apiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AgentID uint        `json:"agent_id"`
			Tasks   []AgentTask `json:"tasks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	if apiResp.Code != 200 {
		fmt.Printf("获取任务列表失败: %s\n", apiResp.Msg)
		return
	}

	tasks := apiResp.Data.Tasks
	if len(tasks) == 0 {
		fmt.Println("当前没有下发的任务")
		return
	}

	fmt.Printf("共 %d 个任务:\n\n", len(tasks))
	for i, task := range tasks {
		fmt.Printf("[%d] ID: %d\n", i+1, task.ID)
		fmt.Printf("    名称: %s\n", task.Name)
		fmt.Printf("    Cron: %s\n", task.Schedule)
		fmt.Printf("    命令: %s\n", task.Command)
		if task.WorkDir != "" {
			fmt.Printf("    工作目录: %s\n", task.WorkDir)
		}
		fmt.Printf("    启用: %v\n", task.Enabled)
		fmt.Println()
	}
}

func cmdLogs() {
	// 检查日志文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Printf("日志文件不存在: %s\n", logFile)
		return
	}

	fmt.Printf("日志文件: %s\n", logFile)
	fmt.Println("按 Ctrl+C 退出")

	// 使用 tail -f 实时跟踪日志
	cmd := exec.Command("tail", "-f", "-n", "50", logFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 处理中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	cmd.Run()
}
