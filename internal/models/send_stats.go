package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SendStats 任务执行统计
type SendStats struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	UUID   string `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	TaskID string `json:"task_id" gorm:"uniqueIndex:idx_task_day_status"`
	Day    string `json:"day" gorm:"size:10;uniqueIndex:idx_task_day_status"` // 格式: 2006-01-02
	Status string `json:"status" gorm:"size:20;uniqueIndex:idx_task_day_status"`
	Num    int    `json:"num" gorm:"default:0"`
}

func (SendStats) TableName() string {
	return constant.TablePrefix + "send_stats"
}

func (ss *SendStats) BeforeCreate(tx *gorm.DB) (err error) {
	if ss.UUID == "" {
		ss.UUID = uuid.New().String()
	}
	return
}
