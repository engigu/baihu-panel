package app

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/engigu/baihu-panel/internal/models"
)

var (
	manifestIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	validEnvTypes   = map[string]bool{
		"string":  true,
		"secret":  true,
		"boolean": true,
		"select":  true,
		"number":  true,
	}
)

// AppManifest 白虎面板应用规范定义文件完整结构 (Baihu Application Specification)
// 100% 对应 apps/BiliBiliToolPro/app.yaml 完整结构规范，用于严格校验与结构化提取
type AppManifest struct {
	SpecVersion string            `json:"spec_version" yaml:"spec_version"`                       // 规范版本 (如 v1)
	ID          string            `json:"id" yaml:"id"`                                           // 应用全局唯一标识符 (如 bilibili-tool-pro)
	Name        string            `json:"name" yaml:"name"`                                       // 应用中文名称
	Version     string            `json:"version" yaml:"version"`                                 // 语义化版本号 (如 2.1.0)
	Author      string            `json:"author,omitempty" yaml:"author,omitempty"`               // 作者或维护者 (如 RayWangQvQ)
	Category    string            `json:"category,omitempty" yaml:"category,omitempty"`           // 所属分类 (如 福利签到)
	Template    interface{}       `json:"template,omitempty" yaml:"template,omitempty"`           // 模板宏变量声明池 (支持 map 或 list)
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`     // 应用功能描述
	Icon        string            `json:"icon,omitempty" yaml:"icon,omitempty"`                   // 应用图标 URL
	Homepage    string            `json:"homepage,omitempty" yaml:"homepage,omitempty"`           // 项目主页或 GitHub 仓库地址
	Sources     []AppSource       `json:"sources,omitempty" yaml:"sources,omitempty"`             // 脚本/代码源列表
	Setup       AppSetup          `json:"setup" yaml:"setup"`                                     // 原生 Shell 环境与依赖编排
	EnvSchema   []AppEnvItem      `json:"env_schema,omitempty" yaml:"env_schema,omitempty"`       // 环境变量声明契约
	SyncRules   *AppSyncRules     `json:"sync_rules,omitempty" yaml:"sync_rules,omitempty"`       // 任务映射规则与任务清单
	Tasks       []AppTaskItem     `json:"tasks,omitempty" yaml:"tasks,omitempty"`                 // 顶层直接声明的任务 (兼容旧格式)
	Scenarios   []AppScenarioItem `json:"scenarios,omitempty" yaml:"scenarios,omitempty"`         // 使用场景预设列表
}

// AppSource 代码源/发布物规范，100% 映射 reposync 参数规范
type AppSource struct {
	ID             string `json:"id" yaml:"id"`                                                       // [必填] 源唯一标识符
	SourceType     string `json:"source_type" yaml:"source_type"`                                     // [必填] 对应 --source-type: git 或 url
	SourceURL      string `json:"source_url" yaml:"source_url"`                                       // [必填] 对应 --source-url: 仓库或文件直链
	Branch         string `json:"branch,omitempty" yaml:"branch,omitempty"`                           // 对应 --branch: 分支名
	Path           string `json:"path,omitempty" yaml:"path,omitempty"`                               // 对应 --path: 稀疏检出相对子目录
	SingleFile     bool   `json:"single_file,omitempty" yaml:"single_file,omitempty"`                 // 对应 --single-file: 单文件下载模式
	Proxy          string `json:"proxy,omitempty" yaml:"proxy,omitempty"`                             // 对应 --proxy: none / ghproxy / mirror / custom
	ProxyURL       string `json:"proxy_url,omitempty" yaml:"proxy_url,omitempty"`                     // 对应 --proxy-url: 自定义代理前缀
	AuthToken      string `json:"auth_token,omitempty" yaml:"auth_token,omitempty"`                   // 对应 --auth-token: 访问私有源 Token
	HttpProxy      string `json:"http_proxy,omitempty" yaml:"http_proxy,omitempty"`                   // 对应 --http-proxy: HTTP/SOCKS 代理
	WhitelistPaths string `json:"whitelist_paths,omitempty" yaml:"whitelist_paths,omitempty"`         // 对应 --whitelist-paths: 白名单保留路径
	Blacklist      string `json:"blacklist,omitempty" yaml:"blacklist,omitempty"`                     // 对应 --blacklist: 过滤目录/文件关键词
	TargetPath     string `json:"target_path,omitempty" yaml:"target_path,omitempty"`                 // 对应 --target-path: 相对存储目录
}

// AppSetup 原生 Shell 环境与依赖编排
type AppSetup struct {
	Check       string `json:"check,omitempty" yaml:"check,omitempty"`               // [可选] 依赖快速探测命令 (退出码 0 即跳过安装)
	Install     string `json:"install" yaml:"install"`                               // [必填] 原生 Shell 依赖安装/构建命令
	PostInstall string `json:"post_install,omitempty" yaml:"post_install,omitempty"` // [可选] 安装/构建完成后执行的后置 Shell 初始化命令
	Uninstall   string `json:"uninstall,omitempty" yaml:"uninstall,omitempty"`       // [可选] 应用卸载清理命令
}

// AppEnvOption 环境变量下拉选择项
type AppEnvOption struct {
	Label string `json:"label" yaml:"label"` // 选项显示名称
	Value string `json:"value" yaml:"value"` // 选项实际写入值
}

// AppEnvItem 环境变量声明契约
type AppEnvItem struct {
	Key         string         `json:"key" yaml:"key"`                             // [必填] 环境变量键名 (如 Ray_BiliBiliCookies__0)
	Label       string         `json:"label" yaml:"label"`                         // [必填] 交互表单显示名称
	Type        string         `json:"type" yaml:"type"`                           // [必填] 类型: string, secret, boolean, select, number
	Required    bool           `json:"required" yaml:"required"`                   // 是否必填
	Default     interface{}    `json:"default,omitempty" yaml:"default,omitempty"` // 默认值
	Tag         string         `json:"tag,omitempty" yaml:"tag,omitempty"`         // 变量分类标签 (用于多账号分组与过滤)
	Description string         `json:"description,omitempty" yaml:"description,omitempty"` // 详细说明
	Placeholder string         `json:"placeholder,omitempty" yaml:"placeholder,omitempty"` // 输入框占位提示
	Options     []AppEnvOption `json:"options,omitempty" yaml:"options,omitempty"`         // select 下拉类型的候选项
}

// AppSyncRules 任务生成与映射规则容器
type AppSyncRules struct {
	Defaults AppTaskDefaults `json:"defaults,omitempty" yaml:"defaults,omitempty"` // 全局任务默认参数
	Tasks    []AppTaskItem   `json:"tasks" yaml:"tasks"`                           // 任务清单定义
}

// AppTaskDefaults 任务默认参数
type AppTaskDefaults struct {
	Timeout       int    `json:"timeout,omitempty" yaml:"timeout,omitempty"`               // 默认超时（分钟）
	RetryCount    int    `json:"retry_count,omitempty" yaml:"retry_count,omitempty"`       // 失败重试次数
	RetryInterval int    `json:"retry_interval,omitempty" yaml:"retry_interval,omitempty"` // 重试间隔（秒）
	WorkDir       string `json:"work_dir,omitempty" yaml:"work_dir,omitempty"`             // 运行时工作目录
	Language      string `json:"language,omitempty" yaml:"language,omitempty"`             // 默认运行语言 (支持空格区分多语言)
	Languages     string `json:"languages,omitempty" yaml:"languages,omitempty"`           // 别名兼容复数形式
	Tag           string `json:"tag,omitempty" yaml:"tag,omitempty"`                       // 默认任务分类标签
}

// AppTaskItem 单个任务定义
type AppTaskItem struct {
	ID          string `json:"id" yaml:"id"`                                                 // [必填] 任务唯一标识符 (如 daily)
	Name        string `json:"name" yaml:"name"`                                             // [必填] 任务显示名称
	Source      string `json:"source,omitempty" yaml:"source,omitempty"`                     // 关联的代码源 ID
	File        string `json:"file,omitempty" yaml:"file,omitempty"`                         // 脚本相对文件路径
	Command     string `json:"command,omitempty" yaml:"command,omitempty"`                   // 直接执行命令 (若无 file)
	Tag         string `json:"tag,omitempty" yaml:"tag,omitempty"`                           // 任务分类标签 (继承 defaults.Tag 或单独指定)
	DefaultCron string `json:"default_cron,omitempty" yaml:"default_cron,omitempty"`         // 默认 Cron 表达式 (推荐 6 位)
	Cron        string `json:"cron,omitempty" yaml:"cron,omitempty"`                         // 兼容 cron 字段
	Timeout     int    `json:"timeout,omitempty" yaml:"timeout,omitempty"`                   // 单独覆盖超时分钟
	Language    string `json:"language,omitempty" yaml:"language,omitempty"`                 // 单独覆盖运行语言 (支持空格区分多语言)
	Languages   string `json:"languages,omitempty" yaml:"languages,omitempty"`               // 别名兼容复数形式
	WorkDir     string `json:"work_dir,omitempty" yaml:"work_dir,omitempty"`                 // 单独覆盖运行时工作目录
	Enabled     bool   `json:"enabled" yaml:"enabled"`                                       // 默认是否开启
}

// LanguageSpec 强类型语言环境规格 (如 name: "python", version: "3.11" 或 name: "node", version: "")
type LanguageSpec struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// String 转换为声明字符串格式 (若有版本则为 name@version，否则为 name)
func (l LanguageSpec) String() string {
	if l.Version == "" {
		return l.Name
	}
	return l.Name + "@" + l.Version
}

// ToMap 转换为底层 TaskLanguages 所用的 map[string]string 格式
func (l LanguageSpec) ToMap() map[string]string {
	return map[string]string{
		"name":    l.Name,
		"version": l.Version,
	}
}

// ParseLanguageSpecs 将语言配置字符串 (如 "python node" 或 "dotnet@8.0.425 node@22.0.0") 解析为 []LanguageSpec 切片
// 空格区分多个语言，支持不带版本 (如 "python") 或带版本 (如 "dotnet@8.0.425")
func ParseLanguageSpecs(langStr string) []LanguageSpec {
	langs := ParseLanguageSpec(langStr)
	specs := make([]LanguageSpec, len(langs))
	for i, l := range langs {
		specs[i] = LanguageSpec{
			Name:    l["name"],
			Version: l["version"],
		}
	}
	return specs
}

// ParseLanguageSpec 将语言配置字符串解析为 models.TaskLanguages (如 []map[string]string)
// 空格区分多个语言，支持不带版本 (如 "python") 或带版本 (如 "dotnet@8.0.425")
func ParseLanguageSpec(langStr string) models.TaskLanguages {
	var taskLangs models.TaskLanguages
	for _, item := range strings.Fields(langStr) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "@", 2)
		langName := strings.TrimSpace(parts[0])
		if langName == "" {
			continue
		}
		langVer := ""
		if len(parts) > 1 {
			langVer = strings.TrimSpace(parts[1])
		}
		taskLangs = append(taskLangs, map[string]string{
			"name":    langName,
			"version": langVer,
		})
	}
	return taskLangs
}

// FormatLanguageSpecs 将规格切片格式化为空格分隔的字符串
func FormatLanguageSpecs(specs []LanguageSpec) string {
	var parts []string
	for _, s := range specs {
		if strings.TrimSpace(s.Name) != "" {
			parts = append(parts, s.String())
		}
	}
	return strings.Join(parts, " ")
}

// ValidateLanguageSpec 校验运行语言声明字符串的格式合法性
// 规则：以空格区分多个语言，支持纯语言名称 (如 "python node") 或带版本格式 (如 "dotnet@8.0.425 node@22.0.0")，不强制要求版本号
func ValidateLanguageSpec(langStr string) error {
	seen := make(map[string]bool)
	for _, item := range strings.Fields(langStr) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "@") {
			return fmt.Errorf("运行环境配置项 '%s' 格式不合法: 缺少语言名称，不能以 '@' 开头", item)
		}
		parts := strings.SplitN(item, "@", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			return fmt.Errorf("运行环境配置项 '%s' 语言名称不能为空", item)
		}
		lowerName := strings.ToLower(name)
		if seen[lowerName] {
			return fmt.Errorf("运行环境配置中存在重复的语言声明: '%s'", name)
		}
		seen[lowerName] = true
	}
	return nil
}

// GetLanguage 获取任务声明的运行语言配置字符串 (优先 Language，次选 Languages)
func (t *AppTaskItem) GetLanguage() string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.Language) != "" {
		return strings.TrimSpace(t.Language)
	}
	return strings.TrimSpace(t.Languages)
}

// GetParsedLanguages 将任务语言配置字符串解析为标准 TaskLanguages 对象 (空格区分多语言，支持纯语言名或带版本号)
func (t *AppTaskItem) GetParsedLanguages() models.TaskLanguages {
	if t == nil {
		return nil
	}
	return ParseLanguageSpec(t.GetLanguage())
}

// GetLanguageSpecs 将任务语言配置解析为强类型 LanguageSpec 切片
func (t *AppTaskItem) GetLanguageSpecs() []LanguageSpec {
	if t == nil {
		return nil
	}
	return ParseLanguageSpecs(t.GetLanguage())
}

// GetLanguageNames 提取任务所声明的所有运行语言名称列表 (如 ["python", "node"])
func (t *AppTaskItem) GetLanguageNames() []string {
	var names []string
	seen := make(map[string]bool)
	for _, s := range t.GetLanguageSpecs() {
		if s.Name != "" && !seen[s.Name] {
			seen[s.Name] = true
			names = append(names, s.Name)
		}
	}
	return names
}

// GetLanguageVersion 提取任务中指定语言的版本号 (若无版本则返回空字符串，若未声明该语言则 found 为 false)
func (t *AppTaskItem) GetLanguageVersion(langName string) (version string, found bool) {
	for _, s := range t.GetLanguageSpecs() {
		if strings.EqualFold(s.Name, langName) {
			return s.Version, true
		}
	}
	return "", false
}

// HasLanguage 判断任务是否依赖指定语言 (忽略大小写)
func (t *AppTaskItem) HasLanguage(langName string) bool {
	_, found := t.GetLanguageVersion(langName)
	return found
}

// ValidateLanguage 校验任务自身声明的运行语言规格合法性
func (t *AppTaskItem) ValidateLanguage() error {
	lang := t.GetLanguage()
	if lang == "" {
		return nil
	}
	return ValidateLanguageSpec(lang)
}

// GetLanguage 获取默认配置中的运行语言字符串 (优先 Language，次选 Languages)
func (d *AppTaskDefaults) GetLanguage() string {
	if d == nil {
		return ""
	}
	if strings.TrimSpace(d.Language) != "" {
		return strings.TrimSpace(d.Language)
	}
	return strings.TrimSpace(d.Languages)
}

// GetParsedLanguages 将默认运行语言配置解析为标准 TaskLanguages 对象
func (d *AppTaskDefaults) GetParsedLanguages() models.TaskLanguages {
	if d == nil {
		return nil
	}
	return ParseLanguageSpec(d.GetLanguage())
}

// GetLanguageSpecs 将默认配置解析为强类型 LanguageSpec 切片
func (d *AppTaskDefaults) GetLanguageSpecs() []LanguageSpec {
	if d == nil {
		return nil
	}
	return ParseLanguageSpecs(d.GetLanguage())
}

// GetLanguageNames 提取默认配置所声明的所有语言名称列表
func (d *AppTaskDefaults) GetLanguageNames() []string {
	var names []string
	seen := make(map[string]bool)
	for _, s := range d.GetLanguageSpecs() {
		if s.Name != "" && !seen[s.Name] {
			seen[s.Name] = true
			names = append(names, s.Name)
		}
	}
	return names
}

// GetLanguageVersion 提取默认配置中指定语言的版本号
func (d *AppTaskDefaults) GetLanguageVersion(langName string) (version string, found bool) {
	for _, s := range d.GetLanguageSpecs() {
		if strings.EqualFold(s.Name, langName) {
			return s.Version, true
		}
	}
	return "", false
}

// HasLanguage 判断默认配置是否包含指定语言
func (d *AppTaskDefaults) HasLanguage(langName string) bool {
	_, found := d.GetLanguageVersion(langName)
	return found
}

// ValidateLanguage 校验默认配置中的运行语言规格合法性
func (d *AppTaskDefaults) ValidateLanguage() error {
	lang := d.GetLanguage()
	if lang == "" {
		return nil
	}
	return ValidateLanguageSpec(lang)
}

// AppTaskPreset 场景模板中的任务覆盖参数
type AppTaskPreset struct {
	Enabled *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"` // 是否启用
	Cron    string `json:"cron,omitempty" yaml:"cron,omitempty"`       // 覆盖的 Cron 表达式
}

