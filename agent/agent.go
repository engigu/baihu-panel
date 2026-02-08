package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gorilla/websocket"
)

// WebSocket 消息类型
const (
	WSTypeHeartbeat     = "heartbeat"
	WSTypeHeartbeatAck  = "heartbeat_ack"
	WSTypeTasks         = "tasks"
	WSTypeTaskResult    = "task_result"
	WSTypeUpdate        = "update"
	WSTypeConnected     = "connected"
	WSTypeDisabled      = "disabled"
	WSTypeEnabled       = "enabled"
	WSTypeFetchTasks    = "fetch_tasks"
	WSTypeTaskLog       = "task_log"
	WSTypeExecute       = "execute"
	WSTypeTaskHeartbeat = "task_heartbeat"
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

func (t *AgentTask) GetID() string {
	return fmt.Sprintf("%d", t.ID)
}

func (t *AgentTask) GetName() string {
	return t.Name
}

func (t *AgentTask) GetCommand() string {
	return t.Command
}

func (t *AgentTask) GetTimeout() int {
	return t.Timeout
}

func (t *AgentTask) GetSchedule() string {
	if t.Schedule != "" {
		return t.Schedule
	}
	return t.Cron
}

type TaskResult struct {
	TaskID    uint   `json:"task_id"`
	LogID     uint   `json:"log_id"`
	AgentID   uint   `json:"agent_id"` // 仅用于 HTTP 上报时后端补充
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
	scheduler     *executor.Scheduler
	cronManager   *executor.CronManager
	tasks         map[uint]*AgentTask // 本地任务缓存，用于执行 lookup
	lastTaskCount int
	mu            sync.RWMutex
	client        *http.Client
	wsConn        *websocket.Conn
	wsMu          sync.Mutex
	stopCh        chan struct{}
	wsStopCh      chan struct{} // 用于停止当前 WebSocket 相关的 goroutine
}

func NewAgent(config *Config, configFile string) *Agent {
	a := &Agent{
		config:     config,
		configFile: configFile,
		machineID:  utils.GenerateMachineID(),
		tasks:      make(map[uint]*AgentTask),
		client:     &http.Client{Timeout: 30 * time.Second},
		stopCh:     make(chan struct{}),
	}

	// 初始化调度器
	handler := &AgentHandler{agent: a}
	schedCfg := executor.SchedulerConfig{
		WorkerCount:  runtime.NumCPU(),
		QueueSize:    100,
		RateInterval: 100 * time.Millisecond,
		Verbose:      true,
	}
	a.scheduler = executor.NewScheduler(schedCfg, handler)
	a.scheduler.SetLogger(logger.NewSchedulerLogger())
	a.cronManager = executor.NewCronManager(a.scheduler)
	a.cronManager.SetLogger(logger.NewSchedulerLogger())

	return a
}

// AgentHandler 实现 executor.SchedulerEventHandler
type AgentHandler struct {
	agent *Agent
}

func (h *AgentHandler) OnTaskScheduled(req *executor.ExecutionRequest) {}

func (h *AgentHandler) OnTaskExecuting(req *executor.ExecutionRequest) (io.Writer, io.Writer, error) {
	if req.LogID > 0 {
		writer := &RealTimeLogWriter{agent: h.agent, logID: req.LogID}
		return writer, writer, nil
	}
	return nil, nil, nil
}

func (h *AgentHandler) OnTaskHeartbeat(req *executor.ExecutionRequest, duration int64) {
	if req.LogID > 0 {
		h.agent.sendWSMessage(WSTypeTaskHeartbeat, map[string]interface{}{
			"log_id":   req.LogID,
			"duration": duration,
		})
	}
}

func (h *AgentHandler) OnTaskStarted(req *executor.ExecutionRequest) {}

func (h *AgentHandler) OnTaskCompleted(req *executor.ExecutionRequest, result *executor.ExecutionResult) {
	var taskID uint
	fmt.Sscanf(req.TaskID, "%d", &taskID)

	h.agent.sendTaskResult(&TaskResult{
		TaskID:    taskID,
		LogID:     result.LogID,
		Command:   req.Command,
		Output:    result.Output,
		Status:    result.Status,
		Duration:  result.Duration,
		ExitCode:  result.ExitCode,
		StartTime: result.StartTime.Unix(),
		EndTime:   result.EndTime.Unix(),
	})
}

func (h *AgentHandler) OnTaskFailed(req *executor.ExecutionRequest, err error) {
	errMsg := fmt.Sprintf("任务执行失败: %v", err)
	// 先发送日志，确保服务端能收到错误信息
	h.agent.sendWSMessage(WSTypeTaskLog, map[string]interface{}{
		"log_id":  req.LogID,
		"content": errMsg,
	})

	var taskID uint
	fmt.Sscanf(req.TaskID, "%d", &taskID)

	h.agent.sendTaskResult(&TaskResult{
		TaskID:    taskID,
		LogID:     req.LogID,
		Command:   req.Command,
		Output:    errMsg,
		Status:    "failed",
		Duration:  0,
		ExitCode:  1,
		StartTime: time.Now().Unix(),
		EndTime:   time.Now().Unix(),
	})
}

func (h *AgentHandler) OnCronNextRun(req *executor.ExecutionRequest, nextRun time.Time) {}

func (a *Agent) Start() error {
	if a.config.Token == "" {
		return fmt.Errorf("缺少令牌，请在配置文件中设置 token")
	}

	logger.Infof("机器识别码: %s", a.machineID[:16]+"...")
	a.scheduler.Start()
	a.cronManager.Start()

	go a.wsLoop()

	logger.Info("Agent 已启动 (时区: Asia/Shanghai, 模式: WebSocket)")
	return nil
}

func (a *Agent) Stop() {
	close(a.stopCh)
	a.closeWS()
	a.cronManager.Stop()
	a.scheduler.Stop()
	logger.Info("Agent 已停止")
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
			logger.Warnf("WebSocket 连接失败: %v，5秒后重试...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		a.readWS()

		logger.Warn("WebSocket 连接断开，5秒后重连...")
		time.Sleep(5 * time.Second)
	}
}

func (a *Agent) connectWS() error {
	serverURL := a.config.ServerURL
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/api/agent/ws?token=%s&machine_id=%s", wsURL, url.QueryEscape(a.config.Token), url.QueryEscape(a.machineID))

	logger.Infof("正在连接 WebSocket: %s", wsURL)
	logger.Infof("Token: %s..., MachineID: %s...", a.config.Token[:8], a.machineID[:16])

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			bodyBytes, _ := io.ReadAll(resp.Body)
			logger.Errorf("WebSocket 握手失败: HTTP %d, Body: %s", resp.StatusCode, string(bodyBytes))
			resp.Body.Close()
		} else {
			logger.Errorf("WebSocket 连接失败: %v", err)
		}
		return err
	}

	a.wsMu.Lock()
	a.wsConn = conn
	a.wsStopCh = make(chan struct{})
	a.wsMu.Unlock()

	logger.Info("WebSocket 已连接")
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
		logger.Info("readWS 退出，准备关闭连接")
		a.closeWS()
	}()

	for {
		a.wsMu.Lock()
		conn := a.wsConn
		a.wsMu.Unlock()

		if conn == nil {
			logger.Warn("readWS: wsConn 为 nil")
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Warnf("WebSocket 读取错误: %v", err)
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
		logger.Info("收到更新指令，开始更新...")
		go a.selfUpdate()
	case WSTypeDisabled:
		logger.Warn("Agent 已被禁用，清空所有任务")
		a.clearAllTasks()
	case WSTypeEnabled:
		logger.Info("Agent 已被启用，主动拉取任务")
		a.fetchTasks()
	case WSTypeExecute:
		a.handleExecute(msg.Data)
	}
}

