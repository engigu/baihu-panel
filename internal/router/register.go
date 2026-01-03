package router

import (
	"baihu/internal/constant"
	"baihu/internal/controllers"
	"baihu/internal/services"
	"baihu/internal/services/tasks"
)

var cronService *tasks.CronService

func RegisterControllers() *Controllers {
	// Initialize services
	settingsService := services.NewSettingsService()
	loginLogService := services.NewLoginLogService()

	// 执行系统初始化（返回 userService）
	initService := services.NewInitService(settingsService)
	userService := initService.Initialize()

	taskService := tasks.NewTaskService()
	envService := services.NewEnvService()
	scriptService := services.NewScriptService()
	sendStatsService := services.NewSendStatsService()
	agentWSManager := services.GetAgentWSManager()
	
	// 创建任务执行服务（需要依赖注入）
	taskExecutionService := tasks.NewTaskExecutionService(agentWSManager, sendStatsService)
	executorService := tasks.NewExecutorService(taskService, taskExecutionService, settingsService, envService)

	// Initialize cron service
	cronService = tasks.NewCronService(taskService, executorService)
	cronService.Start()

	// Initialize and return controllers
	return &Controllers{
		Task:       controllers.NewTaskController(taskService, cronService),
		Auth:       controllers.NewAuthController(userService, settingsService, loginLogService),
		Env:        controllers.NewEnvController(envService),
		Script:     controllers.NewScriptController(scriptService),
		Executor:   controllers.NewExecutorController(executorService),
		File:       controllers.NewFileController(constant.ScriptsWorkDir),
		Dashboard:  controllers.NewDashboardController(cronService, executorService),
		Log:        controllers.NewLogController(),
		Terminal:   controllers.NewTerminalController(),
		Settings:   controllers.NewSettingsController(userService, loginLogService, executorService),
		Dependency: controllers.NewDependencyController(),
		Agent:      controllers.NewAgentController(),
	}
}

// StopCron stops the cron service gracefully
func StopCron() {
	if cronService != nil {
		cronService.Stop()
	}
}
