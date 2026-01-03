package controllers

import (
	"path/filepath"
	"runtime"
	"strconv"

	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/models"
	"baihu/internal/services"
	"baihu/internal/services/tasks"
	"baihu/internal/utils"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/process"
)

type SettingsController struct {
	userService     *services.UserService
	settingsService *services.SettingsService
	loginLogService *services.LoginLogService
	backupService   *services.BackupService
	executorService *tasks.ExecutorService
}

func NewSettingsController(userService *services.UserService, loginLogService *services.LoginLogService, executorService *tasks.ExecutorService) *SettingsController {
	return &SettingsController{
		userService:     userService,
		settingsService: services.NewSettingsService(),
		loginLogService: loginLogService,
		backupService:   services.NewBackupService(),
		executorService: executorService,
	}
}

// ChangePassword 修改密码
func (sc *SettingsController) ChangePassword(c *gin.Context) {
	// 演示模式下禁止修改密码
	if constant.DemoMode {
		utils.BadRequest(c, "演示模式下不能修改密码")
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	// 暂时使用固定用户名 admin
	user := sc.userService.GetUserByUsername("admin")
	if user == nil {
		utils.NotFound(c, "用户不存在")
		return
	}

	if !sc.userService.ValidatePassword(user, req.OldPassword) {
		utils.BadRequest(c, "原密码错误")
		return
	}

	if err := sc.userService.UpdatePassword(user.ID, req.NewPassword); err != nil {
		utils.ServerError(c, "修改密码失败")
		return
	}

	utils.SuccessMsg(c, "密码修改成功")
}

// CleanLogs 清理日志 - 已移除，改为任务级别的日志清理配置

// GetSiteSettings 获取站点设置
func (sc *SettingsController) GetSiteSettings(c *gin.Context) {
	settings := sc.settingsService.GetSection(constant.SectionSite)
	utils.Success(c, settings)
}

// GetPublicSiteSettings 获取公开的站点设置（无需认证）
func (sc *SettingsController) GetPublicSiteSettings(c *gin.Context) {
	settings := sc.settingsService.GetSection(constant.SectionSite)
	// 只返回公开信息
	utils.Success(c, gin.H{
		constant.KeyTitle:    settings[constant.KeyTitle],
		constant.KeySubtitle: settings[constant.KeySubtitle],
		constant.KeyIcon:     settings[constant.KeyIcon],
		"demo_mode":          constant.DemoMode,
	})
}

// UpdateSiteSettings 更新站点设置
func (sc *SettingsController) UpdateSiteSettings(c *gin.Context) {
	var req struct {
		Title      string `json:"title"`
		Subtitle   string `json:"subtitle"`
		Icon       string `json:"icon"`
		PageSize   string `json:"page_size"`
		CookieDays string `json:"cookie_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	values := map[string]string{
		constant.KeyTitle:      req.Title,
		constant.KeySubtitle:   req.Subtitle,
		constant.KeyIcon:       req.Icon,
		constant.KeyPageSize:   req.PageSize,
		constant.KeyCookieDays: req.CookieDays,
	}

	if err := sc.settingsService.SetSection(constant.SectionSite, values); err != nil {
		utils.ServerError(c, "保存失败")
		return
	}

	utils.SuccessMsg(c, "保存成功")
}

// GetSchedulerSettings 获取调度设置
func (sc *SettingsController) GetSchedulerSettings(c *gin.Context) {
	settings := sc.settingsService.GetSection(constant.SectionScheduler)
	utils.Success(c, settings)
}

// UpdateSchedulerSettings 更新调度设置
func (sc *SettingsController) UpdateSchedulerSettings(c *gin.Context) {
	var req struct {
		WorkerCount  string `json:"worker_count"`
		QueueSize    string `json:"queue_size"`
		RateInterval string `json:"rate_interval"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	values := map[string]string{
		constant.KeyWorkerCount:  req.WorkerCount,
		constant.KeyQueueSize:    req.QueueSize,
		constant.KeyRateInterval: req.RateInterval,
	}

	if err := sc.settingsService.SetSection(constant.SectionScheduler, values); err != nil {
		utils.ServerError(c, "保存失败")
		return
	}

	// 重新加载 executor service
	if sc.executorService != nil {
		sc.executorService.Reload()
	}

	utils.SuccessMsg(c, "保存成功")
}

// GetAbout 获取关于信息
func (sc *SettingsController) GetAbout(c *gin.Context) {
	var taskCount, logCount, envCount int64
	database.DB.Model(&models.Task{}).Count(&taskCount)
	database.DB.Model(&models.TaskLog{}).Count(&logCount)
	database.DB.Model(&models.EnvironmentVariable{}).Count(&envCount)

	// 内存使用
	memUsage := "N/A"
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if memInfo, err := p.MemoryInfo(); err == nil {
			memUsage = formatBytes(memInfo.RSS)
		}
	}

	// 运行时间
	uptime := formatDuration(time.Since(constant.StartTime))

	utils.Success(c, gin.H{
		"version":    constant.Version,
		"build_time": constant.BuildTime,
		"mem_usage":  memUsage,
		"goroutines": runtime.NumGoroutine(),
		"uptime":     uptime,
		"task_count": taskCount,
		"log_count":  logCount,
		"env_count":  envCount,
	})
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天%d小时%d分钟%d秒", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟%d秒", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d分钟%d秒", minutes, seconds)
	}
	return fmt.Sprintf("%d秒", seconds)
}

