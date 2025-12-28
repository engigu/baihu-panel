package database

import (
	"baihu/internal/logger"
	"baihu/internal/models"
)

func Migrate() error {
	// 先执行自定义迁移
	if err := customMigrations(); err != nil {
		logger.Warnf("[Database] 自定义迁移警告: %v", err)
	}

	return AutoMigrate(
		&models.User{},
		&models.Task{},
		&models.TaskLog{},
		&models.Script{},
		&models.EnvironmentVariable{},
		&models.Setting{},
		&models.LoginLog{},
		&models.SendStats{},
		&models.Dependency{},
		&models.Agent{},
		&models.AgentRegCode{},
	)
}

// customMigrations 自定义迁移（处理 AutoMigrate 无法自动完成的变更）
func customMigrations() error {
	// 检查 ql_tokens 表是否存在，如果存在则修改 code 列大小为 64
	if DB.Migrator().HasTable("ql_tokens") {
		// MySQL: 修改 code 列大小
		if err := DB.Exec("ALTER TABLE ql_tokens MODIFY COLUMN code VARCHAR(64)").Error; err != nil {
			// 忽略错误（可能是 SQLite 或列已经是正确大小）
			logger.Debugf("[Database] 修改 ql_tokens.code 列: %v", err)
		}
	}
	return nil
}
