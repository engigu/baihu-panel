package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Language struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UUID        string         `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	Plugin      string         `json:"plugin" gorm:"size:100;not null;index"`
	Version     string         `json:"version" gorm:"size:100;not null;index"`
	InstallPath string         `json:"install_path" gorm:"size:255"`
	Source      string         `json:"source" gorm:"size:255"`
	InstalledAt *LocalTime     `json:"installed_at"`
	CreatedAt   LocalTime      `json:"created_at"`
	UpdatedAt   LocalTime      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Language) TableName() string {
	return constant.TablePrefix + "languages"
}

func (l *Language) BeforeCreate(tx *gorm.DB) (err error) {
	if l.UUID == "" {
		l.UUID = uuid.New().String()
	}
	return
}
