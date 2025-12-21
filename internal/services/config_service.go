package services

import (
	"baihu/internal/constant"

	"gopkg.in/ini.v1"
)

type ServerConfig struct {
	Port int    `ini:"port"`
	Host string `ini:"host"`
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

type TaskConfig struct {
	DefaultTimeout   int `ini:"default_timeout"`
	LogRetentionDays int `ini:"log_retention_days"`
}

type AppConfig struct {
	Server   ServerConfig   `ini:"server"`
	Database DatabaseConfig `ini:"database"`
	Security SecurityConfig `ini:"security"`
	Task     TaskConfig     `ini:"task"`
}

var Config *AppConfig

func LoadConfig(path string) (*AppConfig, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	Config = &AppConfig{}
	if err := cfg.MapTo(Config); err != nil {
		return nil, err
	}

	// 设置默认数据库路径
	if Config.Database.Path == "" {
		Config.Database.Path = constant.DefaultDBPath
	}

	// 设置表前缀到 constant 包
	constant.TablePrefix = Config.Database.TablePrefix

	// 设置 Secret 到 constant 包
	constant.Secret = Config.Security.Secret

	return Config, nil
}

func GetConfig() *AppConfig {
	return Config
}
