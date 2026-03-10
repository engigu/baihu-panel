package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// AppLog 统一应用日志与通知记录
type AppLog struct {
	ID          string     `json:"id" gorm:"primaryKey;size:20"`
	Category    string     `json:"category" gorm:"size:50;index;not null"` // 大类：constant.LogCategorySystemNotice(系统通知), constant.LogCategoryPushLog(推送记录) 等
	Title       string     `json:"title" gorm:"size:255"`                  // 消息标题
	Content     BigText    `json:"content"`                                // 详细内容/Payload
	Level       string     `json:"level" gorm:"size:20;index"`             // 级别：constant.LogLevelInfo, constant.LogLevelWarning, constant.LogLevelError
	Status      string     `json:"status" gorm:"size:20;index"`            // 状态：系统通知为 constant.LogStatusRead/constant.LogStatusUnread，推送为 constant.LogStatusSuccess/constant.LogStatusFailed
	RefID       string     `json:"ref_id" gorm:"size:50;index"`            // 关联对象ID（选填，比如绑定的通知渠道ID、任务ID等）
	ErrorMsg    BigText    `json:"error_msg"`                              // 执行错误信息详情
	CreatedAt   LocalTime  `json:"created_at" gorm:"index"`
	ReadAt      *LocalTime `json:"read_at"`               // 已读时间（仅对通知生效）
	ChannelName string     `json:"channel_name" gorm:"-"` // 推送记录的关联渠道名称（动态查询）
}

func (AppLog) TableName() string {
	return constant.TablePrefix + "app_logs"
}
