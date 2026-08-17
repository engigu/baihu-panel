package utils

import (
	"runtime"
	"testing"
)

func TestQuotePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "''"},
		{"/path/to/file", "'/path/to/file'"},
		{"/path's/to/file", ""}, // 动态平台相关
	}

	for _, tc := range tests {
		got := QuotePath(tc.input)
		if tc.input == "/path's/to/file" {
			if runtime.GOOS == "windows" {
				if got != "'/path''s/to/file'" {
					t.Errorf("QuotePath(%q) = %q; want '/path''s/to/file'", tc.input, got)
				}
			} else {
				if got != "''/path'\\''s/to/file''" && got != "'/path'\\''s/to/file'" {
					t.Errorf("QuotePath(%q) = %q; want '/path'\\''s/to/file'", tc.input, got)
				}
			}
		} else {
			if got != tc.expected {
				t.Errorf("QuotePath(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		}
	}
}

func TestGetShellCommand(t *testing.T) {
	cmd := "echo 1"
	shell, args := GetShellCommand(cmd)
	if shell == "" {
		t.Error("GetShellCommand returned empty shell")
	}

	if runtime.GOOS == "windows" {
		foundCommand := false
		for i, arg := range args {
			if arg == "-Command" && i+1 < len(args) && args[i+1] == cmd {
				foundCommand = true
				break
			}
		}
		if !foundCommand {
			t.Errorf("GetShellCommand(%q) args %v; did not find command", cmd, args)
		}
	} else {
		if len(args) < 2 || args[len(args)-2] != "-c" || args[len(args)-1] != cmd {
			t.Errorf("GetShellCommand(%q) args %v; expected -c <cmd>", cmd, args)
		}
	}
}
