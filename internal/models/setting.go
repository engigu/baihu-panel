package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Setting 系统设置
type Setting struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	UUID    string `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	Section string `json:"section" gorm:"size:50;not null;index:idx_section_key"`
	Key     string `json:"key" gorm:"size:100;not null;index:idx_section_key"`
	Value   string `json:"value" gorm:"type:text"`
}

func (Setting) TableName() string {
	return constant.TablePrefix + "settings"
}

func (s *Setting) BeforeCreate(tx *gorm.DB) (err error) {
	if s.UUID == "" {
		s.UUID = uuid.New().String()
	}
	return
}
