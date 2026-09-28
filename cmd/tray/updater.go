package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/engigu/baihu-panel/internal/constant"
)

const (
	githubApiLatestUrl = "https://api.github.com/repos/engigu/baihu-panel/releases/latest"

	mbOK              = 0x00000000
	mbOKCancel        = 0x00000001
	mbYesNo           = 0x00000004
	mbIconInformation = 0x00000040
	mbIconQuestion    = 0x00000020
	mbIconWarning     = 0x00000030
	mbIconError       = 0x00000010
	idYes             = 6
	idOK              = 1

	niifInfo = 0x00000001
	nifInfo  = 0x00000010
)

var (
	messageBox = user32.NewProc("MessageBoxW")

	isDownloadingUpdate = false
	hasNewVersion       = false
	latestReleaseInfo   *githubRelease
	latestAssetInfo     *githubAsset
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Body    string        `json:"body"`
	Assets  []githubAsset `json:"assets"`
}

// showMessage 弹出 Windows 原生消息弹窗
func showMessage(title, text string, style uint32) uint32 {
	tPtr, _ := syscall.UTF16PtrFromString(title)
	mPtr, _ := syscall.UTF16PtrFromString(text)
	ret, _, _ := messageBox.Call(uintptr(hwnd), uintptr(unsafe.Pointer(mPtr)), uintptr(unsafe.Pointer(tPtr)), uintptr(style))
	return uint32(ret)
}

// showBalloonNotification 弹出 Windows 任务栏原生托盘气球通知
func showBalloonNotification(title, msg string) {
	if hwnd == 0 {
		return
	}
	nidCopy := nid
	nidCopy.Flags |= nifInfo
	copy(nidCopy.InfoTitle[:], syscall.StringToUTF16(title))
	copy(nidCopy.Info[:], syscall.StringToUTF16(msg))
	nidCopy.InfoFlags = niifInfo
	shellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nidCopy)))
}

// getCurrentVersion 获取当前本地程序版本
func getCurrentVersion() string {
	if constant.Version != "" && constant.Version != "dev" && constant.Version != "unknown" {
		return constant.Version
	}

	// 若未注入版本号，尝试从同目录 baihu.exe version 获取
	exePath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exePath)
		baihuExe := filepath.Join(dir, "baihu.exe")
		if out, err := exec.Command(baihuExe, "version").Output(); err == nil {
			fields := strings.Fields(string(out))
			if len(fields) >= 2 {
				return fields[1]
			}
		}
	}
	return "v1.0.0"
}

// compareVersions 比较两个语义化版本号 (例如 v1.1.23 与 v1.3.0)
// 返回 1: v1 > v2; 0: v1 == v2; -1: v1 < v2
func compareVersions(v1, v2 string) int {
	clean1 := strings.TrimPrefix(strings.TrimSpace(v1), "v")
	clean2 := strings.TrimPrefix(strings.TrimSpace(v2), "v")

	parts1 := strings.Split(clean1, ".")
	parts2 := strings.Split(clean2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}
	return 0
}

// checkLatestRelease 从 GitHub API 检查最新 Release
func checkLatestRelease() (*githubRelease, *githubAsset, bool, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	apiCandidates := []string{
		githubApiLatestUrl,
		"https://gh-proxy.com/" + githubApiLatestUrl,
		"https://ghproxy.net/" + githubApiLatestUrl,
	}

	var resp *http.Response
	var lastErr error

	for _, targetUrl := range apiCandidates {
		req, err := http.NewRequest("GET", targetUrl, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "baihu-panel-tray")

		r, err := client.Do(req)
		if err == nil && r.StatusCode == http.StatusOK {
			resp = r
			break
		}
		if r != nil {
			r.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", r.StatusCode)
		} else {
			lastErr = err
		}
	}

	if resp == nil {
		return nil, nil, false, fmt.Errorf("无法连接 GitHub API: %w", lastErr)
	}
	defer resp.Body.Close()

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, nil, false, err
	}

	currVer := getCurrentVersion()
	hasUpdate := compareVersions(release.TagName, currVer) > 0

	// 查找 Windows 安装包 asset (如 BaihuPanel-Setup-v1.3.0-windows-amd64.exe)
	var installerAsset *githubAsset
	for i := range release.Assets {
		name := strings.ToLower(release.Assets[i].Name)
		if strings.HasSuffix(name, ".exe") && (strings.Contains(name, "setup") || strings.Contains(name, "windows")) {
			installerAsset = &release.Assets[i]
			break
		}
	}

	// 兜底拼接标准 Release URL
	if installerAsset == nil && release.TagName != "" {
		fallbackName := fmt.Sprintf("BaihuPanel-Setup-%s-windows-amd64.exe", release.TagName)
		fallbackUrl := fmt.Sprintf("https://github.com/engigu/baihu-panel/releases/download/%s/%s", release.TagName, fallbackName)
		installerAsset = &githubAsset{
			Name:               fallbackName,
			BrowserDownloadURL: fallbackUrl,
		}
	}

	return &release, installerAsset, hasUpdate, nil
}

