package repo

import (
	"github.com/engigu/baihu-panel/internal/models"
)

// StandardStrategy 实现默认的解析逻辑
type StandardStrategy struct{}

func (s *StandardStrategy) SupportExtension(ext string, exts []string) bool {
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

func (s *StandardStrategy) ShouldProcess(relRepoPath, filename string, cfg models.RepoConfig) bool {
	// 标准策略未来可能具有不同的过滤规则
	// 目前如果提供了白名单/黑名单，则遵循相同的逻辑，但使用更简单的匹配
	return true
}

func (s *StandardStrategy) ExtractMeta(path string, ext string, cfg models.RepoConfig) (taskName string, taskCron string) {
	// 仅在开启兼容 QL 配置时才解析脚本注释
	if cfg.CommentToTask == "true" {
		return ExtractScriptMeta(path, ext)
	}
	return "", ""
}
