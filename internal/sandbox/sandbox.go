package sandbox

import (
	"os/exec"
	"github.com/engigu/baihu-panel/internal/models"
)

// Apply 适配不同平台的沙箱限制应用
func Apply(cmd *exec.Cmd, sandboxConfig *models.SandboxConfig) {
	applyPlatformSandbox(cmd, sandboxConfig)
}
