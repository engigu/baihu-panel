package services

import (
	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/services/tasks"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AgentService Agent 服务
type AgentService struct{}

// NewAgentService 创建 Agent 服务
func NewAgentService() *AgentService {
	return &AgentService{}
}

// generateToken 生成随机 Token（64位十六进制）
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ========== 令牌管理 ==========

// CreateToken 创建令牌
func (s *AgentService) CreateToken(remark string, maxUses int, expiresAt *time.Time) (*models.AgentToken, error) {
	var expires *models.LocalTime
	if expiresAt != nil {
		t := models.LocalTime(*expiresAt)
		expires = &t
	}

	token := generateToken()

	agentToken := &models.AgentToken{
		Token:     token,
		Remark:    remark,
		MaxUses:   maxUses,
		ExpiresAt: expires,
		Enabled:   true,
	}

	if err := database.DB.Create(agentToken).Error; err != nil {
		return nil, err
	}

	logger.Infof("[Agent] 创建令牌: %s (max_uses=%d)", token[:8]+"...", maxUses)
	return agentToken, nil
}

// ListTokens 获取令牌列表
func (s *AgentService) ListTokens() []models.AgentToken {
	var tokens []models.AgentToken
	database.DB.Order("id DESC").Find(&tokens)
	return tokens
}

// DeleteToken 删除令牌
func (s *AgentService) DeleteToken(id uint) error {
	return database.DB.Delete(&models.AgentToken{}, id).Error
}

// ValidateToken 验证令牌
func (s *AgentService) ValidateToken(token string) (*models.AgentToken, error) {
	var agentToken models.AgentToken
	if err := database.DB.Where("token = ?", token).First(&agentToken).Error; err != nil {
		return nil, &ServiceError{Message: "无效的令牌"}
	}

	if !agentToken.Enabled {
		return nil, &ServiceError{Message: "令牌已禁用"}
	}

	// 检查使用次数
	if agentToken.MaxUses > 0 && agentToken.UsedCount >= agentToken.MaxUses {
		return nil, &ServiceError{Message: "令牌已达到使用上限"}
	}

	// 检查过期时间
	if agentToken.ExpiresAt != nil && time.Time(*agentToken.ExpiresAt).Before(time.Now()) {
		return nil, &ServiceError{Message: "令牌已过期"}
	}

	return &agentToken, nil
}

// UseToken 使用令牌（增加使用计数）
func (s *AgentService) UseToken(id uint) {
	database.DB.Model(&models.AgentToken{}).Where("id = ?", id).UpdateColumn("used_count", gorm.Expr("used_count + 1"))
}

// ========== Agent 注册 ==========

// RegisterByToken 通过令牌注册 Agent（首次 WebSocket 连接时调用）
// 返回: agent, isNewAgent, error
func (s *AgentService) RegisterByToken(token string, machineID string, ip string) (*models.Agent, bool, error) {
	// 验证令牌
	agentToken, err := s.ValidateToken(token)
	if err != nil {
		return nil, false, err
	}

	// 如果提供了 machine_id，先检查是否已存在
	if machineID != "" {
		var existing models.Agent
		if err := database.DB.Where("machine_id = ?", machineID).First(&existing).Error; err == nil {
			// 已存在，更新 token 和状态，复用已有 Agent
			now := models.LocalTime(time.Now())
			database.DB.Model(&existing).Updates(map[string]interface{}{
				"token":     token,
				"ip":        ip,
				"status":    "online",
				"last_seen": now,
			})
			s.UseToken(agentToken.ID)
			logger.Infof("[Agent] Agent #%d 通过 machine_id 复用 (%s)", existing.ID, machineID[:8]+"...")
			return &existing, false, nil
		}
	}

	// 创建 Agent，使用令牌作为认证 Token
	now := models.LocalTime(time.Now())
	agent := &models.Agent{
		Name:      fmt.Sprintf("agent-%d", time.Now().Unix()),
		Token:     token,
		MachineID: machineID,
		IP:        ip,
		Status:    "online",
		LastSeen:  &now,
		Enabled:   true,
	}

	if err := database.DB.Create(agent).Error; err != nil {
		return nil, false, err
	}

	s.UseToken(agentToken.ID)
	logger.Infof("[Agent] Agent 通过令牌注册: #%d (%s)", agent.ID, ip)
	return agent, true, nil
}

// Register Agent 注册（必须使用令牌）- 保留兼容旧版本
func (s *AgentService) Register(req *models.AgentRegisterRequest, ip string) (*models.Agent, string, error) {
	// 必须提供令牌
	if req.Token == "" {
		return nil, "", &ServiceError{Message: "缺少令牌"}
	}

	agentToken, err := s.ValidateToken(req.Token)
	if err != nil {
		return nil, "", err
	}

	// 检查是否已存在同名 Agent
	var existing models.Agent
	if err := database.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, "", &ServiceError{Message: "Agent 名称已存在"}
	}

	// 创建新 Agent，使用令牌作为认证 Token
	now := models.LocalTime(time.Now())
	agent := &models.Agent{
		Name:      req.Name,
		Token:     req.Token,
		Hostname:  req.Hostname,
		Version:   req.Version,
		BuildTime: req.BuildTime,
		IP:        ip,
		Status:    "online",
		LastSeen:  &now,
		Enabled:   true,
	}

	if err := database.DB.Create(agent).Error; err != nil {
		return nil, "", err
	}

	s.UseToken(agentToken.ID)
	logger.Infof("[Agent] Agent 注册成功: %s (%s)", req.Name, ip)
	return agent, req.Token, nil
}

