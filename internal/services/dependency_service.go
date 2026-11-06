package services

import (
	"errors"
	"strings"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/deps"
	"github.com/engigu/baihu-panel/internal/utils"
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
	var existing models.Dependency
	if dep.ID != "" {
		res := database.DB.Where("id = ?", dep.ID).Limit(1).Find(&existing)
		if res.Error == nil && res.RowsAffected > 0 {
			dep.CreatedAt = existing.CreatedAt
			return database.DB.Save(dep).Error
		}
	}

	res := database.DB.Where("name = ? AND language = ? AND lang_version = ?", dep.Name, dep.Language, dep.LangVersion).Limit(1).Find(&existing)
	if res.Error == nil && res.RowsAffected > 0 {
		dep.ID = existing.ID
		dep.CreatedAt = existing.CreatedAt
		return database.DB.Save(dep).Error
	}

	if dep.ID == "" {
		dep.ID = utils.GenerateID()
	}
	return database.DB.Create(dep).Error
}

// Delete 删除依赖记录
func (s *DependencyService) Delete(id string) error {
	return database.DB.Where("id = ?", id).Delete(&models.Dependency{}).Error
}

// Install 安装依赖
func (s *DependencyService) Install(dep *models.Dependency) error {
	m := deps.GetManager(dep.Language)
	if m == nil {
		return errors.New("不支持的依赖类型: " + dep.Language)
	}
	if err := m.Install(dep); err != nil {
		return err
	}
	if dep.Version == "" {
		if ver, err := m.GetInstalledVersion(dep.Name, dep.LangVersion); err == nil && ver != "" {
			dep.Version = ver
		} else if pkgs, err := m.GetInstalledPackages(dep.Language, dep.LangVersion); err == nil {
			targetName := strings.ToLower(dep.Name)
			for _, pkg := range pkgs {
				if strings.ToLower(pkg.Name) == targetName && pkg.Version != "" {
					dep.Version = pkg.Version
					break
				}
			}
		}
	}
	return nil
}

// GetInstalledVersion 获取已安装单个包的版本
func (s *DependencyService) GetInstalledVersion(language, name, langVersion string) (string, error) {
	m := deps.GetManager(language)
	if m == nil {
		return "", errors.New("不支持的依赖类型: " + language)
	}
	ver, err := m.GetInstalledVersion(name, langVersion)
	if err == nil && ver != "" {
		return ver, nil
	}
	pkgs, err := m.GetInstalledPackages(language, langVersion)
	if err != nil {
		return "", err
	}
	targetName := strings.ToLower(name)
	for _, pkg := range pkgs {
		if strings.ToLower(pkg.Name) == targetName && pkg.Version != "" {
			return pkg.Version, nil
		}
	}
	return "", nil
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

// GetBatchInstallCommand 获取批量安装命令
func (s *DependencyService) GetBatchInstallCommand(depsList []models.Dependency) (string, error) {
	if len(depsList) == 0 {
		return "", errors.New("依赖包列表不能为空")
	}

	firstDep := depsList[0]
	m := deps.GetManager(firstDep.Language)
	if m == nil {
		return "", errors.New("不支持的依赖类型: " + firstDep.Language)
	}

	return m.GetBatchInstallCommand(depsList)
}

// ImportDependencies 批量导入依赖并自动入库去重
func (s *DependencyService) ImportDependencies(depsList []models.Dependency) ([]models.Dependency, error) {
	var imported []models.Dependency
	for i := range depsList {
		dep := &depsList[i]
		if err := s.Create(dep); err == nil {
			imported = append(imported, *dep)
		}
	}
	return imported, nil
}
