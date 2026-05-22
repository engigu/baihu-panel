package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

type Language struct {
	ID          string     `json:"id" gorm:"primaryKey;size:20"`
	Plugin      string     `json:"plugin" gorm:"size:100;not null;index"`
	Version     string     `json:"version" gorm:"size:100;not null;index"`
	InstallPath string     `json:"install_path" gorm:"size:255"`
	Source      string     `json:"source" gorm:"size:255"`
	InstalledAt *LocalTime `json:"installed_at"`
	CreatedAt   LocalTime  `json:"created_at"`
	UpdatedAt   LocalTime  `json:"updated_at"`
}

func (Language) TableName() string {
	return constant.TablePrefix + "languages"
}
