package mcp

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/app"
	"github.com/engigu/baihu-panel/internal/services/tasks"
)

// Deps 包含 MCP 服务所需的依赖服务引用
type Deps struct {
	TaskService     *tasks.TaskService
	ExecutorService *tasks.ExecutorService
	TaskLogService  *tasks.TaskLogService
	EnvService      *services.EnvService
	FileService     *services.FileService
	AppService      *app.AppService
	NotifyService   *services.NotificationService
	FileWorkDir     string
}

// InitDefaultDeps 初始化默认依赖（常用于 CLI 模式或默认上下文）
func InitDefaultDeps() *Deps {
	taskService := tasks.NewTaskService()
	envService := services.NewEnvService()
	sendStatsService := services.NewSendStatsService()
	agentWSManager := services.GetAgentWSManager()
	settingsService := services.NewSettingsService()
	taskLogService := tasks.NewTaskLogService(sendStatsService)
	fileService := services.NewFileService(constant.ScriptsWorkDir)
	notifyService := services.NewNotificationService()

	executorService := tasks.NewExecutorService(taskService, taskLogService, agentWSManager, settingsService, envService)

	return &Deps{
		TaskService:     taskService,
		ExecutorService: executorService,
		TaskLogService:  taskLogService,
		EnvService:      envService,
		FileService:     fileService,
		AppService:      app.DefaultAppService,
		NotifyService:   notifyService,
		FileWorkDir:     constant.ScriptsWorkDir,
	}
}
