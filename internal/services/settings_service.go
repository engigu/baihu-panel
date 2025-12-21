package services

import (
	"baihu/internal/cache"
	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/models"
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
			database.DB.Model(&models.Setting{}).Where("section = ? AND key = ?", section, key).Count(&count)
			if count == 0 {
				if err := database.DB.Create(&models.Setting{Section: section, Key: key, Value: value}).Error; err != nil {
					return err
				}
			}
		}
	}
	cache.LoadSiteCache()
	return nil
}

// Get 获取单个设置
func (s *SettingsService) Get(section, key string) string {
	if section == constant.SectionSite {
		return cache.GetSiteCache(key)
	}
	var setting models.Setting
	if err := database.DB.Where("section = ? AND key = ?", section, key).First(&setting).Error; err != nil {
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
	if database.DB.Where("section = ? AND key = ?", section, key).First(&setting).Error != nil {
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
