package services

import (
	"strings"
	"testing"
)

func TestGetAgentBinary_PathTraversal(t *testing.T) {
	svc := NewAgentService()

	invalidCases := []struct {
		osType string
		arch   string
	}{
		{"../etc", "passwd"},
		{"linux", "../../shadow"},
		{"win/../../", "amd64"},
		{"..", "amd64"},
		{"linux", ".."},
		{"", "amd64"},
		{"linux", ""},
		{"linux\\..", "amd64"},
	}

	for _, tc := range invalidCases {
		_, _, err := svc.GetAgentBinary(tc.osType, tc.arch)
		if err == nil {
			t.Errorf("GetAgentBinary(%q, %q) 期望返回错误，但实际返回 nil", tc.osType, tc.arch)
			continue
		}
		if !strings.Contains(err.Error(), "非法") && !strings.Contains(err.Error(), "不能为空") {
			t.Errorf("GetAgentBinary(%q, %q) 错误信息不符合预期: %v", tc.osType, tc.arch, err)
		}
	}
}