// AppScenarioItem 使用场景预设模板
type AppScenarioItem struct {
	ID          string                   `json:"id" yaml:"id"`                                     // [必填] 场景唯一标识符 (如 minimal, standard, hardcore)
	Name        string                   `json:"name" yaml:"name"`                                 // [必填] 场景名称
	Description string                   `json:"description,omitempty" yaml:"description,omitempty"` // 场景说明
	Default     bool                     `json:"default" yaml:"default"`                           // 是否为推荐/默认场景
	TaskPresets map[string]AppTaskPreset `json:"task_presets,omitempty" yaml:"task_presets,omitempty"` // 各任务的预设状态覆盖
}

// ==============================================================================
// 结构数据校验方法 (Validation)
// ==============================================================================

// Validate 对 AppManifest 进行全面严谨的数据完整性与规范校验
func (m *AppManifest) Validate() error {
	if m == nil {
		return errors.New("应用描述对象不能为空")
	}

	// 1. 基础元数据校验
	m.ID = strings.TrimSpace(m.ID)
	if m.ID == "" {
		return errors.New("应用 id 不能为空")
	}
	if !manifestIDRegex.MatchString(m.ID) {
		return fmt.Errorf("应用 id '%s' 包含非法字符，仅允许字母、数字、下划线及中划线", m.ID)
	}

	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("应用名称 (name) 不能为空")
	}

	if strings.TrimSpace(m.Version) == "" {
		m.Version = "1.0.0"
	}
	if strings.TrimSpace(m.SpecVersion) == "" {
		m.SpecVersion = "v1"
	}

	// 2. 代码源合法性校验
	sourceIDSet := make(map[string]bool)
	for i, src := range m.Sources {
		srcID := strings.TrimSpace(src.ID)
		if srcID == "" {
			return fmt.Errorf("第 %d 个代码源 (sources) 的 id 不能为空", i+1)
		}
		if sourceIDSet[srcID] {
			return fmt.Errorf("存在重复的代码源 id: '%s'", srcID)
		}
		sourceIDSet[srcID] = true

		if strings.TrimSpace(src.SourceURL) == "" {
			return fmt.Errorf("代码源 '%s' 的 source_url 不能为空", srcID)
		}
		srcType := strings.ToLower(strings.TrimSpace(src.SourceType))
		if srcType != "git" && srcType != "url" {
			return fmt.Errorf("代码源 '%s' 的 source_type 必须为 'git' 或 'url'", srcID)
		}
	}

	// 3. 任务清单校验
	tasks := m.GetTasks()
	if len(tasks) == 0 {
		return errors.New("应用必须声明至少一个可执行任务 (tasks)")
	}

	taskIDSet := make(map[string]bool)
	for idx, t := range tasks {
		tID := strings.TrimSpace(t.ID)
		if tID == "" {
			return fmt.Errorf("第 %d 个任务的 id 不能为空", idx+1)
		}
		if !manifestIDRegex.MatchString(tID) {
			return fmt.Errorf("任务 id '%s' 包含非法字符，仅允许字母、数字、下划线及中划线", tID)
		}
		if taskIDSet[tID] {
			return fmt.Errorf("存在重复的任务 id: '%s'", tID)
		}
		taskIDSet[tID] = true

		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("任务 '%s' 的显示名称 (name) 不能为空", tID)
		}

		if strings.TrimSpace(t.Command) == "" && strings.TrimSpace(t.File) == "" {
			return fmt.Errorf("任务 '%s' 必须指定 command 执行命令或 file 脚本文件", tID)
		}

		// 检查关联代码源是否存在
		if t.Source != "" && len(m.Sources) > 0 && !sourceIDSet[t.Source] {
			return fmt.Errorf("任务 '%s' 关联的代码源 '%s' 在 sources 列表中不存在", tID, t.Source)
		}
	}

	// 4. 集中校验所有任务与全局默认运行环境配置 (空格区分多语言，支持纯名称如 python node 或带版本号如 dotnet@8.0.425)
	if err := m.ValidateLanguages(); err != nil {
		return err
	}

	// 5. 环境变量声明校验
	envKeySet := make(map[string]bool)
	for i := range m.EnvSchema {
		key := strings.TrimSpace(m.EnvSchema[i].Key)
		if key == "" {
			return fmt.Errorf("第 %d 个环境变量声明的 key 不能为空", i+1)
		}
		if envKeySet[key] {
			return fmt.Errorf("存在重复的环境变量 key: '%s'", key)
		}
		envKeySet[key] = true

		if strings.TrimSpace(m.EnvSchema[i].Label) == "" {
			m.EnvSchema[i].Label = key
		}

		envType := strings.ToLower(strings.TrimSpace(m.EnvSchema[i].Type))
		if envType == "" {
			envType = "string"
			m.EnvSchema[i].Type = "string"
		}
		if !validEnvTypes[envType] {
			return fmt.Errorf("环境变量 '%s' 的类型 '%s' 不合法 (支持: string, secret, boolean, select, number)", key, envType)
		}

		if envType == "select" && len(m.EnvSchema[i].Options) == 0 {
			return fmt.Errorf("select 类型的环境变量 '%s' 必须提供 options 候选列表", key)
		}

		// 标签兜底
		if strings.TrimSpace(m.EnvSchema[i].Tag) == "" {
			m.EnvSchema[i].Tag = m.ID
		}
	}

	// 5. 场景预设校验
	scenarioIDSet := make(map[string]bool)
	for i, sc := range m.Scenarios {
		scID := strings.TrimSpace(sc.ID)
		if scID == "" {
			return fmt.Errorf("第 %d 个场景预设的 id 不能为空", i+1)
		}
		if scenarioIDSet[scID] {
			return fmt.Errorf("存在重复的场景 id: '%s'", scID)
		}
		scenarioIDSet[scID] = true

		if strings.TrimSpace(sc.Name) == "" {
			return fmt.Errorf("场景 '%s' 的 name 不能为空", scID)
		}

		for targetTaskID := range sc.TaskPresets {
			if !taskIDSet[targetTaskID] {
				return fmt.Errorf("场景 '%s' 引用的预设任务 id '%s' 不存在于应用任务列表中", scID, targetTaskID)
			}
		}
	}

	return nil
}