func (a *Agent) fetchTasks() {
	if err := a.sendWSMessage(WSTypeFetchTasks, map[string]interface{}{}); err != nil {
		logger.Warnf("请求任务列表失败: %v", err)
	}
}

func (a *Agent) handleConnected(data json.RawMessage) {
	var resp struct {
		AgentID         uint                   `json:"agent_id"`
		Name            string                 `json:"name"`
		IsNewAgent      bool                   `json:"is_new_agent"`
		MachineID       string                 `json:"machine_id"`
		SchedulerConfig map[string]interface{} `json:"scheduler_config"`
	}
	json.Unmarshal(data, &resp)

	if resp.IsNewAgent {
		logger.Infof("注册成功: Agent #%d, 机器码: %s", resp.AgentID, a.machineID[:16]+"...")
	} else {
		logger.Infof("连接成功: Agent #%d (已存在), 机器码: %s", resp.AgentID, a.machineID[:16]+"...")
	}

	// 更新调度器配置
	if resp.SchedulerConfig != nil {
		a.updateSchedulerConfig(resp.SchedulerConfig)
	}

	a.fetchTasks()
}

func (a *Agent) updateSchedulerConfig(config map[string]interface{}) {
	// 获取当前配置作为基础
	currentCfg := a.scheduler.GetConfig()
	newCfg := currentCfg

	// 更新配置项
	if val, ok := config["worker_count"]; ok {
		if v, ok := val.(float64); ok { // JSON 数字解析为 float64
			newCfg.WorkerCount = int(v)
		}
	}
	if val, ok := config["queue_size"]; ok {
		if v, ok := val.(float64); ok {
			newCfg.QueueSize = int(v)
		}
	}
	if val, ok := config["rate_interval"]; ok {
		if v, ok := val.(float64); ok {
			newCfg.RateInterval = time.Duration(v) * time.Millisecond
		}
	}

	// 只有当配置发生变化时才重新加载
	// 只有当配置发生变化时才重新加载
	if newCfg != currentCfg {
		logger.Infof("收到调度配置更新: workers=%d, queue=%d, rate=%v",
			newCfg.WorkerCount, newCfg.QueueSize, newCfg.RateInterval)
		a.scheduler.Reload(newCfg)
	} else {
		logger.Infof("当前调度配置: workers=%d, queue=%d, rate=%v",
			newCfg.WorkerCount, newCfg.QueueSize, newCfg.RateInterval)
	}
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
		logger.Infof("发现新版本 %s，开始更新...", resp.LatestVersion)
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
		logger.Infof("任务列表更新: %d -> %d 个任务", a.lastTaskCount, newCount)
		a.lastTaskCount = newCount
	}

	a.updateTasks(resp.Tasks)
}

