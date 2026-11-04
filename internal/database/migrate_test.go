package database

import (
	"testing"

	"github.com/engigu/baihu-panel/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 验证启动迁移时，自动清理修复存量数据库中 otp_secret 为空且标记开启两步验证的破损用户数据
func TestRepairCorruptedOtpUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
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

