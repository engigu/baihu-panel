package vo

import (
	"github.com/engigu/baihu-panel/internal/models"
)

// AgentVO 代理视图对象
type AgentVO struct {
	ID          uint              `json:"id"`
	UUID        string            `json:"uuid"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	LastSeen    *models.LocalTime `json:"last_seen"`
	IP          string            `json:"ip"`
	Version     string            `json:"version"`
	BuildTime   string            `json:"build_time"`
	Hostname    string            `json:"hostname"`
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	ForceUpdate bool              `json:"force_update"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   models.LocalTime  `json:"created_at"`
	UpdatedAt   models.LocalTime  `json:"updated_at"`
	// 隐藏 Token 和 MachineID
}

// ToAgentVO 将 Agent 模型转换为 AgentVO
func ToAgentVO(agent *models.Agent) *AgentVO {
	if agent == nil {
		return nil
	}
	return &AgentVO{
		ID:          agent.ID,
		UUID:        agent.UUID,
		Name:        agent.Name,
		Description: agent.Description,
		Status:      agent.Status,
		LastSeen:    agent.LastSeen,
		IP:          agent.IP,
		Version:     agent.Version,
		BuildTime:   agent.BuildTime,
		Hostname:    agent.Hostname,
		OS:          agent.OS,
		Arch:        agent.Arch,
		ForceUpdate: agent.ForceUpdate,
		Enabled:     agent.Enabled,
		CreatedAt:   agent.CreatedAt,
		UpdatedAt:   agent.UpdatedAt,
	}
}

// ToAgentVOList 将 Agent 模型列表转换为 AgentVO 列表
func ToAgentVOList(agents []*models.Agent) []*AgentVO {
	if agents == nil {
		return nil
	}
	vos := make([]*AgentVO, len(agents))
	for i, a := range agents {
		vos[i] = ToAgentVO(a)
	}
	return vos
}

// ToAgentVOListFromModels 将 Agent 模型列表转换为 AgentVO 列表
func ToAgentVOListFromModels(agents []models.Agent) []*AgentVO {
	vos := make([]*AgentVO, len(agents))
	for i := range agents {
		vos[i] = ToAgentVO(&agents[i])
	}
	return vos
}

// AgentTokenVO 代理令牌视图对象
type AgentTokenVO struct {
	ID        uint              `json:"id"`
	UUID      string            `json:"uuid"`
	Token     string            `json:"token"`
	Remark    string            `json:"remark"`
	MaxUses   int               `json:"max_uses"`
	UsedCount int               `json:"used_count"`
	ExpiresAt *models.LocalTime `json:"expires_at"`
	Enabled   bool              `json:"enabled"`
	CreatedAt models.LocalTime  `json:"created_at"`
}

// ToAgentTokenVO 将 AgentToken 模型转换为 AgentTokenVO
func ToAgentTokenVO(token *models.AgentToken) *AgentTokenVO {
	if token == nil {
		return nil
	}
	return &AgentTokenVO{
		ID:        token.ID,
		UUID:      token.UUID,
		Token:     token.Token,
		Remark:    token.Remark,
		MaxUses:   token.MaxUses,
		UsedCount: token.UsedCount,
		ExpiresAt: token.ExpiresAt,
		Enabled:   token.Enabled,
		CreatedAt: token.CreatedAt,
	}
}

// ToAgentTokenVOList 将 AgentToken 模型列表转换为 AgentTokenVO 列表
func ToAgentTokenVOList(tokens []*models.AgentToken) []*AgentTokenVO {
	if tokens == nil {
		return nil
	}
	vos := make([]*AgentTokenVO, len(tokens))
	for i, t := range tokens {
		vos[i] = ToAgentTokenVO(t)
	}
	return vos
}

// ToAgentTokenVOListFromModels 将 AgentToken 模型列表转换为 AgentTokenVO 列表
func ToAgentTokenVOListFromModels(tokens []models.AgentToken) []*AgentTokenVO {
	vos := make([]*AgentTokenVO, len(tokens))
	for i := range tokens {
		vos[i] = ToAgentTokenVO(&tokens[i])
	}
	return vos
}
