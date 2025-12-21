package router

import (
	"baihu/internal/constant"
	"baihu/internal/controllers"
	"baihu/internal/services"
)

var cronService *services.CronService

func RegisterControllers() *Controllers {
	// Initialize services
	settingsService := services.NewSettingsService()
	loginLogService := services.NewLoginLogService()

	// 执行系统初始化（返回 userService）
	initService := services.NewInitService(settingsService)
	userService := initService.Initialize()

	taskService := services.NewTaskService()
	envService := services.NewEnvService()
	scriptService := services.NewScriptService()
	executorService := services.NewExecutorService(taskService)

	// Initialize cron service
	cronService = services.NewCronService(taskService, executorService)
	cronService.Start()

	// Initialize and return controllers
	return &Controllers{
		Task:      controllers.NewTaskController(taskService, cronService),
		Auth:      controllers.NewAuthController(userService, settingsService, loginLogService),
		Env:       controllers.NewEnvController(envService),
		Script:    controllers.NewScriptController(scriptService),
		Executor:  controllers.NewExecutorController(executorService),
		File:      controllers.NewFileController(constant.ScriptsWorkDir),
		Dashboard: controllers.NewDashboardController(cronService, executorService),
		Log:       controllers.NewLogController(),
		Terminal:  controllers.NewTerminalController(),
		Settings:  controllers.NewSettingsController(userService, loginLogService),
	}
}

// StopCron stops the cron service gracefully
func StopCron() {
	if cronService != nil {
		cronService.Stop()
	}
}
