package services

import (
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
)

type ScriptService struct{}

func NewScriptService() *ScriptService {
	return &ScriptService{}
}

func (ss *ScriptService) CreateScript(name, content, userUUID string) *models.Script {
	script := &models.Script{
		Name:    name,
		Content: content,
		UserID:  userUUID,
	}
	database.DB.Create(script)
	return script
}

func (ss *ScriptService) GetScriptsByUserID(userUUID string) []models.Script {
	var scripts []models.Script
	database.DB.Where("user_id = ?", userUUID).Find(&scripts)
	return scripts
}

func (ss *ScriptService) GetScriptByID(id int) *models.Script {
	var script models.Script
	if err := database.DB.First(&script, id).Error; err != nil {
		return nil
	}
	return &script
}

func (ss *ScriptService) UpdateScript(id int, name, content string) *models.Script {
	var script models.Script
	if err := database.DB.First(&script, id).Error; err != nil {
		return nil
	}
	script.Name = name
	script.Content = content
	database.DB.Save(&script)
	return &script
}

func (ss *ScriptService) DeleteScript(id int) bool {
	result := database.DB.Delete(&models.Script{}, id)
	return result.RowsAffected > 0
}
