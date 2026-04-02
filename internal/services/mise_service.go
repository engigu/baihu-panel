package services

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/deps"
	"github.com/engigu/baihu-panel/internal/utils"
	"gorm.io/gorm"
)

type MiseService struct{}

func NewMiseService() *MiseService {
	return &MiseService{}
}

type MiseLanguage struct {
	Plugin      string     `json:"plugin"`
	Version     string     `json:"version"`
	Source      MiseSource `json:"source"`
	IsGlobal    bool       `json:"is_global"`
	InstallPath string     `json:"install_path,omitempty"`
	InstalledAt string     `json:"installed_at,omitempty"` // 安装日期
}

type MiseSource struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// List 实时从系统检测 mise 环境并同步到数据库
func (s *MiseService) List() ([]MiseLanguage, error) {
	langs, err := s.fetchLiveLanguages()
	if err != nil {
		return nil, err
	}

	// 异步同步到数据库，确保列表响应速度
	go s.syncToDB(langs)

	return langs, nil
}

// Sync 实时检测本地 mise 环境并同步到数据库
func (s *MiseService) Sync() error {
	// 获取实时数据
	langs, err := s.fetchLiveLanguages()
	if err != nil {
		return err
	}

	// 同步到数据库
	s.syncToDB(langs)
	return nil
}

// fetchLiveLanguages 实时从系统检测 mise 语言列表
func (s *MiseService) fetchLiveLanguages() ([]MiseLanguage, error) {
	// 使用 --json 获取格式化数据
	cmd := exec.Command("mise", "ls", "--json")
	// 继承父进程环境变量
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "MISE_NO_COLOR=1", "TERM=dumb")

	output, err := cmd.Output()
	if err != nil {
		logger.Warnf("[Mise] mise ls --json failed: %v", err)
		return s.listFallback()
	}

	// 1. 尝试解析为数组格式 [{}, {}]
	var languages []MiseLanguage
	if err := json.Unmarshal(output, &languages); err == nil {
		s.enrichInstallDates(languages)
		s.enrichSourceInfo(languages)
		s.sortByInstallDate(languages)
		return languages, nil
	}

	// 2. 如果失败，尝试解析为对象格式 {"node": [{}], "python": [{}]}
	var langMap map[string][]MiseLanguage
	if err := json.Unmarshal(output, &langMap); err == nil {
		var result []MiseLanguage
		for plugin, items := range langMap {
			for _, item := range items {
				// 填充插件名称（对象格式中插件名通常是 key）
				if item.Plugin == "" {
					item.Plugin = plugin
				}
				result = append(result, item)
			}
		}
		s.enrichInstallDates(result)
		s.enrichSourceInfo(result)
		s.sortByInstallDate(result)
		return result, nil
	}

	return s.listFallback()
}


func (s *MiseService) listFallback() ([]MiseLanguage, error) {
	cmd := exec.Command("mise", "ls")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "MISE_NO_COLOR=1", "TERM=dumb")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mise ls failed: %v, output: %s", err, string(output))
	}

	lines := strings.Split(string(output), "\n")
	languages := []MiseLanguage{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "tool") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		lang := MiseLanguage{
			Plugin:  parts[0],
			Version: parts[1],
		}
		if len(parts) >= 3 {
			lang.Source = MiseSource{Path: parts[2]}
		}
		languages = append(languages, lang)
	}

	return languages, nil
}

// Plugins 获取主流的 mise 插件列表 (固定返回以确保速度和稳定性)
func (s *MiseService) Plugins() ([]string, error) {
	mainstream := constant.MainstreamMisePlugins
	logger.Infof("[Mise] Returning fixed list of %d mainstream plugins", len(mainstream))
	return mainstream, nil
}

// Versions 获取指定插件的所有可用版本
func (s *MiseService) Versions(plugin string) ([]string, error) {
	if plugin == "" {
		return []string{}, nil
	}

	// 只获取最新版本列表
	cmd := exec.Command("mise", "ls-remote", plugin)
	cmd.Env = append(cmd.Env, "MISE_NO_COLOR=1", "TERM=dumb")
	output, err := cmd.CombinedOutput()

	if err != nil && len(output) == 0 {
		logger.Errorf("[Mise] Fetch versions for %s failed: %v", plugin, err)
		return []string{}, nil
	}

	lines := strings.Split(string(output), "\n")
	var versions []string
	// 倒序排列，优先显示新版本
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.Contains(line, " ") || strings.Contains(line, "Usage:") || line == "latest" {
			continue
		}
		versions = append(versions, line)
		if len(versions) >= 300 {
			break
		}
	}

	return versions, nil
}

// enrichInstallDates 为语言列表添加安装日期信息
func (s *MiseService) enrichInstallDates(languages []MiseLanguage) {
	for i := range languages {
		// 判断是否是 global
		path := strings.ToLower(languages[i].Source.Path)
		// 归一化路径分隔符
		normPath := strings.ReplaceAll(path, "\\", "/")
		if languages[i].Source.Type == "global" {
			languages[i].IsGlobal = true
		} else if strings.Contains(normPath, "mise/config.toml") {
			languages[i].IsGlobal = true
		}

		if languages[i].InstallPath != "" {
			if installDate := s.getInstallDate(languages[i].InstallPath); installDate != "" {
				languages[i].InstalledAt = installDate
			}
		}
	}
}

// enrichSourceInfo 为语言列表添加来源信息
func (s *MiseService) enrichSourceInfo(languages []MiseLanguage) {
	for i := range languages {
		// 如果source为空，使用install_path作为来源
		if languages[i].Source.Type == "" && languages[i].Source.Path == "" {
			if languages[i].InstallPath != "" {
				languages[i].Source.Path = languages[i].InstallPath
			}
		}
	}
}

