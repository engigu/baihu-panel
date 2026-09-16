package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/engigu/baihu-panel/internal/constant"
)

// TaskLanguages 自定义语言配置列表类型，处理 JSON 序列化
type TaskLanguages []map[string]string

func (t TaskLanguages) Value() (driver.Value, error) {
	if t == nil {
		return "[]", nil
	}
	b, err := json.Marshal(t)
	return string(b), err
}

func (t *TaskLanguages) Scan(v interface{}) error {
	if v == nil {
		*t = nil
		return nil
	}
	var data []byte
	switch s := v.(type) {
	case string:
		data = []byte(s)
	case []byte:
		data = s
	default:
		return fmt.Errorf("invalid type for TaskLanguages: %T", v)
	}
	return json.Unmarshal(data, t)
}

// CleanConfig 清理配置结构
type CleanConfig struct {
	Type string `json:"type"` // "day" 或 "count"
	Keep int    `json:"keep"` // 保留天数或条数
}

// CommonConfig 通用运行策略
type CommonConfig struct {
	Concurrency int  `json:"task_concurrency"` // 0: disable concurrency, 1: enable concurrency
	AllEnvs     bool `json:"task_all_envs"`    // 开启则注入全部环境变量
}

// UnifiedTaskConfig 统一配置结构体，包含 common, repo, app 分区 map/struct 映射
type UnifiedTaskConfig struct {
	Common *CommonConfig  `json:"common,omitempty"`
	Repo   *RepoConfig    `json:"repo,omitempty"`
	App    *AppTaskConfig `json:"app,omitempty"`
}

// ToJSON 将 UnifiedTaskConfig 序列化为 JSON 字符串
func (c UnifiedTaskConfig) ToJSON() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// ParseUnifiedTaskConfig 解析 JSON 配置到 UnifiedTaskConfig 实体
func ParseUnifiedTaskConfig(raw string) UnifiedTaskConfig {
	var cfg UnifiedTaskConfig
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	return cfg
}

// GetCommon 快捷获取 Common 配置
func (c UnifiedTaskConfig) GetCommon() *CommonConfig {
	return c.Common
}

// GetRepo 快捷获取 Repo 配置
func (c UnifiedTaskConfig) GetRepo() *RepoConfig {
	return c.Repo
}

// GetApp 快捷获取 App 配置
func (c UnifiedTaskConfig) GetApp() *AppTaskConfig {
	return c.App
}

// RepoConfig 仓库同步配置
type RepoConfig struct {
	SourceType     string `json:"source_type,omitempty"`
	SourceURL      string `json:"source_url,omitempty"`
	TargetPath     string `json:"target_path,omitempty"`
	Branch         string `json:"branch,omitempty"`
	SparsePath     string `json:"sparse_path,omitempty"`
	SingleFile     bool   `json:"single_file,omitempty"`
	Proxy          string `json:"proxy,omitempty"`
	ProxyURL       string `json:"proxy_url,omitempty"`
	AuthToken      string `json:"auth_token,omitempty"`
	HttpProxy      string `json:"http_proxy,omitempty"`
	WhitelistPaths string `json:"whitelist_paths,omitempty"`
	Blacklist      string `json:"blacklist,omitempty"`
	Dependence     string `json:"dependence,omitempty"`
	Extensions     string `json:"extensions,omitempty"`
	AutoAddCron    bool   `json:"auto_add_cron,omitempty"`
	CommentToTask  string `json:"commenttotask,omitempty"`
	RepoSource     string `json:"repo_source,omitempty"`
	RepoDirName    string `json:"repo_dir_name,omitempty"`
}