// ==============================================================================
// 结构数据提取方法 (Data Extraction)
// ==============================================================================

// GetTasks 提取所有任务列表（自动继承 sync_rules.defaults 全局默认参数）
func (m *AppManifest) GetTasks() []AppTaskItem {
	var rawList []AppTaskItem
	if m.SyncRules != nil && len(m.SyncRules.Tasks) > 0 {
		rawList = m.SyncRules.Tasks
	} else if len(m.Tasks) > 0 {
		rawList = m.Tasks
	}

	if m.SyncRules == nil {
		return rawList
	}

	// 注入 defaults 默认参数
	defs := m.SyncRules.Defaults
	result := make([]AppTaskItem, len(rawList))
	for i, t := range rawList {
		item := t
		if item.Timeout <= 0 && defs.Timeout > 0 {
			item.Timeout = defs.Timeout
		}
		if item.WorkDir == "" && defs.WorkDir != "" {
			item.WorkDir = defs.WorkDir
		}
		if item.Language == "" && item.Languages == "" {
			if defs.Language != "" {
				item.Language = defs.Language
			} else if defs.Languages != "" {
				item.Languages = defs.Languages
			}
		}
		if item.Tag == "" && defs.Tag != "" {
			item.Tag = defs.Tag
		}
		result[i] = item
	}
	return result
}

