package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UUID      string         `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	Name      string         `json:"name" gorm:"size:255;not null"`
	Value     string         `json:"value" gorm:"type:text"`
	Remark    string         `json:"remark" gorm:"size:500"`
	Hidden    bool           `json:"hidden" gorm:"default:true"`
	UserID    string         `json:"user_id" gorm:"index"`
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (EnvironmentVariable) TableName() string {
	return constant.TablePrefix + "envs"
}

func (ev *EnvironmentVariable) BeforeCreate(tx *gorm.DB) (err error) {
	if ev.UUID == "" {
		ev.UUID = uuid.New().String()
	}
	return
}

// Script represents a script file
type Script struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UUID      string         `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	Name      string         `json:"name" gorm:"size:255;not null"`
	Content   string         `json:"content" gorm:"type:text"`
	UserID    string         `json:"user_id" gorm:"index"`
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Script) TableName() string {
	return constant.TablePrefix + "scripts"
}

func (s *Script) BeforeCreate(tx *gorm.DB) (err error) {
	if s.UUID == "" {
		s.UUID = uuid.New().String()
	}
	return
}
