package models

import (
	"github.com/engigu/baihu-panel/internal/constant"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID        string         `json:"id" gorm:"primaryKey;size:20"`
	Username  string         `json:"username" gorm:"size:100;uniqueIndex;not null"`
	Password  string         `json:"password" gorm:"size:255;not null"`
	Email     string         `json:"email" gorm:"size:255"`
	Role      string         `json:"role" gorm:"size:20;default:user"` // admin, user
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (User) TableName() string {
	return constant.TablePrefix + "users"
}
