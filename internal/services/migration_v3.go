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
		baseName := strings.TrimSuffix(filepath.Base(zipPath), ".zip")
		newPath := filepath.Join(backupDir, fmt.Sprintf("migration_v3_backup_%s.zip", baseName))
		os.Rename(zipPath, newPath)
		logger.Infof("[MigrationV3] 备份成功: %s", newPath)
	}

	mappings := make(map[string]map[uint]string)
	// 根据数据库类型决定是否使用事务：
	// - PostgreSQL: DDL 完全支持事务，使用事务保证原子性
	// - MySQL: DDL 会隐式提交事务，包裹事务无意义
	// - SQLite: DDL+DML 混合在 GORM 事务中会导致数据丢失
	dbType := db.Dialector.Name()
	if dbType == "postgres" {
		err := db.Transaction(func(tx *gorm.DB) error {
			return performHardMigration(tx, mappings)
		})
		if err != nil {
			return err
		}
	} else {
		// SQLite / MySQL: 不使用事务包裹
		if err := performHardMigration(db, mappings); err != nil {
			return err
		}
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

func performHardMigration(db *gorm.DB, mappings map[string]map[uint]string) error {
	allTables := getMigrationTables()

	// ---------------------------------------------------------
	// 第一阶段：全量构建 ID 映射映射表 (Pass 1)
	// ---------------------------------------------------------
	for _, t := range allTables {
		actualName := getTableName(db, t.Model)
		if actualName == "" || !db.Migrator().HasTable(actualName) {
			continue
		}
		mappings[t.EntityName] = make(map[uint]string)
		oldTableName := actualName + "_v2_bak"

		// 如果还没有备份表，说明这是第一次处理该表，先重命名
		if !db.Migrator().HasTable(oldTableName) {
			if isTableStringID(db, t.Model) {
				continue // 已经是字符串 ID 且无备份，跳过
			}
			if err := db.Migrator().RenameTable(actualName, oldTableName); err != nil {
				return fmt.Errorf("重命名表 %s 失败: %v", actualName, err)
			}
			// SQLite 重命名表后，索引名称不变，会导致 AutoMigrate 创建新表时索引冲突
			// 需要先删除备份表上的旧索引
			dropOldIndexes(db, oldTableName)
		}

		// 预先为该表所有记录生成新的 xid
		var rows []map[string]interface{}
		db.Table(oldTableName).Select("id").Find(&rows)
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

	// 构建 fallback ID：FK 映射找不到时使用第一条记录的新 ID
	fallbackIDs := make(map[string]string)
	for _, t := range allTables {
		// 优先从 mappings 中取第一个新 xid（本次迁移生成的）
		if m, ok := mappings[t.EntityName]; ok && len(m) > 0 {
			for _, newID := range m {
				fallbackIDs[t.EntityName] = newID
				break
			}
			continue
		}
		// 如果该表已经是 string ID（被跳过了），从实际表查第一条
		actualName := getTableName(db, t.Model)
		if actualName != "" && db.Migrator().HasTable(actualName) {
			var row map[string]interface{}
			if err := db.Table(actualName).Select("id").Order("id").Limit(1).Find(&row).Error; err == nil {
				if id, ok := getValFromMap(row, "id"); ok && id != nil {
					fallbackIDs[t.EntityName] = fmt.Sprintf("%v", id)
				}
			}
		}
	}

	// ---------------------------------------------------------
	// 第二、三阶段：正式转换数据并处理关联字段 (Pass 2 & 3)
	// ---------------------------------------------------------
	for _, t := range allTables {
		actualName := getTableName(db, t.Model)
		oldTableName := actualName + "_v2_bak"

		if !db.Migrator().HasTable(oldTableName) {
			// 虽然可能已经改过格式，但为了安全还是 AutoMigrate 一下
			db.AutoMigrate(t.Model)
			continue
		}

		logger.Infof("[MigrationV3] Pass 2&3: 正在转换数据并修复关联: %s", actualName)
		db.AutoMigrate(t.Model)

		// 获取新表的有效列名（小写）
		columnTypes, _ := db.Migrator().ColumnTypes(t.Model)
		validColumns := make(map[string]bool)
		for _, ct := range columnTypes {
			validColumns[strings.ToLower(ct.Name())] = true
		}

		var oldData []map[string]interface{}
		if err := db.Table(oldTableName).Find(&oldData).Error; err != nil {
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
					strVal := fmt.Sprintf("%v", val)
					if strVal == "" || strVal == "0" || strVal == "<nil>" {
						row[columnName] = nil
						continue
					}
					// 如果值已经是非数字字符串（例如已迁移过的 xid），直接保留
					if !utils.IsNumeric(strVal) {
						continue
					}
					// 数字型外键，查映射表转换
					ufk := parseUint(val)
					if ufk > 0 {
						if nid, exists := mappings[parentEntity][ufk]; exists {
							row[columnName] = nid
						} else if fbID, hasFB := fallbackIDs[parentEntity]; hasFB {
							logger.Warnf("[MigrationV3] 表 %s 的 %s=%d 在 %s 映射中未找到，使用首条记录 ID: %s", getTableName(db, t.Model), columnName, ufk, parentEntity, fbID)
							row[columnName] = fbID
						} else {
							logger.Warnf("[MigrationV3] 表 %s 的 %s=%d 在 %s 映射中未找到且无 fallback，置空", getTableName(db, t.Model), columnName, ufk, parentEntity)
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
			if err := db.Table(actualName).Create(filteredRow).Error; err != nil {
				return err
			}
		}

		// 迁移完成，清理备份表
		db.Migrator().DropTable(oldTableName)
	}
	return nil
}

// 辅助函数：解析各种数字 ID
func parseUint(val interface{}) uint {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case uint:
		return v
	case int64:
		return uint(v)
	case int:
		return uint(v)
	case uint64:
		return uint(v)
	case float64:
		return uint(v)
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
	case "AgentID":
		return "agent_id"
	case "TaskID":
		return "task_id"
	case "UserID":
		return "user_id"
	case "LogID":
		return "log_id"
	case "Envs":
		return "envs"
	default:
		return strings.ToLower(field)
	}
}

// 辅助函数：处理逗号分隔的 ID 列表
func transformMultiIDs(oldStr string, parentEntity string, mappings map[string]map[uint]string) string {
	parts := strings.Split(oldStr, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
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

// dropOldIndexes 删除备份表上的旧索引，防止 AutoMigrate 创建新表时索引名冲突
// SQLite 重命名表后索引名不变，MySQL/PostgreSQL 也可能存在类似问题
func dropOldIndexes(db *gorm.DB, tableName string) {
	dbType := db.Dialector.Name()
	var indexNames []string

	switch dbType {
	case "sqlite":
		var indexes []struct{ Name string }
		db.Raw("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name=? AND name NOT LIKE 'sqlite_%'", tableName).Scan(&indexes)
		for _, idx := range indexes {
			indexNames = append(indexNames, idx.Name)
		}
	case "mysql":
		var indexes []struct {
			KeyName string `gorm:"column:Key_name"`
		}
		db.Raw("SHOW INDEX FROM `" + tableName + "`").Scan(&indexes)
		seen := make(map[string]bool)
		for _, idx := range indexes {
			if idx.KeyName != "PRIMARY" && !seen[idx.KeyName] {
				indexNames = append(indexNames, idx.KeyName)
				seen[idx.KeyName] = true
			}
		}
	case "postgres":
		var indexes []struct {
			IndexName string `gorm:"column:indexname"`
		}
		db.Raw("SELECT indexname FROM pg_indexes WHERE tablename=?", tableName).Scan(&indexes)
		for _, idx := range indexes {
			if !strings.HasSuffix(idx.IndexName, "_pkey") {
				indexNames = append(indexNames, idx.IndexName)
			}
		}
	}

	for _, name := range indexNames {
		var dropSQL string
		switch dbType {
		case "mysql":
			dropSQL = fmt.Sprintf("DROP INDEX `%s` ON `%s`", name, tableName)
		default:
			dropSQL = fmt.Sprintf("DROP INDEX IF EXISTS \"%s\"", name)
		}
		if err := db.Exec(dropSQL).Error; err != nil {
			logger.Warnf("[MigrationV3] 删除旧索引 %s 失败 (可忽略): %v", name, err)
		} else {
			logger.Infof("[MigrationV3] 已删除备份表旧索引: %s", name)
		}
	}
}
