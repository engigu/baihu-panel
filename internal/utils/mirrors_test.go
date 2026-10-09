package utils

import (
	"strings"
	"testing"
)

func TestMirrorsConfig(t *testing.T) {
	// 1. 验证首选 ghproxy 必须是 gh-proxy.com
	if DefaultGHProxyPrefix != "https://gh-proxy.com/" {
		t.Fatalf("DefaultGHProxyPrefix 期望为 https://gh-proxy.com/，实际为: %s", DefaultGHProxyPrefix)
	}

	if len(GHProxyEndpoints) == 0 || GHProxyEndpoints[0] != "https://gh-proxy.com/" {
		t.Fatalf("GHProxyEndpoints[0] 期望为 https://gh-proxy.com/，实际为: %v", GHProxyEndpoints)
	}

	// 2. 验证 User-Agent 是标准浏览器 UA
	if !strings.Contains(DefaultBrowserUA, "Mozilla/5.0") || !strings.Contains(DefaultBrowserUA, "Chrome/") {
		t.Fatalf("DefaultBrowserUA 必须是标准的现代浏览器 UA: %s", DefaultBrowserUA)
	}

	// 3. 验证代理注入工具
	rawGit := "https://github.com/engigu/baihu-panel.git"
	proxied := BuildProxyURL(rawGit, "ghproxy", "")
	expected := "https://gh-proxy.com/https://github.com/engigu/baihu-panel.git"
	if proxied != expected {
		t.Fatalf("BuildProxyURL 期望: %s，实际: %s", expected, proxied)
	}

	// 重复注入防重检查
	reProxied := BuildProxyURL(proxied, "ghproxy", "")
	if reProxied != expected {
		t.Fatalf("防重注入失败，期望: %s，实际: %s", expected, reProxied)
	}

	// 4. 验证白虎应用市场候选源生成
	yamlCandidates := GetAppStoreAppYAMLCandidates("bilibili-tool-pro")
	if len(yamlCandidates) == 0 {
		t.Fatalf("GetAppStoreAppYAMLCandidates 结果不能为空")
	}

	// 验证包含首选 ghproxy 以及 jsDelivr
	hasGhProxy := false
	hasJsDelivr := false
	for _, c := range yamlCandidates {
		if strings.Contains(c, "https://gh-proxy.com/") {
			hasGhProxy = true
		}
		if strings.Contains(c, "fastly.jsdelivr.net") || strings.Contains(c, "cdn.jsdelivr.net") {
			hasJsDelivr = true
		}
	}

	if !hasGhProxy {
		t.Errorf("yamlCandidates 缺少首选 gh-proxy.com 源")
	}
	if !hasJsDelivr {
		t.Errorf("yamlCandidates 缺少 jsDelivr 源")
	}
}
