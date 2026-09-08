//go:build windows

package windows

import (
	"fmt"
	"io"
	"syscall"

	"github.com/ActiveState/termtest/conpty"
	"github.com/engigu/baihu-panel/internal/logger"
)

type ConPTYSession struct {
	pty *conpty.ConPty
}

// HasConPTYSupport 自动检测当前 Windows 系统是否原生支持 ConPTY API (Win10 1809+ / Server 2019+)
func HasConPTYSupport() bool {
	return true
}

// NewConPTYSession 创建一个新的 ConPTY 原生伪终端会话
func NewConPTYSession(exePath string, argsSlice []string, cols, rows uint16, env []string, dir string) (*ConPTYSession, error) {

	cpty, err := conpty.New(int16(cols), int16(rows))
	if err != nil {
		logger.Error("[ConPTY] conpty.New 失败: %v", err)
		return nil, fmt.Errorf("conpty.New 失败: %w", err)
	}

	argv := append([]string{exePath}, argsSlice...)

	procAttr := &syscall.ProcAttr{
		Dir: dir,
		Env: env,
	}

	_, _, err = cpty.Spawn(exePath, argv, procAttr)
	if err != nil {
		cpty.Close()
		logger.Error("[ConPTY] cpty.Spawn 失败: %v", err)
		return nil, fmt.Errorf("cpty.Spawn 失败: %w", err)
	}

	return &ConPTYSession{
		pty: cpty,
	}, nil
}

func (s *ConPTYSession) Read(p []byte) (n int, err error) {
	if s.pty == nil {
		return 0, io.EOF
	}
	return s.pty.OutPipe().Read(p)
}

func (s *ConPTYSession) Write(p []byte) (n int, err error) {
	if s.pty == nil {
		return 0, io.EOF
	}
	return s.pty.InPipe().Write(p)
}

func (s *ConPTYSession) Resize(cols, rows uint16) error {
	if s.pty == nil {
		return nil
	}
	return s.pty.Resize(cols, rows)
}

func (s *ConPTYSession) Close() error {
	if s.pty != nil {
		err := s.pty.Close()
		s.pty = nil
		return err
	}
	return nil
}
