package services

import (
	"errors"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/deps"
)

type DependencyService struct{}

func NewDependencyService() *DependencyService {
	return &DependencyService{}
}

// List 获取依赖列表
func (s *DependencyService) List(language, langVersion string) ([]models.Dependency, error) {
	var results []models.Dependency
	query := database.DB
	if language != "" {
		query = query.Where("language = ?", language)
	}
	if langVersion != "" {
		query = query.Where("lang_version = ?", langVersion)
	}
	err := query.Order("id desc").Find(&results).Error
	return results, err
}

// Create 创建依赖记录
func (s *DependencyService) Create(dep *models.Dependency) error {
	// 检查是否已存在（名称、语言、版本 必须完全匹配）
	var existing models.Dependency
	err := database.DB.Where("name = ? AND language = ? AND lang_version = ?", dep.Name, dep.Language, dep.LangVersion).First(&existing).Error
	if err == nil {
		return errors.New("依赖已存在")
	}
	return database.DB.Create(dep).Error
}

// Delete 删除依赖记录
func (s *DependencyService) Delete(id int) error {
	return database.DB.Delete(&models.Dependency{}, id).Error
}

// Install 安装依赖
func (s *DependencyService) Install(dep *models.Dependency) error {
	m := deps.GetManager(dep.Language)
	if m == nil {
		return errors.New("不支持的依赖类型: " + dep.Language)
	}
	return m.Install(dep)
}

// Uninstall 卸载依赖
func (s *DependencyService) Uninstall(dep *models.Dependency) error {
	m := deps.GetManager(dep.Language)
	if m == nil {
		return errors.New("不支持的依赖类型: " + dep.Language)
	}
	return m.Uninstall(dep)
}

// GetInstalledPackages 获取已安装的包列表
func (s *DependencyService) GetInstalledPackages(language, langVersion string) ([]models.Dependency, error) {
	m := deps.GetManager(language)
	if m == nil {
		return nil, errors.New("不支持的依赖类型: " + language)
	}
	return m.GetInstalledPackages(language, langVersion)
}

// GetInstallCommand 获取安装命令
func (s *DependencyService) GetInstallCommand(dep *models.Dependency) (string, error) {
	m := deps.GetManager(dep.Language)
	if m == nil {
		return "", errors.New("不支持的依赖类型: " + dep.Language)
	}
	return m.GetInstallCommand(dep)
}

// GetReinstallAllCommand 获取全部重装命令
func (s *DependencyService) GetReinstallAllCommand(language, langVersion string) (string, error) {
	m := deps.GetManager(language)
	if m == nil {
		return "", errors.New("不支持的依赖类型: " + language)
	}

	deps_list, err := s.List(language, langVersion)
	if err != nil {
		return "", err
	}

	return m.GetReinstallAllCommand(deps_list)
}

// GetVerifyCommand 获取环境验证命令
func (s *DependencyService) GetVerifyCommand(language, langVersion string) (string, error) {
	m := deps.GetManager(language)
	if m == nil {
		return "", errors.New("不支持的依赖类型: " + language)
	}
	return m.GetVerifyCommand(langVersion)
}
