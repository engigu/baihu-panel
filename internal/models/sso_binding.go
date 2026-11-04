package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// UserSSOBinding 用户单点登录与第三方身份绑定映射
type UserSSOBinding struct {
	ID           string    `json:"id" gorm:"primaryKey;size:20"`
	UserID       string    `json:"user_id" gorm:"size:20;index;not null"`
	Username     string    `json:"username" gorm:"size:100;index;not null"`
	ChannelType  string    `json:"channel_type" gorm:"size:20;index;not null"` // oidc 或 oauth2
	Subject      string    `json:"subject" gorm:"size:255;index;not null"`      // 第三方唯一标识 (Sub / ID)
	IdentityName string    `json:"identity_name" gorm:"size:255"`              // 第三方用户名或显示名称
	Email        string    `json:"email" gorm:"size:255;index"`                // 第三方邮箱
	BindType     string    `json:"bind_type" gorm:"size:20;default:manual"`    // manual (手动添加) 或 oauth (授权绑定)
	CreatedAt    LocalTime `json:"created_at"`
	UpdatedAt    LocalTime `json:"updated_at"`
}

func (UserSSOBinding) TableName() string {
	return constant.TablePrefix + "user_sso_bindings"
}

