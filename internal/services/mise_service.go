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
	Active      bool       `json:"active"`
	InstallPath string     `json:"install_path,omitempty"`
	InstalledAt string     `json:"installed_at,omitempty"` // 安装日期
}

type MiseSource struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// List 从数据库获取已保存的语言列表
func (s *MiseService) List() ([]MiseLanguage, error) {
	db := database.GetDB()
	var models []models.Language
	if err := db.Order("installed_at DESC, plugin ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]MiseLanguage, 0, len(models))
	for _, m := range models {
		lang := MiseLanguage{
			Plugin:      m.Plugin,
			Version:     m.Version,
			InstallPath: m.InstallPath,
			Active:      true,
		}
		lang.Source.Path = m.Source
		if m.InstalledAt != nil {
			lang.InstalledAt = time.Time(*m.InstalledAt).Format("2006-01-02 15:04:05")
		}
		result = append(result, lang)
	}
	return result, nil
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
	// 强制禁用颜色和交互
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *MiseService) listFallback() ([]MiseLanguage, error) {
	cmd := exec.Command("mise", "ls")
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
			Active:  true,
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

	var currentIds []uint
	for _, lang := range languages {
		var model models.Language
		// 以 plugin 和 version 作为联合唯一标识（业务逻辑上）
		err := db.Where("plugin = ? AND version = ?", lang.Plugin, lang.Version).First(&model).Error

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

		if err != nil {
			// 如果不存在，则创建
			newLang := models.Language{
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
