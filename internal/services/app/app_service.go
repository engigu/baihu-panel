package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/relation"
	"github.com/engigu/baihu-panel/internal/utils"
	"gopkg.in/yaml.v3"
)

// AppDTO 已安装应用的综合数据传输实体（从 baihu_tasks 表中的 App 主任务记录及其 Config 动态反序列化）
type AppDTO struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Version         string           `json:"version,omitempty"`
	Author          string           `json:"author,omitempty"`
	Category        string           `json:"category,omitempty"`
	LastCommit      string           `json:"last_commit,omitempty"`
	Description     string           `json:"description,omitempty"`
	Icon            string           `json:"icon,omitempty"`
	Homepage        string           `json:"homepage,omitempty"`
	CurrentScenario string           `json:"current_scenario,omitempty"`
	Status          string           `json:"status,omitempty"`
	ManifestPath    string                   `json:"manifest_path,omitempty"`
	ManifestRaw     string                   `json:"manifest_raw,omitempty"`
	Template        *models.AppTemplateConfig `json:"template,omitempty"`
	TasksCount      int                      `json:"tasks_count"`
	ScenariosCount  int                      `json:"scenarios_count"`
	CreatedAt       models.LocalTime         `json:"created_at"`
	UpdatedAt       models.LocalTime         `json:"updated_at"`
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
		dto.LastCommit = appCfg.LastCommit
		dto.Description = appCfg.Description
		dto.Icon = appCfg.Icon
		dto.Homepage = appCfg.Homepage
		dto.ManifestPath = appCfg.ManifestPath
		dto.ManifestRaw = FormatManifestYAML(appCfg.ManifestRaw)
		dto.CurrentScenario = appCfg.CurrentScenario
		if appCfg.Status != "" {
			dto.Status = appCfg.Status
		}
		if appCfg.Template != nil {
			dto.Template = appCfg.Template
		}
	}

	if dto.ManifestRaw != "" {
		if manifest, err := ParseManifestFromYAML([]byte(dto.ManifestRaw)); err == nil {
			dto.ScenariosCount = len(manifest.Scenarios)
			if dto.Template == nil {
				dto.Template = manifest.GetTypedTemplateConfig()
			}
		}
	}

	return dto
}

// AppRemoveOptions 应用卸载控制选项
type AppRemoveOptions struct {
	CleanData bool // 是否清理本地代码与产物目录
	CleanEnvs bool // 是否同时清理应用关联的环境变量
}

