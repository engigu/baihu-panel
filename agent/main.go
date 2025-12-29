package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const ServiceName = "baihu-agent"
const ServiceDesc = "Baihu Agent Service"

// 版本信息（通过 ldflags 注入）
var (
	Version   = "dev"
	BuildTime = ""
)

// 东八区时区
var cstZone = time.FixedZone("CST", 8*3600)

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
		}
	}

	switch cmd {
	case "start":
		cmdStart()
	case "stop":
		cmdStop()
	case "status":
		cmdStatus()
	case "tasks":
		cmdTasks()
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
  start       启动 Agent
  stop        停止 Agent
  status      查看运行状态
  tasks       查看已下发的任务列表
  install     安装为系统服务（开机自启）
  uninstall   卸载系统服务
  version     显示版本信息
  help        显示帮助信息

选项:
  -c, --config <file>   配置文件路径 (默认: config.ini)
  -l, --log <file>      日志文件路径 (默认: logs/agent.log)

示例:
  baihu-agent start
  baihu-agent start -c /etc/baihu/config.ini
  baihu-agent install
  baihu-agent status
  baihu-agent tasks
`, Version)
}

func cmdStart() {
	initLogger(logFile)

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

func cmdTasks() {
	config := &Config{Interval: 30}
	if err := loadConfigFile(configFile, config); err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		return
	}

	if config.Token == "" {
		fmt.Println("错误: 缺少令牌，请在配置文件中设置 token")
		return
	}

	agent := &Agent{
		config:    config,
		machineID: generateMachineID(),
		client:    &http.Client{Timeout: 30 * time.Second},
	}

	resp, err := agent.doRequest("GET", "/api/agent/tasks", nil)
	if err != nil {
		fmt.Printf("获取任务列表失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("获取任务列表失败: %s\n", string(body))
		return
	}

	var result struct {
		Tasks []AgentTask `json:"tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return
	}

	if len(result.Tasks) == 0 {
		fmt.Println("当前没有下发的任务")
		return
	}

	fmt.Printf("共 %d 个任务:\n\n", len(result.Tasks))
	for i, task := range result.Tasks {
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
