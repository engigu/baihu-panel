package migrate_scripts

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

// HasMigrated 检查指定的迁移标记是否已执行
func HasMigrated(db *gorm.DB, key string) bool {
	if !db.Migrator().HasTable(&models.Setting{}) {
		return false
	}
	var setting models.Setting
	res := db.Where(&models.Setting{Section: constant.SectionMigrate, Key: key}).Limit(1).Find(&setting)
	return res.RowsAffected > 0 && string(setting.Value) == "true"
}

// MarkMigrated 将指定的迁移标记设置为已执行
func MarkMigrated(db *gorm.DB, key string) {
	if !db.Migrator().HasTable(&models.Setting{}) {
		return
	}
	var setting models.Setting
	res := db.Where(&models.Setting{Section: constant.SectionMigrate, Key: key}).Limit(1).Find(&setting)
	if res.RowsAffected > 0 {
		db.Model(&setting).Update("value", models.BigText("true"))
	} else {
		db.Create(&models.Setting{
			ID:      xid.New().String(),
			Section: constant.SectionMigrate,
			Key:     key,
			Value:   models.BigText("true"),
		})
	}
}

