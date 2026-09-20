package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/models"
	"github.com/goccy/go-yaml"
)

var (
	validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ParseManifestFromYAML 从 YAML 字节切片中解析出 AppManifest
func ParseManifestFromYAML(data []byte) (*AppManifest, error) {
	// 1. 优先使用 template 进行全局宏占位符预处理替换
	processedData, err := PreprocessYAMLTemplate(data)
	if err == nil {
		data = processedData
	}

	var manifest AppManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("YAML 格式解析失败: %w", err)
	}

	if err := ValidateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// PreprocessYAMLTemplate 解析 YAML 中的 template 定义并对整个 YAML 文本执行全局宏占位符替换
func PreprocessYAMLTemplate(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// 1. 轻量提取顶层 template 定义
	var rawRoot struct {
		Template interface{} `yaml:"template"`
	}
	if err := yaml.Unmarshal(data, &rawRoot); err != nil || rawRoot.Template == nil {
		return data, nil
	}

	// 2. 收集所有模板占位符映射
	tplMap := extractTemplateVars(rawRoot.Template)
	if len(tplMap) == 0 {
		return data, nil
	}

	// 3. 执行全局占位符替换（支持 {key} 与 {{key}} 两种常用语法）
	yamlContent := string(data)
	for k, v := range tplMap {
		if k == "" {
			continue
		}
		yamlContent = strings.ReplaceAll(yamlContent, "{"+k+"}", v)
		yamlContent = strings.ReplaceAll(yamlContent, "{{"+k+"}}", v)
	}

	return []byte(yamlContent), nil
}

// extractTemplateVars 递归提取模板参数映射，支持 map 字典或 list 列表定义
func extractTemplateVars(raw interface{}) map[string]string {
	result := make(map[string]string)
	if raw == nil {
		return result
	}

	switch val := raw.(type) {
	case map[string]interface{}:
		for k, v := range val {
			result[k] = fmt.Sprintf("%v", v)
		}
	case map[interface{}]interface{}:
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", v)
		}
	case []interface{}:
		for _, item := range val {
			for k, v := range extractTemplateVars(item) {
				result[k] = v
			}
		}
	}

	return result
}

// ParseManifestFromFile 从本地 YAML 文件路径解析出 AppManifest
func ParseManifestFromFile(filePath string) (*AppManifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取应用描述文件失败 (%s): %w", filePath, err)
	}

	manifest, err := ParseManifestFromYAML(data)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

// ParseManifestFromURL 从远程 URL 下载并解析出 AppManifest
func ParseManifestFromURL(rawURL string, proxy string) (*AppManifest, error) {
	finalURL := strings.TrimSpace(rawURL)
	if proxy != "" && proxy != "none" {
		switch proxy {
		case "ghproxy":
			finalURL = "https://ghfast.top/" + finalURL
		case "mirror":
			finalURL = "https://mirror.ghproxy.com/" + finalURL
		default:
			if strings.HasPrefix(proxy, "http://") || strings.HasPrefix(proxy, "https://") {
				finalURL = strings.TrimRight(proxy, "/") + "/" + finalURL
			}
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", finalURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Baihu-App-Engine/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求远程应用规范文件失败 (%s): %w", finalURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载远程应用规范失败, HTTP 状态码: %d (%s)", resp.StatusCode, finalURL)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取远程数据失败: %w", err)
	}

	return ParseManifestFromYAML(data)
}

// ValidateManifest 校验 Manifest 的合法性
func ValidateManifest(m *AppManifest) error {
	if m == nil {
		return errors.New("应用描述对象不能为空")
	}
	return m.Validate()
}

var preferredKeyOrder = map[string]int{
	"spec_version":  1,
	"id":            2,
	"name":          3,
	"version":       4,
	"author":        5,
	"category":      6,
	"last_commit":   7,
	"template":      8,
	"description":   9,
	"icon":          10,
	"homepage":      11,
	"build_opts":    12,
	"schedule_opts": 13,
	"sources":       14,
	"setup":         15,
	"env_schema":    16,
	"tasks":         17,
	"scenarios":     18,
}

// SortManifestMapSlice 根据白虎应用规范 v1 标准固定节点顺序对 MapSlice 进行稳定排序
func SortManifestMapSlice(ms yaml.MapSlice) yaml.MapSlice {
	sorted := make(yaml.MapSlice, len(ms))
	copy(sorted, ms)
	sort.SliceStable(sorted, func(i, j int) bool {
		keyI := fmt.Sprintf("%v", sorted[i].Key)
		keyJ := fmt.Sprintf("%v", sorted[j].Key)
		orderI, hasI := preferredKeyOrder[keyI]
		orderJ, hasJ := preferredKeyOrder[keyJ]
		if hasI && hasJ {
			return orderI < orderJ
		}
		if hasI {
			return true
		}
		if hasJ {
			return false
		}
		return i < j
	})
	return sorted
}

