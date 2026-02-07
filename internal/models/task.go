package models

import (
	"fmt"

	"github.com/engigu/baihu-panel/internal/constant"

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

// Task represents a scheduled task
type Task struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:255;not null"`
	Command     string         `json:"command" gorm:"type:text"`                // 普通任务的命令
	Type        string         `json:"type" gorm:"size:20;default:'task'"`      // 任务类型: task(普通任务), repo(仓库同步)
	Config      string         `json:"config" gorm:"type:text"`                 // 配置 JSON（仓库同步配置等）
	Schedule    string         `json:"schedule" gorm:"size:100"`                // cron expression
	Timeout     int            `json:"timeout" gorm:"default:30"`               // 超时时间（分钟），默认30分钟
	WorkDir     string         `json:"work_dir" gorm:"size:255;default:''"`     // 工作目录，为空则使用 scripts 目录
	CleanConfig string         `json:"clean_config" gorm:"size:255;default:''"` // 清理配置 JSON
	Envs        string         `json:"envs" gorm:"size:255;default:''"`         // 环境变量ID列表，逗号分隔
	AgentID     *uint          `json:"agent_id" gorm:"index"`                   // Agent ID，为空表示本地执行
	Enabled     bool           `json:"enabled" gorm:"default:true"`
	RunningGo   string         `json:"running_go" gorm:"type:text"` // 正在运行的 go routine id 数组 (JSON)
	LastRun     *LocalTime     `json:"last_run"`
	NextRun     *LocalTime     `json:"next_run"`
	CreatedAt   LocalTime      `json:"created_at"`
	UpdatedAt   LocalTime      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Task) TableName() string {
	return constant.TablePrefix + "tasks"
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

func (t *Task) GetSchedule() string {
	return t.Schedule
}

// TaskLog represents a log entry for task execution
type TaskLog struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	TaskID    uint       `json:"task_id" gorm:"index"`
	AgentID   *uint      `json:"agent_id" gorm:"index"` // Agent ID，为空表示本地执行
	Command   string     `json:"command" gorm:"type:text"`
	Output    string     `json:"-" gorm:"type:longtext"` // gzip+base64 compressed
	Status    string     `json:"status" gorm:"size:20"`  // success, failed
	Duration  int64      `json:"duration"`               // milliseconds
	ExitCode  int        `json:"exit_code"`
	StartTime *LocalTime `json:"start_time"`
	EndTime   *LocalTime `json:"end_time"`
	CreatedAt LocalTime  `json:"created_at"`
}

func (TaskLog) TableName() string {
	return constant.TablePrefix + "task_logs"
}