// GetTask 根据 ID 提取单个任务定义
func (m *AppManifest) GetTask(id string) (*AppTaskItem, bool) {
	for _, t := range m.GetTasks() {
		if t.ID == id {
			return &t, true
		}
	}
	return nil, false
}

// GetEnvSchema 提取环境变量契约列表
func (m *AppManifest) GetEnvSchema() []AppEnvItem {
	return m.EnvSchema
}

// GetEnvItem 根据 key 提取环境变量契约项
func (m *AppManifest) GetEnvItem(key string) (*AppEnvItem, bool) {
	for _, e := range m.EnvSchema {
		if e.Key == key {
			return &e, true
		}
	}
	return nil, false
}

// GetSources 提取代码源列表
func (m *AppManifest) GetSources() []AppSource {
	return m.Sources
}

// GetSource 根据 id 提取代码源
func (m *AppManifest) GetSource(id string) (*AppSource, bool) {
	for _, s := range m.Sources {
		if s.ID == id {
			return &s, true
		}
	}
	return nil, false
}

// GetScenarios 提取所有场景模板列表
func (m *AppManifest) GetScenarios() []AppScenarioItem {
	return m.Scenarios
}

// GetScenario 根据 id 提取指定场景预设
func (m *AppManifest) GetScenario(id string) (*AppScenarioItem, bool) {
	for _, sc := range m.Scenarios {
		if sc.ID == id {
			return &sc, true
		}
	}
	return nil, false
}

