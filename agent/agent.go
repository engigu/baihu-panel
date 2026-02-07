package main

import (
	"bytes"
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
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/robfig/cron/v3"
)

// findAvailableShell 查找可用的 shell
func findAvailableShell() string {
	// 优先使用环境变量中的 SHELL
	if envShell := os.Getenv("SHELL"); envShell != "" {
		return envShell
	}

	// 尝试按优先级查找可用的 shell
	shells := []string{"/bin/bash", "/bin/zsh", "/bin/sh"}
	for _, sh := range shells {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}

	// 最后回退到 sh（应该总是存在）
	return "sh"
}

// WebSocket 消息类型
const (
	WSTypeHeartbeat    = "heartbeat"
	WSTypeHeartbeatAck = "heartbeat_ack"
	WSTypeTasks        = "tasks"
	WSTypeTaskResult   = "task_result"
	WSTypeUpdate       = "update"
	WSTypeConnected    = "connected"
	WSTypeDisabled     = "disabled"
	WSTypeEnabled      = "enabled"
	WSTypeFetchTasks   = "fetch_tasks"
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
	Cron     string `json:"cron"`
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
	lastTaskCount int
	mu            sync.RWMutex
	client        *http.Client
	wsConn        *websocket.Conn
	wsMu          sync.Mutex
	stopCh        chan struct{}
	wsStopCh      chan struct{} // 用于停止当前 WebSocket 相关的 goroutine
}

// generateMachineID 生成机器识别码
func generateMachineID() string {
	var parts []string

	// 主机名
	if hostname, err := os.Hostname(); err == nil {
		parts = append(parts, hostname)
	}

	// 获取所有非回环网卡的 MAC 地址，排序后取第一个（最稳定）
	if interfaces, err := net.Interfaces(); err == nil {
		var macs []string
		for _, iface := range interfaces {
			// 跳过回环接口、没有 MAC 地址的接口、虚拟接口
			if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
				continue
			}
			// 跳过 docker/veth 等虚拟网卡
			name := strings.ToLower(iface.Name)
			if strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "veth") ||
				strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "virbr") {
				continue
			}
			macs = append(macs, iface.HardwareAddr.String())
		}
		sort.Strings(macs)
		// 只使用第一个 MAC 地址（最稳定）
		if len(macs) > 0 {
			parts = append(parts, macs[0])
		}
	}

	// 操作系统和架构
	parts = append(parts, runtime.GOOS, runtime.GOARCH)

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

// wsLoop WebSocket 连接循环
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

		a.readWS()

		log.Warn("WebSocket 连接断开，5秒后重连...")
		time.Sleep(5 * time.Second)
	}
}

func (a *Agent) connectWS() error {
	serverURL := a.config.ServerURL
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/api/agent/ws?token=%s&machine_id=%s", wsURL, url.QueryEscape(a.config.Token), url.QueryEscape(a.machineID))

	log.Infof("正在连接 WebSocket: %s", wsURL)
	log.Infof("Token: %s..., MachineID: %s...", a.config.Token[:8], a.machineID[:16])

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Errorf("WebSocket 握手失败: HTTP %d, Body: %s", resp.StatusCode, string(bodyBytes))
			resp.Body.Close()
		} else {
			log.Errorf("WebSocket 连接失败: %v", err)
		}
		return err
	}

	a.wsMu.Lock()
	a.wsConn = conn
	a.wsStopCh = make(chan struct{})
	a.wsMu.Unlock()

	log.Info("WebSocket 已连接")
	a.sendHeartbeat()
	go a.heartbeatLoop()

	return nil
}

func (a *Agent) closeWS() {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	if a.wsStopCh != nil {
		close(a.wsStopCh)
		a.wsStopCh = nil
	}
	if a.wsConn != nil {
		a.wsConn.Close()
		a.wsConn = nil
	}
}

func (a *Agent) readWS() {
	defer func() {
		log.Info("readWS 退出，准备关闭连接")
		a.closeWS()
	}()

	for {
		a.wsMu.Lock()
		conn := a.wsConn
		a.wsMu.Unlock()

		if conn == nil {
			log.Warn("readWS: wsConn 为 nil")
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Warnf("WebSocket 读取错误: %v", err)
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		a.handleWSMessage(&msg)
	}
}

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
	case "execute":
		a.handleExecute(msg.Data)
	}
}

func (a *Agent) fetchTasks() {
	if err := a.sendWSMessage(WSTypeFetchTasks, map[string]interface{}{}); err != nil {
		log.Warnf("请求任务列表失败: %v", err)
	}
}

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

	a.fetchTasks()
}

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

func (a *Agent) handleTasks(data json.RawMessage) {
	var resp struct {
		Tasks []AgentTask `json:"tasks"`
	}
	json.Unmarshal(data, &resp)

	newCount := len(resp.Tasks)
	if newCount != a.lastTaskCount {
		log.Infof("任务列表更新: %d -> %d 个任务", a.lastTaskCount, newCount)
		a.lastTaskCount = newCount
	}

	a.updateTasks(resp.Tasks)
}

