package models

import (
	"fmt"

	"github.com/engigu/baihu-panel/internal/constant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
	AuthToken  string `json:"auth_token"`  // 认证 Token
}

// TaskConfig  任务配置  RepoConfig+TaskConfig=task.config
type TaskConfig struct {
	Concurrency int `json:"$task_concurrency"` // 0: disable concurrency, 1: enable concurrency
}

// Task 代表一个计划任务
type Task struct {
	ID          uint                `json:"id" gorm:"primaryKey"`
	UUID        string              `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	Name        string              `json:"name" gorm:"size:255;not null"`
	Command     string              `json:"command" gorm:"type:text"`                   // 普通任务的命令
	Tags        string              `json:"tags" gorm:"size:255;default:''"`            // 标签，逗号分隔
	Type        string              `json:"type" gorm:"size:20;default:'task'"`         // 任务类型: constant.TaskTypeNormal, constant.TaskTypeRepo
	TriggerType string              `json:"trigger_type" gorm:"size:25;default:'cron'"` // 触发类型: constant.TriggerTypeCron, constant.TriggerTypeBaihuStartup
	Config      string              `json:"config" gorm:"type:text"`                    // 配置 JSON（仓库同步配置等）
	Schedule    string              `json:"schedule" gorm:"size:100"`                   // cron 表达式
	Timeout     int                 `json:"timeout" gorm:"default:30"`                  // 超时时间（分钟），默认30分钟
	WorkDir     string              `json:"work_dir" gorm:"size:255;default:''"`        // 工作目录，为空则使用 scripts 目录
	CleanConfig string              `json:"clean_config" gorm:"size:255;default:''"`    // 清理配置 JSON
	Envs        string              `json:"envs" gorm:"size:255;default:''"`            // 环境变量ID列表，逗号分隔
	Languages   []map[string]string `json:"languages" gorm:"serializer:json;type:text"` // 针对本地任务的语言配置列表
	AgentID       *string             `json:"agent_id" gorm:"index"`                      // Agent UUID，为空表示本地执行
	RetryCount    int                 `json:"retry_count" gorm:"default:0"`               // 失败重试次数
	RetryInterval int                 `json:"retry_interval" gorm:"default:0"`            // 失败重试间隔(秒)
	Enabled       bool                `json:"enabled" gorm:"default:true"`
	RunningGo     string              `json:"running_go" gorm:"type:text"` // 正在运行的 go routine id 数组 (JSON)
	LastRun       *LocalTime          `json:"last_run"`
	NextRun       *LocalTime          `json:"next_run"`
	CreatedAt     LocalTime           `json:"created_at"`
	UpdatedAt     LocalTime           `json:"updated_at"`
	DeletedAt     gorm.DeletedAt      `json:"-" gorm:"index"`
}

func (Task) TableName() string {
	return constant.TablePrefix + "tasks"
}

func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	if t.UUID == "" {
		t.UUID = uuid.New().String()
	}
	return
}

func (t *Task) GetID() string {
	return fmt.Sprintf("%d", t.ID)
}

func (t *Task) GetName() string {
	return t.Name
}

func (t *Task) GetCommand() string {
	return t.Command
}

func (t *Task) GetTimeout() int {
	return t.Timeout
}

func (t *Task) GetWorkDir() string {
	return t.WorkDir
}

func (t *Task) GetEnvs() string {
	return t.Envs
}

func (t *Task) GetLanguages() []map[string]string {
	return t.Languages
}

func (t *Task) GetUseMise() bool {
	return t.AgentID == nil || *t.AgentID == 0
}

func (t *Task) UseMise() bool {
	return t.GetUseMise()
}

func (t *Task) GetSchedule() string {
	return t.Schedule
}

// TaskLog 代表任务执行的日志记录
type TaskLog struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	UUID      string     `json:"uuid" gorm:"size:36;uniqueIndex;not null"`
	TaskID    string     `json:"task_id" gorm:"index"`
	AgentID   *string    `json:"agent_id" gorm:"index"` // Agent UUID，为空表示本地执行
	Command   string     `json:"command" gorm:"type:text"`
	Output    string     `json:"-" gorm:"type:longtext"`      // gzip+base64 压缩后的日志
	Error     string     `json:"error" gorm:"type:text"`      // 额外的系统错误信息
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

func (tl *TaskLog) BeforeCreate(tx *gorm.DB) (err error) {
	if tl.UUID == "" {
		tl.UUID = uuid.New().String()
	}
	return
}