// FormatManifestYAML 按照白虎规范 18 个标准节点物理顺序重排任意 YAML 字段
func FormatManifestYAML(rawYAML string) string {
	if strings.TrimSpace(rawYAML) == "" {
		return rawYAML
	}
	var ms yaml.MapSlice
	if err := yaml.Unmarshal([]byte(rawYAML), &ms); err != nil || len(ms) == 0 {
		return rawYAML
	}
	ms = SortManifestMapSlice(ms)
	bytes, err := yaml.Marshal(ms)
	if err != nil {
		return rawYAML
	}
	return string(bytes)
}

// getMapSliceValue 从 MapSlice 中按 Key 查找对应节点的 Value
func getMapSliceValue(slice yaml.MapSlice, key string) (interface{}, bool) {
	for _, item := range slice {
		if fmt.Sprintf("%v", item.Key) == key {
			return item.Value, true
		}
	}
	return nil, false
}

// setMapSliceValue 设置或覆盖 MapSlice 中指定 Key 的 Value
func setMapSliceValue(slice *yaml.MapSlice, key string, val interface{}) {
	for i, item := range *slice {
		if fmt.Sprintf("%v", item.Key) == key {
			(*slice)[i].Value = val
			return
		}
	}
	*slice = append(*slice, yaml.MapItem{Key: key, Value: val})
}

// getMapSliceChildValue 递归/二级查找 MapSlice 嵌套节点的子 Key 属性值
func getMapSliceChildValue(slice yaml.MapSlice, parentKey string, childKey string) (interface{}, bool) {
	if parentVal, ok := getMapSliceValue(slice, parentKey); ok {
		if childMS, isMS := parentVal.(yaml.MapSlice); isMS {
			return getMapSliceValue(childMS, childKey)
		}
	}
	return nil, false
}

// updateScenarioItemDefault 统一更新单项场景定义 (支持 yaml.MapSlice 或 map) 的 default 默认选中标记
func updateScenarioItemDefault(scRaw interface{}, targetScenario string) interface{} {
	switch sc := scRaw.(type) {
	case yaml.MapSlice:
		if scIDVal, hasID := getMapSliceValue(sc, "id"); hasID {
			scID := fmt.Sprintf("%v", scIDVal)
			setMapSliceValue(&sc, "default", scID == targetScenario)
		}
		return sc
	case map[string]interface{}:
		if scID, hasID := sc["id"].(string); hasID {
			sc["default"] = (scID == targetScenario)
		}
		return sc
	default:
		return scRaw
	}
}

// SyncManifestYAMLWithConfig 将 AppTaskConfig 中的 build_opts、scenario、schedule 等用户定制项同步写入 YAML 文本中，生成实例级定制 YAML
func SyncManifestYAMLWithConfig(rawYAML string, cfg *models.AppTaskConfig) string {
	if strings.TrimSpace(rawYAML) == "" || cfg == nil {
		return rawYAML
	}

	var ms yaml.MapSlice
	if err := yaml.Unmarshal([]byte(rawYAML), &ms); err != nil || len(ms) == 0 {
		return rawYAML
	}

	// 1. 同步 build_opts
	if cfg.BuildOpts != nil {
		overwriteTaskVal := true
		if cfg.BuildOpts.OverwriteTask != nil {
			overwriteTaskVal = *cfg.BuildOpts.OverwriteTask
		} else if existingVal, ok := getMapSliceChildValue(ms, "build_opts", "overwrite_task"); ok {
			if bVal, isBool := existingVal.(bool); isBool {
				overwriteTaskVal = bVal
			}
		}

		buildOptsMap := yaml.MapSlice{
			{Key: "force_setup", Value: cfg.BuildOpts.ForceSetup},
			{Key: "skip_setup", Value: cfg.BuildOpts.SkipSetup},
			{Key: "skip_sync", Value: cfg.BuildOpts.SkipSync},
			{Key: "overwrite_env", Value: cfg.BuildOpts.OverwriteEnv},
			{Key: "overwrite_task", Value: overwriteTaskVal},
		}
		setMapSliceValue(&ms, "build_opts", buildOptsMap)
	}

	// 2. 同步 schedule
	if cfg.Schedule != "" {
		setMapSliceValue(&ms, "schedule", cfg.Schedule)
	}

	// 3. 场景 default 标记同步（已注释：保持原始 App Manifest 的场景预设声明原汁原味，用户切换场景通过 masterTask 记录控制即可，无需重写 YAML 的 scenarios 节点）
	/*
	if cfg.CurrentScenario != "" {
		if scenariosRaw, ok := getMapSliceValue(ms, "scenarios"); ok {
			if scenariosList, isList := scenariosRaw.([]interface{}); isList {
				newScenarios := make([]interface{}, len(scenariosList))
				for j, scRaw := range scenariosList {
					newScenarios[j] = updateScenarioItemDefault(scRaw, cfg.CurrentScenario)
				}
				setMapSliceValue(&ms, "scenarios", newScenarios)
			}
		}
	}
	*/

	// 4. 依据白虎规范 18 个标准节点顺序进行稳定排序
	ms = SortManifestMapSlice(ms)

	updatedBytes, err := yaml.Marshal(ms)
	if err != nil {
		return rawYAML
	}

	return string(updatedBytes)
}
