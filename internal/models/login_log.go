package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// LoginLog 登录日志
type LoginLog struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	Username  string    `json:"username" gorm:"size:100;index;not null"`
	IP        string    `json:"ip" gorm:"size:50"`
	UserAgent string    `json:"user_agent" gorm:"size:500"`
	Status    string    `json:"status" gorm:"size:20;index"` // success, failed
	Message   string    `json:"message" gorm:"size:255"`
	CreatedAt LocalTime `json:"created_at" gorm:"index"`
}

func (LoginLog) TableName() string {
	return constant.TablePrefix + "login_logs"
}
