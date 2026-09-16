package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/relation"
	"github.com/engigu/baihu-panel/internal/utils"
)

// AppDTO 已安装应用的综合数据传输实体（从 baihu_tasks 表中的 App 主任务记录及其 Config 动态反序列化）
type AppDTO struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Version         string           `json:"version,omitempty"`
	Author          string           `json:"author,omitempty"`
	Category        string           `json:"category,omitempty"`
	Description     string           `json:"description,omitempty"`
	Icon            string           `json:"icon,omitempty"`
	Homepage        string           `json:"homepage,omitempty"`
	CurrentScenario string           `json:"current_scenario,omitempty"`
	Status          string           `json:"status,omitempty"`
	ManifestPath    string           `json:"manifest_path,omitempty"`
	ManifestRaw     string           `json:"manifest_raw,omitempty"`
	TasksCount      int              `json:"tasks_count"`
	ScenariosCount  int              `json:"scenarios_count"`
	CreatedAt       models.LocalTime `json:"created_at"`
	UpdatedAt       models.LocalTime `json:"updated_at"`
}

type AppService struct{}

var DefaultAppService = &AppService{}

// ApplyApp 从文件路径或 URL 直接解析并应用 Manifest
func (s *AppService) ApplyApp(pathOrURL string, opts ApplyOptions) (*ApplyResult, error) {
	opts.ManifestPath = pathOrURL

	var manifest *AppManifest
	var rawData []byte
	var err error

	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		manifest, err = ParseManifestFromURL(pathOrURL, "")
		if err != nil {
			return nil, err
		}
	} else {
		rawData, err = os.ReadFile(pathOrURL)
		if err != nil {
			return nil, fmt.Errorf("读取应用清单文件失败 (%s): %w", pathOrURL, err)
		}
		if processed, pErr := PreprocessYAMLTemplate(rawData); pErr == nil {
			rawData = processed
		}
		manifest, err = ParseManifestFromYAML(rawData)
		if err != nil {
			return nil, err
		}
	}

	return DefaultApplier.Apply(manifest, rawData, opts)
}

// GetApp 从 baihu_tasks 表中读取 type = 'app' 的主任务记录
func (s *AppService) GetApp(id string) (*AppDTO, error) {
	var task models.Task
	res := database.DB.Where("id = ? AND type = ?", id, constant.TaskTypeApp).Limit(1).Find(&task)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, fmt.Errorf("未找到应用: %s", id)
	}

	var childCount int64
	database.DB.Model(&models.Task{}).Where("source_id = ? AND type = ?", task.ID, constant.TaskTypeNormal).Count(&childCount)

	return s.buildAppDTOFromTask(&task, int(childCount)), nil
}

// ListApps 直接从 baihu_tasks 表中查询 type = 'app' 的所有主应用任务记录
func (s *AppService) ListApps() ([]AppDTO, error) {
	var appTasks []models.Task
	err := database.DB.Where("type = ?", constant.TaskTypeApp).Order("created_at desc").Find(&appTasks).Error
	if err != nil {
		return nil, err
	}

	apps := make([]AppDTO, 0, len(appTasks))
	for i := range appTasks {
		task := &appTasks[i]
		var childCount int64
		database.DB.Model(&models.Task{}).Where("source_id = ? AND type = ?", task.ID, constant.TaskTypeNormal).Count(&childCount)
		apps = append(apps, *s.buildAppDTOFromTask(task, int(childCount)))
	}

	return apps, nil
}