// CleanAppTaskAndData 物理清理应用主任务、关联受控子任务、关联环境变量、孤儿标签及本地磁盘代码目录
// 返回所有被删除的任务 ID 列表（包含主任务 ID 与所有受控子任务 ID），便于外层调度器移除 Cron 计划任务
func CleanAppTaskAndData(masterTask *models.Task, opts AppRemoveOptions, out io.Writer) ([]string, error) {
	if masterTask == nil || masterTask.Type != constant.TaskTypeApp {
		return nil, nil
	}

	log := func(format string, args ...interface{}) {
		if out == nil {
			return
		}
		msg := fmt.Sprintf(format, args...)
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		out.Write([]byte(msg))
	}

	// 解析 Manifest 与关联 Tag 列表
	var manifest *AppManifest
	var candidateTags []string
	addCandidateTag := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return
		}
		for _, existing := range candidateTags {
			if existing == tag {
				return
			}
		}
		candidateTags = append(candidateTags, tag)
	}

	if appCfg := masterTask.GetAppConfig(); appCfg != nil {
		if appCfg.Template != nil && appCfg.Template.Tag != "" {
			addCandidateTag(appCfg.Template.Tag)
		}
		if appCfg.ID != "" {
			addCandidateTag(appCfg.ID)
		}
		if appCfg.ManifestRaw != "" {
			if m, err := ParseManifestFromYAML([]byte(appCfg.ManifestRaw)); err == nil {
				manifest = m
				addCandidateTag(m.GetTemplateTag())
				addCandidateTag(m.ID)
			}
		}
	}
	// 同时把主任务在数据库中实际绑定的 task_tag 纳入候选集
	loadedMasterTags := relation.DataRelation.LoadTags([]string{masterTask.ID}, constant.RelationTypeTaskTag)
	for _, t := range loadedMasterTags[masterTask.ID] {
		addCandidateTag(t)
	}

	deletedTaskIDs := []string{masterTask.ID}

	// 1. 清理所有受控子任务 (source_id = masterTask.ID)
	var childTasks []models.Task
	if err := database.DB.Where("source_id = ? AND type = ?", masterTask.ID, constant.TaskTypeNormal).Find(&childTasks).Error; err == nil {
		for _, t := range childTasks {
			log("  - 清理受控子任务: %s (%s)", t.Name, t.ID)
			deletedTaskIDs = append(deletedTaskIDs, t.ID)
			database.DB.Where("type = ? AND data_id = ?", constant.BindingTypeTask, t.ID).Delete(&models.NotifyBinding{})
			relation.DataRelation.CleanRelations(t.ID, constant.RelationTypeTaskTag)
			relation.DataRelation.CleanRelations(t.ID, constant.RelationTypeTaskEnv)
			database.DB.Unscoped().Where("id = ?", t.ID).Delete(&models.Task{})
		}
	}

	// 2. 清理主应用任务关联关系与主记录
	database.DB.Where("type = ? AND data_id = ?", constant.BindingTypeTask, masterTask.ID).Delete(&models.NotifyBinding{})
	relation.DataRelation.CleanRelations(masterTask.ID, constant.RelationTypeTaskTag)
	relation.DataRelation.CleanRelations(masterTask.ID, constant.RelationTypeTaskEnv)
	database.DB.Unscoped().Where("id = ?", masterTask.ID).Delete(&models.Task{})

	// 3. 按需清理应用关联的环境变量 (opts.CleanEnvs)
	if opts.CleanEnvs {
		cleanAppEnvironments(manifest, candidateTags, deletedTaskIDs, log)
	}

	// 4. 兜底检查并清理残留的孤儿标签 (task_tag 与 env_tag)
	if len(candidateTags) > 0 {
		relation.DataRelation.CleanOrphanTagsByNames(constant.RelationTypeTaskTag, candidateTags)
		relation.DataRelation.CleanOrphanTagsByNames(constant.RelationTypeEnvTag, candidateTags)
	}

	// 5. 清理磁盘本地代码与数据文件夹 (data/scripts/apps/{author}-{id})
	manifestID := masterTask.GetManifestID()
	author := masterTask.GetAppAuthor()
	if opts.CleanData && manifestID != "" {
		absScriptsDir := utils.ResolveAbsScriptsDir()
		appDir := masterTask.WorkDir
		if appDir != "" {
			appDir = constant.ResolveScriptPath(appDir)
		}
		if appDir == "" || !strings.Contains(appDir, filepath.Join(absScriptsDir, "apps")) {
			appDir = GetAppDir(absScriptsDir, author, manifestID)
		}
		if appDir != "" && strings.Contains(appDir, filepath.Join(absScriptsDir, "apps")) {
			if _, err := os.Stat(appDir); err == nil {
				// 如果 Manifest 声明了 setup.uninstall，先在 appDir 下执行卸载清理钩子
				if manifest != nil && strings.TrimSpace(manifest.Setup.Uninstall) != "" {
					uninstallScript := strings.ReplaceAll(manifest.Setup.Uninstall, "{app_dir}", appDir)
					log(">> 正在执行应用卸载清理脚本 (setup.uninstall)...")
					uninstallCmd := utils.NewShellCommandCmd(uninstallScript)
					uninstallCmd.Dir = appDir
					uninstallCmd.Env = append(os.Environ(), "APP_DIR="+appDir, "CURR_APP_DIR="+appDir)
					_ = uninstallCmd.Run()
				}
				log(">> 正在清理应用数据目录: %s", appDir)
				_ = os.RemoveAll(appDir)
			}
		}
	}

	return deletedTaskIDs, nil
}

