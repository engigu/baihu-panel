package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/oidc"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSSOBindingTestDB(t *testing.T) (*AuthController, *models.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	database.DB = db
	if err := db.AutoMigrate(&models.Setting{}, &models.User{}, &models.UserSSOBinding{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	userSvc := services.NewUserService()
	admin := userSvc.CreateUser("admin", "admin123", "admin@local", "admin")

	settings := services.NewSettingsService()
	ac := NewAuthController(userSvc, settings, services.NewLoginLogService())
	return ac, admin
}

func TestSSOBindingListAndDelete(t *testing.T) {
	ac, admin := setupSSOBindingTestDB(t)

	// 1. 模拟授权绑定记录落库
	binding := models.UserSSOBinding{
		ID:           utils.GenerateID(),
		UserID:       admin.ID,
		Username:     admin.Username,
		ChannelType:  constant.SSOTypeOIDC,
		Subject:      "oidc-sub-001",
		IdentityName: "sayheya",
		Email:        "sayheya@example.com",
		BindType:     "oauth",
		CreatedAt:    models.Now(),
		UpdatedAt:    models.Now(),
	}
	if err := database.DB.Create(&binding).Error; err != nil {
		t.Fatalf("创建绑定失败: %v", err)
	}
	if len(binding.ID) != 20 {
		t.Errorf("期望 ID 为 20 位 xid，实际为: %s", binding.ID)
	}

	// 2. 查询绑定列表
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/auth/sso/bindings", nil)
	c.Set("userID", admin.ID)
	ac.GetSSOBindings(c)
	if w.Code != http.StatusOK {
		t.Fatalf("获取列表失败: HTTP %d", w.Code)
	}

	// 3. 解除绑定
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request, _ = http.NewRequest(http.MethodDelete, "/api/v1/auth/sso/bindings/"+binding.ID, nil)
	c2.Params = gin.Params{{Key: "id", Value: binding.ID}}
	c2.Set("userID", admin.ID)
	ac.DeleteSSOBinding(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("解绑失败: HTTP %d, body: %s", w2.Code, w2.Body.String())
	}

	// 确认数据库已删除
	var check models.UserSSOBinding
	if err := database.DB.Where("id = ?", binding.ID).First(&check).Error; err == nil {
		t.Errorf("期望解绑后记录已从数据库中删除，但仍然存在")
	}
}

func TestResolveUserWithSSOBinding(t *testing.T) {
	_, admin := setupSSOBindingTestDB(t)

	cfg := &oidc.Config{
		Type: constant.SSOTypeOIDC,
	}

	claims := &oidc.Claims{
		Sub:               "custom-idp-sub-999",
		PreferredUsername: "sayheya",
		Email:             "different-email@corp.com",
	}

	// 1. 未绑定时，必须直接拒绝登录返回 ErrUserNotLinked（不自动创建新账号）
	_, _, err := oidc.ResolveUser(cfg, claims, services.NewUserService())
	if err != oidc.ErrUserNotLinked {
		t.Fatalf("期望返回 ErrUserNotLinked，实际返回: %v", err)
	}

	// 2. 在绑定表中为 admin 绑定该 sub
	database.DB.Create(&models.UserSSOBinding{
		ID:           utils.GenerateID(),
		UserID:       admin.ID,
		Username:     admin.Username,
		ChannelType:  constant.SSOTypeOIDC,
		Subject:      "custom-idp-sub-999",
		IdentityName: "sayheya",
		Email:        "different-email@corp.com",
		BindType:     "oauth",
		CreatedAt:    models.Now(),
		UpdatedAt:    models.Now(),
	})

	// 3. 再次解析，已绑定身份直接匹配对应本地 admin 账号
	resolvedUser, created, err := oidc.ResolveUser(cfg, claims, services.NewUserService())
	if err != nil {
		t.Fatalf("绑定后解析用户失败: %v", err)
	}
	if created {
		t.Errorf("期望 created 为 false，实际为 true")
	}
	if resolvedUser.ID != admin.ID || resolvedUser.Username != "admin" {
		t.Errorf("解析出的用户与 admin 不匹配: %+v", resolvedUser)
	}
}

// 验证历史版本中仅在 users 表存有 oidc_subject 的账号，无需重新绑定，依然能无缝登录并自动补齐绑定映射
func TestLegacyOidcSubjectCanStillLogin(t *testing.T) {
	_, admin := setupSSOBindingTestDB(t)

	// 模拟老版本存量用户：主表有 oidc_subject，但绑定表中暂无记录
	database.DB.Model(admin).Update("oidc_subject", "legacy-sub-8888")

	cfg := &oidc.Config{Type: constant.SSOTypeOIDC}
	claims := &oidc.Claims{
		Sub:               "legacy-sub-8888",
		PreferredUsername: "admin_legacy",
		Email:             "admin@local",
	}

	resolvedUser, created, err := oidc.ResolveUser(cfg, claims, services.NewUserService())
	if err != nil {
		t.Fatalf("历史绑定账号解析失败: %v", err)
	}
	if created {
		t.Errorf("存量账号不应被标记为 newly created")
	}
	if resolvedUser.ID != admin.ID {
		t.Errorf("期望匹配存量用户 %s, 实际匹配 %s", admin.ID, resolvedUser.ID)
	}

	// 验证系统已自动为该存量账号在 sys_user_sso_bindings 中补齐记录
	var backfill models.UserSSOBinding
	res := database.DB.Where("user_id = ? AND subject = ?", admin.ID, "legacy-sub-8888").First(&backfill)
	if res.Error != nil {
		t.Fatalf("未自动补齐绑定记录: %v", res.Error)
	}
	if backfill.Username != admin.Username {
		t.Errorf("补齐记录的用户名不符: %s", backfill.Username)
	}
}
