package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// SandboxProfile 沙箱模板配置
type SandboxProfile struct {
	ID            string    `json:"id" gorm:"primaryKey;size:20"`
	Name          string    `json:"name" gorm:"size:255;not null;unique"` // 模板名称，例如 "标准受限沙箱"
	Description   string    `json:"description" gorm:"size:512"`          // 模板说明描述
	
	// 资源限制
	MemoryLimit   int       `json:"memory_limit" gorm:"default:128"`      // 内存限制 (MB)，为 0 表示不限制
	NprocLimit    int       `json:"nproc_limit" gorm:"default:64"`        // 最大子进程数，防止 fork 炸弹，0 表示不限制
	
	// 权限设置 (仅在 Linux 下生效)
	UID           int       `json:"uid" gorm:"default:10001"`             // 动态指定的执行用户 ID (如 10001, 10002)
	GID           int       `json:"gid" gorm:"default:10001"`             // 动态指定的执行用户组 ID
	
	// 敏感文件隔离
	HideSystemEtc bool      `json:"hide_system_etc" gorm:"default:true"`  // 是否在沙箱子进程中隐藏/隔离 /etc 等敏感目录
	
	CreatedAt     LocalTime `json:"created_at"`
	UpdatedAt     LocalTime `json:"updated_at"`
}

func (SandboxProfile) TableName() string {
	return constant.TablePrefix + "sandbox_profiles"
}

// SandboxConfig 用于在执行器和调度层之间传输的沙箱具体配置
type SandboxConfig struct {
	UseSandbox    bool `json:"use_sandbox"`
	MemoryLimit   int  `json:"memory_limit"`
	NprocLimit    int  `json:"nproc_limit"`
	UID           int  `json:"uid"`
	GID           int  `json:"gid"`
	HideSystemEtc bool `json:"hide_system_etc"`
}
