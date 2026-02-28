package database

import (
	"fmt"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UUIDMigration 迁移处理自增 ID 关联为 UUID 关联
func UUIDMigration(db *gorm.DB) error {
	// 1. 检查是否已经迁移过
	var setting models.Setting
	if err := db.Where("section = 'system' AND `key` = 'changeuuid'").First(&setting).Error; err == nil {
		return nil
	}

	logger.Infof("[Database] 正在执行 UUID 关联迁移...")

	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 为所有表生成缺失的 UUID
		tables := []interface{}{
			&models.User{}, &models.Task{}, &models.Agent{}, &models.AgentToken{},
			&models.TaskLog{}, &models.Setting{}, &models.EnvironmentVariable{},
			&models.Script{}, &models.Dependency{}, &models.LoginLog{},
			&models.Language{}, &models.SendStats{},
		}

		for _, table := range tables {
			var records []map[string]interface{}
			tx.Model(table).Find(&records)
			for _, record := range records {
				if uuidVal, ok := record["uuid"].(string); !ok || uuidVal == "" {
					newUUID := uuid.New().String()
					id := record["id"]
					if err := tx.Model(table).Where("id = ?", id).Update("uuid", newUUID).Error; err != nil {
						return fmt.Errorf("为表生成 UUID 失败: %v", err)
					}
				}
			}
		}

		// 2. 转换业务关联
		// 2.1 TaskLog.TaskID (ID -> UUID)
		if err := migrateRelation(tx, "task_logs", "task_id", "tasks", "id", "uuid"); err != nil {
			return err
		}
		// 2.2 TaskLog.AgentID (ID -> UUID)
		if err := migrateRelation(tx, "task_logs", "agent_id", "agents", "id", "uuid"); err != nil {
			return err
		}
		// 2.3 Tasks.AgentID (ID -> UUID)
		if err := migrateRelation(tx, "tasks", "agent_id", "agents", "id", "uuid"); err != nil {
			return err
		}
		// 2.4 Envs.UserID (ID -> UUID)
		if err := migrateRelation(tx, "envs", "user_id", "users", "id", "uuid"); err != nil {
			return err
		}
		// 2.5 Scripts.UserID (ID -> UUID)
		if err := migrateRelation(tx, "scripts", "user_id", "users", "id", "uuid"); err != nil {
			return err
		}
		// 2.6 SendStats.TaskID (ID -> UUID)
		if err := migrateRelation(tx, "send_stats", "task_id", "tasks", "id", "uuid"); err != nil {
			return err
		}

		// 3. 标记迁移完成
		migratedSetting := models.Setting{
			Section: "system",
			Key:     "changeuuid",
			Value:   "1",
		}
		if err := tx.Create(&migratedSetting).Error; err != nil {
			return err
		}

		logger.Infof("[Database] UUID 关联迁移完成")
		return nil
	})
}

func migrateRelation(tx *gorm.DB, table, column, refTable, refIDCol, refUUIDCol string) error {
	var records []map[string]interface{}
	// 获取所有有在该列有值的记录
	if err := tx.Table(table).Where(fmt.Sprintf("%s IS NOT NULL AND %s != ''", column, column)).Find(&records).Error; err != nil {
		return err
	}

	for _, record := range records {
		oldVal := fmt.Sprintf("%v", record[column])
		// 如果已经是 UUID (36位长)，跳过
		if len(oldVal) == 36 {
			continue
		}

		var refRecord map[string]interface{}
		if err := tx.Table(refTable).Where(fmt.Sprintf("%s = ?", refIDCol), oldVal).First(&refRecord).Error; err == nil {
			newVal := refRecord[refUUIDCol]
			id := record["id"]
			if err := tx.Table(table).Where("id = ?", id).Update(column, newVal).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
