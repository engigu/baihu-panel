package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
)

// AgentSchedulerConfig Agent 调度器配置
type AgentSchedulerConfig struct {
	WorkerCount  int           `json:"worker_count"`
	QueueSize    int           `json:"queue_size"`
	RateInterval time.Duration `json:"rate_interval"`
	Verbose      bool          `json:"verbose"`
	StrictQueue  bool          `json:"strict_queue"`
}

// Value 序列化为数据库字符串
func (c AgentSchedulerConfig) Value() (driver.Value, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

// Scan 反序列化数据库字符串为结构体
func (c *AgentSchedulerConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return errors.New("invalid type for AgentSchedulerConfig")
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, c)
}

// Agent 远程执行代理
type Agent struct {
	ID              string               `json:"id" gorm:"primaryKey;size:20"`
	Name            string               `json:"name" gorm:"size:100;not null"`                 // Agent 名称
	Token           string               `json:"token" gorm:"size:64;index"`                    // 认证 Token（可重复使用）
	MachineID       string               `json:"machine_id" gorm:"size:64;uniqueIndex"`         // 机器识别码（唯一）
	Description     string               `json:"description" gorm:"size:255"`                   // 描述
	Status          string               `json:"status" gorm:"size:20;default:'pending';index"` // 状态: constant.AgentStatusOnline, constant.AgentStatusOffline
	LastSeen        *LocalTime           `json:"last_seen"`                                     // 最后心跳时间
	IP              string               `json:"ip" gorm:"size:45"`                             // Agent IP 地址
	Version         string               `json:"version" gorm:"size:50"`                        // Agent 版本
	BuildTime       string               `json:"build_time" gorm:"size:30"`                     // Agent 构建时间
	Hostname        string               `json:"hostname" gorm:"size:100"`                      // Agent 主机名
	OS              string               `json:"os" gorm:"size:20"`                             // 操作系统
	Arch            string               `json:"arch" gorm:"size:20"`                           // 架构
	ForceUpdate     bool                 `json:"force_update" gorm:"default:false"`             // 强制更新标志
	Enabled         *bool                `json:"enabled" gorm:"default:true"`                   // 是否启用
	SchedulerConfig AgentSchedulerConfig `json:"scheduler_config" gorm:"type:text"`             // 调度配置，以 JSON 字符串形式存储在 Text 类型字段中
	CreatedAt       LocalTime            `json:"created_at"`
	UpdatedAt       LocalTime            `json:"updated_at"`
}

func (Agent) TableName() string {
	return constant.TablePrefix + "agents"
}

// AgentToken Agent 令牌
type AgentToken struct {
	ID        string     `json:"id" gorm:"primaryKey;size:20"`
	Token     string     `json:"token" gorm:"size:64;uniqueIndex;not null"` // 令牌
	Remark    string     `json:"remark" gorm:"size:255"`                    // 备注
	MaxUses   int        `json:"max_uses" gorm:"default:0"`                 // 最大使用次数，0 表示无限制
	UsedCount int        `json:"used_count" gorm:"default:0"`               // 已使用次数
	ExpiresAt *LocalTime `json:"expires_at"`                                // 过期时间，null 表示永不过期
	Enabled   *bool      `json:"enabled" gorm:"default:true"`               // 是否启用
	CreatedAt LocalTime  `json:"created_at"`
	UpdatedAt LocalTime  `json:"updated_at"`
}

func (AgentToken) TableName() string {
	return constant.TablePrefix + "tokens"
}

// AgentTask Agent 任务配置（用于下发给 Agent）
type AgentTask struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Command     string              `json:"command"`
	PreCommand  string              `json:"pre_command"`
	PostCommand string              `json:"post_command"`
	Schedule    string              `json:"schedule"`
	Timeout     int                 `json:"timeout"`
	WorkDir     string              `json:"work_dir"`
	Envs        string              `json:"envs"`
	Languages   []map[string]string `json:"languages"`
	RandomRange int                 `json:"random_range"`
	Secrets     []string            `json:"secrets"`
	Enabled     bool                `json:"enabled"`
}

func (t AgentTask) GetID() string {
	return t.ID
}

func (t AgentTask) GetName() string {
	return t.Name
}

func (t AgentTask) GetCommand() string {
	return t.Command
}

func (t AgentTask) GetPreCommand() string {
	return t.PreCommand
}

func (t AgentTask) GetPostCommand() string {
	return t.PostCommand
}

func (t AgentTask) GetSchedule() string {
	return t.Schedule
}

func (t AgentTask) GetRandomRange() int {
	return t.RandomRange
}

func (t AgentTask) GetSecrets() []string {
	return t.Secrets
}

// AgentTaskResult Agent 上报的任务执行结果
type AgentTaskResult struct {
	TaskID    string `json:"task_id"`
	LogID     string `json:"log_id"`
	AgentID   string `json:"agent_id"`
	Command   string `json:"command"`
	Output    string `json:"output"`
	Error     string `json:"error"`    // 额外的系统错误信息
	Status    string `json:"status"`   // success, failed
	Duration  int64  `json:"duration"` // 耗时（毫秒）
	ExitCode  int    `json:"exit_code"`
	StartTime int64  `json:"start_time"` // Unix 时间戳
	EndTime   int64  `json:"end_time"`   // Unix 时间戳
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
