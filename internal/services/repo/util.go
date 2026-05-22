package repo

import (
	"bufio"
	"fmt"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/robfig/cron/v3"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// envRegex 匹配脚本中的环境名称设置，如 Env("名称")
	envRegex = regexp.MustCompile(`(?i)(?:new[ \t]+)?Env\(['"]?([^'"]+)['"]?\)`)
	// cronRegex 匹配脚本中的 cron 表达式设置
	cronRegex = regexp.MustCompile(`(?i)(?:cron[ \t]*[:=][ \t]*['"]?([^'"\r\n]+))|(?:(?:^|[ \t\*\/])(([0-9\*\/\-,L?#]+[ \t]+){4,5}[0-9\*\/\-,L?#]+))`)
	// cronFormatRegex 用于校验提取出的字符串是否符合 Cron 表达式格式 (5位或6位)
	cronFormatRegex = regexp.MustCompile(`^(([0-9\*\/\-,L?#]+)[ \t]+){4,5}([0-9\*\/\-,L?#]+)$`)
)

// ExtractScriptMeta 读取文件以提取任务名称和 cron 表达式
func ExtractScriptMeta(path string, ext string) (taskName string, taskCron string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var firstCommentLine string
	inBlockComment := false

	// 特殊处理：针对当前文件名的 Cron 关联正则表达式 (对标青龙 perl 逻辑)
	// 寻找类似 "// 0 0 * * * jd_task.js" 的行
	fileNameEscaped := regexp.QuoteMeta(filepath.Base(path))
	associatedCronRegex := regexp.MustCompile(fmt.Sprintf(`(?i)(?:^|[ \t\*\//])(([0-9\*\/\-,L?#]+[ \t]+){4,5}[0-9\*\/\-,L?#]+)[ \t,"]+.*%s`, fileNameEscaped))

	for i := 0; i < 15 && scanner.Scan(); i++ { // 限制扫描范围，避免误匹配代码深处的属性
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 处理块注释开始/结束
		if strings.HasPrefix(line, "/*") {
			inBlockComment = true
			line = strings.TrimPrefix(line, "/*")
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)
		}
		if strings.HasSuffix(line, "*/") {
			inBlockComment = false
			line = strings.TrimSuffix(line, "*/")
			line = strings.TrimSpace(line)
		}

		// 1. 尝试提取任务名称 (优先使用 Env)
		if taskName == "" {
			if envMatch := envRegex.FindStringSubmatch(line); len(envMatch) > 1 {
				taskName = strings.TrimSpace(envMatch[1])
			} else if strings.Contains(line, "name:") {
				// 兼容 name: "xxx" 格式
				nameRegex := regexp.MustCompile(`(?i)name:[ \t]*['"]([^'"]+)['"]`)
				if nameMatch := nameRegex.FindStringSubmatch(line); len(nameMatch) > 1 {
					taskName = strings.TrimSpace(nameMatch[1])
				}
			}
		}

		// 如果还没找到名称，且在注释中，记录第一行非空注释作为备选名称
		if taskName == "" && (inBlockComment || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "#")) {
			cleanLine := line
			if strings.HasPrefix(line, "//") {
				cleanLine = strings.TrimPrefix(line, "//")
			} else if strings.HasPrefix(line, "#") {
				cleanLine = strings.TrimPrefix(line, "#")
			} else if strings.HasPrefix(line, "*") {
				cleanLine = strings.TrimPrefix(line, "*")
			}
			cleanLine = strings.TrimSpace(cleanLine)

			// 排除掉包含 "Env" 或 "cron" 的行, 且排除掉可能是路径或URL的行
			if cleanLine != "" && !strings.Contains(strings.ToLower(cleanLine), "env") &&
				!strings.Contains(strings.ToLower(cleanLine), "cron") &&
				!strings.Contains(cleanLine, "http") &&
				!strings.Contains(cleanLine, "/") &&
				firstCommentLine == "" {
				// 且排除掉纯 cron 表达式
				if !cronRegex.MatchString(cleanLine) {
					firstCommentLine = cleanLine
				}
			}
		}

		// 2. 提取 Cron
		if taskCron == "" {
			// A. 优先查找关联了当前文件名的 Cron (对标 QL)
			if assocMatch := associatedCronRegex.FindStringSubmatch(line); len(assocMatch) > 1 {
				tempCron := strings.Trim(strings.TrimSpace(assocMatch[1]), "\"' \t")
				if isLikelyCron(tempCron) {
					taskCron = tempCron
				}
			}

			// B. 如果没找到，尝试普通的 cron: "..." 或 cron 表达式
			if taskCron == "" {
				if cronMatch := cronRegex.FindStringSubmatch(line); len(cronMatch) > 0 {
					for _, m := range cronMatch[1:] {
						if m != "" {
							tempCron := strings.Trim(strings.TrimSpace(m), "\"' \t")
							if isLikelyCron(tempCron) {
								taskCron = tempCron
								break
							}
						}
					}
				}
			}
		}

		if taskName != "" && taskCron != "" {
			break
		}
	}

	// 如果最后还是没找到 taskName，使用备选名称或文件名
	if taskName == "" {
		if firstCommentLine != "" {
			taskName = firstCommentLine
		} else {
			taskName = strings.TrimSuffix(filepath.Base(path), ext)
		}
	}

	return taskName, taskCron
}