func (a *Agent) handleExecute(data json.RawMessage) {
	var req struct {
		TaskID uint `json:"task_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		log.Errorf("解析立即执行请求失败: %v", err)
		return
	}

	log.Infof("收到立即执行命令: 任务 #%d", req.TaskID)

	// 查找任务
	a.mu.RLock()
	task, exists := a.tasks[req.TaskID]
	a.mu.RUnlock()

	if !exists {
		log.Warnf("任务 #%d 不存在，无法执行", req.TaskID)
		return
	}

	// 立即执行任务
	go a.executeTask(task)
}

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
	if err := a.wsConn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		log.Warnf("发送消息失败 (%s): %v", msgType, err)
		return err
	}
	return nil
}

func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(a.config.Interval) * time.Second)
	defer ticker.Stop()

	a.wsMu.Lock()
	wsStopCh := a.wsStopCh
	a.wsMu.Unlock()

	if wsStopCh == nil {
		return
	}

	for {
		select {
		case <-a.stopCh:
			return
		case <-wsStopCh:
			return
		case <-ticker.C:
			a.wsMu.Lock()
			conn := a.wsConn
			a.wsMu.Unlock()
			if conn == nil {
				return
			}
			a.sendHeartbeat()
		}
	}
}

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

func (a *Agent) sendTaskResult(result *TaskResult) {
	if err := a.sendWSMessage(WSTypeTaskResult, result); err != nil {
		log.Warnf("发送任务结果失败: %v，尝试 HTTP 上报", err)
		a.reportResultHTTP(result)
	}
}

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

func (a *Agent) clearAllTasks() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for id, entryID := range a.entryMap {
		a.cron.Remove(entryID)
		log.Infof("移除任务 #%d", id)
	}

	a.entryMap = make(map[uint]cron.EntryID)
	a.tasks = make(map[uint]*AgentTask)
	a.lastTaskCount = 0
	log.Info("所有任务已清空")
}

func (a *Agent) executeTask(task *AgentTask) {
	log.Infof("执行任务 #%d %s", task.ID, task.Name)

	// 记录进程用户信息
	log.Infof("任务 #%d 进程 UID: %d, GID: %d", task.ID, os.Getuid(), os.Getgid())

	start := time.Now()

	timeout := task.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	finalCommand := task.Command

	if runtime.GOOS == "windows" {
		// Windows: 如果有工作目录，在命令前加 cd
		if task.WorkDir != "" {
			finalCommand = fmt.Sprintf("cd /d %s && %s", task.WorkDir, task.Command)
			log.Infof("任务 #%d 工作目录: %s", task.ID, task.WorkDir)
		}
		cmd = exec.CommandContext(ctx, "cmd", "/c", finalCommand)
	} else {
		// Linux/Unix: 如果有工作目录，在命令前加 cd
		if task.WorkDir != "" {
			finalCommand = fmt.Sprintf("cd %s && %s", task.WorkDir, task.Command)
			log.Infof("任务 #%d 工作目录: %s", task.ID, task.WorkDir)
		}
		// 尝试按优先级查找可用的 shell
		shell := findAvailableShell()
		cmd = exec.CommandContext(ctx, shell, "-c", finalCommand)
	}

	// 处理环境变量（始终继承系统环境变量）
	cmd.Env = os.Environ()
	if task.Envs != "" {
		envVars := a.parseEnvVars(task.Envs)
		if len(envVars) > 0 {
			cmd.Env = append(cmd.Env, envVars...)
			log.Infof("任务 #%d 设置了 %d 个环境变量", task.ID, len(envVars))
		}
	}

	output, err := cmd.CombinedOutput()
	end := time.Now()

	result := &TaskResult{
		TaskID:    task.ID,
		Command:   task.Command,
		Output:    string(output),
		Duration:  end.Sub(start).Milliseconds(),
		StartTime: start.Unix(),
		EndTime:   end.Unix(),
	}

	if err != nil {
		result.Status = "failed"

		// 如果是工作目录错误，添加更明确的提示
		if task.WorkDir != "" && strings.Contains(err.Error(), "chdir") {
			log.Errorf("[工作目录错误] 无法切换到目录: %s\n%s", task.WorkDir, err.Error())
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	} else {
		result.Status = "success"
		result.ExitCode = 0
	}

	a.sendTaskResult(result)
	log.Infof("任务 #%d 执行完成 (%s)", result.TaskID, result.Status)
}

// parseEnvVars 解析环境变量字符串 "KEY1=VALUE1,KEY2=VALUE2"
func (a *Agent) parseEnvVars(envStr string) []string {
	if envStr == "" {
		return nil
	}

	pairs := strings.Split(envStr, ",")
	result := make([]string, 0, len(pairs))

	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		// 解码特殊字符
		pair = strings.ReplaceAll(pair, "{{COMMA}}", ",")
		pair = strings.ReplaceAll(pair, "{{EQUAL}}", "=")
		result = append(result, pair)
	}

	return result
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
	req.Header.Set("X-Machine-ID", a.machineID)

	return a.client.Do(req)
}
