package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	database.DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{})
	return db
}

// 验证当用户两步验证处于破损脏状态（otp_enabled=true 但 otp_secret 为空）时，
// 登录能够自动自愈关闭该状态，并正常签发会话完成登录，防止死锁
func TestLoginWithCorruptedOtpHealsAndLogsIn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	userService := services.NewUserService()
	settingsService := services.NewSettingsService()
	loginLogService := services.NewLoginLogService()

	// 创建测试用户
	user := userService.CreateUser("test_broken_otp", "password123", "test@example.com", constant.DefaultRole)

	// 模拟旧版本备份还原或脏数据注入：otp_enabled 为 true，但 otp_secret 为空
	db.Model(user).Updates(map[string]interface{}{
		"otp_enabled": true,
		"otp_secret":  "",
	})

	ac := NewAuthController(userService, settingsService, loginLogService)
	router := gin.New()
	router.POST("/login", ac.Login)

	body, _ := json.Marshal(map[string]string{
		"username": "test_broken_otp",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际: %d (%s)", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			User       string `json:"user"`
			RequireOtp bool   `json:"require_otp"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Data.RequireOtp {
		t.Fatalf("破损的 OTP 不应要求输入两步验证")
	}
	if resp.Data.User != "test_broken_otp" {
		t.Fatalf("登录用户名不匹配: %s", resp.Data.User)
	}

	// 验证数据库中该用户的两步验证状态已被自动自愈修复为 false
	updatedUser, err := userService.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("获取用户失败: %v", err)
	}
	if updatedUser.OtpEnabled {
		t.Fatalf("破损的 OtpEnabled 期望被自愈修复为 false，实际仍为 true")
	}
}

// 验证两步验证正常配置且密钥合法时，登录流程依然要求输入两步验证
func TestLoginWithValidOtpDemandsVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	userService := services.NewUserService()
	settingsService := services.NewSettingsService()
	loginLogService := services.NewLoginLogService()

	user := userService.CreateUser("test_valid_otp", "password123", "test2@example.com", constant.DefaultRole)
	db.Model(user).Updates(map[string]interface{}{
		"otp_enabled": true,
		"otp_secret":  "JBSWY3DPEHPK3PXP",
	})

	ac := NewAuthController(userService, settingsService, loginLogService)
	router := gin.New()
	router.POST("/login", ac.Login)

	body, _ := json.Marshal(map[string]string{
		"username": "test_valid_otp",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际: %d (%s)", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			RequireOtp      bool   `json:"require_otp"`
			OtpPendingToken string `json:"otp_pending_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !resp.Data.RequireOtp || resp.Data.OtpPendingToken == "" {
		t.Fatalf("正常开启两步验证的用户期望要求输入两步验证码并返回临时凭证")
	}
}

