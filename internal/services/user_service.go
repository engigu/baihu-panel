package services

import (
	"crypto/sha256"
	"encoding/hex"

	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/models"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (us *UserService) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password + constant.Secret))
	return hex.EncodeToString(hash[:])
}

func (us *UserService) CreateUser(username, password, email, role string) *models.User {
	user := &models.User{
		Username: username,
		Password: us.hashPassword(password),
		Email:    email,
		Role:     role,
	}
	database.DB.Create(user)
	return user
}

func (us *UserService) GetUserByUsername(username string) *models.User {
	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil
	}
	return &user
}

func (us *UserService) ValidatePassword(user *models.User, password string) bool {
	return user.Password == us.hashPassword(password)
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

func (us *UserService) UpdatePassword(userID uint, newPassword string) error {
	return database.DB.Model(&models.User{}).Where("id = ?", userID).Update("password", us.hashPassword(newPassword)).Error
}
