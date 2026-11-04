package database

import (
	"testing"

	"github.com/engigu/baihu-panel/internal/database/migrate_scripts"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 验证启动迁移时，自动清理修复存量数据库中 otp_secret 为空且标记开启两步验证的破损用户数据
func TestRepairCorruptedOtpUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	DB = db
	_ = db.AutoMigrate(&models.User{})

	// 插入测试数据
	brokenUser := models.User{
		ID:         "user_broken",
		Username:   "broken_admin",
		Password:   "hash",
		OtpEnabled: true,
		OtpSecret:  "",
	}
	validUser := models.User{
		ID:         "user_valid",
		Username:   "valid_admin",
		Password:   "hash",
		OtpEnabled: true,
		OtpSecret:  "VALID_SECRET",
	}
	db.Create(&brokenUser)
	db.Create(&validUser)

	repairCorruptedOtpUsers()

	var checkBroken models.User
	db.Where("id = ?", "user_broken").First(&checkBroken)
	if checkBroken.OtpEnabled {
		t.Errorf("期望破损用户的 OtpEnabled 被修复为 false，实际为: %v", checkBroken.OtpEnabled)
	}

	var checkValid models.User
	db.Where("id = ?", "user_valid").First(&checkValid)
	if !checkValid.OtpEnabled || checkValid.OtpSecret != "VALID_SECRET" {
		t.Errorf("有效用户的两步验证不应受到影响，实际: %+v", checkValid)
	}
}


// 验证系统中仅有一个本地主账号时，多余的 SSO 账号自动合并至该唯一主账号，并转移所有表的 user_id 与第三方凭据
func TestCleanLegacySSOUsers_SingleAccountFallback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{}, &models.UserSSOBinding{}, &models.EnvironmentVariable{}, &models.Script{})

	adminUser := models.User{
		ID:       "admin_100",
		Username: "admin",
		Email:    "admin@local",
		Role:     "admin",
	}
	sayheyaUser := models.User{
		ID:          "sso_200",
		Username:    "sayheya",
		Email:       "engiguu@gmail.com",
		Role:        "admin",
		OidcSubject: "oauth2:github_999",
	}
	db.Create(&adminUser)
	db.Create(&sayheyaUser)

	// 关联业务数据（环境变量、脚本、SSO 绑定记录）
	envVar := models.EnvironmentVariable{
		ID:     "env_1",
		Name:   "TEST_VAR",
		UserID: "sso_200",
	}
	script := models.Script{
		ID:     "script_1",
		Name:   "test.py",
		UserID: "sso_200",
	}
	binding := models.UserSSOBinding{
		ID:          "bind_1",
		UserID:      "sso_200",
		Username:    "sayheya",
		ChannelType: "oauth2",
		Subject:     "github_999",
	}
	db.Create(&envVar)
	db.Create(&script)
	db.Create(&binding)

	// 执行清理迁移
	migrate_scripts.CleanLegacySSOUsers(db, allModels)

	// 1. 验证 users 表仅保留 admin 一条记录，sayheya 被成功删除
	var remainingUsers []models.User
	db.Find(&remainingUsers)
	if len(remainingUsers) != 1 || remainingUsers[0].ID != "admin_100" {
		t.Fatalf("期望仅保留主账号 admin，实际用户列表: %+v", remainingUsers)
	}
	if remainingUsers[0].Email != "engiguu@gmail.com" {
		t.Errorf("主账号邮箱期望被更新为实际的 engiguu@gmail.com，实际: %s", remainingUsers[0].Email)
	}
	if remainingUsers[0].OidcSubject != "oauth2:github_999" {
		t.Errorf("主账号 oidc_subject 期望同步更新，实际: %s", remainingUsers[0].OidcSubject)
	}

	// 2. 验证各业务表的 user_id 均已转移至 admin_100
	var checkEnv models.EnvironmentVariable
	db.First(&checkEnv, "id = ?", "env_1")
	if checkEnv.UserID != "admin_100" {
		t.Errorf("环境变量 user_id 期望转移至 admin_100，实际: %s", checkEnv.UserID)
	}

	var checkScript models.Script
	db.First(&checkScript, "id = ?", "script_1")
	if checkScript.UserID != "admin_100" {
		t.Errorf("脚本文件 user_id 期望转移至 admin_100，实际: %s", checkScript.UserID)
	}

	var checkBinding models.UserSSOBinding
	db.First(&checkBinding, "id = ?", "bind_1")
	if checkBinding.UserID != "admin_100" || checkBinding.Username != "admin" {
		t.Errorf("SSO 绑定记录期望转移至 admin，实际: UserID=%s Username=%s", checkBinding.UserID, checkBinding.Username)
	}

	// 3. 验证迁移标记已成功落库，防止重复执行
	var setting models.Setting
	res := db.Where("key = ?", "clean_legacy_sso_users_migrated").First(&setting)
	if res.Error != nil || string(setting.Value) != "true" {
		t.Errorf("期望已写入迁移标记 clean_legacy_sso_users_migrated=true，实际: %+v", setting)
	}
}