// GetLoginLogs 获取登录日志
func (sc *SettingsController) GetLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	username := c.Query("username")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	logs, total, err := sc.loginLogService.List(page, pageSize, username)
	if err != nil {
		utils.ServerError(c, "获取登录日志失败")
		return
	}

	utils.Success(c, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateBackup 创建备份
func (sc *SettingsController) CreateBackup(c *gin.Context) {
	_, err := sc.backupService.CreateBackup()
	if err != nil {
		utils.ServerError(c, "创建备份失败: "+err.Error())
		return
	}
	utils.SuccessMsg(c, "备份创建成功")
}

// GetBackupStatus 获取备份状态
func (sc *SettingsController) GetBackupStatus(c *gin.Context) {
	filePath := sc.backupService.GetBackupFile()
	var backupTime string
	if filePath != "" {
		if info, err := os.Stat(filePath); err == nil {
			backupTime = info.ModTime().Format("2006-01-02 15:04:05")
		}
	}
	utils.Success(c, gin.H{
		"has_backup":  filePath != "",
		"backup_time": backupTime,
	})
}

// DownloadBackup 下载备份文件
func (sc *SettingsController) DownloadBackup(c *gin.Context) {
	filePath := sc.backupService.GetBackupFile()
	if filePath == "" {
		utils.NotFound(c, "没有可下载的备份")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		sc.backupService.ClearBackup()
		utils.NotFound(c, "备份文件不存在")
		return
	}

	// 设置响应头
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	c.Header("Content-Type", "application/zip")
	c.File(filePath)

	// 下载后清除备份记录和文件
	go func() {
		time.Sleep(time.Minute * 5) // 等待下载完成
		sc.backupService.ClearBackup()
	}()
}

// RestoreBackup 恢复备份
func (sc *SettingsController) RestoreBackup(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "请上传备份文件")
		return
	}

	// 保存上传的文件
	tempPath := filepath.Join(os.TempDir(), file.Filename)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		utils.ServerError(c, "保存文件失败")
		return
	}
	defer os.Remove(tempPath)

	// 恢复备份
	if err := sc.backupService.Restore(tempPath); err != nil {
		utils.ServerError(c, "恢复失败: "+err.Error())
		return
	}

	utils.SuccessMsg(c, "恢复成功")
}
