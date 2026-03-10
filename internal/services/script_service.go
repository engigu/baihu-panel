package services

import (
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

type ScriptService struct{}

func NewScriptService() *ScriptService {
	return &ScriptService{}
}

func (ss *ScriptService) CreateScript(name, content string, userID string) *models.Script {
	script := &models.Script{
		ID:      utils.GenerateID(),
		Name:    name,
		Content: models.BigText(content),
		UserID:  userID,
	}
	database.DB.Create(script)
	return script
}

func (ss *ScriptService) GetScriptsByUserID(userID string) []models.Script {
	var scripts []models.Script
	database.DB.Where("user_id = ?", userID).Find(&scripts)
	return scripts
}

func (ss *ScriptService) GetScriptByID(id string) *models.Script {
	var script models.Script
	if err := database.DB.Where("id = ?", id).First(&script).Error; err != nil {
		return nil
	}
	return &script
}

func (ss *ScriptService) UpdateScript(id string, name, content string) *models.Script {
	var script models.Script
	if err := database.DB.Where("id = ?", id).First(&script).Error; err != nil {
		return nil
	}
	script.Name = name
	script.Content = models.BigText(content)
	database.DB.Save(&script)
	return &script
}

func (ss *ScriptService) DeleteScript(id string) bool {
	result := database.DB.Where("id = ?", id).Delete(&models.Script{})
	return result.RowsAffected > 0
}
