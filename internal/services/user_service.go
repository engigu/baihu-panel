package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"gorm.io/gorm"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (us *UserService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (us *UserService) legacyHashPassword(password string) string {
	hash := sha256.Sum256([]byte(password + constant.Secret))
	return hex.EncodeToString(hash[:])
}

func (us *UserService) CreateUser(username, password, email, role string) *models.User {
	hashedPassword, _ := us.hashPassword(password)
	user := &models.User{
		ID:           utils.GenerateID(),
		Username:     username,
		Password:     hashedPassword,
		Email:        email,
		Role:         role,
		TokenVersion: 1,
	}
	database.DB.Create(user)
	return user
}

func (us *UserService) GetUserByUsername(username string) *models.User {
	var user models.User
	res := database.DB.Where("username = ?", username).Limit(1).Find(&user)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}
	return &user
}

func (us *UserService) GetUserByID(id string) (*models.User, error) {
	var user models.User
	res := database.DB.Where("id = ?", id).Limit(1).Find(&user)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, res.Error
	}
	return &user, nil
}

func (us *UserService) ValidatePassword(user *models.User, password string) bool {
	// 尝试 bcrypt 校验
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err == nil {
		return true
	}

	// 如果 bcrypt 失败，检查是否为旧的 SHA256 格式
	// 旧格式是 64 位十六进制字符串
	if len(user.Password) == 64 && !strings.HasPrefix(user.Password, "$2") {
		if user.Password == us.legacyHashPassword(password) {
			// 校验成功，迁移到 bcrypt
			newHash, err := us.hashPassword(password)
			if err == nil {
				database.DB.Model(user).Update("password", newHash)
			}
			return true
		}
	}

	return false
}

func (us *UserService) EnsureAdminExists() {
	var count int64
	database.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		us.CreateUser("admin", "admin123", "admin@local", "admin")
	}
}

func (us *UserService) AuthenticateUser(username, password string) bool {
	user := us.GetUserByUsername(username)
	if user == nil {
		return false
	}
	return us.ValidatePassword(user, password)
}

func (us *UserService) UpdatePassword(userID string, newPassword string) error {
	hashedPassword, err := us.hashPassword(newPassword)
	if err != nil {
		return err
	}
	// 修改密码时同时失效旧 Token
	return database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password":      hashedPassword,
		"token_version": gorm.Expr("token_version + 1"),
	}).Error
}

func (us *UserService) InvalidateUserTokens(userID string) error {
	return database.DB.Model(&models.User{}).Where("id = ?", userID).Update("token_version", gorm.Expr("token_version + 1")).Error
}

func (us *UserService) UpdateAccount(userID string, newUsername string) error {
	var user models.User
	res := database.DB.Where("id = ?", userID).Limit(1).Find(&user)
	if res.Error != nil || res.RowsAffected == 0 {
		return fmt.Errorf("未找到对应的用户")
	}

	updates := make(map[string]interface{})
	if newUsername != "" && newUsername != user.Username {
		// 检查用户名是否已存在
		var count int64
		database.DB.Model(&models.User{}).Where("username = ? AND id <> ?", newUsername, userID).Count(&count)
		if count > 0 {
			return fmt.Errorf("用户名 [%s] 已被占用", newUsername)
		}
		updates["username"] = newUsername
		// 用户名变更，必须失效所有 Token，因为 Token 中包含 Username 且中间件会校验
		updates["token_version"] = gorm.Expr("token_version + 1")
	}

	if len(updates) == 0 {
		return nil
	}

	return database.DB.Model(&user).Updates(updates).Error
}

func (us *UserService) UpdateOTP(userID string, secret string, enabled bool) error {
	return database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"otp_secret":  secret,
		"otp_enabled": enabled,
	}).Error
}
