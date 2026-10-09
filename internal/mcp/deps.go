package mcp

import (
	"cmp"

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

// lazyOr 当 val 为零值时才调用 factory，确保严格惰性求值与零额外分配副作用
func lazyOr[T comparable](val T, factory func() T) T {
	var zero T
	if val != zero {
		return val
	}
	return factory()
}

// EnsureDefaults 补全缺失依赖（基于 cmp.Or 与 lazyOr 严格惰性初始化）
func (d *Deps) EnsureDefaults() *Deps {
	if d == nil {
		return InitDefaultDeps()
	}
	d.FileWorkDir = cmp.Or(d.FileWorkDir, constant.ScriptsWorkDir)
	d.AppService = cmp.Or(d.AppService, app.DefaultAppService)
	d.FileService = lazyOr(d.FileService, func() *services.FileService { return services.NewFileService(d.FileWorkDir) })
	d.NotifyService = lazyOr(d.NotifyService, services.NewNotificationService)
	d.TaskService = lazyOr(d.TaskService, tasks.NewTaskService)
	d.EnvService = lazyOr(d.EnvService, services.NewEnvService)
	return d
}

