package services

import (
	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/logger"
	"baihu/internal/models"
	"baihu/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SyncResult 同步执行结果
type SyncResult struct {
	SyncTaskID uint
	Success    bool
	Output     string
	Error      string
	Start      time.Time
	End        time.Time
}

// SyncExecutorService 同步任务执行服务
type SyncExecutorService struct {
	syncTaskService *SyncTaskService
	runningSyncs    map[uint]bool
	mu              sync.RWMutex
}

// NewSyncExecutorService 创建同步执行服务
func NewSyncExecutorService(syncTaskService *SyncTaskService) *SyncExecutorService {
	return &SyncExecutorService{
		syncTaskService: syncTaskService,
		runningSyncs:    make(map[uint]bool),
	}
}

// ExecuteSyncTask 执行同步任务
func (s *SyncExecutorService) ExecuteSyncTask(taskID int) *SyncResult {
	task := s.syncTaskService.GetSyncTaskByID(taskID)
	if task == nil {
		return &SyncResult{
			SyncTaskID: uint(taskID),
			Success:    false,
			Error:      "同步任务不存在",
			Start:      time.Now(),
			End:        time.Now(),
		}
	}

	// 标记任务开始运行
	s.mu.Lock()
	s.runningSyncs[task.ID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningSyncs, task.ID)
		s.mu.Unlock()
	}()

	result := &SyncResult{
		SyncTaskID: task.ID,
		Start:      time.Now(),
	}

	var output string
	var err error

	switch task.SourceType {
	case "url":
		output, err = s.syncFromURL(task)
	case "git":
		output, err = s.syncFromGit(task)
	default:
		err = fmt.Errorf("不支持的源类型: %s", task.SourceType)
	}

	result.End = time.Now()
	result.Output = output

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	// 保存日志
	s.saveLog(task, result)

	// 更新任务状态
	status := "success"
	if !result.Success {
		status = "failed"
	}
	s.syncTaskService.UpdateLastSync(task.ID, status)

	// 清理日志
	s.cleanLogs(task)

	return result
}

// syncFromURL 从 URL 下载文件
func (s *SyncExecutorService) syncFromURL(task *models.SyncTask) (string, error) {
	var output strings.Builder

	// 构建实际下载 URL
	downloadURL := s.buildProxyURL(task.SourceURL, task.Proxy, task.ProxyURL)
	output.WriteString(fmt.Sprintf("下载地址: %s\n", downloadURL))

	// 确保目标目录存在
	targetDir := filepath.Dir(task.TargetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return output.String(), fmt.Errorf("创建目录失败: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return output.String(), fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加认证 Token
	if task.AuthToken != "" {
		req.Header.Set("Authorization", "token "+task.AuthToken)
	}

	// 发送请求
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return output.String(), fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return output.String(), fmt.Errorf("下载失败, HTTP 状态码: %d", resp.StatusCode)
	}

	// 写入文件
	file, err := os.Create(task.TargetPath)
	if err != nil {
		return output.String(), fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return output.String(), fmt.Errorf("写入文件失败: %v", err)
	}

	output.WriteString(fmt.Sprintf("目标路径: %s\n", task.TargetPath))
	output.WriteString(fmt.Sprintf("文件大小: %d 字节\n", written))
	output.WriteString("同步完成\n")

	return output.String(), nil
}

// syncFromGit 从 Git 仓库同步
func (s *SyncExecutorService) syncFromGit(task *models.SyncTask) (string, error) {
	var output strings.Builder

	// 构建实际 Git URL
	gitURL := s.buildProxyURL(task.SourceURL, task.Proxy, task.ProxyURL)
	output.WriteString(fmt.Sprintf("Git 地址: %s\n", gitURL))
	output.WriteString(fmt.Sprintf("目标路径: %s\n", task.TargetPath))

	// 检查目标目录是否存在
	gitDir := filepath.Join(task.TargetPath, ".git")
	isExistingRepo := false
	if _, err := os.Stat(gitDir); err == nil {
		isExistingRepo = true
	}

	var cmd *exec.Cmd
	var cmdOutput bytes.Buffer

	if isExistingRepo {
		// 已存在仓库，执行 git pull
		output.WriteString("检测到已存在仓库，执行 git pull\n")

		// 先切换分支（如果指定了分支）
		if task.Branch != "" {
			checkoutCmd := exec.Command("git", "checkout", task.Branch)
			checkoutCmd.Dir = task.TargetPath
			checkoutCmd.Stdout = &cmdOutput
			checkoutCmd.Stderr = &cmdOutput
			if err := checkoutCmd.Run(); err != nil {
				output.WriteString(fmt.Sprintf("切换分支输出: %s\n", cmdOutput.String()))
			}
			cmdOutput.Reset()
		}

		cmd = exec.Command("git", "pull")
		cmd.Dir = task.TargetPath
	} else {
		// 新仓库，执行 git clone
		output.WriteString("执行 git clone\n")

		// 确保父目录存在
		parentDir := filepath.Dir(task.TargetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return output.String(), fmt.Errorf("创建目录失败: %v", err)
		}

		args := []string{"clone"}
		if task.Branch != "" {
			args = append(args, "-b", task.Branch)
		}
		args = append(args, gitURL, task.TargetPath)
		cmd = exec.Command("git", args...)
	}

	cmd.Stdout = &cmdOutput
	cmd.Stderr = &cmdOutput

	// 设置认证
	if task.AuthToken != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_ASKPASS=echo %s", task.AuthToken))
	}

	err := cmd.Run()
	output.WriteString(cmdOutput.String())

	if err != nil {
		return output.String(), fmt.Errorf("Git 操作失败: %v", err)
	}

	output.WriteString("同步完成\n")
	return output.String(), nil
}

