package models

import (
	"github.com/engigu/baihu-panel/internal/constant"

	"gorm.io/gorm"
)

// Agent 远程执行代理
type Agent struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`           // Agent 名称
	Token       string         `json:"token" gorm:"size:64;index"`              // 认证 Token（可重复使用）
	MachineID   string         `json:"machine_id" gorm:"size:64;uniqueIndex"`   // 机器识别码（唯一）
	Description string         `json:"description" gorm:"size:255"`             // 描述
	Status      string         `json:"status" gorm:"size:20;default:'pending'"` // 状态: pending(待审核), online, offline, blocked(拉黑)
	LastSeen    *LocalTime     `json:"last_seen"`                               // 最后心跳时间
	IP          string         `json:"ip" gorm:"size:45"`                       // Agent IP 地址
	Version     string         `json:"version" gorm:"size:50"`                  // Agent 版本
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

// AgentToken Agent 令牌
type AgentToken struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Token     string         `json:"token" gorm:"size:64;uniqueIndex;not null"` // 令牌
	Remark    string         `json:"remark" gorm:"size:255"`                    // 备注
	MaxUses   int            `json:"max_uses" gorm:"default:0"`                 // 最大使用次数，0 表示无限制
	UsedCount int            `json:"used_count" gorm:"default:0"`               // 已使用次数
	ExpiresAt *LocalTime     `json:"expires_at"`                                // 过期时间，null 表示永不过期
	Enabled   bool           `json:"enabled" gorm:"default:true"`               // 是否启用
	CreatedAt LocalTime      `json:"created_at"`
	UpdatedAt LocalTime      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AgentToken) TableName() string {
	return constant.TablePrefix + "tokens"
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
	LogID     uint   `json:"log_id"`
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
	Name      string `json:"name"`
	Hostname  string `json:"hostname"`
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	Token     string `json:"token"`      // 注册令牌
	MachineID string `json:"machine_id"` // 机器识别码
}
