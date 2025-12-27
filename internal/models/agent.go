package models

import (
	"baihu/internal/constant"

	"gorm.io/gorm"
)

// Agent 远程执行代理
type Agent struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`           // Agent 名称
	Token       string         `json:"token" gorm:"size:64;uniqueIndex"`        // 认证 Token
	Description string         `json:"description" gorm:"size:255"`             // 描述
	Status      string         `json:"status" gorm:"size:20;default:'pending'"` // 状态: pending(待审核), online, offline
	LastSeen    *LocalTime     `json:"last_seen"`                               // 最后心跳时间
	IP          string         `json:"ip" gorm:"size:45"`                       // Agent IP 地址
	Version     string         `json:"version" gorm:"size:20"`                  // Agent 版本
	BuildTime   string         `json:"build_time" gorm:"size:30"`               // Agent 构建时间
	Hostname    string         `json:"hostname" gorm:"size:100"`                // Agent 主机名
	OS          string         `json:"os" gorm:"size:20"`                       // 操作系统
	Arch        string         `json:"arch" gorm:"size:20"`                     // 架构
	ForceUpdate bool           `json:"force_update" gorm:"default:false"`       // 强制更新标志
	Enabled     bool           `json:"enabled" gorm:"default:true"`             // 是否启用
	CreatedAt   LocalTime      `json:"created_at"`
	UpdatedAt   LocalTime      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Agent) TableName() string {
	return constant.TablePrefix + "agents"
}

// AgentTask Agent 任务配置（用于下发给 Agent）
type AgentTask struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	Schedule string `json:"schedule"`
	Timeout  int    `json:"timeout"`
	WorkDir  string `json:"work_dir"`
	Envs     string `json:"envs"`
	Enabled  bool   `json:"enabled"`
}

// AgentTaskResult Agent 上报的任务执行结果
type AgentTaskResult struct {
	TaskID    uint   `json:"task_id"`
	AgentID   uint   `json:"agent_id"`
	Command   string `json:"command"`
	Output    string `json:"output"`
	Status    string `json:"status"`   // success, failed
	Duration  int64  `json:"duration"` // milliseconds
	ExitCode  int    `json:"exit_code"`
	StartTime int64  `json:"start_time"` // unix timestamp
	EndTime   int64  `json:"end_time"`   // unix timestamp
}

// AgentRegisterRequest Agent 注册请求
type AgentRegisterRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Version  string `json:"version"`
}
