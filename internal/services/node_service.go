package services

import (
	"fmt"

	"github.com/engigu/baihu-panel/internal/models"
)

type NodeService struct {
	agentService        *AgentService
	interconnectService *InterconnectService
}

func NewNodeService() *NodeService {
	return &NodeService{
		agentService:        NewAgentService(),
		interconnectService: NewInterconnectService(),
	}
}

// List 获取所有节点（混合了 Runner 和 Panel）
func (s *NodeService) List() ([]*models.NodeDTO, error) {
	var list []*models.NodeDTO

	// 1. 获取所有的 Agent (Runner)
	agents := s.agentService.List()
	for i := range agents {
		list = append(list, models.FromAgent(&agents[i]))
	}

	// 2. 获取所有的 Interconnect (Panel)
	panels, err := s.interconnectService.GetNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interconnect nodes: %w", err)
	}
	for _, p := range panels {
		list = append(list, models.FromInterconnectNode(p))
	}

	return list, nil
}

// CreateNode 创建一个节点（根据类型路由）
func (s *NodeService) CreateNode(nodeType, name, url, token, remark string) (any, error) {
	if nodeType == "panel" {
		return s.interconnectService.CreateNode(name, url, token, remark)
	} else if nodeType == "runner" {
		// Runner 模式：创建用于注册 Agent 的 Token。
		// 返回一个 Token 结构体以供客户端部署使用。
		return s.agentService.CreateToken(remark, 0, nil)
	}
	return nil, fmt.Errorf("unsupported node type: %s", nodeType)
}

// UpdateNode 更新节点数据
func (s *NodeService) UpdateNode(id, nodeType, name, remark string, enabled bool) (any, error) {
	if nodeType == "panel" {
		return s.interconnectService.UpdateNode(id, name, "", "", remark)
	} else if nodeType == "runner" {
		// Agent (Runner) 更新
		err := s.agentService.Update(id, name, remark, enabled, models.AgentSchedulerConfig{})
		if err != nil {
			return nil, err
		}
		return s.agentService.GetByID(id), nil
	}
	return nil, fmt.Errorf("unsupported node type: %s", nodeType)
}

// DeleteNode 删除节点
func (s *NodeService) DeleteNode(id, nodeType string) error {
	if nodeType == "panel" {
		return s.interconnectService.DeleteNode(id)
	} else if nodeType == "runner" {
		return s.agentService.Delete(id)
	}
	return fmt.Errorf("unsupported node type: %s", nodeType)
}
