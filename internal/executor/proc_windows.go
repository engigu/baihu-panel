//go:build windows

package executor

import (
	"fmt"
	"os/exec"
)

func SetProcessGroupAndCancel(cmd *exec.Cmd, usePty bool) {
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
			return killCmd.Run()
		}
		return nil
	}
}