// downloadWithMirrors 下载文件，支持多个国内镜像加速容灾
func downloadWithMirrors(rawUrl, destPath string) error {
	mirrors := []string{
		"https://gh-proxy.com/" + rawUrl,
		"https://ghfast.top/" + rawUrl,
		"https://ghproxy.net/" + rawUrl,
		"https://ghp.ci/" + rawUrl,
		rawUrl,
	}

	client := &http.Client{
		Timeout: 5 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var lastErr error
	for _, targetUrl := range mirrors {
		resp, err := client.Get(targetUrl)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP 状态码异常: %d (%s)", resp.StatusCode, targetUrl)
			continue
		}

		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			resp.Body.Close()
			return fmt.Errorf("创建本地临时安装包失败: %w", err)
		}

		_, err = io.Copy(outFile, resp.Body)
		outFile.Close()
		resp.Body.Close()

		if err == nil {
			// 下载成功
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("所有下载镜像源均失败: %w", lastErr)
}

// executeUpdate 执行更新全流程 (方案 B: 全量安装包静默更新)
func executeUpdate(downloadUrl, versionTag string) {
	if isDownloadingUpdate {
		return
	}
	isDownloadingUpdate = true
	defer func() {
		isDownloadingUpdate = false
	}()

	showBalloonNotification("正在下载更新", fmt.Sprintf("正在下载白虎面板新版本 %s 安装包，请稍候...", versionTag))

	tempDir := os.TempDir()
	installerPath := filepath.Join(tempDir, fmt.Sprintf("BaihuPanel-Setup-%s.exe", versionTag))

	err := downloadWithMirrors(downloadUrl, installerPath)
	if err != nil {
		showMessage("更新失败", fmt.Sprintf("下载新版本安装包失败：%v\n\n请检查网络连接后重试。", err), mbIconError)
		return
	}

	showBalloonNotification("下载完成", "新版本安装包已就绪，正在准备执行安装升级...")

	// 获取当前安装目录
	exePath, err := os.Executable()
	if err != nil {
		showMessage("更新失败", "无法获取当前程序安装路径", mbIconError)
		return
	}
	installDir := filepath.Dir(exePath)

	// 停止当前正在运行的白虎后台服务，释放文件锁
	stopPanelService()

	// 编写批处理启动器：静默安装并重新拉起托盘
	updaterBat := filepath.Join(tempDir, fmt.Sprintf("baihu_updater_%d.bat", time.Now().Unix()))
	batContent := fmt.Sprintf(`@echo off
chcp 65001 >nul
echo 正在安装白虎面板更新，请稍候...
timeout /t 1 /nobreak >nul
"%s" /SILENT /SUPPRESSMSGBOXES /NORESTART
timeout /t 1 /nobreak >nul
if exist "%s\baihu-tray.exe" (
    start "" "%s\baihu-tray.exe"
)
del "%s"
del "%%~f0"
`, installerPath, installDir, installDir, installerPath)

	if err := os.WriteFile(updaterBat, []byte(batContent), 0755); err != nil {
		showMessage("更新失败", fmt.Sprintf("生成更新引导脚本失败：%v", err), mbIconError)
		return
	}

	// 启动批处理脚本并退出托盘自身
	cmd := exec.Command("cmd.exe", "/c", updaterBat)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	if err := cmd.Start(); err != nil {
		showMessage("更新失败", fmt.Sprintf("启动更新程序失败：%v", err), mbIconError)
		return
	}

	// 退出托盘进程，让安装程序顺利覆盖
	os.Exit(0)
}

// checkAndTriggerUpdate 检查更新并提示/触发用户执行
func checkAndTriggerUpdate(manual bool) {
	if isDownloadingUpdate {
		if manual {
			showMessage("正在更新", "新版本正在后台下载中，请稍候...", mbIconInformation)
		}
		return
	}

	rel, asset, hasUpdate, err := checkLatestRelease()
	if err != nil {
		if manual {
			showMessage("检查更新失败", fmt.Sprintf("未能获取最新版本信息：%v\n请检查网络连接。", err), mbIconWarning)
		}
		return
	}

	latestReleaseInfo = rel
	latestAssetInfo = asset
	hasNewVersion = hasUpdate

	currVer := getCurrentVersion()

	if !hasUpdate {
		if manual {
			showMessage("已是最新版本", fmt.Sprintf("白虎面板当前已是最新版本 (%s)。", currVer), mbIconInformation)
		}
		return
	}

	// 发现新版本
	if manual {
		tipText := fmt.Sprintf("发现白虎面板新版本 %s！\n\n当前版本：%s\n最新版本：%s\n\n是否立即下载并升级？",
			rel.TagName, currVer, rel.TagName)
		if strings.TrimSpace(rel.Body) != "" {
			// 附带更新日志前 300 字符
			logSnippet := strings.TrimSpace(rel.Body)
			if len(logSnippet) > 300 {
				logSnippet = logSnippet[:300] + "..."
			}
			tipText += fmt.Sprintf("\n\n【更新日志】\n%s", logSnippet)
		}

		ret := showMessage("发现新版本", tipText, mbYesNo|mbIconQuestion)
		if ret == idYes {
			if asset != nil && asset.BrowserDownloadURL != "" {
				go executeUpdate(asset.BrowserDownloadURL, rel.TagName)
			}
		}
	} else {
		// 自动后台检查，通过托盘气球通知温和提示
		showBalloonNotification("发现新版本", fmt.Sprintf("白虎面板最新版本 %s 已发布，右键点击托盘即可快速升级。", rel.TagName))
	}
}

// startPeriodicUpdateCheck 启动后台定时检查更新 (启动后立即初检，后续每隔 2 小时检查一次)
func startPeriodicUpdateCheck() {
	go func() {
		// 启动后立即在后台异步初检，以便右键菜单快速显示更新状态
		checkAndTriggerUpdate(false)
		trimWorkingSet()

		ticker := time.NewTicker(2 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			checkAndTriggerUpdate(false)
			trimWorkingSet()
		}
	}()
}
