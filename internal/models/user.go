package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// User represents a system user
type User struct {
	ID           string    `json:"id" gorm:"primaryKey;size:20"`
	Username     string    `json:"username" gorm:"size:100;uniqueIndex;not null"`
	Password     string    `json:"password" gorm:"size:255;not null"`
	Email        string    `json:"email" gorm:"size:255"`
	Role         string    `json:"role" gorm:"size:20;default:user"` // admin, user
	TokenVersion int       `json:"-" gorm:"default:1"`               // 用于 JWT 失效校验
	CreatedAt    LocalTime `json:"created_at"`
	UpdatedAt    LocalTime `json:"updated_at"`
}

func (User) TableName() string {
	return constant.TablePrefix + "users"
}