// cleanAppEnvironments 清理属于该应用的环境变量（安全校验未被其他任务或非本应用标签占用）
func cleanAppEnvironments(manifest *AppManifest, candidateTags []string, deletedTaskIDs []string, log func(string, ...interface{})) {
	candidateEnvIDs := make(map[string]bool)
	var candidateEnvs []models.EnvironmentVariable

	// A. 通过应用关联的 env_tag 查找环境变量
	if len(candidateTags) > 0 {
		var storageIDs []string
		database.DB.Model(&models.DataStorage{}).
			Where("type = ? AND name IN ?", constant.RelationTypeEnvTag, candidateTags).
			Pluck("id", &storageIDs)

		if len(storageIDs) > 0 {
			var relEnvIDs []string
			database.DB.Model(&models.DataRelation{}).
				Where("type = ? AND relate_id IN ?", constant.RelationTypeEnvTag, storageIDs).
				Pluck("data_id", &relEnvIDs)

			if len(relEnvIDs) > 0 {
				var taggedEnvs []models.EnvironmentVariable
				database.DB.Where("id IN ?", relEnvIDs).Find(&taggedEnvs)
				for _, e := range taggedEnvs {
					if !candidateEnvIDs[e.ID] {
						candidateEnvIDs[e.ID] = true
						candidateEnvs = append(candidateEnvs, e)
					}
				}
			}
		}
	}

	// B. 通过 Manifest.EnvSchema 中声明的 key 查找环境变量
	if manifest != nil && len(manifest.EnvSchema) > 0 {
		var schemaKeys []string
		for _, item := range manifest.EnvSchema {
			if k := strings.TrimSpace(item.Key); k != "" {
				schemaKeys = append(schemaKeys, k)
			}
		}
		if len(schemaKeys) > 0 {
			var keyEnvs []models.EnvironmentVariable
			database.DB.Where("name IN ?", schemaKeys).Find(&keyEnvs)
			for _, e := range keyEnvs {
				if !candidateEnvIDs[e.ID] {
					candidateEnvIDs[e.ID] = true
					candidateEnvs = append(candidateEnvs, e)
				}
			}
		}
	}

	if len(candidateEnvs) == 0 {
		return
	}

	candidateTagSet := make(map[string]bool)
	for _, t := range candidateTags {
		candidateTagSet[strings.ToLower(t)] = true
	}

	for _, env := range candidateEnvs {
		// 1. 检查是否被系统中的其他任务显式绑定 (task_env)
		var otherTaskRefCount int64
		tx := database.DB.Model(&models.DataRelation{}).Where("type = ? AND relate_id = ?", constant.RelationTypeTaskEnv, env.ID)
		if len(deletedTaskIDs) > 0 {
			tx = tx.Where("data_id NOT IN ?", deletedTaskIDs)
		}
		tx.Count(&otherTaskRefCount)
		if otherTaskRefCount > 0 {
			log("  ~ 保留环境变量 %s (仍被其他 %d 个任务绑定引用)", env.Name, otherTaskRefCount)
			continue
		}

		// 2. 检查该环境变量是否还绑定了其他非本应用的 env_tag
		envTagsMap := relation.DataRelation.LoadTags([]string{env.ID}, constant.RelationTypeEnvTag)
		envTags := envTagsMap[env.ID]
		hasOtherAppTag := false
		var remainingTags []string
		for _, et := range envTags {
			if !candidateTagSet[strings.ToLower(et)] {
				hasOtherAppTag = true
				remainingTags = append(remainingTags, et)
			}
		}
		if hasOtherAppTag {
			// 如果带有其他标签，仅解绑当前应用的标签，保留环境变量本体
			relation.DataRelation.SaveTags(env.ID, constant.RelationTypeEnvTag, strings.Join(remainingTags, ","))
			relation.DataRelation.CleanOrphanTagsByNames(constant.RelationTypeEnvTag, candidateTags)
			log("  ~ 保留共享环境变量 %s (带有其他标签: %s)", env.Name, strings.Join(remainingTags, ","))
			continue
		}

		// 3. 安全物理删除该环境变量及其关联的 env_tag / task_env（并自动回收变成 0 引用的孤儿 env_tag）
		log("  - 清理应用环境变量: %s", env.Name)
		relation.DataRelation.CleanRelations(env.ID, constant.RelationTypeEnvTag)
		database.DB.Where("type = ? AND relate_id = ?", constant.RelationTypeTaskEnv, env.ID).Delete(&models.DataRelation{})
		database.DB.Unscoped().Where("id = ?", env.ID).Delete(&models.EnvironmentVariable{})
	}
}

