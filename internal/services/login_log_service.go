package services

import (
	"baihu/internal/database"
	"baihu/internal/models"
)

type LoginLogService struct{}

func NewLoginLogService() *LoginLogService {
	return &LoginLogService{}
}

// Create 创建登录日志
func (s *LoginLogService) Create(username, ip, userAgent, status, message string) error {
	log := &models.LoginLog{
		Username:  username,
		IP:        ip,
		UserAgent: userAgent,
		Status:    status,
		Message:   message,
	}
	return database.DB.Create(log).Error
}

// List 获取登录日志列表
func (s *LoginLogService) List(page, pageSize int, username string) ([]models.LoginLog, int64, error) {
	var logs []models.LoginLog
	var total int64

	query := database.DB.Model(&models.LoginLog{})
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// CleanOldLogs 清理指定天数前的日志
func (s *LoginLogService) CleanOldLogs(days int) (int64, error) {
	result := database.DB.Exec("DELETE FROM "+models.LoginLog{}.TableName()+" WHERE created_at < datetime('now', ?)", "-"+string(rune(days))+" days")
	return result.RowsAffected, result.Error
}
