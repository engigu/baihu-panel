package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// NotifyFilter 通知匹配/过滤规则表
type NotifyFilter struct {
	ID          string    `json:"id" gorm:"primaryKey;size:20"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Event       string    `json:"event" gorm:"size:50;not null;index"`       // 匹配哪些事件类型 ("all" 或具体事件如 "task_failed")
	Keyword     string    `json:"keyword" gorm:"size:255;not null"`          // 匹配关键字
	MatchField  string    `json:"match_field" gorm:"size:20;not null;index"` // 匹配范围 ("content" 消息正文, "log" 执行日志)
	IsRegex     bool      `json:"is_regex" gorm:"not null"`                  // 是否正则匹配
	Action      string    `json:"action" gorm:"size:20;not null"`            // "silent" (静默/直接拦截), "custom" (自定义并替换)
	CustomTitle string    `json:"custom_title" gorm:"size:255"`              // 自定义通知标题 (Action 为 custom 时使用)
	CustomText  string    `json:"custom_text" gorm:"size:1000"`              // 自定义通知正文 (Action 为 custom 时使用)
	Enabled     bool      `json:"enabled" gorm:"not null;index"`             // 是否启用
	CreatedAt   LocalTime `json:"created_at"`
	UpdatedAt   LocalTime `json:"updated_at"`
}

func (NotifyFilter) TableName() string {
	return constant.TablePrefix + "notify_filters"
}
