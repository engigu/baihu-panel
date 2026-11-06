package services

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 验证恢复旧版本备份数据时，若用户缺失 otp_secret 却标记了开启两步验证，能够自动安全清洗重置为 false
func TestRestoreUsersSanitizesEmptyOtpSecret(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	database.DB = db
	_ = db.AutoMigrate(&models.User{})

	backupService := NewBackupService()

	// 模拟旧版本（如 v1.1.25）导出的 users.json，其中包含开启 2FA 但未导出 otp_secret 的用户
	oldBackupJSON := `[
		{
			"id": "user_old_1",
			"username": "admin_old",
			"password": "hashed_password",
			"email": "admin@example.com",
			"role": "admin",
			"otp_enabled": true
		},
		{
			"id": "user_old_2",
			"username": "user_valid_2fa",
			"password": "hashed_password",
			"email": "user@example.com",
			"role": "user",
			"otp_secret": "MYSECRETKEY12345",
			"otp_enabled": true
		}
	]`

	decoder := json.NewDecoder(strings.NewReader(oldBackupJSON))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('[') {
		t.Fatalf("无效的 JSON 数组测试数据: %v", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		return backupService.restoreUsers(tx, decoder)
	})
	if err != nil {
		t.Fatalf("restoreUsers 失败: %v", err)
	}

	var user1 models.User
	if err := db.Where("id = ?", "user_old_1").First(&user1).Error; err != nil {
		t.Fatalf("查询 user_old_1 失败: %v", err)
	}
	if user1.OtpEnabled {
		t.Errorf("user_old_1 缺少 otp_secret，期望 otp_enabled 被清洗重置为 false，实际为: %v", user1.OtpEnabled)
	}

	var user2 models.User
	if err := db.Where("id = ?", "user_old_2").First(&user2).Error; err != nil {
		t.Fatalf("查询 user_old_2 失败: %v", err)
	}
	if !user2.OtpEnabled || user2.OtpSecret != "MYSECRETKEY12345" {
		t.Errorf("user_old_2 具备有效 otp_secret，期望保持开启两步验证，实际: %+v", user2)
	}
}

// 验证备份恢复解压 scripts 目录时，包含目录穿越 (Zip-Slip) 的条目会被安全阻断
func TestRestoreScriptsDir_ZipSlipPrevention(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "baihu-backup-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempBase)

	origScriptsWorkDir := constant.ScriptsWorkDir
	defer func() {
		constant.ScriptsWorkDir = origScriptsWorkDir
	}()

	mockScriptsDir := filepath.Join(tempBase, "scripts")
	constant.ScriptsWorkDir = mockScriptsDir

	// 构造测试 Zip 内存流
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// 正常文件
	f1, err := zw.Create("scripts/normal.sh")
	if err != nil {
		t.Fatalf("创建 Zip 条目失败: %v", err)
	}
	f1.Write([]byte("echo normal"))

	// 恶意 Zip-Slip 条目（企图跳出 scripts 目录到 tempBase）
	f2, err := zw.Create("scripts/../../evil.sh")
	if err != nil {
		t.Fatalf("创建 Zip 条目失败: %v", err)
	}
	f2.Write([]byte("echo evil"))

	// 正常多级目录文件
	f3, err := zw.Create("scripts/sub/deep.sh")
	if err != nil {
		t.Fatalf("创建 Zip 条目失败: %v", err)
	}
	f3.Write([]byte("echo deep"))

	if err := zw.Close(); err != nil {
		t.Fatalf("关闭 zip.Writer 失败: %v", err)
	}

	zipPath := filepath.Join(tempBase, "test_backup.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("写入测试 zip 文件失败: %v", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("打开 zip 读取失败: %v", err)
	}
	defer r.Close()

	backupService := NewBackupService()
	backupService.restoreScriptsDir(r)

	// 验证正常文件恢复成功
	if _, err := os.Stat(filepath.Join(mockScriptsDir, "normal.sh")); os.IsNotExist(err) {
		t.Errorf("normal.sh 应该被正常恢复到 scripts 目录")
	}
	if _, err := os.Stat(filepath.Join(mockScriptsDir, "sub", "deep.sh")); os.IsNotExist(err) {
		t.Errorf("sub/deep.sh 应该被正常恢复到 scripts 目录")
	}

	// 验证恶意文件未被写入
	evilPath := filepath.Join(tempBase, "evil.sh")
	if _, err := os.Stat(evilPath); !os.IsNotExist(err) {
		t.Errorf("evil.sh 发生 Zip-Slip 越界逃逸，未被拦截: %s", evilPath)
	}
}


