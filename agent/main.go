package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"gopkg.in/natefinch/lumberjack.v2"
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

// 日志实例
var log = logrus.New()

// 全局配置
var (
	configFile = "config.ini"
	logFile    = "logs/agent.log"
)

func main() {
	// 获取程序所在目录
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
`, Version)
}

// ========== 命令实现 ==========

func cmdStart() {
	// 初始化日志
	initLogger(logFile)

	// 加载配置
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

	// 验证配置
	if config.ServerURL == "" {
		log.Fatal("请在配置文件中设置 server_url")
	}
	if config.Name == "" {
		hostname, _ := os.Hostname()
		config.Name = hostname
	}

	log.Infof("Baihu Agent v%s", Version)
	if BuildTime != "" {
		log.Infof("构建时间: %s", BuildTime)
	}
	log.Infof("服务器: %s", config.ServerURL)
	log.Infof("名称: %s", config.Name)

	// 写入 PID 文件
	writePidFile()

	// 创建并启动 Agent
	agent := NewAgent(config, configFile)
	if err := agent.Start(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在停止...")
	agent.Stop()
	removePidFile()
}

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

// ========== PID 文件管理 ==========

func getPidFile() string {
	return filepath.Join(filepath.Dir(configFile), "agent.pid")
}

func writePidFile() {
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

// ========== 日志初始化 ==========

// CustomFormatter 自定义日志格式
type CustomFormatter struct{}

// ANSI 颜色代码
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[36m"
	colorGray   = "\033[37m"
)

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	level := strings.ToUpper(entry.Level.String())

	var levelColor string
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = colorGray
	case logrus.InfoLevel:
		levelColor = colorBlue
	case logrus.WarnLevel:
		levelColor = colorYellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = colorRed
	default:
		levelColor = colorBlue
	}

	msg := fmt.Sprintf("[%s]%s[%s]%s %s\n", timestamp, levelColor, level, colorReset, entry.Message)
	return []byte(msg), nil
}

func initLogger(logFile string) {
	logDir := filepath.Dir(logFile)
	if logDir != "" && logDir != "." {
		os.MkdirAll(logDir, 0755)
	}

	log.SetFormatter(&CustomFormatter{})
	log.SetLevel(logrus.InfoLevel)

	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     0,
		Compress:   false,
	}

	log.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))
}


// ========== 配置相关 ==========

type Config struct {
	ServerURL  string
	Name       string
	Token      string
	Interval   int
	AutoUpdate bool
}

func loadConfigFile(path string, config *Config) error {
	cfg, err := ini.Load(path)
	if err != nil {
		return err
	}

	section := cfg.Section("agent")
	if v := section.Key("server_url").String(); v != "" {
		config.ServerURL = v
	}
	if v := section.Key("name").String(); v != "" {
		config.Name = v
	}
	if v := section.Key("token").String(); v != "" {
		config.Token = v
	}
	if v := section.Key("interval").String(); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			config.Interval = i
		}
	}
	if v := section.Key("auto_update").String(); v != "" {
		config.AutoUpdate = v == "true" || v == "1"
	}
	return nil
}

func saveConfigFile(path string, config *Config) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}

	cfg := ini.Empty()
	section := cfg.Section("agent")
	section.Key("server_url").SetValue(config.ServerURL)
	section.Key("name").SetValue(config.Name)
	section.Key("token").SetValue(config.Token)
	section.Key("interval").SetValue(strconv.Itoa(config.Interval))
	if config.AutoUpdate {
		section.Key("auto_update").SetValue("true")
	} else {
		section.Key("auto_update").SetValue("false")
	}

	return cfg.SaveTo(path)
}

// ========== Agent 结构 ==========

type AgentTask struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	Schedule string `json:"schedule"`
	Timeout  int    `json:"timeout"`
	WorkDir  string `json:"work_dir"`
	Envs     string `json:"envs"`
	Enabled  bool   `json:"enabled"`
}

type TaskResult struct {
	TaskID    uint   `json:"task_id"`
	Command   string `json:"command"`
	Output    string `json:"output"`
	Status    string `json:"status"`
	Duration  int64  `json:"duration"`
	ExitCode  int    `json:"exit_code"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

type Agent struct {
	config     *Config
	configFile string
	cron       *cron.Cron
	tasks      map[uint]*AgentTask
	entryMap   map[uint]cron.EntryID
	mu         sync.RWMutex
	client     *http.Client
}

func NewAgent(config *Config, configFile string) *Agent {
	return &Agent{
		config:     config,
		configFile: configFile,
		cron:       cron.New(cron.WithSeconds(), cron.WithLocation(cstZone)),
		tasks:      make(map[uint]*AgentTask),
		entryMap:   make(map[uint]cron.EntryID),
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *Agent) Start() error {
	if a.config.Token == "" {
		log.Info("未找到 Token，开始注册流程...")
		if err := a.registerAndWait(); err != nil {
			return err
		}
	}

	if err := a.heartbeat(); err != nil {
		log.Warnf("首次心跳失败: %v（将继续重试）", err)
	}

	if err := a.syncTasks(); err != nil {
		log.Warnf("同步任务失败: %v（将继续重试）", err)
	}

	a.cron.Start()
	go a.heartbeatLoop()
	go a.syncTasksLoop()

	log.Info("Agent 已启动 (时区: Asia/Shanghai)")
	return nil
}

func (a *Agent) Stop() {
	ctx := a.cron.Stop()
	<-ctx.Done()
	log.Info("Agent 已停止")
}

func (a *Agent) registerAndWait() error {
	hostname, _ := os.Hostname()

	body := map[string]string{
		"name":     a.config.Name,
		"hostname": hostname,
		"version":  Version,
	}

	resp, err := a.doRequestNoAuth("POST", "/api/agent/register", body)
	if err != nil {
		return fmt.Errorf("注册失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("注册失败: %s", string(data))
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			AgentID uint   `json:"agent_id"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	log.Infof("注册成功 (ID: %d)，等待管理员审核...", result.Data.AgentID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		statusResp, err := a.doRequestNoAuth("POST", "/api/agent/status", map[string]string{
			"name": a.config.Name,
		})
		if err != nil {
			log.Warnf("检查状态失败: %v", err)
			continue
		}

		if statusResp.StatusCode != http.StatusOK {
			statusResp.Body.Close()
			continue
		}

		var statusResult struct {
			Code int `json:"code"`
			Data struct {
				AgentID uint   `json:"agent_id"`
				Status  string `json:"status"`
				Token   string `json:"token"`
			} `json:"data"`
		}
		json.NewDecoder(statusResp.Body).Decode(&statusResult)
		statusResp.Body.Close()

		if statusResult.Data.Status != "pending" && statusResult.Data.Token != "" {
			a.config.Token = statusResult.Data.Token
			if err := saveConfigFile(a.configFile, a.config); err != nil {
				log.Warnf("保存配置文件失败: %v", err)
			} else {
				log.Infof("Token 已保存到 %s", a.configFile)
			}
			log.Info("审核通过，开始工作...")
			return nil
		}

		log.Debug("等待审核中...")
	}
}

func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(a.config.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := a.heartbeat(); err != nil {
			log.Warnf("心跳失败: %v", err)
		}
	}
}

func (a *Agent) heartbeat() error {
	hostname, _ := os.Hostname()
	body := map[string]interface{}{
		"version":     Version,
		"build_time":  BuildTime,
		"hostname":    hostname,
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"auto_update": a.config.AutoUpdate,
	}

	resp, err := a.doRequest("POST", "/api/agent/heartbeat", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("心跳失败: %s", string(data))
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			AgentID       uint   `json:"agent_id"`
			Name          string `json:"name"`
			NeedUpdate    bool   `json:"need_update"`
			ForceUpdate   bool   `json:"force_update"`
			LatestVersion string `json:"latest_version"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil // 忽略解析错误
	}

	// 检查是否需要更新
	if result.Data.NeedUpdate && (a.config.AutoUpdate || result.Data.ForceUpdate) {
		log.Infof("发现新版本 %s，开始更新...", result.Data.LatestVersion)
		go a.selfUpdate()
	}

	return nil
}

func (a *Agent) syncTasksLoop() {
	ticker := time.NewTicker(time.Duration(a.config.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := a.syncTasks(); err != nil {
			log.Warnf("同步任务失败: %v", err)
		}
	}
}

func (a *Agent) syncTasks() error {
	resp, err := a.doRequest("GET", "/api/agent/tasks", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("获取任务失败: %s", string(data))
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			AgentID uint        `json:"agent_id"`
			Tasks   []AgentTask `json:"tasks"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	a.updateTasks(result.Data.Tasks)
	return nil
}

func (a *Agent) updateTasks(tasks []AgentTask) {
	a.mu.Lock()
	defer a.mu.Unlock()

	newTasks := make(map[uint]*AgentTask)
	for i := range tasks {
		newTasks[tasks[i].ID] = &tasks[i]
	}

	for id, entryID := range a.entryMap {
		if _, exists := newTasks[id]; !exists {
			a.cron.Remove(entryID)
			delete(a.entryMap, id)
			delete(a.tasks, id)
			log.Infof("移除任务 #%d", id)
		}
	}

	for id, task := range newTasks {
		oldTask, exists := a.tasks[id]
		if !exists || oldTask.Schedule != task.Schedule || oldTask.Command != task.Command {
			if entryID, ok := a.entryMap[id]; ok {
				a.cron.Remove(entryID)
			}

			taskCopy := *task
			entryID, err := a.cron.AddFunc(task.Schedule, func() {
				a.executeTask(&taskCopy)
			})
			if err != nil {
				log.Errorf("添加任务 #%d 失败: %v", id, err)
				continue
			}

			a.entryMap[id] = entryID
			a.tasks[id] = task
			log.Infof("调度任务 #%d %s (%s)", id, task.Name, task.Schedule)
		}
	}
}

func (a *Agent) executeTask(task *AgentTask) {
	log.Infof("执行任务 #%d %s", task.ID, task.Name)

	start := time.Now()
	result := &TaskResult{
		TaskID:    task.ID,
		Command:   task.Command,
		StartTime: start.Unix(),
	}

	timeout := task.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", task.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", task.Command)
	}

	if task.WorkDir != "" {
		cmd.Dir = task.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	end := time.Now()

	result.EndTime = end.Unix()
	result.Duration = end.Sub(start).Milliseconds()
	result.Output = stdout.String()

	if err != nil {
		result.Status = "failed"
		result.Output += "\n[ERROR]\n" + stderr.String() + "\n" + err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.Status = "success"
		result.ExitCode = 0
	}

	if err := a.reportResult(result); err != nil {
		log.Errorf("上报结果失败: %v", err)
	}
}

func (a *Agent) reportResult(result *TaskResult) error {
	resp, err := a.doRequest("POST", "/api/agent/report", result)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上报失败: %s", string(data))
	}

	log.Infof("任务 #%d 执行完成 (%s)", result.TaskID, result.Status)
	return nil
}

func (a *Agent) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, a.config.ServerURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.config.Token)
	req.Header.Set("Content-Type", "application/json")

	return a.client.Do(req)
}

func (a *Agent) doRequestNoAuth(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, a.config.ServerURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return a.client.Do(req)
}

// selfUpdate 自动更新
func (a *Agent) selfUpdate() {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		log.Errorf("获取可执行文件路径失败: %v", err)
		return
	}
	exePath, _ = filepath.Abs(exePath)
	exeDir := filepath.Dir(exePath)

	// 下载新版本 tar.gz
	downloadURL := fmt.Sprintf("%s/api/agent/download?os=%s&arch=%s", a.config.ServerURL, runtime.GOOS, runtime.GOARCH)
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		log.Errorf("创建下载请求失败: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.config.Token)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("下载新版本失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf("下载新版本失败: HTTP %d", resp.StatusCode)
		return
	}

	// 读取 tar.gz 内容
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Errorf("解压 gzip 失败: %v", err)
		return
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// 解压并找到二进制文件
	var newBinary []byte
	binaryName := "baihu-agent"
	if runtime.GOOS == "windows" {
		binaryName = "baihu-agent.exe"
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("读取 tar 失败: %v", err)
			return
		}

		if header.Typeflag == tar.TypeReg && header.Name == binaryName {
			newBinary, err = io.ReadAll(tarReader)
			if err != nil {
				log.Errorf("读取二进制文件失败: %v", err)
				return
			}
			break
		}
	}

	if newBinary == nil {
		log.Errorf("tar.gz 中未找到 %s", binaryName)
		return
	}

	// 保存到临时文件
	tmpFile := filepath.Join(exeDir, binaryName+".new")
	if err := os.WriteFile(tmpFile, newBinary, 0755); err != nil {
		log.Errorf("保存新版本失败: %v", err)
		return
	}

	// 备份旧版本
	backupFile := exePath + ".bak"
	os.Remove(backupFile)
	if err := os.Rename(exePath, backupFile); err != nil {
		log.Errorf("备份旧版本失败: %v", err)
		os.Remove(tmpFile)
		return
	}

	// 替换为新版本
	if err := os.Rename(tmpFile, exePath); err != nil {
		log.Errorf("替换新版本失败: %v", err)
		os.Rename(backupFile, exePath) // 恢复旧版本
		return
	}

	log.Info("更新完成，正在重启...")

	// 重启服务
	a.restart()
}

// restart 重启服务
func (a *Agent) restart() {
	exePath, _ := os.Executable()

	if runtime.GOOS == "windows" {
		// Windows: 启动新进程后退出
		cmd := exec.Command(exePath, "start")
		cmd.Start()
		os.Exit(0)
	} else {
		// Linux/macOS: 使用 exec 替换当前进程
		syscall.Exec(exePath, []string{exePath, "start"}, os.Environ())
	}
}
