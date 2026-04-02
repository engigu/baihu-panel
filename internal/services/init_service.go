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
	logger.Info("[Initialize] 开始初始化系统...")

	// 初始化默认设置
	if err := s.settingsService.InitSettings(); err != nil {
		logger.Warnf("[Initialize] 初始化设置失败: %v", err)
	}

	// 创建 UserService
	userService := NewUserService()

	// 创建管理员账号
	s.initializeAdmin(userService)
	
	// 初始化语言环境
	s.initializeLanguages()

	return userService
}

// initializeLanguages 初始化同步语言环境
func (s *InitService) initializeLanguages() {
	logger.Info("[Languages] 开始初始化编程语言环境...")
	miseService := NewMiseService()
	if err := miseService.Sync(); err != nil {
		logger.Errorf("[Languages] 初始化同步语言环境失败: %v", err)
	} else {
		logger.Info("[Languages] 初始化语言环境同步完成")
	}
}

// initializeAdmin 创建管理员账号
func (s *InitService) initializeAdmin(userService *UserService) {
	existingUser := userService.GetUserByUsername("admin")
	if existingUser != nil {
		logger.Info("[Init] 管理员账号已存在，跳过创建")
		return
	}

	password := utils.RandomString(12)
	userService.CreateUser("admin", password, "admin@local", "admin")
	logger.Infof("--------------------------------------------------")
	logger.Infof("[Init] 管理员账号创建成功:")
	logger.Infof("[Init] 用户名: admin")
	logger.Infof("[Init] 密  码: %s", password)
	logger.Infof("[Init] 请妥善保管您的密码，并登录后及时修改。")
	logger.Infof("--------------------------------------------------")
}
