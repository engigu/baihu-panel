package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/systime"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB
var DBConfig *Config

type Config struct {
	Type     string // sqlite, mysql, postgres
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Path     string // for sqlite
	DSN      string // for mysql/mariadb unix socket or custom dsn
	SSLMode  string // postgres: disable/require/verify-ca/verify-full; mysql: true/skip-verify
}

func Init(cfg *Config) error {
	var err error
	DBConfig = cfg
	// 设置东八区时区
	loc := systime.CST
	time.Local = loc

	dsn, err := buildDSN(cfg)
	if err != nil {
		return err
	}

	var dialector gorm.Dialector

	switch cfg.Type {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	newLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		gormlogger.Config{
			SlowThreshold:             time.Millisecond * 500, // 慢 SQL 阈值，默认是 200ms，这里改为 500ms
			LogLevel:                  gormlogger.Warn,        // 日志级别
			IgnoreRecordNotFoundError: true,                   // 忽略 ErrRecordNotFound（找不到记录）错误
			Colorful:                  true,                   // 禁用彩色打印
		},
	)

	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: newLogger,
		NowFunc: func() time.Time {
			return time.Now().In(loc)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	logger.Infof("[Database] 已连接 %s 数据库 (时区: Asia/Shanghai)", cfg.Type)

	// SQLite 特殊优化：开启 WAL 模式，提升并发性能
	if cfg.Type == "sqlite" {
		sqlDB, _ := DB.DB()
		if sqlDB != nil {
			sqlDB.SetMaxOpenConns(1) // SQLite 只允许单写连接
			sqlDB.Exec("PRAGMA journal_mode=WAL")
			sqlDB.Exec("PRAGMA synchronous=NORMAL")
		}
	}

	return nil
}

func AutoMigrate(models ...interface{}) error {
	return DB.AutoMigrate(models...)
}

func GetDB() *gorm.DB {
	return DB
}

func buildDSN(cfg *Config) (string, error) {
	switch cfg.Type {
	case "sqlite":
		return cfg.Path + "?_busy_timeout=5000", nil
	case "mysql":
		if cfg.DSN != "" {
			return cfg.DSN, nil
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FShanghai",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
		if cfg.SSLMode != "" {
			dsn += "&tls=" + cfg.SSLMode
		}
		return dsn, nil
	case "postgres":
		if cfg.DSN != "" {
			return cfg.DSN, nil
		}
		sslMode := cfg.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslMode)
		return dsn, nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}
