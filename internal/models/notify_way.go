package models

import (
	"github.com/engigu/baihu-panel/internal/constant"

	"gorm.io/gorm"
)

// NotifyWay 消息推送渠道
type NotifyWay struct {
	ID        string         `json:"id" gorm:"primaryKey;size:20"`
	Name      string         `json:"name" gorm:"size:100;not null"`
	Type      string         `json:"type" gorm:"size:50;not null;index"`
	Config    BigText        `json:"config"`
	Enabled   bool           `json:"enabled" gorm:"default:true;index"`
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (NotifyWay) TableName() string {
	return constant.TablePrefix + "notify_ways"
}
