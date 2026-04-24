package database

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"github.com/engigu/baihu-panel/internal/utils/idgen"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
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
}

func Migrate() error {
	// 1. 自动指纹识别，大幅提升远程数据库启动进度
	sig := getModelSignature(allModels)
	if DB.Migrator().HasTable(&models.Setting{}) {
		var sigSetting models.Setting
		res := DB.Where(&models.Setting{Section: "system", Key: "schema_signature"}).Limit(1).Find(&sigSetting)
		if res.RowsAffected > 0 && string(sigSetting.Value) == sig {
			logger.Info("[Database] 模型指纹一致，跳过自动表结构同步")
			return nil
		}
	}

	// 执行自定义迁移
	logger.Info("[Database] 正在执行自定义迁移与表结构同步...")
	if err := customMigrations(); err != nil {
		logger.Warnf("[Database] 自定义迁移警告: %v", err)
	}

	logger.Infof("[Database] 正在同步 %d 个数据模型的表结构...", len(allModels))
	if err := AutoMigrate(allModels...); err != nil {
		return err
	}

	// 3. 更新指纹记录
	if DB.Migrator().HasTable(&models.Setting{}) {
		var sigSetting models.Setting
		res := DB.Where(&models.Setting{Section: "system", Key: "schema_signature"}).Limit(1).Find(&sigSetting)
		if res.RowsAffected > 0 {
			DB.Model(&sigSetting).Update("value", models.BigText(sig))
		} else {
			DB.Create(&models.Setting{
				ID:      "sys_schema_sig",
				Section: "system",
				Key:     "schema_signature",
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

// hasGormTypeText 检查 gorm tag 中是否包含 type:text
func hasGormTypeText(gormTag string) bool {
	for _, part := range strings.Split(gormTag, ";") {
		if kv := strings.SplitN(strings.TrimSpace(part), ":", 2); len(kv) == 2 {
			if strings.TrimSpace(kv[0]) == "type" && strings.EqualFold(strings.TrimSpace(kv[1]), "text") {
				return true
			}
		}
	}
	return false
}

// parseGormColumn 从 gorm tag 中提取 column:xxx
func parseGormColumn(gormTag string) string {
	for _, part := range strings.Split(gormTag, ";") {
		if kv := strings.SplitN(strings.TrimSpace(part), ":", 2); len(kv) == 2 {
			if strings.TrimSpace(kv[0]) == "column" {
				return strings.TrimSpace(kv[1])
			}
		}
	}
	return ""
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

	// 迁移标签到关联关系表
	if err := migrateTagsToRelations(); err != nil {
		logger.Warnf("[Database] 标签数据迁移失败: %v", err)
	}

	return nil
}

// migrateTagsToRelations 迁移旧的逗号分隔标签到 DataRelation 和 DataStorage 表
func migrateTagsToRelations() error {
	// 检查迁移标志
	var flag models.Setting
	res := DB.Where(&models.Setting{Section: "system", Key: "migration_tags_v1"}).Limit(1).Find(&flag)
	if res.RowsAffected > 0 && string(flag.Value) == "true" {
		return nil
	}

	// 确保表已创建
	if !DB.Migrator().HasTable(&models.DataRelation{}) || !DB.Migrator().HasTable(&models.DataStorage{}) {
		return nil
	}

	logger.Info("[Database] 正在执行标签数据到关联表的迁移...")

	var tasks []models.Task
	if err := DB.Find(&tasks).Error; err != nil {
		return err
	}

	for _, task := range tasks {
		if task.Tags == "" {
			continue
		}
		tags := strings.Split(task.Tags, ",")
		for i, tagName := range tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}

			// 查找或创建 DataStorage (type=tag)
			var storage models.DataStorage
			findRes := DB.Where(&models.DataStorage{Type: "tag", Key: tagName}).Limit(1).Find(&storage)
			if findRes.RowsAffected == 0 {
				dataJson, _ := json.Marshal(map[string]string{"name": tagName})
				storage = models.DataStorage{
					ID:   idgen.GenerateID(),
					Type: "tag",
					Key:  tagName,
					Data: models.BigText(dataJson),
				}
				if err := DB.Create(&storage).Error; err != nil {
					logger.Errorf("[Database] 创建标签存储失败: %v", err)
					continue
				}
			}

			// 创建 DataRelation (type=task_tag)，避免重复创建
			var relation models.DataRelation
			relRes := DB.Where(&models.DataRelation{
				DataID:   task.ID,
				RelateID: storage.ID,
				Type:     "task_tag",
			}).Limit(1).Find(&relation)

			if relRes.RowsAffected == 0 {
				relation = models.DataRelation{
					ID:        idgen.GenerateID(),
					DataID:    task.ID,
					RelateID:  storage.ID,
					Type:      "task_tag",
					SortOrder: i,
				}
				if err := DB.Create(&relation).Error; err != nil {
					logger.Errorf("[Database] 创建标签关联失败: %v", err)
					continue
				}
			}
		}
	}

	// 标记迁移完成
	if res.RowsAffected > 0 {
		DB.Model(&flag).Update("value", models.BigText("true"))
	} else {
		DB.Create(&models.Setting{
			ID:      idgen.GenerateID(),
			Section: "system",
			Key:     "migration_tags_v1",
			Value:   models.BigText("true"),
		})
	}

	logger.Info("[Database] 标签数据迁移完成")
	return nil
}
