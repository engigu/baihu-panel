package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
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

	log.Infof("Baihu Agent Version: %s", Version)
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

// WebSocket 消息类型
const (
	WSTypeHeartbeat    = "heartbeat"
	WSTypeHeartbeatAck = "heartbeat_ack"
	WSTypeTasks        = "tasks"
	WSTypeTaskResult   = "task_result"
	WSTypeUpdate       = "update"
	WSTypeConnected    = "connected"
	WSTypeDisabled     = "disabled"    // Agent 被禁用
	WSTypeEnabled      = "enabled"     // Agent 被启用
	WSTypeFetchTasks   = "fetch_tasks" // Agent 请求任务列表
)

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

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
	config        *Config
	configFile    string
	machineID     string
	cron          *cron.Cron
	tasks         map[uint]*AgentTask
	entryMap      map[uint]cron.EntryID
	lastTaskCount int // 上次任务数量，用于判断是否需要打印日志
	mu            sync.RWMutex
	client        *http.Client
	wsConn        *websocket.Conn
	wsMu          sync.Mutex
	stopCh        chan struct{}
}

// generateMachineID 生成机器识别码（基于 hostname + MAC 地址）
func generateMachineID() string {
	var parts []string

	// 1. Hostname
	if hostname, err := os.Hostname(); err == nil {
		parts = append(parts, hostname)
	}

	// 2. MAC 地址（取所有非回环网卡的 MAC）
	if interfaces, err := net.Interfaces(); err == nil {
		var macs []string
		for _, iface := range interfaces {
			// 跳过回环和无 MAC 的接口
			if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
				continue
			}
			macs = append(macs, iface.HardwareAddr.String())
		}
		// 排序确保顺序一致
		sort.Strings(macs)
		parts = append(parts, macs...)
	}

	// 3. 操作系统和架构
	parts = append(parts, runtime.GOOS, runtime.GOARCH)

	// 生成 SHA256 哈希
	data := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func NewAgent(config *Config, configFile string) *Agent {
	return &Agent{
		config:     config,
		configFile: configFile,
		machineID:  generateMachineID(),
		cron:       cron.New(cron.WithSeconds(), cron.WithLocation(cstZone)),
		tasks:      make(map[uint]*AgentTask),
		entryMap:   make(map[uint]cron.EntryID),
		client:     &http.Client{Timeout: 30 * time.Second},
		stopCh:     make(chan struct{}),
	}
}

func (a *Agent) Start() error {
	if a.config.Token == "" {
		return fmt.Errorf("缺少令牌，请在配置文件中设置 token")
	}

	log.Infof("机器识别码: %s", a.machineID[:16]+"...")
	a.cron.Start()

	// 启动 WebSocket 连接
	go a.wsLoop()

	log.Info("Agent 已启动 (时区: Asia/Shanghai, 模式: WebSocket)")
	return nil
}

func (a *Agent) Stop() {
	close(a.stopCh)
	a.closeWS()
	ctx := a.cron.Stop()
	<-ctx.Done()
	log.Info("Agent 已停止")
}

