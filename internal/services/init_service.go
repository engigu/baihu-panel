package services

import (
	"baihu/internal/logger"
)

type InitService struct {
	settingsService *SettingsService
}

func NewInitService(settingsService *SettingsService) *InitService {
	return &InitService{
		settingsService: settingsService,
	}
}

// Initialize 执行系统初始化，返回 UserService
func (s *InitService) Initialize() *UserService {
	logger.Info("开始初始化系统...")

	// 初始化默认设置
	if err := s.settingsService.InitSettings(); err != nil {
		logger.Warnf("初始化设置失败: %v", err)
	}

	// 创建 UserService
	userService := NewUserService()

	// 创建管理员账号
	s.initializeAdmin(userService)

	return userService
}

// initializeAdmin 创建管理员账号
func (s *InitService) initializeAdmin(userService *UserService) {
	existingUser := userService.GetUserByUsername("admin")
	if existingUser != nil {
		logger.Info("管理员账号已存在，跳过创建")
		return
	}

	userService.CreateUser("admin", "123456", "admin@local", "admin")
	logger.Info("管理员账号创建成功: admin / 123456")
}