// RemoveApp 卸载已安装应用：根据应用 TaskID 物理清理主应用记录、关联受控子任务、环境变量、标签及本地代码目录
func (s *AppService) RemoveApp(taskID string, cleanData bool, out io.Writer) error {
	_, err := s.RemoveAppWithOptions(taskID, AppRemoveOptions{
		CleanData: cleanData,
		CleanEnvs: true,
	}, out)
	return err
}

// RemoveAppWithOptions 使用完整卸载选项卸载已安装应用，并返回所有被清理的任务 ID 列表
func (s *AppService) RemoveAppWithOptions(taskID string, opts AppRemoveOptions, out io.Writer) ([]string, error) {
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

	log(">> 正在卸载应用 [%s]...", taskID)

	// 1. 使用 taskID 查找主应用任务实体（同时兼容传入 manifestID 的 CLI 场景）
	var masterTask models.Task
	if err := database.DB.Where("id = ? AND type = ?", taskID, constant.TaskTypeApp).First(&masterTask).Error; err != nil {
		if err2 := database.DB.Where("source_id = ? AND type = ?", "app:"+taskID, constant.TaskTypeApp).First(&masterTask).Error; err2 != nil {
			return nil, fmt.Errorf("未找到应用记录: %s", taskID)
		}
	}

	// 2. 调用公共逻辑执行清理
	deletedTaskIDs, err := CleanAppTaskAndData(&masterTask, opts, out)
	if err != nil {
		return nil, err
	}

	log(">> ✓ 应用 [%s] 卸载与关联资源清理完成！", masterTask.Name)
	return deletedTaskIDs, nil
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

// MarketplaceResult 应用市场数据载荷
type MarketplaceResult struct {
	Source          string                   `json:"source"`
	Apps            []map[string]interface{} `json:"apps"`
	GeneratedAtUTC8 string                   `json:"generated_at_utc8,omitempty"`
	BuildTime       string                   `json:"build_time,omitempty"`
	GeneratedAt     string                   `json:"generated_at,omitempty"`
}

// FetchMarketplace 从官方/镜像源矩阵拉取白虎应用市场索引并完成数据修饰
func (s *AppService) FetchMarketplace() (*MarketplaceResult, error) {
	branch := strings.TrimSpace(os.Getenv("BH_APPSTORE_BRANCH"))
	if branch == "" {
		branch = "main"
	}

	remoteCandidates := []string{
		"https://engigu.github.io/baihu-appstore/apps.json",
		fmt.Sprintf("https://ghproxy.net/https://raw.githubusercontent.com/engigu/baihu-appstore/%s/apps.json", branch),
		fmt.Sprintf("https://ghp.ci/https://raw.githubusercontent.com/engigu/baihu-appstore/%s/apps.json", branch),
		fmt.Sprintf("https://raw.githubusercontent.com/engigu/baihu-appstore/%s/apps.json", branch),
		fmt.Sprintf("https://cdn.jsdelivr.net/gh/engigu/baihu-appstore@%s/apps.json", branch),
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var resp *http.Response
	var err error
	nowTs := time.Now().Unix()

	for _, targetURL := range remoteCandidates {
		fetchURL := fmt.Sprintf("%s?t=%d", targetURL, nowTs)
		resp, err = client.Get(fetchURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		return &MarketplaceResult{
			Source: "none",
			Apps:   []map[string]interface{}{},
		}, nil
	}

	defer resp.Body.Close()
	body, rErr := io.ReadAll(resp.Body)
	if rErr != nil {
		return nil, fmt.Errorf("读取应用市场响应失败: %w", rErr)
	}

	var rawJSON interface{}
	if err := json.Unmarshal(body, &rawJSON); err != nil {
		return nil, fmt.Errorf("解析应用市场 JSON 失败: %w", err)
	}

	var appList []interface{}
	var rawMap map[string]interface{}
	if m, ok := rawJSON.(map[string]interface{}); ok {
		rawMap = m
		if arr, exists := m["apps"].([]interface{}); exists {
			appList = arr
		}
	} else if arr, ok := rawJSON.([]interface{}); ok {
		appList = arr
	}

	EnrichMarketplaceApps(appList)

	resultApps := make([]map[string]interface{}, 0, len(appList))
	for _, it := range appList {
		if m, ok := it.(map[string]interface{}); ok {
			resultApps = append(resultApps, m)
		}
	}

	res := &MarketplaceResult{
		Source: "remote",
		Apps:   resultApps,
	}
	if rawMap != nil {
		if v, ok := rawMap["generated_at_utc8"].(string); ok {
			res.GeneratedAtUTC8 = v
		}
		if v, ok := rawMap["build_time"].(string); ok {
			res.BuildTime = v
		}
		if v, ok := rawMap["generated_at"].(string); ok {
			res.GeneratedAt = v
		}
	}

	return res, nil
}

// EnrichMarketplaceApps 遍历应用列表，统一提取并注入结构化的 template 与规范契约
func EnrichMarketplaceApps(appList []interface{}) {
	for _, rawItem := range appList {
		appMap, isMap := rawItem.(map[string]interface{})
		if !isMap {
			continue
		}

		var manifest *AppManifest
		if rawYAML, hasYAML := appMap["manifest_raw"].(string); hasYAML && rawYAML != "" {
			manifest, _ = ParseManifestFromYAML([]byte(rawYAML))
		} else if tmpl, hasTmpl := appMap["template"]; hasTmpl {
			manifest = &AppManifest{Template: tmpl}
		}

		if manifest != nil {
			tplCfg := manifest.GetTypedTemplateConfig()
			if tplCfg != nil {
				appMap["template"] = tplCfg
			}
		}

		delete(appMap, "tag")
		delete(appMap, "languages")
	}
}

// GetMarketplaceApp 从应用市场拉取并获取指定 appID 的结构化 AppManifest 及原始属性字典
func (s *AppService) GetMarketplaceApp(appID string) (*AppManifest, map[string]interface{}, error) {
	appID = strings.TrimSpace(strings.ToLower(appID))
	if appID == "" {
		return nil, nil, fmt.Errorf("应用 ID 不能为空")
	}

	market, err := s.FetchMarketplace()
	if err != nil {
		return nil, nil, err
	}

	for _, appMap := range market.Apps {
		curID, _ := appMap["id"].(string)
		if strings.EqualFold(strings.TrimSpace(curID), appID) {
			rawYAMLBytes, err := yaml.Marshal(appMap)
			if err != nil {
				return nil, nil, fmt.Errorf("序列化应用定义失败: %w", err)
			}
			processedBytes, pErr := PreprocessYAMLTemplate(rawYAMLBytes)
			if pErr == nil {
				rawYAMLBytes = processedBytes
			}
			manifest, mErr := ParseManifestFromYAML(rawYAMLBytes)
			if mErr != nil {
				return nil, nil, fmt.Errorf("解析应用定义失败: %w", mErr)
			}
			return manifest, appMap, nil
		}
	}

	return nil, nil, fmt.Errorf("应用市场中未找到应用: %s", appID)
}

// ApplyMarketplaceApp 从应用市场一键直接部署安装指定应用
func (s *AppService) ApplyMarketplaceApp(appID string, opts ApplyOptions) (*ApplyResult, error) {
	manifest, appMap, err := s.GetMarketplaceApp(appID)
	if err != nil {
		return nil, err
	}

	rawYAMLBytes, _ := yaml.Marshal(appMap)
	opts.ManifestPath = fmt.Sprintf("marketplace:%s", manifest.ID)
	return DefaultApplier.Apply(manifest, rawYAMLBytes, opts)
}

