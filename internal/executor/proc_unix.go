//go:build !windows

package executor

import (
	"os/exec"
	"syscall"
)

func SetProcessGroupAndCancel(cmd *exec.Cmd, usePty bool) {
	if !usePty {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			// Kill the entire process group by sending SIGKILL to negative PID
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
}
