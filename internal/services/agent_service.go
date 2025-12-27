package services

import (
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/utils"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentService Agent 服务
type AgentService struct{}

// NewAgentService 创建 Agent 服务
func NewAgentService() *AgentService {
	return &AgentService{}
}

// generateToken 生成随机 Token
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Register Agent 注册（进入待审核状态）
func (s *AgentService) Register(req *models.AgentRegisterRequest, ip string) (*models.Agent, error) {
	// 检查是否已存在同名待审核的 Agent
	var existing models.Agent
	if err := database.DB.Where("name = ? AND status = ?", req.Name, "pending").First(&existing).Error; err == nil {
		// 更新现有记录
		now := models.LocalTime(time.Now())
		database.DB.Model(&existing).Updates(map[string]interface{}{
			"hostname":  req.Hostname,
			"version":   req.Version,
			"ip":        ip,
			"last_seen": now,
		})
		return &existing, nil
	}

	// 创建新的待审核 Agent
	now := models.LocalTime(time.Now())
	agent := &models.Agent{
		Name:     req.Name,
		Hostname: req.Hostname,
		Version:  req.Version,
		IP:       ip,
		Status:   "pending",
		LastSeen: &now,
		Enabled:  true,
	}

	if err := database.DB.Create(agent).Error; err != nil {
		return nil, err
	}

	logger.Infof("[Agent] 新 Agent 注册: %s (%s)", req.Name, ip)
	return agent, nil
}

// Approve 审核通过 Agent，生成 Token
func (s *AgentService) Approve(id uint) (*models.Agent, error) {
	agent := s.GetByID(id)
	if agent == nil {
		return nil, &ServiceError{Message: "Agent 不存在"}
	}

	if agent.Status != "pending" {
		return nil, &ServiceError{Message: "Agent 状态不是待审核"}
	}

	token := generateToken()
	now := models.LocalTime(time.Now())

	if err := database.DB.Model(agent).Updates(map[string]interface{}{
		"token":     token,
		"status":    "online",
		"last_seen": now,
	}).Error; err != nil {
		return nil, err
	}

	agent.Token = token
	agent.Status = "online"
	agent.LastSeen = &now

	logger.Infof("[Agent] Agent 已审核通过: %s (#%d)", agent.Name, agent.ID)
	return agent, nil
}

// Reject 拒绝 Agent
func (s *AgentService) Reject(id uint) error {
	return database.DB.Delete(&models.Agent{}, id).Error
}

// Update 更新 Agent
func (s *AgentService) Update(id uint, name, description string, enabled bool) error {
	return database.DB.Model(&models.Agent{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":        name,
		"description": description,
		"enabled":     enabled,
	}).Error
}

// Delete 删除 Agent
func (s *AgentService) Delete(id uint) error {
	// 检查是否有关联任务
	var count int64
	database.DB.Model(&models.Task{}).Where("agent_id = ?", id).Count(&count)
	if count > 0 {
		return &ServiceError{Message: "该 Agent 下还有关联任务，无法删除"}
	}

	return database.DB.Delete(&models.Agent{}, id).Error
}

// GetByID 根据 ID 获取 Agent
func (s *AgentService) GetByID(id uint) *models.Agent {
	var agent models.Agent
	if err := database.DB.First(&agent, id).Error; err != nil {
		return nil
	}
	return &agent
}

// GetByToken 根据 Token 获取 Agent
func (s *AgentService) GetByToken(token string) *models.Agent {
	var agent models.Agent
	if err := database.DB.Where("token = ?", token).First(&agent).Error; err != nil {
		return nil
	}
	return &agent
}

// List 获取已审核的 Agent 列表
func (s *AgentService) List() []models.Agent {
	var agents []models.Agent
	database.DB.Where("status != ?", "pending").Order("id DESC").Find(&agents)
	return agents
}

// ListPending 获取待审核的 Agent 列表
func (s *AgentService) ListPending() []models.Agent {
	var agents []models.Agent
	database.DB.Where("status = ?", "pending").Order("id DESC").Find(&agents)
	return agents
}

// RegenerateToken 重新生成 Token
func (s *AgentService) RegenerateToken(id uint) (string, error) {
	newToken := generateToken()
	if err := database.DB.Model(&models.Agent{}).Where("id = ?", id).Update("token", newToken).Error; err != nil {
		return "", err
	}
	return newToken, nil
}

// Heartbeat Agent 心跳
func (s *AgentService) Heartbeat(token, ip, version, buildTime, hostname, osType, arch string) (*models.Agent, error) {
	agent := s.GetByToken(token)
	if agent == nil {
		return nil, &ServiceError{Message: "无效的 Token"}
	}

	if !agent.Enabled {
		return nil, &ServiceError{Message: "Agent 已禁用"}
	}

	now := models.LocalTime(time.Now())
	updates := map[string]interface{}{
		"status":    "online",
		"last_seen": now,
		"ip":        ip,
	}
	if version != "" {
		updates["version"] = version
	}
	if buildTime != "" {
		updates["build_time"] = buildTime
	}
	if hostname != "" {
		updates["hostname"] = hostname
	}
	if osType != "" {
		updates["os"] = osType
	}
	if arch != "" {
		updates["arch"] = arch
	}

	database.DB.Model(&models.Agent{}).Where("id = ?", agent.ID).Updates(updates)

	agent.Status = "online"
	agent.LastSeen = &now
	agent.IP = ip
	agent.Version = version
	agent.BuildTime = buildTime
	agent.Hostname = hostname
	agent.OS = osType
	agent.Arch = arch

	return agent, nil
}

// CheckPendingAgent 检查待审核 Agent 的状态（用于 Agent 轮询）
func (s *AgentService) CheckPendingAgent(name, ip string) (*models.Agent, error) {
	var agent models.Agent
	if err := database.DB.Where("name = ? AND ip = ?", name, ip).First(&agent).Error; err != nil {
		return nil, &ServiceError{Message: "Agent 未注册"}
	}

	// 更新最后心跳时间
	now := models.LocalTime(time.Now())
	database.DB.Model(&agent).Update("last_seen", now)

	return &agent, nil
}

// GetTasks 获取 Agent 的任务列表
func (s *AgentService) GetTasks(agentID uint) []models.AgentTask {
	var tasks []models.Task
	database.DB.Where("agent_id = ? AND enabled = ?", agentID, true).Find(&tasks)

	result := make([]models.AgentTask, len(tasks))
	for i, task := range tasks {
		result[i] = models.AgentTask{
			ID:       task.ID,
			Name:     task.Name,
			Command:  task.Command,
			Schedule: task.Schedule,
			Timeout:  task.Timeout,
			WorkDir:  task.WorkDir,
			Envs:     task.Envs,
			Enabled:  task.Enabled,
		}
	}

	return result
}

// ReportResult Agent 上报执行结果
func (s *AgentService) ReportResult(result *models.AgentTaskResult) error {
	// 压缩输出
	compressed, err := utils.CompressToBase64(result.Output)
	if err != nil {
		logger.Errorf("[Agent] 压缩日志失败: %v", err)
		compressed = ""
	}

	taskLog := &models.TaskLog{
		TaskID:   result.TaskID,
		AgentID:  &result.AgentID,
		Command:  result.Command,
		Output:   compressed,
		Status:   result.Status,
		Duration: result.Duration,
		ExitCode: result.ExitCode,
	}

	if err := database.DB.Create(taskLog).Error; err != nil {
		return err
	}

	// 更新任务的 last_run
	database.DB.Model(&models.Task{}).Where("id = ?", result.TaskID).Update("last_run", time.Now())

	// 更新统计
	sendStatsService := NewSendStatsService()
	sendStatsService.IncrementStats(result.TaskID, result.Status)

	logger.Infof("[Agent] 收到任务结果 #%d (agent=%d, status=%s)", result.TaskID, result.AgentID, result.Status)
	return nil
}

// UpdateOfflineAgents 更新离线 Agent 状态（超过 2 分钟无心跳）
func (s *AgentService) UpdateOfflineAgents() {
	cutoff := time.Now().Add(-2 * time.Minute)
	database.DB.Model(&models.Agent{}).
		Where("status = ? AND last_seen < ?", "online", cutoff).
		Update("status", "offline")
}

// GetLatestVersion 获取最新 Agent 版本
func (s *AgentService) GetLatestVersion() string {
	// 优先从 /opt/agent 读取（容器内）
	versionFile := "/opt/agent/version.txt"
	data, err := os.ReadFile(versionFile)
	if err != nil {
		// 回退到 data/agent（本地开发）
		data, err = os.ReadFile("data/agent/version.txt")
		if err != nil {
			return ""
		}
	}
	return strings.TrimSpace(string(data))
}

// GetAvailablePlatforms 获取可用的平台列表
func (s *AgentService) GetAvailablePlatforms() []map[string]string {
	platforms := []map[string]string{}
	
	// 优先从 /opt/agent 读取（容器内）
	agentDir := "/opt/agent"
	files, err := os.ReadDir(agentDir)
	if err != nil {
		// 回退到 data/agent（本地开发）
		agentDir = "data/agent"
		files, err = os.ReadDir(agentDir)
		if err != nil {
			return platforms
		}
	}

	for _, f := range files {
		name := f.Name()
		// baihu-agent-linux-amd64.tar.gz
		if strings.HasPrefix(name, "baihu-agent-") && strings.HasSuffix(name, ".tar.gz") {
			// 去掉 .tar.gz 后缀
			baseName := strings.TrimSuffix(name, ".tar.gz")
			parts := strings.Split(baseName, "-")
			if len(parts) >= 4 {
				platforms = append(platforms, map[string]string{
					"os":       parts[2],
					"arch":     parts[3],
					"filename": name,
				})
			}
		}
	}

	return platforms
}

// GetAgentBinary 获取 Agent 压缩包
func (s *AgentService) GetAgentBinary(osType, arch string) ([]byte, string, error) {
	filename := fmt.Sprintf("baihu-agent-%s-%s.tar.gz", osType, arch)

	// 优先从 /opt/agent 读取（容器内）
	filePath := filepath.Join("/opt/agent", filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// 回退到 data/agent（本地开发）
		filePath = filepath.Join("data/agent", filename)
		data, err = os.ReadFile(filePath)
		if err != nil {
			return nil, "", &ServiceError{Message: "未找到对应平台的 Agent 程序"}
		}
	}

	return data, filename, nil
}

// SetForceUpdate 设置强制更新标志
func (s *AgentService) SetForceUpdate(id uint) error {
	return database.DB.Model(&models.Agent{}).Where("id = ?", id).Update("force_update", true).Error
}

// ClearForceUpdate 清除强制更新标志
func (s *AgentService) ClearForceUpdate(id uint) error {
	return database.DB.Model(&models.Agent{}).Where("id = ?", id).Update("force_update", false).Error
}

// ServiceError 服务错误
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}
