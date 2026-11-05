package services

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
		{
			name: "历史未加密明文机密需纳入脱敏列表",
			envs: []models.EnvironmentVariable{
				{Name: "LEGACY_PLAIN", Value: "plain-legacy-secret", Type: constant.EnvTypeSecret, Enabled: boolPtr(true)},
			},
			// 明文机密无法解密，但仍应作为机密注入并纳入脱敏列表，避免泄露到日志
			expectedVars:   []string{"LEGACY_PLAIN=plain-legacy-secret"},
			expectedSecs:   []string{"plain-legacy-secret"},
			expectedNonSec: nil,
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

func setupUpsertSecretTestDB(t *testing.T) *EnvService {
	t.Helper()
	os.Setenv("BAIHU_SECRET_KEY", "12345678901234567890123456789012")
	utils.InitSecretKey()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.EnvironmentVariable{}, &models.DataStorage{}, &models.DataRelation{}, &models.Agent{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	database.DB = db
	return NewEnvService()
}

func TestEnvService_UpsertSecretByName_CreateAndUpdate(t *testing.T) {
	es := setupUpsertSecretTestDB(t)

	created, wasCreated, err := es.UpsertSecretByName("user-1", "API_TOKEN", "token-v1", "first", "test", true)
	if err != nil || !wasCreated {
		t.Fatalf("创建机密失败: created=%v, err=%v", wasCreated, err)
	}
	if created.Type != constant.EnvTypeSecret || created.Value == "token-v1" {
		t.Fatalf("创建结果类型或加密值错误: type=%s, value=%s", created.Type, created.Value)
	}

	updated, wasCreated, err := es.UpsertSecretByName("user-1", "API_TOKEN", "token-v2", "second", "test", false)
	if err != nil || wasCreated {
		t.Fatalf("更新机密失败: created=%v, err=%v", wasCreated, err)
	}
	if updated.ID != created.ID || updated.Value == "token-v2" || updated.Type != constant.EnvTypeSecret {
		t.Fatalf("更新结果错误: id=%s/%s, value=%s, type=%s", created.ID, updated.ID, updated.Value, updated.Type)
	}
	plain, err := utils.Decrypt(string(updated.Value))
	if err != nil || plain != "token-v2" {
		t.Fatalf("机密未正确加密保存: plain=%s, err=%v", plain, err)
	}
}

func TestEnvService_UpsertSecretByName_ValidationAndConflicts(t *testing.T) {
	es := setupUpsertSecretTestDB(t)

	if _, _, err := es.UpsertSecretByName("user-1", "BAD-NAME", "value", "", "", true); !errors.Is(err, ErrInvalidSecretName) {
		t.Fatalf("期望非法名称错误，实际: %v", err)
	}
	if _, _, err := es.UpsertSecretByName("user-1", "API_TOKEN", "", "", "", true); !errors.Is(err, ErrEmptySecretValue) {
		t.Fatalf("期望空值错误，实际: %v", err)
	}
	es.CreateEnvVar("NORMAL_NAME", "value", "", constant.EnvTypeNormal, true, true, "user-1")
	if _, _, err := es.UpsertSecretByName("user-1", "NORMAL_NAME", "value", "", "", true); err == nil {
		t.Fatalf("期望同名普通变量冲突，实际成功")
	}

	es.CreateEnvVar("DUPLICATE", "value-1", "", constant.EnvTypeSecret, true, true, "user-1")
	es.CreateEnvVar("DUPLICATE", "value-2", "", constant.EnvTypeSecret, true, true, "user-1")
	if _, _, err := es.UpsertSecretByName("user-1", "DUPLICATE", "value-3", "", "", true); !errors.Is(err, ErrDuplicateSecretName) {
		t.Fatalf("期望重复名称错误，实际: %v", err)
	}
}

func TestEnvService_UpsertSecretByName_LegacySecretName(t *testing.T) {
	es := setupUpsertSecretTestDB(t)

	legacy := es.CreateEnvVar("legacy-token", "legacy-value", "", constant.EnvTypeSecret, true, true, "user-1")
	if legacy == nil {
		t.Fatalf("创建存量小写机密失败")
	}

	updated, wasCreated, err := es.UpsertSecretByName("user-1", "legacy-token", "legacy-updated", "", "", true)
	if err != nil || wasCreated {
		t.Fatalf("更新存量小写机密失败: wasCreated=%v, err=%v", wasCreated, err)
	}
	if updated.ID != legacy.ID || updated.Name != legacy.Name {
		t.Fatalf("存量机密更新错误: id=%s/%s, name=%s/%s", legacy.ID, updated.ID, legacy.Name, updated.Name)
	}
	plain, err := utils.Decrypt(string(updated.Value))
	if err != nil || plain != "legacy-updated" {
		t.Fatalf("存量机密未正确加密更新: plain=%s, err=%v", plain, err)
	}

	created, wasCreated, err := es.UpsertSecretByName("user-1", "new_token", "new-value", "", "", true)
	if err != nil || !wasCreated || created.Type != constant.EnvTypeSecret {
		t.Fatalf("新小写机密应可创建: wasCreated=%v, err=%v", wasCreated, err)
	}

	if _, _, err := es.UpsertSecretByName("user-1", "new-token", "value", "", "", true); !errors.Is(err, ErrInvalidSecretName) {
		t.Fatalf("期望新非法机密命名错误，实际: %v", err)
	}
}

func TestEnvService_UpsertSecretByName_UserScope(t *testing.T) {
	es := setupUpsertSecretTestDB(t)
	if _, wasCreated, err := es.UpsertSecretByName("user-1", "API_TOKEN", "token-1", "", "", true); err != nil || !wasCreated {
		t.Fatalf("创建 user-1 机密失败: %v", err)
	}
	if _, wasCreated, err := es.UpsertSecretByName("user-2", "API_TOKEN", "token-2", "", "", true); err != nil || !wasCreated {
		t.Fatalf("user-2 不应看到 user-1 的机密: created=%v, err=%v", wasCreated, err)
	}
}
