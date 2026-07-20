//go:build windows

package sandbox

import (
	"os/exec"
	"github.com/engigu/baihu-panel/internal/models"
)

func applyPlatformSandbox(cmd *exec.Cmd, sandboxConfig *models.SandboxConfig) {
	ApplyWindowsSandbox(cmd, sandboxConfig)
}