// buildProxyURL 构建代理 URL
func (s *SyncExecutorService) buildProxyURL(sourceURL, proxy, customProxyURL string) string {
	if proxy == "" || proxy == "none" {
		return sourceURL
	}

	var proxyBase string
	switch proxy {
	case "ghproxy":
		proxyBase = "https://ghproxy.com/"
	case "mirror":
		proxyBase = "https://mirror.ghproxy.com/"
	case "custom":
		proxyBase = customProxyURL
		if proxyBase != "" && !strings.HasSuffix(proxyBase, "/") {
			proxyBase += "/"
		}
	default:
		return sourceURL
	}

	if proxyBase == "" {
		return sourceURL
	}

	return proxyBase + sourceURL
}

// saveLog 保存同步日志
func (s *SyncExecutorService) saveLog(task *models.SyncTask, result *SyncResult) {
	output := result.Output
	if result.Error != "" {
		output += "\n[ERROR]\n" + result.Error
	}

	compressed, err := utils.CompressToBase64(output)
	if err != nil {
		logger.Errorf("压缩日志失败: %v", err)
		compressed = ""
	}

	status := "success"
	if !result.Success {
		status = "failed"
	}

	log := &models.SyncTaskLog{
		SyncTaskID: task.ID,
		SourceURL:  task.SourceURL,
		TargetPath: task.TargetPath,
		Output:     compressed,
		Status:     status,
		Duration:   result.End.Sub(result.Start).Milliseconds(),
	}

	if err := database.DB.Create(log).Error; err != nil {
		logger.Errorf("保存同步日志失败: %v", err)
	}
}

// cleanLogs 清理日志
func (s *SyncExecutorService) cleanLogs(task *models.SyncTask) {
	if task.CleanConfig == "" {
		return
	}

	var config struct {
		Type string `json:"type"`
		Keep int    `json:"keep"`
	}
	if err := json.Unmarshal([]byte(task.CleanConfig), &config); err != nil {
		logger.Errorf("解析清理配置失败: %v", err)
		return
	}

	if config.Keep <= 0 {
		return
	}

	var deleted int64
	switch config.Type {
	case "day":
		cutoff := time.Now().AddDate(0, 0, -config.Keep)
		result := database.DB.Where("sync_task_id = ? AND created_at < ?", task.ID, cutoff).Delete(&models.SyncTaskLog{})
		deleted = result.RowsAffected
	case "count":
		var boundaryLog models.SyncTaskLog
		err := database.DB.Where("sync_task_id = ?", task.ID).Order("id DESC").Offset(config.Keep - 1).Limit(1).First(&boundaryLog).Error
		if err == nil {
			result := database.DB.Where("sync_task_id = ? AND id < ?", task.ID, boundaryLog.ID).Delete(&models.SyncTaskLog{})
			deleted = result.RowsAffected
		}
	}

	if deleted > 0 {
		logger.Infof("清理了 %d 条同步任务 %d 的日志", deleted, task.ID)
	}
}

// GetRunningCount 获取正在运行的同步任务数量
func (s *SyncExecutorService) GetRunningCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.runningSyncs)
}

// ValidateTargetPath 验证目标路径是否在允许的目录内
func (s *SyncExecutorService) ValidateTargetPath(targetPath string) bool {
	// 空路径使用默认 scripts 目录，是有效的
	if targetPath == "" {
		return true
	}

	// 获取绝对路径
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	// 获取 scripts 目录的绝对路径
	absScripts, err := filepath.Abs(constant.ScriptsWorkDir)
	if err != nil {
		return false
	}

	// 检查目标路径是否在 scripts 目录下
	return strings.HasPrefix(absTarget, absScripts)
}
