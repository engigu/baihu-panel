package models

import (
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
)

// Dependency 依赖包模型
type Dependency struct {
	ID          string    `json:"id" gorm:"primaryKey;size:20"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Version     string    `json:"version" gorm:"size:50"`
	Language    string    `json:"language" gorm:"size:100;index"`     // 关联语言 (node, python...)
	LangVersion string    `json:"lang_version" gorm:"size:100;index"` // 关联语言版本
	Remark      string    `json:"remark" gorm:"size:255"`
	Log         BigText   `json:"log"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Dependency) TableName() string {
	return constant.TablePrefix + "deps"
}
