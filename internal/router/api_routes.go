package router

import (
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/gin-gonic/gin"
)

func initPublicAPIRoutes(api *gin.RouterGroup, c *Controllers) {
	// Health check (无需认证)
	api.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "pong"})
	})

	// api.GET("/debug/goroutines", func(ctx *gin.Context) {
	// 	buf := make([]byte, 1024*1024)
	// 	n := runtime.Stack(buf, true)
	// 	ctx.Data(200, "text/plain; charset=utf-8", buf[:n])
	// })

	// Authentication routes (无需认证)
	auth := api.Group("/auth")
	{
		auth.POST("/login", c.Auth.Login)
		auth.POST("/logout", c.Auth.Logout)
		// auth.POST("/register", c.Auth.Register)
	}

	// 公开的站点设置（无需认证）
	api.GET("/settings/public", c.Settings.GetPublicSiteSettings)

	// 隧道模式 (被控端反向连入，使用独立 Token 做 WebSocket 鉴权)
	api.GET("/interconnect/tunnel", c.Interconnect.HandleTunnel)
	// 子节点主动上报监控数据 (无中间件鉴权，内部鉴权)
	api.POST("/interconnect/report", c.Interconnect.ReportMonitorData)

	// 内部使用的 API（仅限本地调用，无需 Bearer 认证）
	internalAPI := api.Group("/internal")
	internalAPI.Use(middleware.LocalhostOnly())
	{
		internalAPI.POST("/tasks/sync-repo-status", c.Task.SyncRepoTasks)
		internalAPI.POST("/tasks/execute/:id", c.Executor.ExecuteTask)
		internalAPI.POST("/tasks/toggle/:id", c.Task.ToggleTask)
	}
}

func initAuthorizedAPIRoutes(api *gin.RouterGroup, c *Controllers) {
	authorized := api.Group("")
	authorized.Use(middleware.AuthRequired())
	{
		// 获取当前用户 (普通用户即可访问)
		authorized.GET("/auth/me", c.Auth.GetCurrentUser)

		// 以下管理接口需要管理员权限
		adminOnly := authorized.Group("")
		adminOnly.Use(middleware.AdminRequired())
		{
			registerDashboardRoutes(adminOnly, c)
			registerTaskRoutes(adminOnly, c)
			registerEnvRoutes(adminOnly, c)
			registerSandboxRoutes(adminOnly, c)
			registerScriptRoutes(adminOnly, c)
			registerFileRoutes(adminOnly, c)
			registerLogRoutes(adminOnly, c)
			registerTerminalRoutes(adminOnly, c)
			registerSettingsRoutes(adminOnly, c)
			registerDependencyRoutes(adminOnly, c)
			registerAgentRoutes(adminOnly, c)
			registerMiseRoutes(adminOnly, c)
			registerNotificationRoutes(adminOnly, c)
			registerAppLogRoutes(adminOnly, c)
			registerSystemWSRoutes(adminOnly, c)
			registerWebUIRoutes(adminOnly, c)
			registerMonitorRoutes(adminOnly, c)
			registerInterconnectRoutes(adminOnly, c)
			registerSystemRoutes(adminOnly, c)
		}
	}

	// 通知发送 API（使用通知 Token 认证，供脚本调用）
	notifyAPI := api.Group("/notify")
	notifyAPI.Use(middleware.NotifyTokenAuth())
	{
		notifyAPI.POST("/send", c.Notification.SendNotification)
	}
}

func registerDashboardRoutes(g *gin.RouterGroup, c *Controllers) {
	g.GET("/stats", c.Dashboard.GetStats)
	g.GET("/sentence", c.Dashboard.GetSentence)
	g.GET("/sendstats", c.Dashboard.GetSendStats)
	g.GET("/taskstats", c.Dashboard.GetTaskStats)
}

