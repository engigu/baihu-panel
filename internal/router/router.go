package router

import (
	"io/fs"
	"net/http"
	"strings"

	"baihu/internal/controllers"
	"baihu/internal/middleware"
	"baihu/internal/services"
	"baihu/internal/static"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Task       *controllers.TaskController
	Auth       *controllers.AuthController
	Env        *controllers.EnvController
	Script     *controllers.ScriptController
	Executor   *controllers.ExecutorController
	File       *controllers.FileController
	Dashboard  *controllers.DashboardController
	Log        *controllers.LogController
	Terminal   *controllers.TerminalController
	Settings   *controllers.SettingsController
	Dependency *controllers.DependencyController
	Agent      *controllers.AgentController
}

func mustSubFS(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// cacheControl 返回设置 Cache-Control header 的中间件
func cacheControl(value string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", value)
		c.Next()
	}
}

func Setup(c *Controllers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.GinLogger(), middleware.GinRecovery())

	// 获取 URL 前缀
	cfg := services.GetConfig()
	urlPrefix := strings.TrimSuffix(cfg.Server.URLPrefix, "/")

	// 创建一个路由组，如果有前缀则使用前缀，否则使用根路径
	var root *gin.RouterGroup
	if urlPrefix != "" {
		root = router.Group(urlPrefix)
	} else {
		root = router.Group("")
	}

	// Serve embedded Vue SPA static files with cache headers
	staticFS := static.GetFS()
	assetsGroup := root.Group("/assets")
	assetsGroup.Use(cacheControl("public, max-age=31536000, immutable")) // 1 year cache for hashed assets
	assetsGroup.StaticFS("/", http.FS(mustSubFS(staticFS, "assets")))

	// Serve logo.svg with short cache
	root.GET("/logo.svg", func(ctx *gin.Context) {
		data, err := static.ReadFile("logo.svg")
		if err != nil {
			ctx.Status(404)
			return
		}
		ctx.Header("Cache-Control", "public, max-age=86400") // 1 day
		ctx.Data(200, "image/svg+xml", data)
	})

	// API routes
	api := root.Group("/api/v1")
	{
		// Health check (无需认证)
		api.GET("/ping", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{"message": "pong"})
		})

		// Authentication routes (无需认证)
		auth := api.Group("/auth")
		{
			auth.POST("/login", c.Auth.Login)
			auth.POST("/logout", c.Auth.Logout)
			auth.POST("/register", c.Auth.Register)
		}

		// 公开的站点设置（无需认证）
		api.GET("/settings/public", c.Settings.GetPublicSiteSettings)

		// 需要认证的路由
		authorized := api.Group("")
		authorized.Use(middleware.AuthRequired())
		{
			// 获取当前用户
			authorized.GET("/auth/me", c.Auth.GetCurrentUser)

			// Dashboard stats
			authorized.GET("/stats", c.Dashboard.GetStats)
			authorized.GET("/sentence", c.Dashboard.GetSentence)
			authorized.GET("/sendstats", c.Dashboard.GetSendStats)
			authorized.GET("/taskstats", c.Dashboard.GetTaskStats)

			// Task routes
			tasks := authorized.Group("/tasks")
			{
				tasks.POST("", c.Task.CreateTask)
				tasks.GET("", c.Task.GetTasks)
				tasks.GET("/:id", c.Task.GetTask)
				tasks.PUT("/:id", c.Task.UpdateTask)
				tasks.DELETE("/:id", c.Task.DeleteTask)
			}

			// Task execution routes
			execution := authorized.Group("/execute")
			{
				execution.POST("/task/:id", c.Executor.ExecuteTask)
				execution.POST("/command", c.Executor.ExecuteCommand)
				execution.GET("/results", c.Executor.GetLastResults)
			}

			// Environment variable routes
			env := authorized.Group("/env")
			{
				env.POST("", c.Env.CreateEnvVar)
				env.GET("", c.Env.GetEnvVars)
				env.GET("/all", c.Env.GetAllEnvVars)
				env.GET("/:id", c.Env.GetEnvVar)
				env.PUT("/:id", c.Env.UpdateEnvVar)
				env.DELETE("/:id", c.Env.DeleteEnvVar)
			}

			// Script routes
			scripts := authorized.Group("/scripts")
			{
				scripts.POST("", c.Script.CreateScript)
				scripts.GET("", c.Script.GetScripts)
				scripts.GET("/:id", c.Script.GetScript)
				scripts.PUT("/:id", c.Script.UpdateScript)
				scripts.DELETE("/:id", c.Script.DeleteScript)
			}

			// File routes
			files := authorized.Group("/files")
			{
				files.GET("/tree", c.File.GetFileTree)
				files.GET("/content", c.File.GetFileContent)
				files.POST("/content", c.File.SaveFileContent)
				files.POST("/create", c.File.CreateFile)
				files.POST("/delete", c.File.DeleteFile)
				files.POST("/rename", c.File.RenameFile)
				files.POST("/upload", c.File.UploadArchive)
				files.POST("/uploadfiles", c.File.UploadFiles)
			}

			// Log routes
			logs := authorized.Group("/logs")
			{
				logs.GET("", c.Log.GetLogs)
				logs.GET("/:id", c.Log.GetLogDetail)
			}

			// Terminal routes
			authorized.GET("/terminal/ws", c.Terminal.HandleWebSocket)
			authorized.POST("/terminal/exec", c.Terminal.ExecuteShellCommand)

			// Settings routes
			settings := authorized.Group("/settings")
			{
				settings.POST("/password", c.Settings.ChangePassword)
				settings.GET("/site", c.Settings.GetSiteSettings)
				settings.PUT("/site", c.Settings.UpdateSiteSettings)
				settings.GET("/scheduler", c.Settings.GetSchedulerSettings)
				settings.PUT("/scheduler", c.Settings.UpdateSchedulerSettings)
				settings.GET("/about", c.Settings.GetAbout)
				settings.GET("/loginlogs", c.Settings.GetLoginLogs)
				settings.POST("/backup", c.Settings.CreateBackup)
				settings.GET("/backup/status", c.Settings.GetBackupStatus)
				settings.GET("/backup/download", c.Settings.DownloadBackup)
				settings.POST("/restore", c.Settings.RestoreBackup)
			}

			// Dependency routes (依赖管理)
			deps := authorized.Group("/deps")
			{
				deps.GET("", c.Dependency.List)
				deps.POST("", c.Dependency.Create)
				deps.DELETE("/:id", c.Dependency.Delete)
				deps.POST("/install", c.Dependency.Install)
				deps.POST("/uninstall/:id", c.Dependency.Uninstall)
				deps.POST("/reinstall/:id", c.Dependency.Reinstall)
				deps.POST("/reinstall-all", c.Dependency.ReinstallAll)
				deps.GET("/installed", c.Dependency.GetInstalled)
			}

			// Agent routes (Agent 管理)
			agents := authorized.Group("/agents")
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
		}

		// Agent API（供远程 Agent 调用）
		agentAPI := api.Group("/agent")
		{
			agentAPI.POST("/heartbeat", c.Agent.Heartbeat)
			agentAPI.GET("/tasks", c.Agent.GetTasks)
			agentAPI.POST("/report", c.Agent.ReportResult)
			agentAPI.GET("/download", c.Agent.Download)
			agentAPI.GET("/ws", c.Agent.WSConnect) // WebSocket 连接
		}
	}

	// SPA fallback - serve index.html (no cache for HTML)
	// 必须在最后注册，作为兜底路由
	router.NoRoute(func(ctx *gin.Context) {
		// 如果配置了前缀，只处理带前缀的路径
		if urlPrefix != "" && !strings.HasPrefix(ctx.Request.URL.Path, urlPrefix) {
			ctx.Status(404)
			return
		}
		
		data, err := static.ReadFile("index.html")
		if err != nil {
			ctx.String(500, "index.html not found")
			return
		}
		ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		ctx.Data(200, "text/html; charset=utf-8", data)
	})

	return router
}
