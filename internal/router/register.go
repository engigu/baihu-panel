package router

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/controllers"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/tasks"
)

var executorService *tasks.ExecutorService

func RegisterControllers() *Controllers {
	// 初始化服务
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

	taskLogService := tasks.NewTaskLogService(sendStatsService)
	workflowService := services.NewWorkflowService()

	// 创建任务执行服务（需要依赖注入）

	// 清理 task 运行状态的任务可以直接由 executorService 承担或在此处通过 Database 直接清理
	// 简单期间，我们使用一个新方法 tasks.CleanupRunningTasks() 或者让 executorService 启动时清理

	executorService = tasks.NewExecutorService(taskService, taskLogService, agentWSManager, settingsService, envService)
	// 启动时清理残留的运行状态
	_ = executorService.CleanupRunningTasks()

	// 启动计划任务
	executorService.StartCron()

	// 初始化并返回控制器
	return &Controllers{
		Task:       controllers.NewTaskController(taskService, executorService),
		Auth:       controllers.NewAuthController(userService, settingsService, loginLogService),
		Env:        controllers.NewEnvController(envService),
		Script:     controllers.NewScriptController(scriptService),
		Executor:   controllers.NewExecutorController(executorService),
		File:       controllers.NewFileController(constant.ScriptsWorkDir),
		Dashboard:  controllers.NewDashboardController(executorService),
		Log:        controllers.NewLogController(),
		LogWS:      controllers.NewLogWSController(),
		Terminal:   controllers.NewTerminalController(envService),
		Settings:   controllers.NewSettingsController(userService, loginLogService, executorService),
		Dependency: controllers.NewDependencyController(),
		Agent:      controllers.NewAgentController(settingsService),
		Mise:       controllers.NewMiseController(services.NewMiseService()),
		Workflow:   controllers.NewWorkflowController(workflowService, executorService),
	}
}

// StopCron 停止计划任务服务
func StopCron() {
	if executorService != nil {
		executorService.Stop()
	}
}
