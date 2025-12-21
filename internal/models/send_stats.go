package models

import (
	"time"

	"baihu/internal/constant"
)

// SendStats 任务执行统计
type SendStats struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    uint      `json:"task_id" gorm:"index"`
	Status    string    `json:"status" gorm:"size:20;not null"` // success, failed
	Num       int       `json:"num" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
}

func (SendStats) TableName() string {
	return constant.TablePrefix + "send_stats"
}