// wsLoop WebSocket 连接循环（自动重连）
func (a *Agent) wsLoop() {
	for {
		select {
		case <-a.stopCh:
			return
		default:
		}

		if err := a.connectWS(); err != nil {
			log.Warnf("WebSocket 连接失败: %v，5秒后重试...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// 连接成功，开始读取消息
		a.readWS()

		// 连接断开，等待后重连
		log.Warn("WebSocket 连接断开，5秒后重连...")
		time.Sleep(5 * time.Second)
	}
}

// connectWS 建立 WebSocket 连接
func (a *Agent) connectWS() error {
	// 构建 WebSocket URL
	serverURL := a.config.ServerURL
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/api/agent/ws?token=%s&machine_id=%s", wsURL, url.QueryEscape(a.config.Token), url.QueryEscape(a.machineID))

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	a.wsMu.Lock()
	a.wsConn = conn
	a.wsMu.Unlock()

	log.Info("WebSocket 已连接")

	// 发送首次心跳
	a.sendHeartbeat()

	// 启动心跳协程
	go a.heartbeatLoop()

	return nil
}

// closeWS 关闭 WebSocket 连接
func (a *Agent) closeWS() {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	if a.wsConn != nil {
		a.wsConn.Close()
		a.wsConn = nil
	}
}

// readWS 读取 WebSocket 消息
func (a *Agent) readWS() {
	for {
		a.wsMu.Lock()
		conn := a.wsConn
		a.wsMu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		a.handleWSMessage(&msg)
	}
}

// handleWSMessage 处理 WebSocket 消息
func (a *Agent) handleWSMessage(msg *WSMessage) {
	switch msg.Type {
	case WSTypeConnected:
		a.handleConnected(msg.Data)

	case WSTypeHeartbeatAck:
		a.handleHeartbeatAck(msg.Data)

	case WSTypeTasks:
		a.handleTasks(msg.Data)

	case WSTypeUpdate:
		log.Info("收到更新指令，开始更新...")
		go a.selfUpdate()

	case WSTypeDisabled:
		log.Warn("Agent 已被禁用，清空所有任务")
		a.clearAllTasks()

	case WSTypeEnabled:
		log.Info("Agent 已被启用，主动拉取任务")
		a.fetchTasks()
	}
}

// fetchTasks 主动请求任务列表
func (a *Agent) fetchTasks() {
	if err := a.sendWSMessage(WSTypeFetchTasks, map[string]interface{}{}); err != nil {
		log.Warnf("请求任务列表失败: %v", err)
	}
}

// handleConnected 处理连接成功消息
func (a *Agent) handleConnected(data json.RawMessage) {
	var resp struct {
		AgentID    uint   `json:"agent_id"`
		Name       string `json:"name"`
		IsNewAgent bool   `json:"is_new_agent"`
		MachineID  string `json:"machine_id"`
	}
	json.Unmarshal(data, &resp)

	if resp.IsNewAgent {
		log.Infof("注册成功: Agent #%d, 机器码: %s", resp.AgentID, a.machineID[:16]+"...")
	} else {
		log.Infof("连接成功: Agent #%d (已存在), 机器码: %s", resp.AgentID, a.machineID[:16]+"...")
	}

	// 连接成功后主动拉取任务
	a.fetchTasks()
}

// handleHeartbeatAck 处理心跳响应
func (a *Agent) handleHeartbeatAck(data json.RawMessage) {
	var resp struct {
		AgentID       uint   `json:"agent_id"`
		Name          string `json:"name"`
		NeedUpdate    bool   `json:"need_update"`
		ForceUpdate   bool   `json:"force_update"`
		LatestVersion string `json:"latest_version"`
	}
	json.Unmarshal(data, &resp)

	if resp.NeedUpdate && (a.config.AutoUpdate || resp.ForceUpdate) {
		log.Infof("发现新版本 %s，开始更新...", resp.LatestVersion)
		go a.selfUpdate()
	}
}

// handleTasks 处理任务列表
func (a *Agent) handleTasks(data json.RawMessage) {
	var resp struct {
		Tasks []AgentTask `json:"tasks"`
	}
	json.Unmarshal(data, &resp)

	// 只在任务数量变化时打印日志
	newCount := len(resp.Tasks)
	if newCount != a.lastTaskCount {
		log.Infof("任务列表更新: %d -> %d 个任务", a.lastTaskCount, newCount)
		a.lastTaskCount = newCount
	}

	a.updateTasks(resp.Tasks)
}

// sendWSMessage 发送 WebSocket 消息
func (a *Agent) sendWSMessage(msgType string, data interface{}) error {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()

	if a.wsConn == nil {
		return fmt.Errorf("WebSocket 未连接")
	}

	dataBytes, _ := json.Marshal(data)
	msg := WSMessage{Type: msgType, Data: dataBytes}
	msgBytes, _ := json.Marshal(msg)

	a.wsConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return a.wsConn.WriteMessage(websocket.TextMessage, msgBytes)
}

// heartbeatLoop 心跳循环
func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(a.config.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.wsMu.Lock()
			conn := a.wsConn
			a.wsMu.Unlock()
			if conn == nil {
				return // 连接已断开，退出心跳循环
			}
			a.sendHeartbeat()
		}
	}
}

// sendHeartbeat 发送心跳
func (a *Agent) sendHeartbeat() {
	hostname, _ := os.Hostname()
	data := map[string]interface{}{
		"version":     Version,
		"build_time":  BuildTime,
		"hostname":    hostname,
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"auto_update": a.config.AutoUpdate,
	}
	if err := a.sendWSMessage(WSTypeHeartbeat, data); err != nil {
		log.Warnf("发送心跳失败: %v", err)
	}
}

// sendTaskResult 发送任务结果
func (a *Agent) sendTaskResult(result *TaskResult) {
	if err := a.sendWSMessage(WSTypeTaskResult, result); err != nil {
		log.Warnf("发送任务结果失败: %v，尝试 HTTP 上报", err)
		// 降级到 HTTP
		a.reportResultHTTP(result)
	}
}

// reportResultHTTP HTTP 方式上报结果（降级方案）
func (a *Agent) reportResultHTTP(result *TaskResult) error {
	resp, err := a.doRequest("POST", "/api/agent/report", result)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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

// clearAllTasks 清空所有任务（Agent 被禁用时调用）
func (a *Agent) clearAllTasks() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for id, entryID := range a.entryMap {
		a.cron.Remove(entryID)
		log.Infof("移除任务 #%d", id)
	}

	a.entryMap = make(map[uint]cron.EntryID)
	a.tasks = make(map[uint]*AgentTask)
	log.Info("所有任务已清空")
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

	// 使用 WebSocket 上报结果
	a.sendTaskResult(result)
	log.Infof("任务 #%d 执行完成 (%s)", result.TaskID, result.Status)
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
