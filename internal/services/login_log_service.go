package services

import (
	"fmt"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

type LoginLogService struct{}

func NewLoginLogService() *LoginLogService {
	return &LoginLogService{}
}

// Create 创建登录日志
func (s *LoginLogService) Create(username, ip, userAgent, status, message string) error {
	level := constant.LogLevelInfo
	if status != "success" {
		level = constant.LogLevelWarning
	}
	log := &models.AppLog{
		ID:       utils.GenerateID(),
		Category: constant.LogCategoryLoginLog,
		Title:    username,
		Content:  models.BigText(userAgent),
		Level:    level,
		Status:   status,
		RefID:    ip,
		ErrorMsg: models.BigText(message),
	}
	err := database.DB.Create(log).Error
	if err == nil {
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type:    constant.EventAppLogAdded,
			Payload: log,
		})
	}
	return err
}

// SubscribeEvents 注册订阅事件
func (s *LoginLogService) SubscribeEvents(bus *eventbus.EventBus) {
	// 用户登录事件
	bus.Subscribe(constant.EventUserLogin, func(e eventbus.Event) {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return
		}

		username, _ := payload["username"].(string)
		ip, _ := payload["ip"].(string)
		userAgent, _ := payload["userAgent"].(string)
		status, _ := payload["status"].(string)
		message, _ := payload["message"].(string)

		s.Create(username, ip, userAgent, status, message)

		// 如果登录成功，触发系统通知
		if status == "success" {
			bus.Publish(eventbus.Event{
				Type: constant.EventSystemNotice,
				Payload: map[string]interface{}{
					"title":   "登录提醒",
					"content": fmt.Sprintf("用户 %s 已从 IP %s 登录系统", username, ip),
					"level":   constant.LogLevelWarning,
				},
			})
		}
	})

	// 暴力破解防御触发事件
	bus.Subscribe(constant.EventBruteForceLogin, func(e eventbus.Event) {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return
		}

		username, _ := payload["username"].(string)
		ip, _ := payload["ip"].(string)
		userAgent, _ := payload["userAgent"].(string)

		s.Create(username, ip, userAgent, "failed", "尝试次数过多，由于暴力破解防御机制已锁定")

		// 触发系统通知
		bus.Publish(eventbus.Event{
			Type: constant.EventSystemNotice,
			Payload: map[string]interface{}{
				"title":   "系统安全警告",
				"content": fmt.Sprintf("检测到 IP %s 正在尝试暴力破解用户 %s", ip, username),
				"level":   constant.LogLevelError,
			},
		})
	})
}

// List 获取登录日志列表
func (s *LoginLogService) List(page, pageSize int, username string) ([]models.AppLog, int64, error) {
	var logs []models.AppLog
	var total int64

	query := database.DB.Model(&models.AppLog{}).Where("category = ?", constant.LogCategoryLoginLog)
	if username != "" {
		query = query.Where("title LIKE ?", "%"+username+"%")
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
	deadline := time.Now().AddDate(0, 0, -days)
	result := database.DB.Unscoped().Where("category = ? AND created_at < ?", constant.LogCategoryLoginLog, deadline).Delete(&models.AppLog{})
	return result.RowsAffected, result.Error
}