// isLikelyCron 校验字符串是否符合 Cron 表达式的格式特征且数值合法
func isLikelyCron(s string) bool {
	if !cronFormatRegex.MatchString(s) {
		return false
	}

	// 语义校验：确保数值在合法范围内（解决如 JS 数组数字超出 Cron 范围的问题）
	fields := strings.Fields(s)
	testCron := s
	if len(fields) == 5 {
		testCron = "0 " + s
	}

	// 使用与面板执行器一致的 Parser 进行校验
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(testCron)
	return err == nil
}

// splitKeywords 按竖线或逗号分割字符串
func splitKeywords(s string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	if strings.Contains(s, "|") {
		parts = strings.Split(s, "|")
	} else if strings.Contains(s, ",") {
		parts = strings.Split(s, ",")
	} else {
		parts = []string{s}
	}

	var res []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

// getValidExtensions 返回支持的文件扩展名列表，优先考虑自定义扩展名
func getValidExtensions(customExtensions string) []string {
	exts := []string{".js", ".py", ".ts", ".sh"}
	if customExtensions != "" {
		customExts := splitKeywords(customExtensions)
		if len(customExts) > 0 {
			exts = nil
			for _, e := range customExts {
				e = strings.TrimSpace(e)
				if e != "" {
					if !strings.HasPrefix(e, ".") {
						e = "." + e
					}
					exts = append(exts, e)
				}
			}
		}
	}
	return exts
}

// sanitizeIdentifier 将非字母数字字符替换为下划线
func sanitizeIdentifier(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	res := reg.ReplaceAllString(s, "_")
	return strings.ToLower(strings.Trim(res, "_"))
}

// normalizeCron 确保 cron 表达式具有 6 个字段
func normalizeCron(cron string) string {
	fields := strings.Fields(cron)
	if len(fields) == 5 {
		return "0 " + cron
	}
	return cron
}

// pathExists 检查路径是否存在
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isDir 检查路径是否为目录
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// getCommandByExt 根据文件扩展名返回默认执行命令
func getCommandByExt(ext, path string) string {
	quotedPath := utils.QuotePath(path)
	switch ext {
	case ".js", ".ts":
		return fmt.Sprintf("node %s", quotedPath)
	case ".py":
		return fmt.Sprintf("python %s", quotedPath)
	case ".sh":
		return fmt.Sprintf("bash %s", quotedPath)
	case ".php":
		return fmt.Sprintf("php %s", quotedPath)
	case ".cs":
		return fmt.Sprintf("dotnet run %s", quotedPath)
	}
	return quotedPath
}
