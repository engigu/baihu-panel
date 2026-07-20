package database

import (
	"crypto/md5"
	"encoding/hex"
	"reflect"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/rs/xid"
)

var allModels = []interface{}{
	&models.AppLog{},
	&models.User{},
	&models.Task{},
	&models.TaskLog{},
	&models.Script{},
	&models.EnvironmentVariable{},
	&models.Setting{},
	&models.SendStats{},
	&models.Dependency{},
	&models.Agent{},
	&models.AgentToken{},
	&models.Language{},
	&models.NotifyWay{},
	&models.NotifyBinding{},
	&models.DataRelation{},
	&models.DataStorage{},
	&models.InterconnectNode{},
	&models.SandboxProfile{},
}

func Migrate() error {
	// 1. 自动指纹识别，大幅提升远程数据库启动进度
	sig := getModelSignature(allModels)
	if DB.Migrator().HasTable(&models.Setting{}) {
		var sigSetting models.Setting
		res := DB.Where(&models.Setting{Section: constant.SectionSystem, Key: constant.KeySchemaSignature}).Limit(1).Find(&sigSetting)
		if res.RowsAffected > 0 && string(sigSetting.Value) == sig {
			logger.Info("[Database] 模型指纹一致，跳过自动表结构同步")
			
			// 即使表结构一致，也要执行后置数据迁移（内部有幂等检查），防止有漏网之鱼
			logger.Info("[Database] 正在执行后置数据迁移...")
			if err := postMigrations(); err != nil {
				logger.Warnf("[Database] 后置数据迁移警告: %v", err)
			}
			return nil
		}
	}

	// 执行前置结构迁移
	logger.Info("[Database] 正在执行前置结构迁移与表结构同步...")
	if err := preMigrations(); err != nil {
		logger.Warnf("[Database] 前置结构迁移警告: %v", err)
	}

	logger.Infof("[Database] 正在同步 %d 个数据模型的表结构...", len(allModels))
	if err := AutoMigrate(allModels...); err != nil {
		return err
	}

	// 执行后置数据迁移，依赖完整的表结构
	logger.Info("[Database] 正在执行后置数据迁移...")
	if err := postMigrations(); err != nil {
		logger.Warnf("[Database] 后置数据迁移警告: %v", err)
	}

	// 3. 更新指纹记录
	if DB.Migrator().HasTable(&models.Setting{}) {
		var sigSetting models.Setting
		res := DB.Where(&models.Setting{Section: constant.SectionSystem, Key: constant.KeySchemaSignature}).Limit(1).Find(&sigSetting)
		if res.RowsAffected > 0 {
			DB.Model(&sigSetting).Update("value", models.BigText(sig))
		} else {
			DB.Create(&models.Setting{
				ID:      constant.IDSchemaSignature,
				Section: constant.SectionSystem,
				Key:     constant.KeySchemaSignature,
				Value:   models.BigText(sig),
			})
		}
	}

	return nil
}

// getModelSignature 生成数据模型的结构指纹
func getModelSignature(models []interface{}) string {
	var sb strings.Builder
	// 包含表前缀，确保前缀变更时也能触发迁移
	sb.WriteString(constant.TablePrefix)
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		sb.WriteString(t.Name())
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous {
				continue
			}
			sb.WriteString(f.Name)
			sb.WriteString(f.Type.String())
			sb.WriteString(f.Tag.Get("gorm"))
		}
	}
	hash := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

// preMigrations 前置结构迁移，处理 AutoMigrate 无法自动解决的变更
func preMigrations() error {
	// 检查 ql_tokens 表是否存在
	if DB.Migrator().HasTable(constant.TableMigrateQlTokens) {
		// 如果 code 列存在，且 token 列不存在，则重命名
		if DB.Migrator().HasColumn(&models.AgentToken{}, constant.ColumnMigrateQlTokenCode) {
			if err := DB.Migrator().RenameColumn(&models.AgentToken{}, constant.ColumnMigrateQlTokenCode, constant.ColumnMigrateQlTokenToken); err != nil {
				logger.Debugf("[Database] 重命名 ql_tokens.code 失败: %v", err)
			}
		}
	}
	// 移除 deps 表中的 type 字段（如果存在）
	if DB.Migrator().HasColumn(&models.Dependency{}, constant.ColumnMigrateDependencyType) {
		if err := DB.Migrator().DropColumn(&models.Dependency{}, constant.ColumnMigrateDependencyType); err != nil {
			logger.Debugf("[Database] 移除 deps.type 字段失败: %v", err)
		} else {
			logger.Infof("[Database] 已成功移除 deps 表中的 type 字段")
		}
	}

	return nil
}

// postMigrations 数据后置迁移，用于需要依赖完整表结构的数据搬运
func postMigrations() error {
	// 迁移任务标签到通用的数据关联表中
	migrateTaskTags()
	// 迁移任务绑定的环境变量到通用的数据关联表中
	migrateTaskEnvs()
	return nil
}

