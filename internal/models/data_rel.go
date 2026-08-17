package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// DataRelation 通用数据关联表
type DataRelation struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	DataID    string    `json:"data_id" gorm:"size:20;index;not null"`
	RelateID  string    `json:"relate_id" gorm:"size:20;index;not null"`
	Type      string    `json:"type" gorm:"size:50;index;not null"`
	CreatedAt LocalTime `json:"created_at"`
	UpdatedAt LocalTime `json:"updated_at"`
}

func (DataRelation) TableName() string {
	return constant.TablePrefix + "data_relations"
}

// DataStorage 通用数据存储表
type DataStorage struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	Type      string    `json:"type" gorm:"size:50;index;not null"`
	Name      string    `json:"name" gorm:"size:255;index;not null"`
	Data      BigText   `json:"data"`
	CreatedAt LocalTime `json:"created_at"`
	UpdatedAt LocalTime `json:"updated_at"`
}

func (DataStorage) TableName() string {
	return constant.TablePrefix + "data_storages"
}
