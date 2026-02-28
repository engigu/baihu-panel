package services

import (
	"github.com/engigu/baihu-panel/internal/cache"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

type SettingsService struct{}

func NewSettingsService() *SettingsService {
	return &SettingsService{}
}

// InitSettings 初始化默认设置
func (s *SettingsService) InitSettings() error {
	for section, keys := range constant.DefaultSettings {
		for key, value := range keys {
			var count int64
			database.DB.Model(&models.Setting{}).Where("section = ? AND `key` = ?", section, key).Count(&count)
			if count == 0 {
				if err := database.DB.Create(&models.Setting{Section: section, Key: key, Value: value}).Error; err != nil {
					return err
				}
			}
		}
	}
	// 初始化或获取 JWT Secret 密码
	var secCount int64
	database.DB.Model(&models.Setting{}).Where("section = ? AND `key` = ?", constant.SectionSecurity, constant.KeySecret).Count(&secCount)
	var secretValue string
	if secCount == 0 {
		// 先尝试从配置文件读取遗留下来的旧设
		if Config != nil && Config.Security.Secret != "" {
			secretValue = Config.Security.Secret
		} else {
			secretValue = utils.RandomString(32)
		}
		if err := database.DB.Create(&models.Setting{Section: constant.SectionSecurity, Key: constant.KeySecret, Value: secretValue}).Error; err != nil {
			return err
		}
	} else {
		secretValue = s.Get(constant.SectionSecurity, constant.KeySecret)
	}
	constant.Secret = secretValue

	cache.LoadSiteCache()
	return nil
}

// Get 获取单个设置
func (s *SettingsService) Get(section, key string) string {
	if section == constant.SectionSite {
		return cache.GetSiteCache(key)
	}
	var setting models.Setting
	if err := database.DB.Where("section = ? AND `key` = ?", section, key).First(&setting).Error; err != nil {
		if def, ok := constant.DefaultSettings[section][key]; ok {
			return def
		}
		return ""
	}
	return setting.Value
}

// Set 设置单个值
func (s *SettingsService) Set(section, key, value string) error {
	var setting models.Setting
	if database.DB.Where("section = ? AND `key` = ?", section, key).First(&setting).Error != nil {
		return database.DB.Create(&models.Setting{Section: section, Key: key, Value: value}).Error
	}
	return database.DB.Model(&setting).Update("value", value).Error
}

// GetSection 获取整个 section 的设置
func (s *SettingsService) GetSection(section string) map[string]string {
	if section == constant.SectionSite {
		return cache.GetSiteCacheAll()
	}
	result := make(map[string]string)
	if defaults, ok := constant.DefaultSettings[section]; ok {
		for k, v := range defaults {
			result[k] = v
		}
	}
	var settings []models.Setting
	database.DB.Where("section = ?", section).Find(&settings)
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	return result
}

// SetSection 批量设置
func (s *SettingsService) SetSection(section string, values map[string]string) error {
	for key, value := range values {
		if err := s.Set(section, key, value); err != nil {
			return err
		}
	}
	if section == constant.SectionSite {
		cache.SetSiteCacheBatch(values)
	}
	return nil
}