// migrateTaskEnvs 迁移旧的任务绑定环境变量到通用数据关联表
func migrateTaskEnvs() {
	// 检查 settings 表中是否已经记录了迁移状态
	if DB.Migrator().HasTable(&models.Setting{}) {
		var setting models.Setting
		res := DB.Where(&models.Setting{Section: constant.SectionSystem, Key: constant.KeyTaskEnvsMigrated}).Limit(1).Find(&setting)
		if res.RowsAffected > 0 && string(setting.Value) == "true" {
			return
		}
	}

	if !DB.Migrator().HasColumn(&models.Task{}, "envs") {
		markTaskEnvsMigrated()
		return
	}
	logger.Infof("[Database] 正在迁移旧任务环境变量绑定...")

	type TaskMigration struct {
		ID   string
		Envs models.BigText
	}
	var tasks []TaskMigration
	DB.Table((&models.Task{}).TableName()).Select("id, envs").Where("envs IS NOT NULL AND envs != ?", "").Find(&tasks)

	for _, task := range tasks {
		envs := strings.Split(string(task.Envs), ",")
		for _, envID := range envs {
			envID = strings.TrimSpace(envID)
			if envID == "" {
				continue
			}
			var count int64
			DB.Model(&models.DataRelation{}).Where("data_id = ? AND relate_id = ? AND type = ?", task.ID, envID, constant.RelationTypeTaskEnv).Count(&count)
			if count == 0 {
				relation := models.DataRelation{
					ID:        xid.New().String(),
					DataID:    task.ID,
					RelateID:  envID,
					Type:      constant.RelationTypeTaskEnv,
					CreatedAt: models.Now(),
					UpdatedAt: models.Now(),
				}
				DB.Create(&relation)
			}
		}
	}

	// if err := DB.Migrator().DropColumn(&models.Task{}, "envs"); err != nil {
	// 	logger.Debugf("[Database] 移除 bh_tasks.envs 字段失败: %v", err)
	// } else {
	// 	logger.Infof("[Database] 成功迁移 %d 个环境变量绑定的任务，并删除了旧 envs 字段", len(tasks))
	// }
	logger.Infof("[Database] 成功迁移 %d 个环境变量绑定的任务", len(tasks))
	markTaskEnvsMigrated()
}

func markTaskEnvsMigrated() {
	if !DB.Migrator().HasTable(&models.Setting{}) {
		return
	}
	var setting models.Setting
	res := DB.Where(&models.Setting{Section: constant.SectionSystem, Key: constant.KeyTaskEnvsMigrated}).Limit(1).Find(&setting)
	if res.RowsAffected > 0 {
		DB.Model(&setting).Update("value", models.BigText("true"))
	} else {
		DB.Create(&models.Setting{
			ID:      xid.New().String(),
			Section: constant.SectionSystem,
			Key:     constant.KeyTaskEnvsMigrated,
			Value:   models.BigText("true"),
		})
	}
}

// migrateTaskTags 迁移旧的任务标签到通用数据关联表
func migrateTaskTags() {
	// 检查 settings 表中是否已经记录了迁移状态
	if DB.Migrator().HasTable(&models.Setting{}) {
		var setting models.Setting
		res := DB.Where(&models.Setting{Section: constant.SectionSystem, Key: constant.KeyTaskTagsMigrated}).Limit(1).Find(&setting)
		if res.RowsAffected > 0 && string(setting.Value) == "true" {
			return
		}
	}

	if !DB.Migrator().HasColumn(&models.Task{}, "tags") {
		markTaskTagsMigrated()
		return
	}
	logger.Infof("[Database] 正在迁移旧任务标签...")

	type TaskMigration struct {
		ID   string
		Tags string
	}
	var tasks []TaskMigration
	DB.Table((&models.Task{}).TableName()).Select("id, tags").Where("tags != ?", "").Find(&tasks)

	for _, task := range tasks {
		tags := strings.Split(task.Tags, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			var storage models.DataStorage
			res := DB.Where("type = ? AND name = ?", constant.RelationTypeTaskTag, tag).Limit(1).Find(&storage)
			if res.RowsAffected == 0 {
				storage = models.DataStorage{
					ID:        xid.New().String(),
					Type:      constant.RelationTypeTaskTag,
					Name:      tag,
					CreatedAt: models.Now(),
					UpdatedAt: models.Now(),
				}
				DB.Create(&storage)
			}
			var count int64
			DB.Model(&models.DataRelation{}).Where("data_id = ? AND relate_id = ? AND type = ?", task.ID, storage.ID, constant.RelationTypeTaskTag).Count(&count)
			if count == 0 {
				relation := models.DataRelation{
					ID:        xid.New().String(),
					DataID:    task.ID,
					RelateID:  storage.ID,
					Type:      constant.RelationTypeTaskTag,
					CreatedAt: models.Now(),
					UpdatedAt: models.Now(),
				}
				DB.Create(&relation)
			}
		}
	}

	// if err := DB.Migrator().DropColumn(&models.Task{}, "tags"); err != nil {
	// 	logger.Debugf("[Database] 移除 bh_tasks.tags 字段失败: %v", err)
	// } else {
	// 	logger.Infof("[Database] 成功迁移 %d 个带有标签的任务，并删除了旧 tags 字段", len(tasks))
	// }
	logger.Infof("[Database] 成功迁移 %d 个带有标签的任务", len(tasks))
	markTaskTagsMigrated()
}

func markTaskTagsMigrated() {
	if !DB.Migrator().HasTable(&models.Setting{}) {
		return
	}
	var setting models.Setting
	res := DB.Where(&models.Setting{Section: constant.SectionSystem, Key: constant.KeyTaskTagsMigrated}).Limit(1).Find(&setting)
	if res.RowsAffected > 0 {
		DB.Model(&setting).Update("value", models.BigText("true"))
	} else {
		DB.Create(&models.Setting{
			ID:      xid.New().String(),
			Section: constant.SectionSystem,
			Key:     constant.KeyTaskTagsMigrated,
			Value:   models.BigText("true"),
		})
	}
}
