package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

// DefaultGitIgnoreRules 默认提供的几个过滤目录与文件规则
var DefaultGitIgnoreRules = []string{
	".git/",
	"node_modules/",
	"__pycache__/",
	".idea/",
	".vscode/",
	"*.log",
	".DS_Store",
}

// GitIgnoreRule 单条解析后的 gitignore 规则
type GitIgnoreRule struct {
	Raw     string
	Negate  bool           // 是否为取反规则 (!)
	OnlyDir bool           // 是否只匹配目录 (以 / 结尾)
	Regex   *regexp.Regexp // 编译后的正则表达式
}

// GitIgnoreMatcher gitignore 规则匹配器
type GitIgnoreMatcher struct {
	rules []GitIgnoreRule
}

// CompileGitIgnore 将规则字符串列表编译为匹配器
func CompileGitIgnore(rules []string) *GitIgnoreMatcher {
	matcher := &GitIgnoreMatcher{}
	for _, raw := range rules {
		rule, ok := parseGitIgnoreRule(raw)
		if ok {
			matcher.rules = append(matcher.rules, rule)
		}
	}
	return matcher
}

func parseGitIgnoreRule(raw string) (GitIgnoreRule, bool) {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return GitIgnoreRule{}, false
	}

	negate := false
	if strings.HasPrefix(line, "!") {
		negate = true
		line = strings.TrimPrefix(line, "!")
		line = strings.TrimSpace(line)
		if line == "" {
			return GitIgnoreRule{}, false
		}
	}

	onlyDir := false
	if strings.HasSuffix(line, "/") {
		onlyDir = true
		line = strings.TrimSuffix(line, "/")
	}

	isRooted := strings.HasPrefix(line, "/")
	if isRooted {
		line = strings.TrimPrefix(line, "/")
	}

	hasSlash := strings.Contains(line, "/")

	re := patternToRegex(line, hasSlash || isRooted)
	if re == nil {
		return GitIgnoreRule{}, false
	}

	return GitIgnoreRule{
		Raw:     raw,
		Negate:  negate,
		OnlyDir: onlyDir,
		Regex:   re,
	}, true
}

func patternToRegex(pattern string, rooted bool) *regexp.Regexp {
	var sb strings.Builder
	if rooted {
		sb.WriteString(`^`)
	} else {
		sb.WriteString(`(?:^|/)`)
	}

	chars := []rune(pattern)
	n := len(chars)
	for i := 0; i < n; i++ {
		c := chars[i]
		if c == '*' {
			if i+1 < n && chars[i+1] == '*' {
				// **
				i++
				if i+1 < n && chars[i+1] == '/' {
					// **/
					i++
					sb.WriteString(`(?:^|.+/)`)
				} else {
					sb.WriteString(`.*`)
				}
			} else {
				// 单个 *
				sb.WriteString(`[^/]*`)
			}
		} else if c == '?' {
			sb.WriteString(`[^/]`)
		} else if c == '.' || c == '+' || c == '(' || c == ')' || c == '{' || c == '}' || c == '^' || c == '$' || c == '|' || c == '\\' {
			sb.WriteRune('\\')
			sb.WriteRune(c)
		} else {
			sb.WriteRune(c)
		}
	}
	sb.WriteString(`$`)

	re, err := regexp.Compile(sb.String())
	if err != nil {
		return nil
	}
	return re
}

// Match 检查给定的相对路径是否应该被忽略
func (m *GitIgnoreMatcher) Match(relPath string, isDir bool) bool {
	if m == nil || len(m.rules) == 0 {
		return false
	}

	cleanPath := filepath.ToSlash(filepath.Clean(relPath))
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	cleanPath = strings.TrimSuffix(cleanPath, "/")

	if cleanPath == "" || cleanPath == "." {
		return false
	}

	matched := false
	for _, rule := range m.rules {
		if rule.OnlyDir && !isDir {
			continue
		}
		if rule.Regex.MatchString(cleanPath) {
			matched = !rule.Negate
		}
	}
	return matched
}
