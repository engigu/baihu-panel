package database

import (
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
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
		&models.AgentToken{},
		&models.Language{},
	)
}

// customMigrations 自定义迁移（处理 AutoMigrate 无法自动完成的变更）
func customMigrations() error {
	// 检查 ql_tokens 表是否存在
	if DB.Migrator().HasTable("ql_tokens") {
		// 将 code 列重命名为 token（如果 code 列存在）
		if DB.Migrator().HasColumn(&models.AgentToken{}, "code") {
			if err := DB.Migrator().RenameColumn(&models.AgentToken{}, "code", "token"); err != nil {
				logger.Debugf("[Database] 重命名 ql_tokens.code 列: %v", err)
			}
		}
	}
	// 移除 deps 表中的 type 字段（如果存在）
	if DB.Migrator().HasColumn(&models.Dependency{}, "type") {
		if err := DB.Migrator().DropColumn(&models.Dependency{}, "type"); err != nil {
			logger.Debugf("[Database] 移除 deps.type 列失败: %v", err)
		} else {
			logger.Infof("[Database] 已成功移除 deps 表中的冗余 type 列")
		}
	}

	return nil
}
