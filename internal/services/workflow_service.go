package services

import (
	"errors"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"gorm.io/gorm"
)

type WorkflowService struct{}

func NewWorkflowService() *WorkflowService {
	return &WorkflowService{}
}

// List 获取工作流列表
func (s *WorkflowService) List(page, pageSize int, name string) ([]models.Workflow, int64, error) {
	var workflows []models.Workflow
	var total int64
	query := database.DB.Model(&models.Workflow{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&workflows).Error
	return workflows, total, err
}

// GetByID 根据 ID 获取工作流
func (s *WorkflowService) GetByID(id uint) (*models.Workflow, error) {
	var workflow models.Workflow
	err := database.DB.First(&workflow, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("workflow not found")
		}
		return nil, err
	}
	return &workflow, nil
}

// Create 创建工作流
func (s *WorkflowService) Create(workflow *models.Workflow) error {
	return database.DB.Create(workflow).Error
}

// Update 更新工作流
func (s *WorkflowService) Update(workflow *models.Workflow) error {
	// 使用 Updates 可以只更新非零值或指定的列
	return database.DB.Model(&models.Workflow{}).Where("id = ?", workflow.ID).Updates(map[string]interface{}{
		"name":        workflow.Name,
		"description": workflow.Description,
		"schedule":    workflow.Schedule,
		"enabled":     workflow.Enabled,
		"flow_data":   workflow.FlowData,
		"next_run":    workflow.NextRun, // 更新调度时间
	}).Error
}

// Delete 删除工作流
func (s *WorkflowService) Delete(id uint) error {
	return database.DB.Delete(&models.Workflow{}, id).Error
}

// ToggleStatus 切换工作流状态
func (s *WorkflowService) ToggleStatus(id uint, enabled bool) error {
	return database.DB.Model(&models.Workflow{}).Where("id = ?", id).Update("enabled", enabled).Error
}
