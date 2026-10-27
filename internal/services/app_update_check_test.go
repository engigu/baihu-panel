package services

import (
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/services/app"
)

func TestFormatAppVersion(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"V2.1.0", "v2.1.0"},
		{"  v0.9.1  ", "v0.9.1"},
		{"", ""},
	}
	for _, c := range cases {
		actual := formatAppVersion(c.input)
		if actual != c.expected {
			t.Errorf("formatAppVersion(%q) = %q, want %q", c.input, actual, c.expected)
		}
	}
}

func TestBuildUpdateNoticeContent(t *testing.T) {
	items := []app.AppUpdateInfo{
		{
			TaskID:           "task-1",
			AppName:          "BiliBiliToolPro",
			UpdateSource:     "marketplace",
			InstalledVersion: "1.0.0",
			LatestVersion:    "v1.1.0",
			Reason:           "version_changed",
			Updatable:        true,
		},
		{
			TaskID:           "task-2",
			AppName:          "CustomApp",
			UpdateSource:     "url",
			InstalledVersion: "v2.0.0",
			LatestVersion:    "2.0.0",
			Reason:           "manifest_changed",
			Updatable:        true,
		},
	}

	content := buildUpdateNoticeContent(items)
	if !strings.Contains(content, "检测到 2 个已装应用可更新") {
		t.Errorf("content does not contain expected count: %s", content)
	}
	if !strings.Contains(content, "BiliBiliToolPro（marketplace）：v1.0.0 → v1.1.0（版本更新）") {
		t.Errorf("content missing task-1 line: %s", content)
	}
	if !strings.Contains(content, "CustomApp（url）：v2.0.0 → v2.0.0（应用定义有更新）") {
		t.Errorf("content missing task-2 line: %s", content)
	}
	if !strings.Contains(content, "调度实体 → 应用行详情与更新") {
		t.Errorf("content missing tail instruction: %s", content)
	}
}

func TestDailyCheckStateNilSafety(t *testing.T) {
	state := loadDailyCheckState(nil)
	if state == nil || len(state) != 0 {
		t.Errorf("expected empty state on nil service, got %v", state)
	}
	// Should not panic
	saveDailyCheckState(nil, map[string]string{"task-1": "1.0.0"})
}