func (a *Agent) handleExecute(data json.RawMessage) {
	var req struct {
		TaskID uint `json:"task_id"`
		LogID  uint `json:"log_id"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		logger.Errorf("解析立即执行请求失败: %v", err)
		return
	}

	// 查找任务
	a.mu.RLock()
	task, exists := a.tasks[req.TaskID]
	a.mu.RUnlock()

	if !exists {
		logger.Warnf("任务 #%d 不存在，无法执行", req.TaskID)
		return
	}

	// 准备执行请求
	execReq := &executor.ExecutionRequest{
		TaskID:  fmt.Sprintf("%d", task.ID),
		LogID:   req.LogID,
		Name:    task.Name,
		Command: task.Command,
		WorkDir: task.WorkDir,
		Envs:    executor.ParseEnvVars(task.Envs),
		Timeout: task.Timeout,
		Type:    executor.TaskTypeManual,
	}

	// 立即执行任务（加入队列）
	a.scheduler.EnqueueOrExecute(execReq)
}

// RealTimeLogWriter 实时日志写入器，通过 WebSocket 发送日志
type RealTimeLogWriter struct {
	agent *Agent
	logID uint
}

func (w *RealTimeLogWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// 构造消息
	msg := map[string]interface{}{
		"log_id":  w.logID,
		"content": string(p),
	}

	// 发送消息
	if err := w.agent.sendWSMessage(WSTypeTaskLog, msg); err != nil {
		// 如果发送失败，不阻塞程序执行，只记录日志
		return len(p), nil
	}

	return len(p), nil
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
		logger.Warnf("发送消息失败 (%s): %v", msgType, err)
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
		logger.Warnf("发送心跳失败: %v", err)
	}
}

func (a *Agent) sendTaskResult(result *TaskResult) {
	if err := a.sendWSMessage(WSTypeTaskResult, result); err != nil {
		logger.Warnf("发送任务结果失败: %v，尝试 HTTP 上报", err)
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

	// 1. 移除不再存在的任务
	for id := range a.tasks {
		if _, exists := newTasks[id]; !exists {
			a.cronManager.RemoveTask(fmt.Sprintf("%d", id))
			delete(a.tasks, id)
			logger.Infof("移除任务 #%d", id)
		}
	}

	// 2. 添加或更新任务
	for id, task := range newTasks {
		oldTask, exists := a.tasks[id]
		if !exists || oldTask.Schedule != task.Schedule || oldTask.Command != task.Command || oldTask.Enabled != task.Enabled {
			if task.Enabled {
				err := a.cronManager.AddTask(task)
				if err != nil {
					logger.Errorf("调度任务 #%d 失败: %v", id, err)
					continue
				}
				logger.Infof("已调度任务 #%d %s (%s)", id, task.Name, task.GetSchedule())
			} else {
				a.cronManager.RemoveTask(fmt.Sprintf("%d", id))
				logger.Infof("任务 #%d 已禁用", id)
			}
			a.tasks[id] = task
		}
	}
}

func (a *Agent) clearAllTasks() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for id := range a.tasks {
		a.cronManager.RemoveTask(fmt.Sprintf("%d", id))
		logger.Infof("移除任务 #%d", id)
	}

	a.tasks = make(map[uint]*AgentTask)
	a.lastTaskCount = 0
	logger.Info("所有任务已清空")
}

// executeTask 已被 AgentHandler.OnTaskCompleted 代替，此处删除旧实现

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