// GetDefaultScenario 提取默认场景预设（若无显式 default 则返回第一个场景）
func (m *AppManifest) GetDefaultScenario() *AppScenarioItem {
	for _, sc := range m.Scenarios {
		if sc.Default {
			return &sc
		}
	}
	if len(m.Scenarios) > 0 {
		return &m.Scenarios[0]
	}
	return nil
}

// ResolveScenarioTaskState 计算指定场景下某个任务的启停状态与 Cron 表达式
func (m *AppManifest) ResolveScenarioTaskState(scenarioID string, taskID string) (enabled bool, cron string) {
	task, found := m.GetTask(taskID)
	if !found {
		return false, ""
	}

	enabled = task.Enabled
	cron = task.Cron
	if cron == "" {
		cron = task.DefaultCron
	}

	sc, foundSc := m.GetScenario(scenarioID)
	if foundSc && sc.TaskPresets != nil {
		if preset, ok := sc.TaskPresets[taskID]; ok {
			if preset.Enabled != nil {
				enabled = *preset.Enabled
			}
			if preset.Cron != "" {
				cron = preset.Cron
			}
		}
	}

	return enabled, cron
}

// GetDefaultLanguage 提取应用全局默认语言原始配置字符串
func (m *AppManifest) GetDefaultLanguage() string {
	if m.SyncRules != nil {
		return m.SyncRules.Defaults.GetLanguage()
	}
	return ""
}

