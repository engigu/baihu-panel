package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// selfUpdate 自动更新
func (a *Agent) selfUpdate() {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		log.Errorf("获取可执行文件路径失败: %v", err)
		return
	}
	exePath, _ = filepath.Abs(exePath)

	// 下载新版本 tar.gz
	downloadURL := a.config.ServerURL + "/api/agent/download?os=" + runtime.GOOS + "&arch=" + runtime.GOARCH
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		log.Errorf("创建下载请求失败: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.config.Token)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("下载新版本失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf("下载新版本失败: HTTP %d", resp.StatusCode)
		return
	}

	// 读取 tar.gz 内容
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Errorf("解压 gzip 失败: %v", err)
		return
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// 解压并找到二进制文件
	var newBinary []byte
	binaryName := "baihu-agent"
	if runtime.GOOS == "windows" {
		binaryName = "baihu-agent.exe"
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("读取 tar 失败: %v", err)
			return
		}

		if header.Typeflag == tar.TypeReg && header.Name == binaryName {
			newBinary, err = io.ReadAll(tarReader)
			if err != nil {
				log.Errorf("读取二进制文件失败: %v", err)
				return
			}
			break
		}
	}

	if newBinary == nil {
		log.Errorf("tar.gz 中未找到 %s", binaryName)
		return
	}

	// 保存到临时文件（放到 data 目录）
	os.MkdirAll(dataDir, 0755)
	tmpFile := filepath.Join(dataDir, binaryName+".new")
	if err := os.WriteFile(tmpFile, newBinary, 0755); err != nil {
		log.Errorf("保存新版本失败: %v", err)
		return
	}

	// 计算基础路径（去掉所有 .bak 后缀）
	basePath := exePath
	for strings.HasSuffix(basePath, ".bak") {
		basePath = strings.TrimSuffix(basePath, ".bak")
	}
	backupFile := basePath + ".bak"

	// 如果当前运行的就是 .bak 文件，直接删除它（更新后会用新版本）
	// 否则需要备份当前文件
	if exePath != backupFile {
		os.Remove(backupFile)
		if err := os.Rename(exePath, backupFile); err != nil {
			log.Errorf("备份旧版本失败: %v", err)
			os.Remove(tmpFile)
			return
		}
	}

	// 替换为新版本（放到 basePath，即不带 .bak 的路径）
	if err := os.Rename(tmpFile, basePath); err != nil {
		log.Errorf("替换新版本失败: %v", err)
		if exePath != backupFile {
			os.Rename(backupFile, exePath) // 恢复旧版本
		}
		return
	}

	// 如果之前运行的是 .bak 文件，现在可以删除它了
	if exePath == backupFile {
		os.Remove(exePath)
	}

	log.Info("更新完成，正在重启...")

	// 重启服务
	a.restart()
}

// restart 重启服务
func (a *Agent) restart() {
	exePath, _ := os.Executable()

	// 计算基础路径（去掉所有 .bak 后缀），确保启动的是正确的可执行文件
	basePath := exePath
	for strings.HasSuffix(basePath, ".bak") {
		basePath = strings.TrimSuffix(basePath, ".bak")
	}

	if runtime.GOOS == "windows" {
		// Windows: 启动新进程后退出
		cmd := exec.Command(basePath, "start")
		cmd.Start()
		os.Exit(0)
	} else {
		// Linux/macOS: 使用 exec 替换当前进程
		syscall.Exec(basePath, []string{basePath, "start"}, os.Environ())
	}
}
