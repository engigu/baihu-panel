package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
	"gorm.io/gorm"
)

// MigrationTable 定义迁移配置
type MigrationTable struct {
	Model      any
	EntityName string
	FKs        map[string]string // 单个外键映射: 字段名 -> 实体名
	MultiFKs   map[string]string // 复合外键映射 (如逗号分隔): 字段名 -> 实体名
}

func getMigrationTables() []MigrationTable {
	return []MigrationTable{
		{&models.User{}, "users", nil, nil},
		{&models.Agent{}, "agents", nil, nil},
		{&models.AgentToken{}, "tokens", nil, nil},
		{&models.EnvironmentVariable{}, "envs", map[string]string{"UserID": "users"}, nil},
		{&models.Task{}, "tasks", map[string]string{"AgentID": "agents"}, map[string]string{"Envs": "envs"}},
		{&models.TaskLog{}, "task_logs", map[string]string{"TaskID": "tasks", "AgentID": "agents"}, nil},
		{&models.Script{}, "scripts", map[string]string{"UserID": "users"}, nil},
		{&models.Setting{}, "settings", nil, nil},
		{&models.SendStats{}, "send_stats", map[string]string{"TaskID": "tasks"}, nil},
		{&models.LoginLog{}, "login_logs", nil, nil},
		{&models.Language{}, "languages", nil, nil},
		{&models.Dependency{}, "deps", nil, nil},
	}
}

func getTableName(db *gorm.DB, model any) string {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return ""
	}
	return stmt.Schema.Table
}

func isTableStringID(db *gorm.DB, model any) bool {
	if !db.Migrator().HasTable(model) {
		return true
	}
	columnTypes, err := db.Migrator().ColumnTypes(model)
	if err != nil {
		return true
	}
	for _, ct := range columnTypes {
		if strings.ToLower(ct.Name()) == "id" {
			typeName := strings.ToLower(ct.DatabaseTypeName())
			return strings.Contains(typeName, "char") || strings.Contains(typeName, "text") || strings.Contains(typeName, "string")
		}
	}
	return false
}

func getValFromMap(m map[string]interface{}, key string) (interface{}, bool) {
	lowerKey := strings.ToLower(key)
	for k, v := range m {
		if strings.ToLower(k) == lowerKey {
			return v, true
		}
	}
	return nil, false
}

func RunMigrationV3() error {
	db := database.DB
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 0. 检查迁移标记，防止重复迁移逻辑被误判触发
	if db.Migrator().HasTable(&models.Setting{}) {
		var migrationFlag models.Setting
		err := db.Where("section = ? AND `key` = ?", "system", "migration_v3_success").First(&migrationFlag).Error
		if err == nil && migrationFlag.Value == "true" {
			// 如果已经是字符串 ID 模式，双重确认
			return nil
		}
	}

	tables := getMigrationTables()
	needMigration := false
	for _, t := range tables {
		if !isTableStringID(db, t.Model) {
			needMigration = true
			logger.Infof("[MigrationV3] 表 [%s] ID 为数字，需迁移", getTableName(db, t.Model))
			break
		}
	}

	if !needMigration {
		// 没有数字 ID 表了，但也可能还没打标（比如之前手动修过表），此时补一个标
		return markMigrationSuccess(db)
	}

	// 1. 备份过程
	backupDir := "./data"
	os.MkdirAll(backupDir, 0755)
	// 如果已经有还原过的记录，为了防止反复循环，我们可以检查还原标记
	backups, _ := filepath.Glob(filepath.Join(backupDir, "migration_v3_backup_*.zip"))
	if len(backups) == 0 {
		logger.Infof("[MigrationV3] 执行关键备份...")
		backupService := NewBackupService()
		zipPath, err := backupService.CreateBackup()
		if err != nil {
			return fmt.Errorf("自动备份失败，流程终止: %v", err)
		}
		newPath := filepath.Join(backupDir, fmt.Sprintf("migration_v3_backup_%s.zip", filepath.Base(zipPath)))
		os.Rename(zipPath, newPath)
		logger.Infof("[MigrationV3] 备份成功: %s", newPath)
	}

	mappings := make(map[string]map[uint]string)
	err := db.Transaction(func(tx *gorm.DB) error {
		return performHardMigration(tx, mappings)
	})

	if err != nil {
		return err
	}

	// 3. 标记成功
	return markMigrationSuccess(db)
}

func markMigrationSuccess(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Setting{}) {
		return nil
	}
	var flag models.Setting
	err := db.Where("section = ? AND `key` = ?", "system", "migration_v3_success").First(&flag).Error
	if err != nil {
		// 创建或更新
		flag = models.Setting{
			ID:      utils.GenerateID(),
			Section: "system",
			Key:     "migration_v3_success",
			Value:   "true",
		}
		return db.Create(&flag).Error
	}
	return db.Model(&flag).Update("value", "true").Error
}

