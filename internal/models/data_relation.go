package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// DataRelation 通用关联关系表
type DataRelation struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	DataID    string    `json:"data_id" gorm:"size:20;index"`
	RelateID  string    `json:"relate_id" gorm:"size:20;index"`
	Type      string    `json:"type" gorm:"size:50;index"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt LocalTime `json:"created_at"`
}

func (DataRelation) TableName() string {
	return constant.TablePrefix + "data_relations"
}