// getInstallDate 获取安装路径的创建时间
func (s *MiseService) getInstallDate(installPath string) string {
	if installPath == "" {
		return ""
	}

	fileInfo, err := os.Stat(installPath)
	if err != nil {
		logger.Debugf("[Mise] Failed to stat install path %s: %v", installPath, err)
		return ""
	}

	// 获取修改时间作为安装时间的近似值
	modTime := fileInfo.ModTime()
	return modTime.Format("2006-01-02 15:04:05")
}

// sortByInstallDate 按安装时间降序排序（最新的在前面）
func (s *MiseService) sortByInstallDate(languages []MiseLanguage) {
	sort.Slice(languages, func(i, j int) bool {
		// 如果都有安装时间，按时间降序排序
		if languages[i].InstalledAt != "" && languages[j].InstalledAt != "" {
			return languages[i].InstalledAt > languages[j].InstalledAt
		}
		// 有安装时间的排在前面
		if languages[i].InstalledAt != "" {
			return true
		}
		if languages[j].InstalledAt != "" {
			return false
		}
		// 都没有安装时间，按插件名排序
		return languages[i].Plugin < languages[j].Plugin
	})
}

// syncToDB 将实时检测到的语言信息同步到数据库表中
func (s *MiseService) syncToDB(languages []MiseLanguage) {
	db := database.GetDB()
	if db == nil {
		return
	}

	var currentIds []string
	for _, lang := range languages {
		var model models.Language
		// 以 plugin 和 version 作为联合唯一标识（业务逻辑上）
		res := db.Where("plugin = ? AND version = ?", lang.Plugin, lang.Version).Limit(1).Find(&model)
		queryErr := res.Error
		rowsAffected := res.RowsAffected

		sourceStr := ""
		if lang.Source.Path != "" {
			sourceStr = lang.Source.Path
		} else if lang.Source.Type != "" {
			sourceStr = lang.Source.Type
		}

		var installTime *models.LocalTime
		if lang.InstalledAt != "" {
			t, err := time.Parse("2006-01-02 15:04:05", lang.InstalledAt)
			if err == nil {
				lt := models.LocalTime(t)
				installTime = &lt
			}
		}

		if queryErr == nil && rowsAffected == 0 {
				// 如果不存在，则创建
			newLang := models.Language{
				ID:          utils.GenerateID(),
				Plugin:      lang.Plugin,
				Version:     lang.Version,
				InstallPath: lang.InstallPath,
				Source:      sourceStr,
				InstalledAt: installTime,
			}
			if err := db.Create(&newLang).Error; err == nil {
				currentIds = append(currentIds, newLang.ID)
			}
		} else {
			// 如果已存在，更新可能变动的信息
			updates := map[string]interface{}{
				"install_path": lang.InstallPath,
				"source":       sourceStr,
				"installed_at": installTime,
			}
			db.Model(&model).Updates(updates)
			currentIds = append(currentIds, model.ID)
		}
	}

	// 清理数据库中存在但实际 mise 已经卸载的记录
	if len(currentIds) > 0 {
		db.Where("id NOT IN ?", currentIds).Delete(&models.Language{})
	} else {
		// 如果本地一个都没有，清空表
		db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Language{})
	}
}

// GetVerifyCommand 获取环境验证命令
func (s *MiseService) GetVerifyCommand(plugin, version string) (string, error) {
	m := deps.GetManager(plugin)
	if m == nil {
		// 如果没找到对应的包管理器（比如 java 等不支持依赖管理的），
		// 则提供一个基础的通用验证命令
		return utils.BuildMiseCommandSimple(plugin+" --version", plugin, version), nil
	}
	return m.GetVerifyCommand(version)
}
// UseGlobal 设置全局默认版本
func (s *MiseService) UseGlobal(plugin, version string) error {
	cmd := exec.Command("mise", "use", "-g", fmt.Sprintf("%s@%s", plugin, version))
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mise use -g failed: %v, output: %s", err, string(output))
	}
	return nil
}

// UnsetGlobal 取消全局默认版本
func (s *MiseService) UnsetGlobal(plugin, version string) error {
	// 使用 mise unuse -g <tool>@<version> 移出全局配置
	target := plugin
	if version != "" {
		target = fmt.Sprintf("%s@%s", plugin, version)
	}
	cmd := exec.Command("mise", "unuse", "-g", target)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mise unuse global failed: %v, output: %s", err, string(output))
	}
	return nil
}
// Envs 获取全局环境变量
func (s *MiseService) Envs() (map[string]string, error) {
	cmd := exec.Command("mise", "set")
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mise set failed: %v, output: %s", err, string(output))
	}

	envs := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		// 期望格式: KEY VALUE SOURCE
		// 跳过标题行或不合规行
		if len(fields) < 3 || strings.ToLower(fields[0]) == "key" {
			continue
		}
		// 仅显示来自全局配置文件的环境变量
		if strings.Contains(strings.ToLower(fields[len(fields)-1]), "config.toml") {
			envs[fields[0]] = fields[1]
		}
	}
	return envs, nil
}

// SetEnv 设置全局环境变量
func (s *MiseService) SetEnv(key, value string) error {
	cmd := exec.Command("mise", "set", "-g", fmt.Sprintf("%s=%s", key, value))
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mise set -g failed: %v, output: %s", err, string(output))
	}
	return nil
}

// UnsetEnv 取消全局环境变量
func (s *MiseService) UnsetEnv(key string) error {
	cmd := exec.Command("mise", "unset", "-g", key)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mise unset -g failed: %v, output: %s", err, string(output))
	}
	return nil
}
