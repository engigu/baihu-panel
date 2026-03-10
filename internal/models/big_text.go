package models

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// BigText 自定义大数据文本类型，自动处理跨数据库类型差异
// MySQL: LONGTEXT (4GB)
// PostgreSQL: TEXT
// SQLite: TEXT
type BigText string

func (BigText) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql":
		return "LONGTEXT"
	case "postgres":
		return "TEXT"
	case "sqlite":
		return "TEXT"
	}
	return "TEXT"
}
