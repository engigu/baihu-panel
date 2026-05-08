package utils

import (
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	data := map[string]interface{}{
		"task_id":    "T1001",
		"task_name":  "测试任务",
		"status":     "success",
		"duration":   500,
		"start_time": "2024-05-07 20:00:00",
		"output":     "Hello World",
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "基础替换",
			template: "任务 {{task_name}} ({{task_id}})",
			want:     "任务 测试任务 (T1001)",
		},
		{
			name:     "带空格的标签",
			template: "状态: {{ status }}, 耗时: {{   duration   }}ms",
			want:     "状态: success, 耗时: 500ms",
		},
		{
			name:     "多行模板",
			template: "ID: {{task_id}}\nName: {{task_name}}",
			want:     "ID: T1001\nName: 测试任务",
		},
		{
			name:     "变量缺失",
			template: "来自 {{author}} 的消息",
			want:     "来自 <no value> 的消息", // Go text/template 的默认行为
		},
		{
			name:     "空模板",
			template: "",
			want:     "",
		},
		{
			name:     "if/else 分支判断 - 成功",
			template: "任务 {{if eq .status \"success\"}}执行成功 ✅{{else}}执行失败 ❌{{end}}",
			want:     "任务 执行成功 ✅",
		},
		{
			name:     "if/else 分支判断 - 失败",
			template: "任务 {{if eq .status \"failed\"}}执行成功 ✅{{else}}执行失败 ❌{{end}}",
			want:     "任务 执行失败 ❌",
		},
		{
			name:     "contains 关键字包含 - 匹配",
			template: "结果: {{if contains .output \"Hello\"}}发现了问候语{{else}}无匹配{{end}}",
			want:     "结果: 发现了问候语",
		},
		{
			name:     "contains 关键字包含 - 不匹配",
			template: "结果: {{if contains .output \"Error\"}}发现了错误{{else}}无匹配{{end}}",
			want:     "结果: 无匹配",
		},
		{
			name:     "混合模式: 变量+逻辑",
			template: "{{task_name}} 的状态是 {{status}}，{{if eq .status \"success\"}}请放心{{else}}请检查{{end}}",
			want:     "测试任务 的状态是 success，请放心",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderTemplate(tt.template, data); got != tt.want {
				t.Errorf("RenderTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNotifyExtra(t *testing.T) {
	v2Json := `{
		"version": "v2",
		"enable_log": true,
		"log_limit": 500,
		"templates": {
			"task_success": {
				"title": "成功了",
				"content": "内容 {{task_name}}"
			}
		}
	}`

	t.Run("有效v2配置解析", func(t *testing.T) {
		extra, err := ParseNotifyExtra(v2Json)
		if err != nil {
			t.Fatalf("ParseNotifyExtra() error = %v", err)
		}
		if extra.Version != "v2" {
			t.Errorf("Expected version v2, got %s", extra.Version)
		}
		if extra.Templates["task_success"].Title != "成功了" {
			t.Errorf("Title mismatch, got %s", extra.Templates["task_success"].Title)
		}
		if extra.LogLimit != 500 {
			t.Errorf("LogLimit mismatch, got %d", extra.LogLimit)
		}
	})

	t.Run("空字符串处理", func(t *testing.T) {
		extra, err := ParseNotifyExtra("")
		if err != nil {
			t.Errorf("Empty string should not return error, got %v", err)
		}
		if extra != nil {
			t.Errorf("Empty string should return nil extra")
		}
	})

	t.Run("非法JSON处理", func(t *testing.T) {
		_, err := ParseNotifyExtra("{invalid}")
		if err == nil {
			t.Errorf("Expected error for invalid JSON, got nil")
		}
	})
}
