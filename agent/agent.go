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
}

// generateMachineID 生成机器识别码
func generateMachineID() string {
	var parts []string

	if hostname, err := os.Hostname(); err == nil {
		parts = append(parts, hostname)
	}

	if interfaces, err := net.Interfaces(); err == nil {
		var macs []string
		for _, iface := range interfaces {
			if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
				continue
			}
			macs = append(macs, iface.HardwareAddr.String())
		}
		sort.Strings(macs)
		parts = append(parts, macs...)
	}

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

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	a.wsMu.Lock()
	a.wsConn = conn
	a.wsMu.Unlock()

	log.Info("WebSocket 已连接")
	a.sendHeartbeat()
	go a.heartbeatLoop()

	return nil
}

func (a *Agent) closeWS() {
	a.wsMu.Lock()
	defer a.wsMu.Unlock()
	if a.wsConn != nil {
		a.wsConn.Close()
		a.wsConn = nil
	}
}

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
