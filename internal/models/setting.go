package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// Setting 系统设置
type Setting struct {
	ID      string `json:"id" gorm:"primaryKey;size:20"`
	Section string `json:"section" gorm:"size:50;not null;index:idx_section_key"`
	Key     string `json:"key" gorm:"size:100;not null;index:idx_section_key"`
	Value   BigText `json:"value"`
}

func (Setting) TableName() string {
	return constant.TablePrefix + "settings"
}
