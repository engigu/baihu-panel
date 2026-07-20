//go:build windows

package sandbox

import (
	"os/exec"
	"github.com/engigu/baihu-panel/internal/models"
)

// ApplyWindowsSandbox 在 Windows 环境下应用轻量级资源限制 (Windows 降级实现，预留 API 接口)
func ApplyWindowsSandbox(cmd *exec.Cmd, sandbox *models.SandboxConfig) {
	if sandbox == nil || !sandbox.UseSandbox {
		return
	}
	
	// 在 Windows 上暂时做空处理，或基于 Job Object 限制（未来可集成）
}
