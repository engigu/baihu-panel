package models

import (
	"baihu/internal/constant"

	"gorm.io/gorm"
)

// SyncTask 同步任务实体
type SyncTask struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:255;not null"`           // 任务名称
	SourceType  string         `json:"source_type" gorm:"size:20;not null"`     // 源类型: url, git
	SourceURL   string         `json:"source_url" gorm:"size:500;not null"`     // 源地址
	Branch      string         `json:"branch" gorm:"size:100;default:''"`       // Git 分支
	TargetPath  string         `json:"target_path" gorm:"size:255;not null"`    // 目标路径
	Schedule    string         `json:"schedule" gorm:"size:100"`                // cron 表达式
	Proxy       string         `json:"proxy" gorm:"size:100;default:''"`        // 代理: none, ghproxy, mirror, custom
	ProxyURL    string         `json:"proxy_url" gorm:"size:255;default:''"`    // 自定义代理 URL
	AuthToken   string         `json:"auth_token" gorm:"size:500;default:''"`   // 认证 Token
	CleanConfig string         `json:"clean_config" gorm:"size:255;default:''"` // 清理配置 JSON
	Enabled     bool           `json:"enabled" gorm:"default:true"`             // 是否启用
	LastSync    *LocalTime     `json:"last_sync"`                               // 上次同步时间
	NextSync    *LocalTime     `json:"next_sync"`                               // 下次同步时间
	LastStatus  string         `json:"last_status" gorm:"size:20;default:''"`   // 上次同步状态
	CreatedAt   LocalTime      `json:"created_at"`
	UpdatedAt   LocalTime      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (SyncTask) TableName() string {
	return constant.TablePrefix + "sync_tasks"
}

// SyncTaskLog 同步任务执行日志
type SyncTaskLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	SyncTaskID uint      `json:"sync_task_id" gorm:"index"`
	SourceURL  string    `json:"source_url" gorm:"size:500"`
	TargetPath string    `json:"target_path" gorm:"size:255"`
	Output     string    `json:"-" gorm:"type:longtext"` // gzip+base64 compressed
	Status     string    `json:"status" gorm:"size:20"`  // success, failed
	Duration   int64     `json:"duration"`               // milliseconds
	CreatedAt  LocalTime `json:"created_at"`
}

func (SyncTaskLog) TableName() string {
	return constant.TablePrefix + "sync_task_logs"
}
