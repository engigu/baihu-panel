//go:build !windows

package sandbox

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"github.com/engigu/baihu-panel/internal/models"
)

// ApplyUnixSandbox 在 Unix 环境下应用用户权限降权与 rlimit 限制
func ApplyUnixSandbox(cmd *exec.Cmd, sandbox *models.SandboxConfig) {
	if sandbox == nil || !sandbox.UseSandbox {
		return
	}

	var sysAttr *syscall.SysProcAttr
	if cmd.SysProcAttr != nil {
		sysAttr = cmd.SysProcAttr
	} else {
		sysAttr = &syscall.SysProcAttr{}
	}

	// 如果是 chpst，则降权直接由 chpst 自身设定完成，此处无需在 Go 级别附加 Credential，否则会导致冲突
	if cmd.Path != "chpst" && !strings.HasSuffix(cmd.Path, "/chpst") {
		sysAttr.Credential = &syscall.Credential{
			Uid: uint32(sandbox.UID),
			Gid: uint32(sandbox.GID),
		}
	}
	
	// 关键防护逻辑：将子进程的标准输出/错误输出及额外管道全部 Chown 过户给沙箱 UID，
	// 否则降权后的子进程因为无权写入原本属于 root 的 Pipe/Pty，会导致日志全空没有任何输出。
	chownFile := func(f interface{}) {
		if f == nil {
			return
		}
		if file, ok := f.(*os.File); ok && file != nil {
			_ = file.Chown(int(sandbox.UID), int(sandbox.GID))
		}
	}
	chownFile(cmd.Stdout)
	chownFile(cmd.Stderr)
	chownFile(cmd.Stdin)
	for _, f := range cmd.ExtraFiles {
		if f != nil {
			_ = f.Chown(int(sandbox.UID), int(sandbox.GID))
		}
	}

	// 在 Linux 上，标准库的 syscall 包其实直接支持并且有且仅有这一套标准 Uid/Gid 降权
	// 如果需要额外控制 Rlimit，可以直接在子进程启动前，利用 SysProcAttr 中原生的系统调用完成，
	// 或者通过 sysAttr.Cloneflags 启动独立挂载空间。
	if sandbox.HideSystemEtc {
		// 启用挂载隔离空间 (Mount Namespace)
		sysAttr.Cloneflags = sysAttr.Cloneflags | syscall.CLONE_NEWNS
	}

	cmd.SysProcAttr = sysAttr
}

