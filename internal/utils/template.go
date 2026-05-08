package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// NotifyExtraV2 v2 版本的通知额外配置
type NotifyExtraV2 struct {
	Version    string                                `json:"version"`
	EnableLog  bool                                  `json:"enable_log"`
	LogLimit   int                                   `json:"log_limit"`
	Templates  map[string]NotifyTemplate             `json:"templates"`
}

type NotifyTemplate struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

var (
	tagRegex     = regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)\s*}}`)
	expressionFn = map[string]interface{}{
		"contains": strings.Contains,
		"eq":       func(a, b interface{}) bool { return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) },
		"ne":       func(a, b interface{}) bool { return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b) },
	}
)

/**
 * 渲染模板变量 (v2) - 支持表达式逻辑
 */
func RenderTemplate(templateStr string, data map[string]interface{}) string {
	if templateStr == "" {
		return ""
	}

	// 预处理：将简单变量 {{var}} 转换为 {{.var}} 以匹配 Go 模板语法
	// 排除掉逻辑关键字，如 if, else, end, range
	processedTmpl := tagRegex.ReplaceAllStringFunc(templateStr, func(match string) string {
		key := strings.TrimSpace(match[2 : len(match)-2])
		// 如果是逻辑关键字或已经带点了，不处理
		keywords := map[string]bool{"if": true, "else": true, "end": true, "range": true, "with": true}
		if keywords[key] || strings.HasPrefix(key, ".") {
			return match
		}
		return "{{." + key + "}}"
	})

	tmpl, err := template.New("notify").Funcs(expressionFn).Parse(processedTmpl)
	if err != nil {
		return "模板语法错误: " + err.Error()
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "执行错误: " + err.Error()
	}

	return buf.String()
}

/**
 * 从 Extra 字符串中解析 v2 配置
 */
func ParseNotifyExtra(extraStr string) (*NotifyExtraV2, error) {
	if extraStr == "" {
		return nil, nil
	}
	var extra NotifyExtraV2
	err := json.Unmarshal([]byte(extraStr), &extra)
	if err != nil {
		return nil, err
	}
	return &extra, nil
}
