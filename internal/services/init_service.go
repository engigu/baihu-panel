package services

import (
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/utils"
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

// initializeLanguages 初始化同步语言环境
func (s *InitService) initializeLanguages() {
	logger.Info("开始初始化编程语言环境...")
	miseService := NewMiseService()
	if err := miseService.Sync(); err != nil {
		logger.Errorf("初始化同步语言环境失败: %v", err)
	} else {
		logger.Info("初始化语言环境同步完成")
	}
}

// initializeAdmin 创建管理员账号
func (s *InitService) initializeAdmin(userService *UserService) {
	existingUser := userService.GetUserByUsername("admin")
	if existingUser != nil {
		logger.Info("管理员账号已存在，跳过创建")
		return
	}

	password := utils.RandomString(12)
	userService.CreateUser("admin", password, "admin@local", "admin")
	logger.Infof("--------------------------------------------------")
	logger.Infof("管理员账号创建成功:")
	logger.Infof("用户名: admin")
	logger.Infof("密  码: %s", password)
	logger.Infof("请妥善保管您的密码，并登录后及时修改。")
	logger.Infof("--------------------------------------------------")
}
