package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/engigu/baihu-panel/internal/constant"

	"gorm.io/gorm"
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

// RepoConfig 仓库同步配置
type RepoConfig struct {
	SourceType string `json:"source_type"` // url 或 git
	SourceURL  string `json:"source_url"`  // 源地址
	TargetPath string `json:"target_path"` // 目标路径
	Branch     string `json:"branch"`      // Git 分支
	SparsePath string `json:"sparse_path"` // 稀疏检出路径（仅拉取指定目录或文件）
	SingleFile bool   `json:"single_file"` // 单文件模式（直接下载文件而非 sparse-checkout）
	Proxy      string `json:"proxy"`       // 代理类型: none, ghproxy, mirror, custom
	ProxyURL   string `json:"proxy_url"`   // 自定义代理地址
	AuthToken      string `json:"auth_token"`      // 认证 Token
	WhitelistPaths string `json:"whitelist_paths"` // 同步时保留的路径及脚本筛选白名单关键词，逗号或竖线分割
	Blacklist      string `json:"blacklist"`       // 脚本筛选黑名单关键词，竖线分割
	Dependence     string `json:"dependence"`      // 脚本依赖文件关键词，竖线分割
	Extensions     string `json:"extensions"`      // 脚本文件后缀关键词，竖线分割
	AutoAddCron    bool   `json:"auto_add_cron"`   // 自动解析脚本注释添加定时任务
	RepoSource     string `json:"repo_source"`     // 仓库来源，如果是选择了这个 ql 导入的仓库，= ql
}

// TaskConfig  任务配置  RepoConfig+TaskConfig=task.config
type TaskConfig struct {
	Concurrency int  `json:"$task_concurrency"` // 0: disable concurrency, 1: enable concurrency
	AllEnvs     bool `json:"$task_all_envs"`    // 开启则注入全部环境变量
}

// Task 代表一个计划任务
type Task struct {
	ID            string              `json:"id" gorm:"primaryKey;size:20"`
	Name          string              `json:"name" gorm:"size:255;not null"`
	Remark        string              `json:"remark" gorm:"size:255;default:''"`
	Command       BigText             `json:"command"`                   // 普通任务的命令
	Tags          string              `json:"tags" gorm:"size:255;default:''"`            // 标签，逗号分隔
	Type          string              `json:"type" gorm:"size:20;default:'task'"`         // 任务类型: constant.TaskTypeNormal, constant.TaskTypeRepo
	TriggerType   string              `json:"trigger_type" gorm:"size:25;default:'cron'"` // 触发类型: constant.TriggerTypeCron, constant.TriggerTypeBaihuStartup
	Config        BigText             `json:"config"`                    // 配置 JSON（仓库同步配置等）
	Schedule      string              `json:"schedule" gorm:"size:100"`                   // cron 表达式
	Timeout       int                 `json:"timeout" gorm:"default:30"`                  // 超时时间（分钟），默认30分钟
	WorkDir       string              `json:"work_dir" gorm:"size:255;default:''"`        // 工作目录，为空则使用 scripts 目录
	CleanConfig   string              `json:"clean_config" gorm:"size:255;default:''"`    // 清理配置 JSON
	Envs          BigText             `json:"envs"`                      // 环境变量ID列表，逗号分隔
	Languages     TaskLanguages       `json:"languages" gorm:"type:text"`                      // 针对本地任务的语言配置列表
	AgentID       *string             `json:"agent_id" gorm:"size:20;index"`              // Agent ID，为空表示本地执行
	RetryCount    int                 `json:"retry_count" gorm:"default:0"`               // 失败重试次数
	RetryInterval int                 `json:"retry_interval" gorm:"default:0"`            // 失败重试间隔(秒)
	RandomRange   int                 `json:"random_range" gorm:"default:0"`              // 随机延迟范围(秒)
	Enabled       bool                `json:"enabled" gorm:"default:true"`
	RunningGo     BigText             `json:"running_go"` // 正在运行的 go routine id 数组 (JSON)
	RuntimeEnvs   []string            `json:"-" gorm:"-"`                  // 运行时环境变量（非持久化）
	RuntimeSecrets []string           `json:"-" gorm:"-"`                  // 运行时安全机密（非持久化）
	LastRun       *LocalTime          `json:"last_run"`
	NextRun       *LocalTime          `json:"next_run"`
	SourceID      string              `json:"source_id" gorm:"size:255;index"`            // 脚本资源唯一标识（路径 sanitized）
	RepoTaskID    string              `json:"repo_task_id" gorm:"size:20;index"`          // 所属的仓库任务 ID
	CreatedAt     LocalTime           `json:"created_at"`
	UpdatedAt     LocalTime           `json:"updated_at"`
	DeletedAt     gorm.DeletedAt      `json:"-" gorm:"index"`
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

// TaskLog 代表任务执行的日志记录
type TaskLog struct {
	ID        string     `json:"id" gorm:"primaryKey;size:20"`
	TaskID    string     `json:"task_id" gorm:"size:20;index"`
	AgentID   *string    `json:"agent_id" gorm:"size:20;index"` // Agent ID，为空表示本地执行
	Command   BigText    `json:"command"`
	Output    BigText    `json:"-"`          // gzip+base64 压缩后的日志
	Error     BigText    `json:"error"`      // 额外的系统错误信息
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