// GetDefaultLanguages 提取应用全局默认运行语言解析列表
func (m *AppManifest) GetDefaultLanguages() models.TaskLanguages {
	if m.SyncRules != nil {
		return m.SyncRules.Defaults.GetParsedLanguages()
	}
	return nil
}

// GetAllLanguages 提取整个应用所声明的全部运行环境配置 (自动汇总 Defaults 与 Tasks 并去重)
// 空格区分多语言，支持纯名称如 "python node" 或带版本号如 "dotnet@8.0.425"
func (m *AppManifest) GetAllLanguages() models.TaskLanguages {
	seen := make(map[string]bool)
	var allLangs models.TaskLanguages

	appendLang := func(l map[string]string) {
		name := l["name"]
		if name == "" {
			return
		}
		version := l["version"]
		key := name + "@" + version
		if !seen[key] {
			seen[key] = true
			allLangs = append(allLangs, map[string]string{
				"name":    name,
				"version": version,
			})
		}
	}

	if m.SyncRules != nil {
		for _, l := range m.SyncRules.Defaults.GetParsedLanguages() {
			appendLang(l)
		}
	}

	for _, t := range m.GetTasks() {
		for _, l := range t.GetParsedLanguages() {
			appendLang(l)
		}
	}

	return allLangs
}

