package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// 官方应用市场仓库常量与请求头定义
const (
	DefaultAppStoreOwner = "engigu"
	DefaultAppStoreRepo  = "baihu-appstore"
	DefaultAppStoreBranch = "main"

	// DefaultBrowserUA 标准通用 Chrome 浏览器 User-Agent，防止被 CDN / WAF / 代理节点以爬虫策略拦截
	DefaultBrowserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
)

// 统一代理与镜像节点前缀
var (
	// DefaultGHProxyPrefix 默认推荐的高可用 ghproxy 代理前缀
	DefaultGHProxyPrefix = "https://gh-proxy.com/"
	// DefaultMirrorPrefix 默认镜像代理前缀
	DefaultMirrorPrefix = "https://mirror.ghproxy.com/"

	// GHProxyEndpoints 全局代理候选前缀列表（按可用性与速度排序，gh-proxy.com 默认首选）
	GHProxyEndpoints = []string{
		"https://gh-proxy.com/",
		"https://ghfast.top/",
		"https://ghproxy.net/",
		"https://mirror.ghproxy.com/",
		"https://ghp.ci/",
	}

	// JsDelivrEndpoints 全局 jsDelivr CDN 候选列表
	JsDelivrEndpoints = []string{
		"https://fastly.jsdelivr.net/gh/",
		"https://cdn.jsdelivr.net/gh/",
		"https://gcore.jsdelivr.net/gh/",
	}
)

// GetAppStoreBranch 获取当前配置的应用市场分支，优先读取环境变量 BH_APPSTORE_BRANCH
func GetAppStoreBranch() string {
	branch := strings.TrimSpace(os.Getenv("BH_APPSTORE_BRANCH"))
	if branch == "" {
		return DefaultAppStoreBranch
	}
	return branch
}

// BuildProxyURL 根据代理类型为指定 URL 注入代理前缀，避免硬编码零散定义
func BuildProxyURL(rawURL string, proxyType string, customProxy string) string {
	cleanURL := strings.TrimSpace(rawURL)
	if cleanURL == "" || proxyType == "" || proxyType == "none" {
		return cleanURL
	}

	// 容错：如果 URL 已经包含明显的代理前缀，则跳过注入
	if strings.Contains(cleanURL, "ghproxy") ||
		strings.Contains(cleanURL, "ghfast.top") ||
		strings.Contains(cleanURL, "ghp.ci") ||
		strings.Contains(cleanURL, "googo.win") ||
		(proxyType == "custom" && customProxy != "" && strings.HasPrefix(cleanURL, customProxy)) {
		return cleanURL
	}

	base := ""
	switch proxyType {
	case "ghproxy":
		base = DefaultGHProxyPrefix
	case "mirror":
		base = DefaultMirrorPrefix
	case "custom":
		if customProxy != "" {
			base = strings.TrimRight(customProxy, "/") + "/"
		}
	default:
		if strings.HasPrefix(proxyType, "http://") || strings.HasPrefix(proxyType, "https://") {
			base = strings.TrimRight(proxyType, "/") + "/"
		}
	}

	if base != "" && strings.HasPrefix(cleanURL, "http") && !strings.HasPrefix(cleanURL, base) {
		return base + cleanURL
	}
	return cleanURL
}

// GetGitHubRawCandidates 生成指定 GitHub 仓库文件的多源容灾候选列表
func GetGitHubRawCandidates(owner, repo, branch, filePath string) []string {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "main"
	}
	filePath = strings.TrimLeft(filePath, "/")

	candidates := make([]string, 0, 8)

	// 1. jsDelivr CDN 矩阵 (静态文件速度极快)
	for _, cdn := range JsDelivrEndpoints {
		candidates = append(candidates, fmt.Sprintf("%s%s/%s@%s/%s", cdn, owner, repo, branch, filePath))
	}

	// 2. 原生 Raw 直连
	rawDirect := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, filePath)
	candidates = append(candidates, rawDirect)

	// 3. ghproxy 加速代理矩阵
	for _, proxy := range GHProxyEndpoints {
		candidates = append(candidates, fmt.Sprintf("%s%s", proxy, rawDirect))
	}

	return candidates
}

// GetAppStoreAppsJSONCandidates 获取白虎应用市场 apps.json 的全部候选源
func GetAppStoreAppsJSONCandidates() []string {
	branch := GetAppStoreBranch()
	candidates := make([]string, 0, 10)

	// 1. GitHub Pages (如开启，通常速度快且支持跨域)
	candidates = append(candidates, fmt.Sprintf("https://%s.github.io/%s/apps.json", DefaultAppStoreOwner, DefaultAppStoreRepo))

	// 2. 注入全局 GitHub Raw 候选源
	rawCandidates := GetGitHubRawCandidates(DefaultAppStoreOwner, DefaultAppStoreRepo, branch, "apps.json")
	candidates = append(candidates, rawCandidates...)

	return candidates
}

// GetAppStoreAppYAMLCandidates 获取白虎应用市场指定应用 app.yaml 的候选源
func GetAppStoreAppYAMLCandidates(appID string) []string {
	branch := GetAppStoreBranch()
	relPath := fmt.Sprintf("apps/%s/app.yaml", strings.TrimSpace(appID))
	return GetGitHubRawCandidates(DefaultAppStoreOwner, DefaultAppStoreRepo, branch, relPath)
}

// FetchWithFallback 顺序请求候选 URL，返回首个成功的结果数据及所使用的 URL
func FetchWithFallback(candidates []string, timeout time.Duration) ([]byte, string, error) {
	if len(candidates) == 0 {
		return nil, "", fmt.Errorf("候选源列表为空")
	}

	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := &http.Client{Timeout: timeout}
	var lastErr error
	nowTs := time.Now().Unix()

	for _, targetURL := range candidates {
		fetchURL := targetURL
		if !strings.Contains(fetchURL, "?") {
			fetchURL = fmt.Sprintf("%s?t=%d", targetURL, nowTs)
		}

		req, err := http.NewRequest("GET", fetchURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", DefaultBrowserUA)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			body, rErr := io.ReadAll(resp.Body)
			if rErr == nil && len(body) > 0 {
				return body, targetURL, nil
			}
			if rErr != nil {
				lastErr = rErr
			}
		} else {
			if err != nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("HTTP 状态码: %d (%s)", resp.StatusCode, targetURL)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}

	return nil, "", fmt.Errorf("所有候选镜像源均请求失败: %w", lastErr)
}
