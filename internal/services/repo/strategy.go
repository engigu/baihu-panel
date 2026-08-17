package repo

import (
	"github.com/engigu/baihu-panel/internal/models"
)

// RepoParserStrategy 定义不同仓库解析策略的接口
type RepoParserStrategy interface {
	// SupportExtension 判断给定后缀的文件是否应该被处理
	SupportExtension(ext string, exts []string) bool

	// ShouldProcess 应用白名单/黑名单过滤，决定是否处理该文件
	ShouldProcess(relRepoPath, filename string, cfg models.RepoConfig) bool

	// ExtractMeta 从脚本文件中提取任务元数据（名称和 cron 表达式）
	ExtractMeta(path string, ext string, cfg models.RepoConfig) (taskName string, taskCron string)
}

// GetParserStrategy 根据来源类型返回相应的策略实现
func GetParserStrategy(sourceType string) RepoParserStrategy {
	switch sourceType {
	case "ql":
		return &QinglongStrategy{}
	default:
		return &StandardStrategy{}
	}
}
