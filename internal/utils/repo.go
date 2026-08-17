package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// GetRepoIdentifier 返回根据仓库URL和分支生成的作者_仓库名标识符
func GetRepoIdentifier(url string, branch string) string {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	repoName := url[strings.LastIndex(url, "/")+1:]

	author := ""
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash != -1 {
		prefix := url[:lastSlash]
		if strings.Contains(prefix, ":") {
			parts := strings.Split(prefix, ":")
			prefix = parts[len(parts)-1]
		}
		lastSlashPrefix := strings.LastIndex(prefix, "/")
		if lastSlashPrefix != -1 {
			author = prefix[lastSlashPrefix+1:]
		} else {
			author = prefix
		}
	}

	if dotIdx := strings.LastIndex(author, "."); dotIdx != -1 {
		author = author[dotIdx+1:]
	}

	identifier := ""
	if author != "" {
		identifier = author + "_" + repoName
	} else {
		identifier = repoName
	}

	if branch != "" && branch != "master" && branch != "main" {
		identifier = identifier + "_" + branch
	}

	// Replace any invalid characters for tags or paths
	identifier = strings.ReplaceAll(identifier, "/", "_")
	identifier = strings.ReplaceAll(identifier, ".", "_")
	return identifier
}

// GetActualRepoDir 返回仓库真实的物理目录
func GetActualRepoDir(targetPath, sourceURL, branch, sourceType string) string {
	repoDir := targetPath
	if sourceType == "git" && sourceURL != "" {
		repoName := GetRepoIdentifier(sourceURL, branch)
		// 检查 targetPath 是否已存在且是 Git 仓库
		gitDir := filepath.Join(repoDir, ".git")
		if info, err := os.Stat(repoDir); err == nil && info.IsDir() {
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				// 只有当目标目录存在但不是 Git 仓库时，才追加仓库名
				repoDir = filepath.Join(repoDir, repoName)
			}
		}
	}
	return repoDir
}
