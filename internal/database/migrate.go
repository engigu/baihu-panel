package database

import (
	"crypto/md5"
	"encoding/hex"
	"reflect"
	"strings"

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
