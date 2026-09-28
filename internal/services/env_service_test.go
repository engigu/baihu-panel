package services

import (
	"os"
	"reflect"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
)

func TestEnvService_FormatEnvVars_Equivalence(t *testing.T) {
	// 设置 32 字节密钥并初始化
	os.Setenv("BAIHU_SECRET_KEY", "12345678901234567890123456789012")
	utils.InitSecretKey()

	es := NewEnvService()

	boolPtr := func(b bool) *bool { return &b }

	// 加密一个假 Secret 准备测试
	secretPlain := "my-secret-key"
	secretEncrypted, err := utils.Encrypt(secretPlain)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	testCases := []struct {
		name           string
		envs           []models.EnvironmentVariable
		expectedVars   []string
		expectedSecs   []string
		expectedNonSec []string
	}{
		{
			name:           "空切片",
			envs:           nil,
			expectedVars:   nil,
			expectedSecs:   nil,
			expectedNonSec: nil,
		},
		{
			name: "普通变量与禁用变量",
			envs: []models.EnvironmentVariable{
				{Name: "VAR1", Value: "val1", Type: constant.EnvTypeNormal, Enabled: boolPtr(true)},
				{Name: "VAR2", Value: "val2", Type: constant.EnvTypeNormal, Enabled: boolPtr(false)},
				{Name: "VAR3", Value: "val3", Type: constant.EnvTypeNormal, Enabled: nil}, // 默认为启用
			},
			expectedVars:   []string{"VAR1=val1", "VAR3=val3"},
			expectedSecs:   nil,
			expectedNonSec: []string{"VAR1=val1", "VAR3=val3"},
		},
		{
			name: "同名变量聚合(&拼接)",
			envs: []models.EnvironmentVariable{
				{Name: "MULTI", Value: "first", Type: constant.EnvTypeNormal, Enabled: boolPtr(true)},
				{Name: "MULTI", Value: "disabled", Type: constant.EnvTypeNormal, Enabled: boolPtr(false)},
				{Name: "MULTI", Value: "second", Type: constant.EnvTypeNormal, Enabled: boolPtr(true)},
			},
			expectedVars:   []string{"MULTI=first&second"},
			expectedSecs:   nil,
			expectedNonSec: []string{"MULTI=first&second"},
		},
		{
			name: "包含 Secret 变量",
			envs: []models.EnvironmentVariable{
				{Name: "NORMAL", Value: "hello", Type: constant.EnvTypeNormal, Enabled: boolPtr(true)},
				{Name: "SECRET_PASS", Value: models.BigText(secretEncrypted), Type: constant.EnvTypeSecret, Enabled: boolPtr(true)},
				{Name: "SECRET_DISABLED", Value: models.BigText(secretEncrypted), Type: constant.EnvTypeSecret, Enabled: boolPtr(false)},
			},
			// formatEnvVarsAndSecrets 结果（包含解密后的 Secret，禁用项不包含）
			expectedVars: []string{"NORMAL=hello", "SECRET_PASS=" + secretPlain},
			expectedSecs: []string{secretPlain},
			// formatEnvVars 结果（过滤所有 Secret）
			expectedNonSec: []string{"NORMAL=hello"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. 测试 formatEnvVarsAndSecrets
			resVars, resSecs := es.formatEnvVarsAndSecrets(tc.envs)
			if !reflect.DeepEqual(resVars, tc.expectedVars) {
				t.Errorf("formatEnvVarsAndSecrets vars 不匹配: got %v, want %v", resVars, tc.expectedVars)
			}
			if !reflect.DeepEqual(resSecs, tc.expectedSecs) {
				t.Errorf("formatEnvVarsAndSecrets secrets 不匹配: got %v, want %v", resSecs, tc.expectedSecs)
			}

			// 2. 测试 formatEnvVars
			resNonSec := es.formatEnvVars(tc.envs)
			if !reflect.DeepEqual(resNonSec, tc.expectedNonSec) {
				t.Errorf("formatEnvVars 不匹配: got %v, want %v", resNonSec, tc.expectedNonSec)
			}
		})
	}
}
