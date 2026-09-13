package clibase

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestVisualFormatAndFormatCell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		align    Alignment
		expected string
		exactW   int
	}{
		{
			name:     "ASCII left align padding",
			input:    "hello",
			width:    10,
			align:    AlignLeft,
			expected: "hello     ",
			exactW:   10,
		},
		{
			name:     "ASCII right align padding",
			input:    "123",
			width:    6,
			align:    AlignRight,
			expected: "   123",
			exactW:   6,
		},
		{
			name:     "ASCII center align padding",
			input:    "test",
			width:    8,
			align:    AlignCenter,
			expected: "  test  ",
			exactW:   8,
		},
		{
			name:     "Chinese characters exact width",
			input:    "任务管理",
			width:    12,
			align:    AlignLeft,
			expected: "任务管理    ",
			exactW:   12,
		},
		{
			name:     "Chinese characters truncation with ..",
			input:    "超长任务名称测试超出设定的列宽限制测试",
			width:    16,
			align:    AlignLeft,
			expected: "超长任务名称测..", // 7个汉字(14列宽) + ".." (2列宽) = 16列宽
			exactW:   16,
		},
		{
			name:     "Emoji characters width",
			input:    "🚀启动",
			width:    10,
			align:    AlignLeft,
			expected: "🚀启动    ",
			exactW:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := FormatCell(tt.input, tt.width, tt.align)
			actualW := runewidth.StringWidth(res)
			if actualW != tt.exactW {
				t.Errorf("FormatCell(%q, %d, %v) visual width = %d, expected %d, got %q",
					tt.input, tt.width, tt.align, actualW, tt.exactW, res)
			}
			if res != tt.expected {
				t.Errorf("FormatCell(%q, %d, %v) = %q, expected %q",
					tt.input, tt.width, tt.align, res, tt.expected)
			}

			if tt.align == AlignLeft {
				vfRes := VisualFormat(tt.input, tt.width)
				if vfRes != res {
					t.Errorf("VisualFormat(%q, %d) = %q, expected identical to FormatCell %q",
						tt.input, tt.width, vfRes, res)
				}
			}
		})
	}
}

func TestTableRenderWithColumns(t *testing.T) {
	table := NewTableWithColumns(
		Column{Title: "ID", Width: 6},
		Column{Title: "名称", Width: 10},
		Column{Title: "状态", Width: 6, Align: AlignCenter},
		Column{Title: "耗时", Width: 8, Align: AlignRight},
	)

	table.AddRow("1", "每日签到", "成功", "120 ms")
	table.AddRow("2", "超长任务名称测试超出列宽", "运行中", "15 ms")

	output := table.String()
	lines := strings.Split(output, "\n")

	if len(lines) != 6 {
		t.Fatalf("Expected 6 lines in rendered table, got %d:\n%s", len(lines), output)
	}

	// 检查每行视觉宽度是否严丝合缝一致
	expectedWidth := runewidth.StringWidth(lines[0])
	for i, line := range lines {
		w := runewidth.StringWidth(line)
		if w != expectedWidth {
			t.Errorf("Line %d visual width = %d, expected %d. Line: %q", i, w, expectedWidth, line)
		}
	}

	// 顶部分隔线应全由 '=' 组成
	if strings.Trim(lines[0], "=") != "" {
		t.Errorf("Line 0 should be '=', got %q", lines[0])
	}
	// 表头与数据行分隔线应全由 '-' 组成
	if strings.Trim(lines[2], "-") != "" {
		t.Errorf("Line 2 should be '-', got %q", lines[2])
	}
	// 底部分隔线应全由 '=' 组成
	if strings.Trim(lines[5], "=") != "" {
		t.Errorf("Line 5 should be '=', got %q", lines[5])
	}
}

func TestTableAutoWidthAndAnyTypes(t *testing.T) {
	// 直接传表头字符串，全自适应列宽，且支持链式配置
	table := NewTable("任务ID", "类型", "退出码", "状态").
		SetAlign(2, AlignRight).
		SetAlign(3, AlignCenter)

	// AddRow 支持字符串和非字符串类型 (如数字 0, 127 等)
	table.AddRow("task-001", "Cron", 0, "成功")
	table.AddRow("task-002", "Manual-Trigger", 127, "失败")
	table.AddRow("task-003", "Repo-Sync", 0, "成功")

	if table.RowCount() != 3 {
		t.Errorf("Expected RowCount = 3, got %d", table.RowCount())
	}

	output := table.String()
	lines := strings.Split(output, "\n")

	// 顶部分隔线 + 表头 + 分隔线 + 3行数据 + 底部分隔线 = 7行
	if len(lines) != 7 {
		t.Fatalf("Expected 7 lines in rendered table, got %d:\n%s", len(lines), output)
	}

	// 验证所有行的视觉宽度绝对一致
	expectedWidth := runewidth.StringWidth(lines[0])
	for i, line := range lines {
		w := runewidth.StringWidth(line)
		if w != expectedWidth {
			t.Errorf("Auto width table line %d visual width = %d, expected %d. Line: %q", i, w, expectedWidth, line)
		}
	}
}

func TestTableMaxWidthConstraint(t *testing.T) {
	table := NewTable("ID", "长描述").
		SetMaxWidth(1, 10) // 限制第2列最大视觉宽度为 10

	table.AddRow("1", "这是一个非常非常长的文本描述")

	output := table.String()
	lines := strings.Split(output, "\n")

	expectedWidth := runewidth.StringWidth(lines[0])
	for i, line := range lines {
		w := runewidth.StringWidth(line)
		if w != expectedWidth {
			t.Errorf("MaxWidth table line %d visual width = %d, expected %d", i, w, expectedWidth)
		}
	}

	// 确认第2列内容被截断且包含 ".."
	if !strings.Contains(lines[3], "..") {
		t.Errorf("Expected cell to be truncated with '..', got line: %q", lines[3])
	}
}
