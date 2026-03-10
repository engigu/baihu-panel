package models

import (
	"github.com/engigu/baihu-panel/internal/constant"

	"gorm.io/gorm"
)

// NotifyBinding 事件绑定表
type NotifyBinding struct {
	ID        string         `json:"id" gorm:"primaryKey;size:20"`
	Type      string         `json:"type" gorm:"size:20;not null;index"`  // system 或 task
	Event     string         `json:"event" gorm:"size:50;not null;index"` // 事件类型
	WayID     string         `json:"way_id" gorm:"size:20;not null;index"` // 通知渠道ID
	DataID    string         `json:"data_id" gorm:"size:20;index"`        // 关联ID，系统事件为空，任务事件为任务ID
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (NotifyBinding) TableName() string {
	return constant.TablePrefix + "notify_bindings"
}
