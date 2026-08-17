package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Value     BigText   `json:"value"`
	Remark    string    `json:"remark" gorm:"size:500"`
	Type      string    `json:"type" gorm:"size:20;default:'normal'"`
	Hidden    *bool     `json:"hidden" gorm:"default:true"`
	Enabled   *bool     `json:"enabled" gorm:"default:true"`
	UserID    string    `json:"user_id" gorm:"size:20;index"`
	Tags      string    `json:"-" gorm:"-"`
	CreatedAt LocalTime `json:"created_at"`
	UpdatedAt LocalTime `json:"updated_at"`
}

func (EnvironmentVariable) TableName() string {
	return constant.TablePrefix + "envs"
}

// Script represents a script file
type Script struct {
	ID        string    `json:"id" gorm:"primaryKey;size:20"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Content   BigText   `json:"content"`
	UserID    string    `json:"user_id" gorm:"size:20;index"`
	CreatedAt LocalTime `json:"created_at"`
	UpdatedAt LocalTime `json:"updated_at"`
}

func (Script) TableName() string {
	return constant.TablePrefix + "scripts"
}
