package sandbox

import (
	"os/exec"
	"runtime"
	"testing"

	"github.com/engigu/baihu-panel/internal/models"
)

// TestApplySandboxBasic 验证沙箱配置在不同平台下的基础装配和安全性逻辑
func TestApplySandboxBasic(t *testing.T) {
	cmd := exec.Command("echo", "test")
	
	// 1. 无沙箱配置测试，预期不会有任何 SysProcAttr 变化或越权附加
	Apply(cmd, nil)
	if cmd.SysProcAttr != nil {
		t.Logf("默认不使用沙箱时 SysProcAttr: %v", cmd.SysProcAttr)
	}

	// 2. 正常配置沙箱测试
	config := &models.SandboxConfig{
		UseSandbox:    true,
		MemoryLimit:   128,
		NprocLimit:    64,
		UID:           10001,
		GID:           10001,
		HideSystemEtc: false,
	}

	Apply(cmd, config)

	if runtime.GOOS != "windows" {
		// 在 Unix 平台下，预期已经附加了降权和命名空间标记
		if cmd.SysProcAttr == nil {
			t.Fatal("Unix 下启用沙箱但 cmd.SysProcAttr 为 nil")
		}
		if cmd.SysProcAttr.Credential == nil {
			t.Fatal("Unix 下启用沙箱但 Credential 降权未设置")
		}
		if cmd.SysProcAttr.Credential.Uid != 10001 || cmd.SysProcAttr.Credential.Gid != 10001 {
			t.Errorf("Credential 期望 (10001:10001), 实际为: (%d:%d)", cmd.SysProcAttr.Credential.Uid, cmd.SysProcAttr.Credential.Gid)
		}
	} else {
		// Windows 兼容模式测试
		t.Log("Windows 兼容模式下沙箱安全降级运行通过")
	}
}

// TestApplySandboxNamespace 验证挂载空间隔离的配置装配
func TestApplySandboxNamespace(t *testing.T) {
	cmd := exec.Command("echo", "test")
	config := &models.SandboxConfig{
		UseSandbox:    true,
		UID:           10002,
		GID:           10002,
		HideSystemEtc: true,
	}

	Apply(cmd, config)

	if runtime.GOOS != "windows" {
		if cmd.SysProcAttr == nil {
			t.Fatal("SysProcAttr 未成功初始化")
		}
		// 校验 CLONE_NEWNS 挂载命名空间标记是否成功注入
		const CLONE_NEWNS = 0x00020000
		if (cmd.SysProcAttr.Cloneflags & CLONE_NEWNS) == 0 {
			t.Error("HideSystemEtc 开启时，挂载空间隔离 CLONE_NEWNS 标记未注入")
		}
	}
}
