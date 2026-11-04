package migrate_scripts

import (
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestHasMigratedAndMarkMigrated(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}

	testKey := "test_migration_key_123"

	// 1. 无 Setting 表时返回 false，且 MarkMigrated 静默退出不报错
	if HasMigrated(db, testKey) {
		t.Errorf("期望无 Setting 表时返回 false")
	}
	MarkMigrated(db, testKey)

	// 2. 创建 Setting 表
	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("迁移 Setting 表失败: %v", err)
	}

	// 3. 初始状态未迁移
	if HasMigrated(db, testKey) {
		t.Errorf("期望初始未记录时返回 false")
	}

	// 4. 首次标记迁移
	MarkMigrated(db, testKey)
	if !HasMigrated(db, testKey) {
		t.Errorf("期望标记后 HasMigrated 返回 true")
	}

	// 验证落库的 section 为 constant.SectionMigrate ("migrate")
	var saved models.Setting
	if err := db.Where("key = ?", testKey).First(&saved).Error; err != nil {
		t.Fatalf("未找到迁移记录: %v", err)
	}
	if saved.Section != constant.SectionMigrate {
		t.Errorf("期望 Section 为 %s，实际为: %s", constant.SectionMigrate, saved.Section)
	}

	// 5. 重复标记迁移（幂等性验证）
	MarkMigrated(db, testKey)
	if !HasMigrated(db, testKey) {
		t.Errorf("期望重复标记后仍然返回 true")
	}

	var count int64
	db.Model(&models.Setting{}).Where("section = ? AND key = ?", constant.SectionMigrate, testKey).Count(&count)
	if count != 1 {
		t.Errorf("期望记录数唯一为 1，实际为: %d", count)
	}
}

