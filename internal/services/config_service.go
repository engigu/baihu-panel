package services

import (
	"baihu/internal/constant"
	"baihu/internal/logger"
	"os"
	"strconv"

	"gopkg.in/ini.v1"
)

type ServerConfig struct {
	Port      int    `ini:"port"`
	Host      string `ini:"host"`
	URLPrefix string `ini:"url_prefix"`
}

type DatabaseConfig struct {
	Type        string `ini:"type"`
	Host        string `ini:"host"`
	Port        int    `ini:"port"`
	User        string `ini:"user"`
	Password    string `ini:"password"`
	DBName      string `ini:"dbname"`
	Path        string `ini:"path"`
	TablePrefix string `ini:"table_prefix"`
}

type SecurityConfig struct {
	Secret string `ini:"secret"`
}

type AppConfig struct {
	Server   ServerConfig   `ini:"server"`
	Database DatabaseConfig `ini:"database"`
	Security SecurityConfig `ini:"security"`
}

var Config *AppConfig

// getEnvStr 获取环境变量字符串
func getEnvStr(key string, target *string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}

// getEnvInt 获取环境变量整数
func getEnvInt(key string, target *int) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*target = n
		}
	}
}

func LoadConfig(path string) (*AppConfig, error) {
	// 初始化默认配置
	Config = &AppConfig{
		Server: ServerConfig{
			Port: 8052,
			Host: "0.0.0.0",
		},
		Database: DatabaseConfig{
			Type:        "sqlite",
			Host:        "localhost",
			Port:        3306,
			User:        "root",
			Password:    "",
			DBName:      "baihu",
			Path:        constant.DefaultDBPath,
			TablePrefix: "baihu_",
		},
		Security: SecurityConfig{
			Secret: "",
		},
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(path); err == nil {
		// 配置文件存在，从文件加载
		logger.Infof("[Config] 从文件加载配置: %s", path)
		cfg, err := ini.Load(path)
		if err != nil {
			return nil, err
		}
		if err := cfg.MapTo(Config); err != nil {
			return nil, err
		}
	} else {
		// 配置文件不存在，使用环境变量
		logger.Info("[Config] 配置文件不存在，从环境变量加载")
		applyEnvOverrides()
	}

	// 设置默认数据库路径
	if Config.Database.Path == "" {
		Config.Database.Path = constant.DefaultDBPath
	}

	// 设置表前缀到 constant 包
	constant.TablePrefix = Config.Database.TablePrefix

	// 设置 Secret 到 constant 包
	constant.Secret = Config.Security.Secret

	// 设置演示模式
	if v := os.Getenv("BH_DEMO_MODE"); v == "true" || v == "1" {
		constant.DemoMode = true
		logger.Info("[Config] 演示模式已启用")
	}

	// 输出配置信息（隐藏敏感信息）
	logger.Infof("[Config] 服务地址: %s:%d", Config.Server.Host, Config.Server.Port)
	if Config.Server.URLPrefix != "" {
		logger.Infof("[Config] URL前缀: %s", Config.Server.URLPrefix)
	}
	logger.Infof("[Config] 数据库: type=%s, host=%s, port=%d, dbname=%s",
		Config.Database.Type, Config.Database.Host, Config.Database.Port, Config.Database.DBName)

	return Config, nil
}

// applyEnvOverrides 从环境变量加载配置
func applyEnvOverrides() {
	// Server
	getEnvInt("BH_SERVER_PORT", &Config.Server.Port)
	getEnvStr("BH_SERVER_HOST", &Config.Server.Host)
	getEnvStr("BH_SERVER_URL_PREFIX", &Config.Server.URLPrefix)

	// Database
	getEnvStr("BH_DB_TYPE", &Config.Database.Type)
	getEnvStr("BH_DB_HOST", &Config.Database.Host)
	getEnvInt("BH_DB_PORT", &Config.Database.Port)
	getEnvStr("BH_DB_USER", &Config.Database.User)
	getEnvStr("BH_DB_PASSWORD", &Config.Database.Password)
	getEnvStr("BH_DB_NAME", &Config.Database.DBName)
	getEnvStr("BH_DB_PATH", &Config.Database.Path)
	getEnvStr("BH_DB_TABLE_PREFIX", &Config.Database.TablePrefix)

	// Security
	getEnvStr("BH_SECRET", &Config.Security.Secret)
}

func GetConfig() *AppConfig {
	return Config
}