// buildAppDTOFromTask 从 Task 实体及其 Config 中的 AppTaskConfig JSON 构建 AppDTO
func (s *AppService) buildAppDTOFromTask(task *models.Task, taskCount int) *AppDTO {
	appID := task.ID
	dto := &AppDTO{
		ID:         appID,
		Name:       task.Name,
		Status:     "installed",
		TasksCount: taskCount,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
	}

	if appCfg := task.GetAppConfig(); appCfg != nil {
		dto.Version = appCfg.Version
		dto.Author = appCfg.Author
		dto.Category = appCfg.Category
		dto.Description = appCfg.Description
		dto.Icon = appCfg.Icon
		dto.Homepage = appCfg.Homepage
		dto.ManifestPath = appCfg.ManifestPath
		dto.ManifestRaw = appCfg.ManifestRaw
		dto.CurrentScenario = appCfg.CurrentScenario
		if appCfg.Status != "" {
			dto.Status = appCfg.Status
		}
	}

	if dto.ManifestRaw != "" {
		if manifest, err := ParseManifestFromYAML([]byte(dto.ManifestRaw)); err == nil {
			dto.ScenariosCount = len(manifest.Scenarios)
		}
	}

	return dto
}

// RemoveApp 卸载已安装应用：物理清理 baihu_tasks 表中的 source_id 子任务以及 type = 'app' 的主应用任务
func (s *AppService) RemoveApp(id string, cleanData bool, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		out.Write([]byte(msg))
	}

	log(">> 正在卸载应用 [%s]...", id)

	// 1. 先查找主应用任务实体 (id = id 或 source_id = 'app:'+id 或 id 匹配)
	var masterTask models.Task
	masterID := id
	if err := database.DB.Where("(id = ? OR source_id = ?) AND type = ?", id, "app:"+id, constant.TaskTypeApp).Limit(1).Find(&masterTask).Error; err == nil && masterTask.ID != "" {
		masterID = masterTask.ID
	}

	// 2. 删除所属受控子任务 (source_id 等于 masterID 或 id 或 'app:'+id)
	var childTasks []models.Task
	if err := database.DB.Where("source_id IN ? AND type = ?", []string{masterID, id, "app:" + id}, constant.TaskTypeNormal).Find(&childTasks).Error; err == nil {
		for _, t := range childTasks {
			log("  - 清理受控子任务: %s", t.Name)
			relation.DataRelation.CleanRelations(t.ID, constant.RelationTypeTaskTag)
			relation.DataRelation.CleanRelations(t.ID, constant.RelationTypeTaskEnv)
			database.DB.Unscoped().Where("id = ?", t.ID).Delete(&models.Task{})
		}
	}

	// 3. 删除主应用任务记录 (type = 'app')
	relation.DataRelation.CleanRelations(masterID, constant.RelationTypeTaskTag)
	relation.DataRelation.CleanRelations(masterID, constant.RelationTypeTaskEnv)
	database.DB.Unscoped().Where("(id = ? OR source_id = ?) AND type = ?", masterID, "app:"+id, constant.TaskTypeApp).Delete(&models.Task{})

	// 3. 清理磁盘文件
	if cleanData {
		absScriptsDir := utils.ResolveAbsScriptsDir()
		appDir := filepath.Join(absScriptsDir, "apps", id)
		if _, err := os.Stat(appDir); err == nil {
			log(">> 正在清理应用数据目录: %s", appDir)
			_ = os.RemoveAll(appDir)
		}
	}

	log(">> ✓ 应用 [%s] 卸载与关联任务清理完成！", id)
	return nil
}

// SwitchScenario 切换应用场景预设
func (s *AppService) SwitchScenario(appID string, scenarioID string, out io.Writer) error {
	appDTO, err := s.GetApp(appID)
	if err != nil {
		return err
	}

	if appDTO.ManifestRaw == "" {
		return fmt.Errorf("应用缺少 Manifest 定义，无法重新编排场景")
	}

	manifest, err := ParseManifestFromYAML([]byte(appDTO.ManifestRaw))
	if err != nil {
		return fmt.Errorf("解析应用定义失败: %w", err)
	}

	opts := ApplyOptions{
		ScenarioID: scenarioID,
		SkipSetup:  true, // 场景切换无需重复安装依赖
		SkipSync:   true, // 无需重复拉取源码
		LogWriter:  out,
	}

	_, err = DefaultApplier.Apply(manifest, []byte(appDTO.ManifestRaw), opts)
	return err
}
