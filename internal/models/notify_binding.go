package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// NotifyBinding 事件绑定表
type NotifyBinding struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	Type      string    `json:"type" gorm:"size:20;not null;index"`   // system 或 task
	Event     string    `json:"event" gorm:"size:50;not null;index"`  // 事件类型
	WayID     string    `json:"way_id" gorm:"size:20;not null;index"` // 通知渠道ID
	DataID    string    `json:"data_id" gorm:"size:20;index"`         // 关联ID，系统事件为空，任务事件为任务ID
	Extra     BigText   `json:"extra"`                                // 额外配置（如是否开启日志推送等，对应 BindingExtra 结构）
	CreatedAt LocalTime `json:"created_at"`
	UpdatedAt LocalTime `json:"updated_at"`
}

// BindingExtra 存储在 Extra 字段中的 JSON 配置
type BindingExtra struct {
	EnableLog bool `json:"enable_log"`
	LogLimit  int  `json:"log_limit"` // 日志字数限制，默认 1000
}

func (NotifyBinding) TableName() string {
	return constant.TablePrefix + "notify_bindings"
}
