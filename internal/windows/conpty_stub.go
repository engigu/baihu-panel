//go:build !windows

package windows

import (
	"errors"
	"io"
)

type ConPTYSession struct{}

// HasConPTYSupport 非 Windows 平台固定返回 false
func HasConPTYSupport() bool {
	return false
}

// NewConPTYSession 非 Windows 平台存根
func NewConPTYSession(cmdStr string, cols, rows uint16, env []string, dir string) (*ConPTYSession, error) {
	return nil, errors.New("ConPTY 仅在 Windows 平台上支持")
}

func (s *ConPTYSession) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (s *ConPTYSession) Write(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (s *ConPTYSession) Resize(cols, rows uint16) error {
	return nil
}

func (s *ConPTYSession) Close() error {
	return nil
}
