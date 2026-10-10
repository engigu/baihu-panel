package memopt

import (
	"os"
	"strconv"

	"github.com/engigu/baihu-panel/internal/cache"
	"github.com/engigu/baihu-panel/internal/constant"
)

// Config 容器内存自适应回收策略配置
type Config struct {
	Enabled          bool    // 智能自适应回收总开关
	WatermarkRate    float64 // 内存警戒水位线比例 (0.0 ~ 1.0)
	ThresholdMB      int     // 触发缓存清理的阈值 (MB)
	TaskFinishedTrim bool    // 任务结束后是否执行尾部回收
}

// GetConfig 获取当前生效的内存配置
// 优先级：系统环境变量硬覆盖 > 数据库/系统设置缓存 (SiteCache) > 默认常量
func GetConfig() Config {
	cfg := Config{
		Enabled:          true,
		WatermarkRate:    constant.DefaultMemWatermarkRate,
		ThresholdMB:      constant.DefaultCacheMaxMB,
		TaskFinishedTrim: true,
	}

	// 1. 从系统设置缓存读取用户在 UI 界面配置的参数
	if val := cache.GetSiteCache(constant.KeyCacheTrimEnabled); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Enabled = b
		}
	}
	if val := cache.GetSiteCache(constant.KeyMemWatermarkRate); val != "" {
		if v, err := strconv.ParseFloat(val, 64); err == nil && v > 0 {
			rate := v
			if rate > 1.0 {
				rate = rate / 100.0 // 前端输入的是 80 表示 80%
			}
			if rate >= 0.10 && rate <= 0.99 {
				cfg.WatermarkRate = rate
			}
		}
	}
	if val := cache.GetSiteCache(constant.KeyCacheMaxMB); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			cfg.ThresholdMB = v
		}
	}
	if val := cache.GetSiteCache(constant.KeyTaskFinishedTrim); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.TaskFinishedTrim = b
		}
	}

	// 2. 环境变量硬覆盖（若容器启动参数明确指定，具备最高优先级）
	if envVal := os.Getenv(constant.EnvKeyCacheMaxMB); envVal != "" {
		if v, err := strconv.Atoi(envVal); err == nil && v > 0 {
			cfg.ThresholdMB = v
		}
	}
	if envVal := os.Getenv(constant.EnvKeyMemWatermarkRate); envVal != "" {
		if v, err := strconv.ParseFloat(envVal, 64); err == nil && v > 0 {
			rate := v
			if rate > 1.0 {
				rate = rate / 100.0
			}
			if rate >= 0.10 && rate <= 0.99 {
				cfg.WatermarkRate = rate
			}
		}
	}

	return cfg
}

// GetTrimSpec 获取后台定时巡检 Cron 表达式
// 优先级：系统环境变量 > 数据库/系统设置缓存 > 默认常量 (@every 5m)
func GetTrimSpec() string {
	if spec := os.Getenv(constant.EnvKeyCacheTrimSpec); spec != "" {
		return spec
	}
	if dbSpec := cache.GetSiteCache(constant.KeyCacheTrimSpec); dbSpec != "" {
		return dbSpec
	}
	return constant.DefaultCacheTrimSpec
}