// AppTaskConfig 应用任务配置
type AppTaskConfig struct {
	ID              string            `json:"id,omitempty"`
	Name            string            `json:"name,omitempty"`
	Version         string            `json:"version,omitempty"`
	Author          string            `json:"author,omitempty"`
	Category        string            `json:"category,omitempty"`
	Description     string            `json:"description,omitempty"`
	Icon            string            `json:"icon,omitempty"`
	Homepage        string            `json:"homepage,omitempty"`
	ManifestPath    string            `json:"manifest_path,omitempty"`
	ManifestRaw     string            `json:"manifest_raw,omitempty"`
	CurrentScenario string            `json:"current_scenario,omitempty"`
	Status          string            `json:"status,omitempty"`
	EnvValues       map[string]string `json:"env_values,omitempty"`
	BuildOpts       *AppBuildOpts     `json:"build_opts,omitempty"`
}

// AppBuildOpts 高级构建与部署控制选项
type AppBuildOpts struct {
	ForceSetup bool `json:"force_setup"` // 强制重新编译
	SkipSetup  bool `json:"skip_setup"`  // 跳过环境与依赖安装
	SkipSync   bool `json:"skip_sync"`   // 跳过代码源同步
}

// Task 代表一个计划任务
type Task struct {
	ID             string        `json:"id" gorm:"primaryKey;size:20"`
	Name           string        `json:"name" gorm:"size:255;not null"`
	Remark         string        `json:"remark" gorm:"size:255;default:''"`
	PinType        string        `json:"pin_type" gorm:"size:20;default:none;index"` // 置顶类型: constant.PinTypeNone, constant.PinTypeTop
	Command        BigText       `json:"command"`                                    // 普通任务的命令
	PreCommand     BigText       `json:"pre_command"`                                // 执行前的命令
	PostCommand    BigText       `json:"post_command"`                               // 执行后的命令
	Tags           string        `json:"tags" gorm:"-"`                              // 标签，逗号分隔
	Type           string        `json:"type" gorm:"size:20;default:'task'"`         // 任务类型: constant.TaskTypeNormal, constant.TaskTypeRepo
	TriggerType    string        `json:"trigger_type" gorm:"size:25;default:'cron'"` // 触发类型: constant.TriggerTypeCron, constant.TriggerTypeBaihuStartup
	Config         BigText       `json:"config"`                                     // 旧配置 JSON（保留不删）
	UnifiedConfig  BigText       `json:"unified_config"`                             // 升级后的统一 JSON 配置 {'common':{}, 'repo':{}, 'app':{}}
	Schedule       string        `json:"schedule" gorm:"size:100"`                   // cron 表达式
	Timeout        int           `json:"timeout" gorm:"default:30"`                  // 超时时间（分钟），默认30分钟
	WorkDir        string        `json:"work_dir" gorm:"size:255;default:''"`        // 工作目录，为空则使用 scripts 目录
	CleanConfig    string        `json:"clean_config" gorm:"size:255;default:''"`    // 清理配置 JSON
	Envs           BigText       `json:"envs" gorm:"-"`                              // 环境变量ID列表，逗号分隔
	Languages      TaskLanguages `json:"languages" gorm:"type:text"`                 // 针对本地任务的语言配置列表
	AgentID        *string       `json:"agent_id" gorm:"size:20;index"`              // Agent ID，为空表示本地执行
	RetryCount     int           `json:"retry_count" gorm:"default:0"`               // 失败重试次数
	RetryInterval  int           `json:"retry_interval" gorm:"default:0"`            // 失败重试间隔(秒)
	RandomRange    int           `json:"random_range" gorm:"default:0"`              // 随机延迟范围(秒)
	Enabled        *bool         `json:"enabled" gorm:"default:true"`
	RunningGo      BigText       `json:"running_go"` // 正在运行的 go routine id 数组 (JSON)
	RuntimeEnvs    []string      `json:"-" gorm:"-"` // 运行时环境变量（非持久化）
	RuntimeSecrets []string      `json:"-" gorm:"-"` // 运行时安全机密（非持久化）
	LastRun        *LocalTime    `json:"last_run"`
	NextRun        *LocalTime    `json:"next_run"`
	SourceID       string        `json:"source_id" gorm:"size:255;index"`   // 脚本资源唯一标识（路径 sanitized / app:yml_id）
	RepoTaskID     string        `json:"repo_task_id" gorm:"size:20;index"` // 所属的仓库任务 ID
	CreatedAt      LocalTime     `json:"created_at"`
	UpdatedAt      LocalTime     `json:"updated_at"`
}

