package repo

import (
	"github.com/engigu/baihu-panel/internal/models"
	"regexp"
	"strings"
)

// QinglongStrategy 实现与青龙兼容的解析逻辑
type QinglongStrategy struct{}

func (s *QinglongStrategy) SupportExtension(ext string, exts []string) bool {
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

func (s *QinglongStrategy) ShouldProcess(relRepoPath, filename string, cfg models.RepoConfig) bool {
	// 只有在显式设置了白名单时才进行白名单校验 (青龙行为)
	if cfg.WhitelistPaths != "" {
		if !matchesQLPattern(relRepoPath, filename, cfg.WhitelistPaths) {
			return false
		}
	}

	// 校验黑名单
	if cfg.Blacklist != "" {
		if matchesQLPattern(relRepoPath, filename, cfg.Blacklist) {
			return false
		}
	}
	return true
}

func (s *QinglongStrategy) ExtractMeta(path string, ext string, cfg models.RepoConfig) (taskName string, taskCron string) {
	return ExtractScriptMeta(path, ext)
}

// matchesQLPattern 应用关键字过滤逻辑（正则或包含匹配）
func matchesQLPattern(rel, filename string, keywordsStr string) bool {
	if keywordsStr == "" {
		return false
	}

	keywords := splitKeywords(keywordsStr)
	for _, k := range keywords {
		// 1. 尝试作为正则整体进行匹配，默认不区分大小写 (?i)
		pattern := k
		if !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}

		reg, err := regexp.Compile(pattern)
		if err == nil {
			// 优先匹配文件名（解决 ^jd[^_] 这种锚点在相对路径下失效的问题）
			if reg.MatchString(filename) || reg.MatchString(rel) {
				return true
			}
		} else {
			// 回退逻辑：全小写包含判断
			kLower := strings.ToLower(k)
			if strings.Contains(strings.ToLower(rel), kLower) || strings.Contains(strings.ToLower(filename), kLower) {
				return true
			}
		}
	}
	return false
}