func registerTaskRoutes(g *gin.RouterGroup, c *Controllers) {
	tasks := g.Group("/tasks")
	{
		tasks.POST("", c.Task.CreateTask)
		tasks.GET("", c.Task.GetTasks)
		tasks.GET("/:id", c.Task.GetTask)
		tasks.POST("/bulk_save", c.Task.BulkSaveTask)
		tasks.PUT("/:id", c.Task.UpdateTask)
		tasks.DELETE("/:id", c.Task.DeleteTask)
		tasks.POST("/batch-delete", c.Task.BatchDeleteTasks)
		tasks.DELETE("/batch-by-query", c.Task.BatchDeleteByQuery)
		tasks.POST("/stop/:logID", c.Task.StopTask)
		tasks.GET("/tags", c.Task.GetTags)
	}

	execution := g.Group("/execute")
	{
		execution.POST("/task/:id", c.Executor.ExecuteTask)
		execution.POST("/command", c.Executor.ExecuteCommand)
		execution.GET("/results", c.Executor.GetLastResults)
	}
}

func registerEnvRoutes(g *gin.RouterGroup, c *Controllers) {
	env := g.Group("/env")
	{
		env.GET("/secret-status", c.Env.GetSecretStatus)
		env.GET("/tags", c.Env.GetTags)
		env.POST("", c.Env.CreateEnvVar)
		env.POST("/bulk_save", c.Env.BulkSaveEnv)
		env.GET("", c.Env.GetEnvVars)
		env.GET("/all", c.Env.GetAllEnvVars)
		env.GET("/:id", c.Env.GetEnvVar)
		env.GET("/:id/tasks", c.Env.GetAssociatedTasks)
		env.PUT("/:id", c.Env.UpdateEnvVar)
		env.DELETE("/:id", c.Env.DeleteEnvVar)
	}
}

func registerSandboxRoutes(g *gin.RouterGroup, c *Controllers) {
	sandboxes := g.Group("/sandboxes")
	{
		sandboxes.GET("", c.Sandbox.GetSandboxProfiles)
		sandboxes.POST("", c.Sandbox.CreateSandboxProfile)
		sandboxes.POST("/repair", c.Sandbox.RepairSandboxDirectories)
		sandboxes.GET("/:id", c.Sandbox.GetSandboxProfileByID)
		sandboxes.PUT("/:id", c.Sandbox.UpdateSandboxProfile)
		sandboxes.DELETE("/:id", c.Sandbox.DeleteSandboxProfile)
	}
}

func registerScriptRoutes(g *gin.RouterGroup, c *Controllers) {
	scripts := g.Group("/scripts")
	{
		scripts.POST("", c.Script.CreateScript)
		scripts.GET("", c.Script.GetScripts)
		scripts.GET("/:id", c.Script.GetScript)
		scripts.PUT("/:id", c.Script.UpdateScript)
		scripts.DELETE("/:id", c.Script.DeleteScript)
	}
}

func registerFileRoutes(g *gin.RouterGroup, c *Controllers) {
	files := g.Group("/files")
	{
		files.GET("/tree", c.File.GetFileTree)
		files.GET("/content", c.File.GetFileContent)
		files.GET("/download", c.File.DownloadFile)
		files.GET("/download-zip", c.File.DownloadZip)
		files.POST("/content", c.File.SaveFileContent)
		files.POST("/create", c.File.CreateFile)
		files.POST("/delete", c.File.DeleteFile)
		files.POST("/rename", c.File.RenameFile)
		files.POST("/move", c.File.MoveFile)
		files.POST("/copy", c.File.CopyFile)
		files.POST("/upload", c.File.UploadArchive)
		files.POST("/uploadfiles", c.File.UploadFiles)
	}
}

func registerLogRoutes(g *gin.RouterGroup, c *Controllers) {
	logs := g.Group("/logs")
	{
		logs.GET("", c.Log.GetLogs)
		logs.POST("/clear", c.Log.ClearLogs)
		logs.GET("/sse", c.LogSSE.StreamLog)
		logs.GET("/:id", c.Log.GetLogDetail)
		logs.DELETE("/:id", c.Log.DeleteLog)
	}
}

func registerTerminalRoutes(g *gin.RouterGroup, c *Controllers) {
	g.GET("/terminal/ws", c.Terminal.HandleWebSocket)
	// g.POST("/terminal/exec", c.Terminal.ExecuteShellCommand) // 暂未使用，已注释
	g.GET("/terminal/cmds", c.Terminal.GetCommands)
}

