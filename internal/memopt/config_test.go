package memopt

import (
	"os"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
)

func TestGetConfig_DefaultsAndOverrides(t *testing.T) {
	// 1. 测试默认值
	cfg := GetConfig()
	if !cfg.Enabled {
		t.Errorf("期望默认 Enabled 为 true, 实际为 %v", cfg.Enabled)
	}
	if cfg.ThresholdMB != constant.DefaultCacheMaxMB {
		t.Errorf("期望默认 ThresholdMB 为 %d, 实际为 %d", constant.DefaultCacheMaxMB, cfg.ThresholdMB)
	}
	if cfg.WatermarkRate != constant.DefaultMemWatermarkRate {
		t.Errorf("期望默认 WatermarkRate 为 %v, 实际为 %v", constant.DefaultMemWatermarkRate, cfg.WatermarkRate)
	}
	if !cfg.TaskFinishedTrim {
		t.Errorf("期望默认 TaskFinishedTrim 为 true, 实际为 %v", cfg.TaskFinishedTrim)
	}

	// 2. 测试环境变量百分比格式覆盖 (如 BH_MEM_WATERMARK_RATE=85)
	os.Setenv(constant.EnvKeyMemWatermarkRate, "85")
	os.Setenv(constant.EnvKeyCacheMaxMB, "120")
	defer func() {
		os.Unsetenv(constant.EnvKeyMemWatermarkRate)
		os.Unsetenv(constant.EnvKeyCacheMaxMB)
	}()

	cfgOverridden := GetConfig()
	if cfgOverridden.ThresholdMB != 120 {
		t.Errorf("期望环境变量覆盖 ThresholdMB 为 120, 得到 %d", cfgOverridden.ThresholdMB)
	}
	if cfgOverridden.WatermarkRate != 0.85 {
		t.Errorf("期望环境变量 85 转换为 0.85, 得到 %v", cfgOverridden.WatermarkRate)
	}

	// 3. 测试环境变量小数格式覆盖 (如 BH_MEM_WATERMARK_RATE=0.75)
	os.Setenv(constant.EnvKeyMemWatermarkRate, "0.75")
	cfgDecimal := GetConfig()
	if cfgDecimal.WatermarkRate != 0.75 {
		t.Errorf("期望环境变量 0.75 为 0.75, 得到 %v", cfgDecimal.WatermarkRate)
	}
}

func TestGetTrimSpec_Fallback(t *testing.T) {
	spec := GetTrimSpec()
	if spec == "" {
		t.Error("期望 GetTrimSpec 返回非空表达式")
	}

	// 环境变量覆盖
	os.Setenv(constant.EnvKeyCacheTrimSpec, "@every 10m")
	defer os.Unsetenv(constant.EnvKeyCacheTrimSpec)

	if GetTrimSpec() != "@every 10m" {
		t.Errorf("期望环境变量返回 @every 10m, 得到 %s", GetTrimSpec())
	}
}
