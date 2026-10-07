package utils

import (
	"testing"
)

func TestGitIgnoreMatcher(t *testing.T) {
	rules := []string{
		".git/",
		"node_modules/",
		"__pycache__/",
		"*.log",
		"!important.log",
		"build/output",
	}

	m := CompileGitIgnore(rules)

	tests := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{".git", true, true},
		{".git", false, false}, // .git/ only matches dir
		{"node_modules", true, true},
		{"src/node_modules", true, true},
		{"test.log", false, true},
		{"sub/dir/app.log", false, true},
		{"important.log", false, false}, // negated by !important.log
		{"sub/important.log", false, false},
		{"build/output", false, true},
		{"src/main.go", false, false},
		{"dist/bundle.js", false, false},
	}

	for _, tt := range tests {
		got := m.Match(tt.path, tt.isDir)
		if got != tt.expected {
			t.Errorf("Match(%q, isDir=%v) = %v; want %v", tt.path, tt.isDir, got, tt.expected)
		}
	}
}
