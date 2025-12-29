package main

import (
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/ini.v1"
)

// Config Agent 配置
type Config struct {
	ServerURL  string
	Name       string
	Token      string
	Interval   int
	AutoUpdate bool
}

func loadConfigFile(path string, config *Config) error {
	cfg, err := ini.Load(path)
	if err != nil {
		return err
	}

	section := cfg.Section("agent")
	if v := section.Key("server_url").String(); v != "" {
		config.ServerURL = v
	}
	if v := section.Key("name").String(); v != "" {
		config.Name = v
	}
	if v := section.Key("token").String(); v != "" {
		config.Token = v
	}
	if v := section.Key("interval").String(); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			config.Interval = i
		}
	}
	if v := section.Key("auto_update").String(); v != "" {
		config.AutoUpdate = v == "true" || v == "1"
	}
	return nil
}

func saveConfigFile(path string, config *Config) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}

	cfg := ini.Empty()
	section := cfg.Section("agent")
	section.Key("server_url").SetValue(config.ServerURL)
	section.Key("name").SetValue(config.Name)
	section.Key("token").SetValue(config.Token)
	section.Key("interval").SetValue(strconv.Itoa(config.Interval))
	if config.AutoUpdate {
		section.Key("auto_update").SetValue("true")
	} else {
		section.Key("auto_update").SetValue("false")
	}

	return cfg.SaveTo(path)
}