func registerSettingsRoutes(g *gin.RouterGroup, c *Controllers) {
	settings := g.Group("/settings")
	{
		settings.POST("/password", c.Settings.ChangePassword)
		settings.GET("/site", c.Settings.GetSiteSettings)
		settings.PUT("/site", c.Settings.UpdateSiteSettings)
		settings.POST("/site/openapi-token/generate", c.Settings.GenerateOpenapiToken)
		settings.GET("/paths", c.Settings.GetPaths)
		settings.GET("/scheduler", c.Settings.GetSchedulerSettings)
		settings.PUT("/scheduler", c.Settings.UpdateSchedulerSettings)
		settings.GET("/about", c.Settings.GetAbout)
		settings.GET("/changelog", c.Settings.GetChangelog)
		settings.GET("/loginlogs", c.Settings.GetLoginLogs)
		settings.POST("/backup", c.Settings.CreateBackup)
		settings.GET("/backup/status", c.Settings.GetBackupStatus)
		settings.GET("/backup/download", c.Settings.DownloadBackup)
		settings.POST("/restore", c.Settings.RestoreBackup)
		// 通用设置接口
		settings.GET("/:section", c.Settings.GetSectionSettings)
		settings.PUT("/:section", c.Settings.UpdateSectionSettings)
		settings.GET("/:section/:key", c.Settings.GetSetting)
		settings.POST("/:section/:key/generate", c.Settings.GenerateSettingToken)
	}
}

func registerDependencyRoutes(g *gin.RouterGroup, c *Controllers) {
	deps := g.Group("/deps")
	{
		deps.GET("", c.Dependency.List)
		deps.POST("", c.Dependency.Create)
		deps.DELETE("/:id", c.Dependency.Delete)
		deps.POST("/install", c.Dependency.Install)
		deps.POST("/install-cmd", c.Dependency.GetInstallCommand)
		deps.POST("/uninstall/:id", c.Dependency.Uninstall)
		deps.POST("/reinstall/:id", c.Dependency.Reinstall)
		deps.POST("/reinstall-all", c.Dependency.ReinstallAll)
		deps.POST("/reinstall-all-cmd", c.Dependency.GetReinstallAllCommand)
		deps.POST("/batch-install-cmd", c.Dependency.GetBatchInstallCommand)
		deps.POST("/import", c.Dependency.ParseAndImport)
		deps.GET("/installed", c.Dependency.GetInstalled)
		deps.GET("/install-suggest-cmd", c.Dependency.GetDepInstallCommand)
	}
}

func registerAgentRoutes(g *gin.RouterGroup, c *Controllers) {
	agents := g.Group("/agents")
	{
		agents.GET("", c.Agent.List)
		agents.GET("/version", c.Agent.GetVersion)
		agents.PUT("/:id", c.Agent.Update)
		agents.DELETE("/:id", c.Agent.Delete)
		agents.POST("/:id/token", c.Agent.RegenerateToken)
		agents.POST("/:id/update", c.Agent.ForceUpdate)
		// 令牌管理
		agents.GET("/tokens", c.Agent.ListTokens)
		agents.POST("/tokens", c.Agent.CreateToken)
		agents.DELETE("/tokens/:id", c.Agent.DeleteToken)
	}

	// Agent API（供前端调用，保持在 v1 下）
	agentAPIv1 := g.Group("/agent")
	{
		agentAPIv1.GET("/download", c.Agent.Download)
	}
}

func registerMiseRoutes(g *gin.RouterGroup, c *Controllers) {
	mise := g.Group("/mise")
	{
		mise.GET("/ls", c.Mise.List)
		mise.POST("/sync", c.Mise.Sync)
		mise.GET("/plugins", c.Mise.Plugins)
		mise.GET("/versions", c.Mise.Versions)
		mise.GET("/verify-cmd", c.Mise.VerifyCommand)
		mise.POST("/use-global", c.Mise.UseGlobal)
		mise.POST("/unset-global", c.Mise.UnsetGlobal)
		mise.GET("/envs", c.Mise.Envs)
		mise.POST("/envs", c.Mise.SetEnv)
		mise.DELETE("/envs", c.Mise.UnsetEnv)
	}
}

