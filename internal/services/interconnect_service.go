package services

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

type InterconnectService struct{}

func NewInterconnectService() *InterconnectService {
	return &InterconnectService{}
}

func (s *InterconnectService) GetNodes() ([]*models.InterconnectNode, error) {
	var nodes []*models.InterconnectNode
	err := database.DB.Find(&nodes).Error
	return nodes, err
}

func (s *InterconnectService) GetNodeByID(id string) (*models.InterconnectNode, error) {
	var node models.InterconnectNode
	err := database.DB.Where("id = ?", id).First(&node).Error
	return &node, err
}

func (s *InterconnectService) CreateNode(name, url, token, remark string) (*models.InterconnectNode, error) {
	nodeID := utils.GenerateID()
	if url == "" {
		url = "tunnel://" + nodeID
	}
	node := &models.InterconnectNode{
		ID:        nodeID,
		Name:      name,
		URL:       url,
		Token:     strings.ToLower(token),
		Remark:    remark,
		CreatedAt: models.Now(),
		UpdatedAt: models.Now(),
	}
	err := database.DB.Create(node).Error
	return node, err
}

func (s *InterconnectService) UpdateNode(id, name, url, token, remark string) (*models.InterconnectNode, error) {
	node, err := s.GetNodeByID(id)
	if err != nil {
		return nil, err
	}
	node.Name = name
	if url != "" {
		node.URL = url
	}
	node.Token = token
	node.Remark = remark
	node.UpdatedAt = models.Now()

	err = database.DB.Save(node).Error
	return node, err
}

func (s *InterconnectService) DeleteNode(id string) error {
	return database.DB.Where("id = ?", id).Delete(&models.InterconnectNode{}).Error
}

func (s *InterconnectService) GetNodeByToken(token string) (*models.InterconnectNode, error) {
	var node models.InterconnectNode
	err := database.DB.Where("token = ?", token).First(&node).Error
	return &node, err
}

func (s *InterconnectService) UpdateNodeMonitorData(id string, metrics models.NodeMetrics) error {
	now := models.Now()
	return database.DB.Model(&models.InterconnectNode{}).
		Where("id = ?", id).
		Select("status", "metrics", "last_heartbeat_at", "updated_at").
		Updates(models.InterconnectNode{
			Status:          "online",
			Metrics:         metrics,
			LastHeartbeatAt: &now,
			UpdatedAt:       now,
		}).Error
}
