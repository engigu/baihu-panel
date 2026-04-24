package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// DataStorage 通用数据存储表
type DataStorage struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	Type      string    `json:"type" gorm:"size:50;index:idx_type_key"`
	Key       string    `json:"key" gorm:"size:255;index:idx_type_key"`
	Data      BigText   `json:"data"`
	CreatedAt LocalTime `json:"created_at"`
	UpdatedAt LocalTime `json:"updated_at"`
}

func (DataStorage) TableName() string {
	return constant.TablePrefix + "data_storages"
}