func registerNotificationRoutes(g *gin.RouterGroup, c *Controllers) {
	notify := g.Group("/notify")
	{
		notify.GET("/types", c.Notification.GetChannelTypes)
		notify.GET("/channels", c.Notification.GetChannels)
		notify.POST("/channels", c.Notification.SaveChannel)
		notify.DELETE("/channels/:id", c.Notification.DeleteChannel)
		notify.POST("/channels/test", c.Notification.TestChannel)
		notify.GET("/bindings", c.Notification.GetBindings)
		notify.POST("/bindings", c.Notification.SaveBinding)
		notify.POST("/bindings/batch", c.Notification.BatchSaveBindings)
		notify.DELETE("/bindings/:id", c.Notification.DeleteBinding)
	}
}

func registerAppLogRoutes(g *gin.RouterGroup, c *Controllers) {
	appLogs := g.Group("/app-logs")
	{
		appLogs.GET("", c.AppLog.GetLogs)
		appLogs.POST("/read", c.AppLog.MarkAsRead)
		appLogs.POST("/clear", c.AppLog.ClearLogs)
	}
}

func registerSystemWSRoutes(g *gin.RouterGroup, c *Controllers) {
	g.GET("/ws/events", c.SystemWS.HandleEvents)
}

func registerMonitorRoutes(g *gin.RouterGroup, c *Controllers) {
	monitor := g.Group("/monitor")
	{
		monitor.GET("", c.Monitor.GetSystemMonitor)
		monitor.GET("/sse", c.Monitor.MonitorSSE)
	}
}

func initAgentAPIRoutes(root *gin.RouterGroup, c *Controllers) {
	// Agent API（供远程 Agent 调用，不使用 /v1 版本号）
	agentAPI := root.Group("/api/agent")
	{
		agentAPI.POST("/heartbeat", c.Agent.Heartbeat)
		agentAPI.GET("/tasks", c.Agent.GetTasks)
		agentAPI.POST("/report", c.Agent.ReportResult)
		agentAPI.GET("/download", c.Agent.Download) // 也在这里注册，兼容 Agent 调用
		agentAPI.GET("/ws", c.Agent.WSConnect)      // WebSocket 连接
	}
}

func registerWebUIRoutes(g *gin.RouterGroup, c *Controllers) {
	webuiGroup := g.Group("/webui")
	{
		webuiGroup.GET("", c.WebUI.GetWebUIs)
		webuiGroup.POST("/upload", c.WebUI.UploadWebUI)
		webuiGroup.PUT("/active", c.WebUI.SetActiveWebUI)
		webuiGroup.DELETE("/:name", c.WebUI.DeleteWebUI)
	}
}

func registerInterconnectRoutes(g *gin.RouterGroup, c *Controllers) {
	interconnect := g.Group("/interconnect")
	{
		interconnect.GET("/nodes", c.Interconnect.GetNodes)
		interconnect.POST("/nodes", c.Interconnect.CreateNode)
		interconnect.PUT("/nodes/:id", c.Interconnect.UpdateNode)
		interconnect.DELETE("/nodes/:id", c.Interconnect.DeleteNode)
		interconnect.GET("/nodes/:id/status", c.Interconnect.GetNodeStatus)
		interconnect.POST("/sync/script", c.Interconnect.SyncScript)
		interconnect.POST("/sync/env", c.Interconnect.SyncEnv)
		interconnect.POST("/sync/task", c.Interconnect.SyncTask)
		
		interconnect.GET("/child/status", c.Interconnect.GetChildStatus)
		
		// 代理模式 (面板穿越)
		interconnect.Any("/proxy/:node_id/*path", c.Interconnect.ProxyRequest)
	}
}

func registerSystemRoutes(g *gin.RouterGroup, c *Controllers) {
	systemAPI := g.Group("/system")
	{
		systemAPI.POST("/export", c.Data.ExportBusinessData)
		systemAPI.POST("/import", c.Data.ImportBusinessData)
	}
}