func (t *Task) IsRunning() bool {
	if string(t.RunningGo) == "" || string(t.RunningGo) == "[]" {
		return false
	}
	return true
}

func (Task) TableName() string {
	return constant.TablePrefix + "tasks"
}

func (t *Task) GetID() string {
	return t.ID
}

func (t *Task) GetName() string {
	return t.Name
}

func (t *Task) GetCommand() string {
	return string(t.Command)
}

func (t *Task) GetPreCommand() string {
	return string(t.PreCommand)
}

func (t *Task) GetPostCommand() string {
	return string(t.PostCommand)
}

func (t *Task) GetTimeout() int {
	return t.Timeout
}

func (t *Task) GetWorkDir() string {
	return t.WorkDir
}

func (t *Task) GetEnvs() string {
	return string(t.Envs)
}

func (t *Task) GetLanguages() []map[string]string {
	return []map[string]string(t.Languages)
}

func (t *Task) GetEnvVars() []string {
	return t.RuntimeEnvs
}

func (t *Task) GetSecrets() []string {
	return t.RuntimeSecrets
}

func (t *Task) GetUseMise() bool {
	return t.AgentID == nil || *t.AgentID == ""
}

func (t *Task) UseMise() bool {
	return t.GetUseMise()
}

// CronTask 计划任务接口
func (t *Task) GetSchedule() string {
	return t.Schedule
}

func (t *Task) GetRandomRange() int {
	return t.RandomRange
}

// GetUnifiedConfig 获取并解析当前 Task 的 UnifiedTaskConfig 配置结构
func (t *Task) GetUnifiedConfig() UnifiedTaskConfig {
	if t == nil || string(t.UnifiedConfig) == "" {
		return UnifiedTaskConfig{}
	}
	return ParseUnifiedTaskConfig(string(t.UnifiedConfig))
}

// GetCommonConfig 快捷获取当前 Task 的 CommonConfig 配置
func (t *Task) GetCommonConfig() *CommonConfig {
	cfg := t.GetUnifiedConfig()
	return cfg.Common
}

// GetRepoConfig 快捷获取当前 Task 的 RepoConfig 配置
func (t *Task) GetRepoConfig() *RepoConfig {
	cfg := t.GetUnifiedConfig()
	return cfg.Repo
}

// GetAppConfig 快捷获取当前 Task 的 AppTaskConfig 配置
func (t *Task) GetAppConfig() *AppTaskConfig {
	cfg := t.GetUnifiedConfig()
	return cfg.App
}

// GetManifestID 快捷从 UnifiedConfig 获取声明式 App 的 Manifest ID
func (t *Task) GetManifestID() string {
	if appCfg := t.GetAppConfig(); appCfg != nil {
		return appCfg.ID
	}
	return ""
}

// TaskLog 代表任务执行的日志记录
type TaskLog struct {
	ID        string     `json:"id" gorm:"primaryKey;size:20"`
	TaskID    string     `json:"task_id" gorm:"size:20;index"`
	AgentID   *string    `json:"agent_id" gorm:"size:20;index"` // Agent ID，为空表示本地执行
	Command   BigText    `json:"command"`
	Output    BigText    `json:"-"`                           // gzip+base64 压缩后的日志
	Error     BigText    `json:"error"`                       // 额外的系统错误信息
	Status    string     `json:"status" gorm:"size:20;index"` // success, failed
	Duration  int64      `json:"duration"`                    // 执行耗时（毫秒）
	ExitCode  int        `json:"exit_code"`
	StartTime *LocalTime `json:"start_time"`
	EndTime   *LocalTime `json:"end_time"`
	CreatedAt LocalTime  `json:"created_at"`
}

func (TaskLog) TableName() string {
	return constant.TablePrefix + "task_logs"
}