// Update 更新 Agent
func (s *AgentService) Update(id uint, name, description string, enabled bool) error {
	return database.DB.Model(&models.Agent{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":        name,
		"description": description,
		"enabled":     enabled,
	}).Error
}

// Delete 删除 Agent（物理删除）
func (s *AgentService) Delete(id uint) error {
	// 检查是否有关联任务
	var count int64
	database.DB.Model(&models.Task{}).Where("agent_id = ?", id).Count(&count)
	if count > 0 {
		return &ServiceError{Message: "该 Agent 下还有关联任务，无法删除"}
	}

	return database.DB.Unscoped().Delete(&models.Agent{}, id).Error
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

// GetByMachineID 根据 MachineID 获取 Agent
func (s *AgentService) GetByMachineID(machineID string) *models.Agent {
	var agent models.Agent
	if err := database.DB.Where("machine_id = ?", machineID).First(&agent).Error; err != nil {
		return nil
	}
	return &agent
}

// List 获取 Agent 列表
func (s *AgentService) List() []models.Agent {
	var agents []models.Agent
	database.DB.Order("id DESC").Find(&agents)
	return agents
}

// RegenerateToken 重新生成 Token - 已废弃，保留空实现避免路由错误
func (s *AgentService) RegenerateToken(id uint) (string, error) {
	return "", &ServiceError{Message: "此功能已禁用"}
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

// GetTasks 获取 Agent 的任务列表
func (s *AgentService) GetTasks(agentID uint) []models.AgentTask {
	var tasks []models.Task
	database.DB.Where("agent_id = ? AND enabled = ?", agentID, true).Find(&tasks)

	result := make([]models.AgentTask, len(tasks))
	for i, task := range tasks {
		// 将环境变量 ID 转换为实际的环境变量键值对
		envVarsStr := s.buildEnvVarsString(task.Envs)
		
		result[i] = models.AgentTask{
			ID:       task.ID,
			Name:     task.Name,
			Command:  task.Command,
			Schedule: task.Schedule,
			Timeout:  task.Timeout,
			WorkDir:  task.WorkDir,
			Envs:     envVarsStr, // 传递 "KEY1=VALUE1,KEY2=VALUE2" 格式
			Enabled:  task.Enabled,
		}
	}

	return result
}

// buildEnvVarsString 将环境变量 ID 列表转换为键值对字符串
func (s *AgentService) buildEnvVarsString(envIDs string) string {
	if envIDs == "" {
		return ""
	}

	var envVars []models.EnvironmentVariable
	ids := strings.Split(envIDs, ",")
	database.DB.Where("id IN ?", ids).Find(&envVars)

	if len(envVars) == 0 {
		return ""
	}

	// 构建 "KEY1=VALUE1,KEY2=VALUE2" 格式
	pairs := make([]string, 0, len(envVars))
	for _, env := range envVars {
		// 对值进行转义，避免特殊字符问题
		encodedValue := strings.ReplaceAll(env.Value, ",", "{{COMMA}}")
		encodedValue = strings.ReplaceAll(encodedValue, "=", "{{EQUAL}}")
		pairs = append(pairs, fmt.Sprintf("%s=%s", env.Name, encodedValue))
	}
	return strings.Join(pairs, ",")
}

// ReportResult Agent 上报执行结果
func (s *AgentService) ReportResult(result *models.AgentTaskResult) error {
	// 获取依赖的服务
	agentWSManager := GetAgentWSManager()
	sendStatsService := NewSendStatsService()
	taskExecutionService := tasks.NewTaskExecutionService(agentWSManager, sendStatsService)
	
	// 使用统一的结果处理流程
	return taskExecutionService.ProcessAgentResult(result)
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

// GetLatestBuildTime 获取最新 Agent 构建时间
func (s *AgentService) GetLatestBuildTime() string {
	return constant.BuildTime
}

// CheckNeedUpdate 检查 Agent 是否需要更新
// 比较 version 和 build_time，任一不同则需要更新
func (s *AgentService) CheckNeedUpdate(agentVersion, agentBuildTime string) bool {
	latestVersion := s.GetLatestVersion()
	latestBuildTime := s.GetLatestBuildTime()

	// 如果没有最新版本信息，不需要更新
	if latestVersion == "" {
		return false
	}

	// 如果 Agent 没有版本信息，需要更新
	if agentVersion == "" {
		return true
	}

	// 版本不同，需要更新
	if agentVersion != latestVersion {
		return true
	}

	// 版本相同但构建时间不同，也需要更新
	if latestBuildTime != "" && latestBuildTime != "unknown" && agentBuildTime != "" && agentBuildTime != latestBuildTime {
		return true
	}

	return false
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