// GetAllLanguageNames 提取应用所有任务与默认配置中声明的语言名称去重列表 (如 ["dotnet", "node", "python"])
func (m *AppManifest) GetAllLanguageNames() []string {
	var names []string
	seen := make(map[string]bool)
	for _, l := range m.GetAllLanguages() {
		name := l["name"]
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// GetAllLanguageSpecs 提取应用所声明的全部运行环境强类型规格列表
func (m *AppManifest) GetAllLanguageSpecs() []LanguageSpec {
	langs := m.GetAllLanguages()
	specs := make([]LanguageSpec, len(langs))
	for i, l := range langs {
		specs[i] = LanguageSpec{
			Name:    l["name"],
			Version: l["version"],
		}
	}
	return specs
}

// GetTaskLanguages 提取指定任务最终生效的运行语言列表 (若任务未单独配置则自动继承 defaults)
func (m *AppManifest) GetTaskLanguages(taskID string) (models.TaskLanguages, bool) {
	t, found := m.GetTask(taskID)
	if !found {
		return nil, false
	}
	return t.GetParsedLanguages(), true
}

// GetTasksByLanguage 筛选依赖指定语言名称的所有任务清单 (忽略大小写，如筛选出所有包含 "dotnet" 的任务)
func (m *AppManifest) GetTasksByLanguage(langName string) []AppTaskItem {
	var matched []AppTaskItem
	for _, t := range m.GetTasks() {
		if t.HasLanguage(langName) {
			matched = append(matched, t)
		}
	}
	return matched
}

// HasLanguage 判断整个应用是否依赖某种语言 (Defaults 或任意任务中存在即返回 true)
func (m *AppManifest) HasLanguage(langName string) bool {
	for _, name := range m.GetAllLanguageNames() {
		if strings.EqualFold(name, langName) {
			return true
		}
	}
	return false
}

// ValidateLanguages 集中校验应用中所有任务与全局默认配置的运行语言规格
func (m *AppManifest) ValidateLanguages() error {
	if m.SyncRules != nil {
		if err := m.SyncRules.Defaults.ValidateLanguage(); err != nil {
			return fmt.Errorf("全局任务默认配置 (sync_rules.defaults.language) %w", err)
		}
	}
	for idx, t := range m.GetTasks() {
		tID := t.ID
		if tID == "" {
			tID = fmt.Sprintf("#%d", idx+1)
		}
		if err := t.ValidateLanguage(); err != nil {
			return fmt.Errorf("任务 '%s' 的运行环境 (language) %w", tID, err)
		}
	}
	return nil
}

// ExtractMetadata 提取用于应用市场 (apps.json) 索引与列表展示的标准元数据摘要
func (m *AppManifest) ExtractMetadata() map[string]interface{} {
	tasks := m.GetTasks()
	tasksSummary := make([]map[string]interface{}, len(tasks))
	for i, t := range tasks {
		tasksSummary[i] = map[string]interface{}{
			"id":        t.ID,
			"name":      t.Name,
			"tag":       t.Tag,
			"command":   t.Command,
			"cron":      t.DefaultCron,
			"enabled":   t.Enabled,
			"language":  t.GetLanguage(),
			"languages": t.GetParsedLanguages(),
		}
	}

	scenariosSummary := make([]map[string]interface{}, len(m.Scenarios))
	for i, s := range m.Scenarios {
		scenariosSummary[i] = map[string]interface{}{
			"id":          s.ID,
			"name":        s.Name,
			"description": s.Description,
			"default":     s.Default,
		}
	}

	return map[string]interface{}{
		"spec_version":    m.SpecVersion,
		"id":              m.ID,
		"name":            m.Name,
		"version":         m.Version,
		"author":          m.Author,
		"category":        m.Category,
		"description":     m.Description,
		"icon":            m.Icon,
		"homepage":        m.Homepage,
		"languages":       m.GetAllLanguages(),
		"tasks_count":     len(tasks),
		"tasks":           tasksSummary,
		"env_count":       len(m.EnvSchema),
		"env_schema":      m.EnvSchema,
		"scenarios_count": len(m.Scenarios),
		"scenarios":       scenariosSummary,
	}
}