// 验证存在多个本地账号时，优先按邮箱精准匹配合并
func TestCleanLegacySSOUsers_EmailMatchMultipleAccounts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{}, &models.UserSSOBinding{}, &models.EnvironmentVariable{}, &models.Script{})

	adminUser := models.User{
		ID:       "admin_1",
		Username: "admin",
		Email:    "admin@company.com",
	}
	devUser := models.User{
		ID:       "dev_2",
		Username: "developer",
		Email:    "dev@company.com",
	}
	ssoDevUser := models.User{
		ID:          "sso_dev_3",
		Username:    "dev_github",
		Email:       "dev@company.com",
		OidcSubject: "oidc:dev_sub_777",
	}
	db.Create(&adminUser)
	db.Create(&devUser)
	db.Create(&ssoDevUser)

	migrate_scripts.CleanLegacySSOUsers(db, allModels)

	// sso_dev_3 应该合并至相同邮箱的 dev_2，而不是 admin_1
	var devCheck models.User
	db.First(&devCheck, "id = ?", "dev_2")
	if devCheck.OidcSubject != "oidc:dev_sub_777" {
		t.Errorf("期望 devUser 继承 oidc_subject，实际: %s", devCheck.OidcSubject)
	}
	var remaining []models.User
	db.Find(&remaining)
	if len(remaining) != 2 {
		t.Errorf("期望剩余 2 个用户（admin 与 developer），实际剩余 %d 个", len(remaining))
	}
}

// 验证主账号用户名非 admin 但拥有默认邮箱 admin@local 时，仍可精准判定为主账号并完成合并
func TestCleanLegacySSOUsers_DefaultEmailIdentification(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{}, &models.UserSSOBinding{}, &models.EnvironmentVariable{}, &models.Script{})

	customAdmin := models.User{
		ID:       "custom_master",
		Username: "my_custom_admin", // 用户名非默认 admin
		Email:    "admin@local",     // 默认邮箱
		Role:     "admin",
	}
	ssoUser := models.User{
		ID:          "sso_300",
		Username:    "sayheya",
		Email:       "engiguu@gmail.com",
		Role:        "admin",
		OidcSubject: "oauth2:github_custom",
	}
	db.Create(&customAdmin)
	db.Create(&ssoUser)

	migrate_scripts.CleanLegacySSOUsers(db, allModels)

	var remaining []models.User
	db.Find(&remaining)
	if len(remaining) != 1 || remaining[0].ID != "custom_master" {
		t.Fatalf("期望仅保留 custom_master 账号，实际: %+v", remaining)
	}
	if remaining[0].Email != "engiguu@gmail.com" {
		t.Errorf("邮箱期望更新为第三方真实邮箱，实际: %s", remaining[0].Email)
	}
}

// 自定义带创建者与修改者字段的模型，验证 transferUserReferences 的动态字段过户能力
type TestAuditResource struct {
	ID        string `json:"id" gorm:"primaryKey"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
	UpdatedBy string `json:"updated_by"`
}

func (TestAuditResource) TableName() string {
	return "test_audit_resources"
}

// 验证迁移脚本不仅处理 user_id，还能自动处理创建者 (created_by) 与修改者 (updated_by) 字段
func TestCleanLegacySSOUsers_CreatedByAndUpdatedByTransfer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{}, &TestAuditResource{})

	admin := models.User{
		ID:       "admin_primary",
		Username: "admin",
		Email:    "admin@local",
	}
	legacySSO := models.User{
		ID:          "sso_legacy_99",
		Username:    "legacy_sso",
		Email:       "sso@example.com",
		OidcSubject: "oidc:sub_audit_1",
	}
	db.Create(&admin)
	db.Create(&legacySSO)

	res := TestAuditResource{
		ID:        "res_1",
		Name:      "ResourceWithAudit",
		CreatedBy: "sso_legacy_99",
		UpdatedBy: "sso_legacy_99",
	}
	db.Create(&res)

	customModels := append(allModels, &TestAuditResource{})
	migrate_scripts.CleanLegacySSOUsers(db, customModels)

	// 验证创建者与修改者已成功转更为主账号 ID
	var checkRes TestAuditResource
	db.First(&checkRes, "id = ?", "res_1")
	if checkRes.CreatedBy != "admin_primary" {
		t.Errorf("创建者 created_by 期望更新为 admin_primary，实际: %s", checkRes.CreatedBy)
	}
	if checkRes.UpdatedBy != "admin_primary" {
		t.Errorf("修改者 updated_by 期望更新为 admin_primary，实际: %s", checkRes.UpdatedBy)
	}
}