func performHardMigration(tx *gorm.DB, mappings map[string]map[uint]string) error {
	allTables := getMigrationTables()

	// ---------------------------------------------------------
	// 第一阶段：全量构建 ID 映射映射表 (Pass 1)
	// ---------------------------------------------------------
	for _, t := range allTables {
		actualName := getTableName(tx, t.Model)
		if actualName == "" || !tx.Migrator().HasTable(actualName) {
			continue
		}
		mappings[t.EntityName] = make(map[uint]string)
		oldTableName := actualName + "_v2_bak"

		// 如果还没有备份表，说明这是第一次处理该表，先重命名
		if !tx.Migrator().HasTable(oldTableName) {
			if isTableStringID(tx, t.Model) {
				continue // 已经是字符串 ID 且无备份，跳过
			}
			if err := tx.Migrator().RenameTable(actualName, oldTableName); err != nil {
				return fmt.Errorf("重命名表 %s 失败: %v", actualName, err)
			}
		}

		// 预先为该表所有记录生成新的 xid
		var rows []map[string]interface{}
		tx.Table(oldTableName).Select("id").Find(&rows)
		for _, row := range rows {
			if val, ok := getValFromMap(row, "id"); ok {
				uid := parseUint(val)
				if uid > 0 {
					mappings[t.EntityName][uid] = utils.GenerateID()
				}
			}
		}
		logger.Infof("[MigrationV3] Pass 1: 构建关键表 %s 的 ID 映射, 共 %d 条", actualName, len(mappings[t.EntityName]))
	}

	// ---------------------------------------------------------
	// 第二、三阶段：正式转换数据并处理关联字段 (Pass 2 & 3)
	// ---------------------------------------------------------
	for _, t := range allTables {
		actualName := getTableName(tx, t.Model)
		oldTableName := actualName + "_v2_bak"

		if !tx.Migrator().HasTable(oldTableName) {
			// 虽然可能已经改过格式，但为了安全还是 AutoMigrate 一下
			tx.AutoMigrate(t.Model)
			continue
		}

		logger.Infof("[MigrationV3] Pass 2&3: 正在转换数据并修复关联: %s", actualName)
		tx.AutoMigrate(t.Model)

		// 获取新表的有效列名（小写）
		columnTypes, _ := tx.Migrator().ColumnTypes(t.Model)
		validColumns := make(map[string]bool)
		for _, ct := range columnTypes {
			validColumns[strings.ToLower(ct.Name())] = true
		}

		var oldData []map[string]interface{}
		if err := tx.Table(oldTableName).Find(&oldData).Error; err != nil {
			return err
		}

		for _, row := range oldData {
			// 1. 处理主键 ID
			if val, ok := getValFromMap(row, "id"); ok {
				uid := parseUint(val)
				if nid, exists := mappings[t.EntityName][uid]; exists {
					row["id"] = nid
				} else {
					row["id"] = utils.GenerateID() 
				}
			}

			// 2. 处理单外键关联 (Phase 3)
			for field, parentEntity := range t.FKs {
				columnName := getColumnName(field)
				if val, ok := getValFromMap(row, columnName); ok && val != nil {
					ufk := parseUint(val)
					if ufk > 0 {
						if nid, exists := mappings[parentEntity][ufk]; exists {
							row[columnName] = nid
						} else {
							row[columnName] = nil
						}
					} else {
						row[columnName] = nil
					}
				}
			}

			// 3. 处理复合多外键字段 (envs)
			for field, parentEntity := range t.MultiFKs {
				columnName := getColumnName(field)
				if val, ok := getValFromMap(row, columnName); ok && val != nil {
					if strVal, ok := val.(string); ok && strVal != "" {
						row[columnName] = transformMultiIDs(strVal, parentEntity, mappings)
					}
				}
			}

			// 4. 过滤不存在的列并插入
			filteredRow := make(map[string]interface{})
			for k, v := range row {
				if validColumns[strings.ToLower(k)] {
					filteredRow[k] = v
				}
			}
			if err := tx.Table(actualName).Create(filteredRow).Error; err != nil {
				return err
			}
		}

		// 迁移完成，清理备份表
		tx.Migrator().DropTable(oldTableName)
	}
	return nil
}

// 辅助函数：解析各种数字 ID
func parseUint(val interface{}) uint {
	if val == nil { return 0 }
	switch v := val.(type) {
	case uint: return v
	case int64: return uint(v)
	case int: return uint(v)
	case uint64: return uint(v)
	case float64: return uint(v)
	case string:
		var u uint
		fmt.Sscanf(v, "%d", &u)
		return u
	}
	return 0
}

// 辅助函数：字段名转列名
func getColumnName(field string) string {
	switch field {
	case "AgentID": return "agent_id"
	case "TaskID": return "task_id"
	case "UserID": return "user_id"
	case "LogID": return "log_id"
	case "Envs": return "envs"
	default: return strings.ToLower(field)
	}
}

// 辅助函数：处理逗号分隔的 ID 列表
func transformMultiIDs(oldStr string, parentEntity string, mappings map[string]map[uint]string) string {
	parts := strings.Split(oldStr, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" { continue }
		if len(p) == 20 && !utils.IsNumeric(p) {
			result = append(result, p) // 已经是 xid，保留
			continue
		}
		uid := parseUint(p)
		if uid > 0 {
			if nid, exists := mappings[parentEntity][uid]; exists {
				result = append(result, nid)
			}
		}
	}
	return strings.Join(result, ",")
}
