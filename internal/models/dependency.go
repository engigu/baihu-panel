package models

import (
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Dependency 依赖包模型
type Dependency struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	UUID        string    `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Version     string    `json:"version" gorm:"size:50"`
	Language    string    `json:"language" gorm:"size:100;index"`     // 关联语言 (node, python...)
	LangVersion string    `json:"lang_version" gorm:"size:100;index"` // 关联语言版本
	Remark      string    `json:"remark" gorm:"size:255"`
	Log         string    `json:"log" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Dependency) TableName() string {
	return constant.TablePrefix + "deps"
}

func (d *Dependency) BeforeCreate(tx *gorm.DB) (err error) {
	if d.UUID == "" {
		d.UUID = uuid.New().String()
	}
	return
}
