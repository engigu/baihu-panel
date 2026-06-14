package services

import (
	"os"
	"strconv"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/utils"

	"gopkg.in/ini.v1"
)

type ServerConfig struct {
	Port         int    `ini:"port"`
	Host         string `ini:"host"`
	URLPrefix    string `ini:"url_prefix"`
	PprofEnabled bool   `ini:"pprof_enabled"`
}

type DatabaseConfig struct {
	Type        string `ini:"type"`
	Host        string `ini:"host"`
	Port        int    `ini:"port"`
	User        string `ini:"user"`
	Password    string `ini:"password"`
	DBName      string `ini:"dbname"`
	Path        string `ini:"path"`
	DSN         string `ini:"dsn"`
	TablePrefix string `ini:"table_prefix"`
	SSLMode     string `ini:"ssl_mode"`
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
		_ = os.Unsetenv(key)
	}
}

// getEnvBool 获取环境变量布尔值
func getEnvBool(key string, target *bool) {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			*target = b
		}
		_ = os.Unsetenv(key)
	}
}

// getEnvInt 获取环境变量整数
func getEnvInt(key string, target *int) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*target = n
		}
		_ = os.Unsetenv(key)
	}
}

func LoadConfig(path string) (*AppConfig, error) {
	// 路径发现逻辑：参数优先 -> 环境变量优先 -> 默认常量
	if path == "" {
		if envPath := os.Getenv("BH_CONFIG_PATH"); envPath != "" {
			path = envPath
		} else {
			path = constant.ConfigPath
		}
	}

	// 初始化默认配置
	Config = &AppConfig{
		Server: ServerConfig{
			Port:         8052,
			Host:         "0.0.0.0",
			PprofEnabled: false,
		},
		Database: DatabaseConfig{
			Type:        "sqlite",
			Host:        "localhost",
			Port:        3306,
			User:        "root",
			Password:    "",
			DBName:      "github.com/engigu/baihu-panel",
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
	constant.RuntimeDBType = Config.Database.Type
	constant.RuntimeDBHost = Config.Database.Host
	constant.RuntimeDBPort = Config.Database.Port
	constant.RuntimeDBUser = Config.Database.User
	constant.RuntimeDBPassword = Config.Database.Password
	constant.RuntimeDBName = Config.Database.DBName
	constant.RuntimeDBPath = Config.Database.Path
	constant.RuntimeDBDSN = Config.Database.DSN
	constant.RuntimeDBTablePrefix = Config.Database.TablePrefix
	constant.RuntimeDBSSLMode = Config.Database.SSLMode

	// 暂存旧的 Secret，不再直接给 constant 赋值（改为到 settings 初始化时判断）
	// constant.Secret = Config.Security.Secret

	// 设置演示模式
	if v := os.Getenv("BH_DEMO_MODE"); v == "true" || v == "1" {
		constant.DemoMode = true
		logger.Info("[Config] 演示模式已启用")
		_ = os.Unsetenv("BH_DEMO_MODE")
	}

	// 输出配置信息（隐藏敏感信息）
	logger.Infof("[Config] 服务地址: %s:%d", Config.Server.Host, Config.Server.Port)
	if Config.Server.URLPrefix != "" {
		logger.Infof("[Config] URL前缀: %s", Config.Server.URLPrefix)
	}

	maskedHost := utils.MaskString(Config.Database.Host)
	maskedDBName := utils.MaskString(Config.Database.DBName)
	logger.Infof("[Config] 数据库: type=%s, host=%s, port=%d, dbname=%s, dsn=%v",
		Config.Database.Type, maskedHost, Config.Database.Port, maskedDBName, Config.Database.DSN != "")

	return Config, nil
}

// applyEnvOverrides 从环境变量加载配置
func applyEnvOverrides() {
	// Server
	getEnvInt("BH_SERVER_PORT", &Config.Server.Port)
	getEnvStr("BH_SERVER_HOST", &Config.Server.Host)
	getEnvStr("BH_SERVER_URL_PREFIX", &Config.Server.URLPrefix)
	getEnvBool("BH_SERVER_PPROF", &Config.Server.PprofEnabled)

	// Database
	getEnvStr("BH_DB_TYPE", &Config.Database.Type)
	getEnvStr("BH_DB_HOST", &Config.Database.Host)
	getEnvInt("BH_DB_PORT", &Config.Database.Port)
	getEnvStr("BH_DB_USER", &Config.Database.User)
	getEnvStr("BH_DB_PASSWORD", &Config.Database.Password)
	getEnvStr("BH_DB_NAME", &Config.Database.DBName)
	getEnvStr("BH_DB_PATH", &Config.Database.Path)
	getEnvStr("BH_DB_DSN", &Config.Database.DSN)
	getEnvStr("BH_DB_TABLE_PREFIX", &Config.Database.TablePrefix)
	getEnvStr("BH_DB_SSL_MODE", &Config.Database.SSLMode)

	// Security
	getEnvStr("BH_SECRET", &Config.Security.Secret)

}

func GetConfig() *AppConfig {
	return Config
}
