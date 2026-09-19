package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
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

// SyncManifestYAMLWithConfig 将 AppTaskConfig 中的 build_opts、scenario、schedule 等用户定制项同步写入 YAML 文本中，生成实例级定制 YAML
func SyncManifestYAMLWithConfig(rawYAML string, cfg *models.AppTaskConfig) string {
	if strings.TrimSpace(rawYAML) == "" || cfg == nil {
		return rawYAML
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal([]byte(rawYAML), &m); err != nil || m == nil {
		return rawYAML
	}

	// 1. 同步 build_opts
	if cfg.BuildOpts != nil {
		buildOptsMap := map[string]interface{}{
			"force_setup":   cfg.BuildOpts.ForceSetup,
			"skip_setup":    cfg.BuildOpts.SkipSetup,
			"skip_sync":     cfg.BuildOpts.SkipSync,
			"overwrite_env": cfg.BuildOpts.OverwriteEnv,
		}
		if cfg.BuildOpts.OverwriteTask != nil {
			buildOptsMap["overwrite_task"] = *cfg.BuildOpts.OverwriteTask
		} else {
			buildOptsMap["overwrite_task"] = true
		}
		m["build_opts"] = buildOptsMap
	}

	// 2. 同步 schedule
	if cfg.Schedule != "" {
		m["schedule"] = cfg.Schedule
	}

	// 3. 同步场景（若有默认场景且被覆盖）
	if cfg.CurrentScenario != "" {
		if scenariosRaw, ok := m["scenarios"].([]interface{}); ok {
			for _, scRaw := range scenariosRaw {
				if scMap, isMap := scRaw.(map[string]interface{}); isMap {
					if scID, hasID := scMap["id"].(string); hasID {
						scMap["default"] = (scID == cfg.CurrentScenario)
					}
				}
			}
		}
	}

	// 4. 重新序列化为 YAML 文本
	updatedBytes, err := yaml.Marshal(m)
	if err != nil {
		return rawYAML
	}

	return string(updatedBytes)
}
