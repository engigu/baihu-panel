package cache

import (
	"sync"

	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/models"
)

// siteCache 站点设置内存缓存
var (
	siteCache     = make(map[string]string)
	siteCacheMu   sync.RWMutex
	siteCacheInit bool
)

// LoadSiteCache 从数据库加载站点设置到缓存
func LoadSiteCache() {
	siteCacheMu.Lock()
	defer siteCacheMu.Unlock()

	// 先填充默认值
	if defaults, ok := constant.DefaultSettings[constant.SectionSite]; ok {
		for k, v := range defaults {
			siteCache[k] = v
		}
	}

	// 从数据库加载覆盖
	var settings []models.Setting
	database.DB.Where("section = ?", constant.SectionSite).Find(&settings)
	for _, setting := range settings {
		siteCache[setting.Key] = setting.Value
	}
	siteCacheInit = true
}

// ensureSiteCache 确保站点缓存已初始化
func ensureSiteCache() {
	siteCacheMu.RLock()
	init := siteCacheInit
	siteCacheMu.RUnlock()

	if !init {
		LoadSiteCache()
	}
}

// GetSiteCache 从缓存获取站点设置
func GetSiteCache(key string) string {
	ensureSiteCache()

	siteCacheMu.RLock()
	defer siteCacheMu.RUnlock()

	if val, ok := siteCache[key]; ok {
		return val
	}
	if def, ok := constant.DefaultSettings[constant.SectionSite][key]; ok {
		return def
	}
	return ""
}

// SetSiteCache 更新缓存中的站点设置
func SetSiteCache(key, value string) {
	siteCacheMu.Lock()
	siteCache[key] = value
	siteCacheMu.Unlock()
}

// GetSiteCacheAll 获取整个站点设置缓存
func GetSiteCacheAll() map[string]string {
	ensureSiteCache()

	siteCacheMu.RLock()
	defer siteCacheMu.RUnlock()

	result := make(map[string]string)
	for k, v := range siteCache {
		result[k] = v
	}
	return result
}

// SetSiteCacheBatch 批量更新缓存
func SetSiteCacheBatch(values map[string]string) {
	siteCacheMu.Lock()
	for k, v := range values {
		siteCache[k] = v
	}
	siteCacheMu.Unlock()
}
