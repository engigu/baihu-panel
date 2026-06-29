package models

// NodeDTO 统一的节点传输对象 (合并 Agent 与 InterconnectNode)
type NodeDTO struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Type       string      `json:"type"` // "runner" (原 Agent) 或 "panel" (原子面板)
	Status     string      `json:"status"`
	IP         string      `json:"ip"`
	Hostname   string      `json:"hostname"`
	OS         string      `json:"os"`
	Arch       string      `json:"arch"`
	Version    string      `json:"version"`
	LastSeenAt *LocalTime  `json:"last_seen_at"`
	URL        string      `json:"url,omitempty"` // 仅 Panel 有效
	Remark     string      `json:"remark"`        // 统一 Remark / Description
	Metrics    NodeMetrics `json:"metrics"`       // 统一系统指标
	Enabled    bool        `json:"enabled"`
	CreatedAt  LocalTime   `json:"created_at"`
	UpdatedAt  LocalTime   `json:"updated_at"`
}

// FromAgent 将 Agent 转换为 NodeDTO
func FromAgent(agent *Agent) *NodeDTO {
	if agent == nil {
		return nil
	}
	enabled := true
	if agent.Enabled != nil {
		enabled = *agent.Enabled
	}

	return &NodeDTO{
		ID:         agent.ID,
		Name:       agent.Name,
		Type:       "runner",
		Status:     agent.Status,
		IP:         agent.IP,
		Hostname:   agent.Hostname,
		OS:         agent.OS,
		Arch:       agent.Arch,
		Version:    agent.Version,
		LastSeenAt: agent.LastSeen,
		Remark:     agent.Description,
		Enabled:    enabled,
		CreatedAt:  agent.CreatedAt,
		UpdatedAt:  agent.UpdatedAt,
	}
}

// FromInterconnectNode 将 InterconnectNode 转换为 NodeDTO
func FromInterconnectNode(node *InterconnectNode) *NodeDTO {
	if node == nil {
		return nil
	}
	return &NodeDTO{
		ID:         node.ID,
		Name:       node.Name,
		Type:       "panel",
		Status:     node.Status,
		URL:        node.URL,
		Remark:     node.Remark,
		Metrics:    node.Metrics,
		LastSeenAt: node.LastHeartbeatAt,
		Enabled:    true, // 互联面板默认启用
		CreatedAt:  node.CreatedAt,
		UpdatedAt:  node.UpdatedAt,
	}
}
