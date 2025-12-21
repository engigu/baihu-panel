package router

import (
	"io/fs"
	"net/http"

	"baihu/internal/controllers"
	"baihu/internal/middleware"
	"baihu/internal/static"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Task      *controllers.TaskController
	Auth      *controllers.AuthController
	Env       *controllers.EnvController
	Script    *controllers.ScriptController
	Executor  *controllers.ExecutorController
	File      *controllers.FileController
	Dashboard *controllers.DashboardController
	Log       *controllers.LogController
	Terminal  *controllers.TerminalController
	Settings  *controllers.SettingsController
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

	// Serve embedded Vue SPA static files with cache headers
	staticFS := static.GetFS()
	assetsGroup := router.Group("/assets")
	assetsGroup.Use(cacheControl("public, max-age=31536000, immutable")) // 1 year cache for hashed assets
	assetsGroup.StaticFS("/", http.FS(mustSubFS(staticFS, "assets")))

	// Serve logo.svg with short cache
	router.GET("/logo.svg", func(ctx *gin.Context) {
		data, err := static.ReadFile("logo.svg")
		if err != nil {
			ctx.Status(404)
			return
		}
		ctx.Header("Cache-Control", "public, max-age=86400") // 1 day
		ctx.Data(200, "image/svg+xml", data)
	})

	// SPA fallback - serve index.html (no cache for HTML)
	router.NoRoute(func(ctx *gin.Context) {
		data, err := static.ReadFile("index.html")
		if err != nil {
			ctx.String(500, "index.html not found")
			return
		}
		ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		ctx.Data(200, "text/html; charset=utf-8", data)
	})

	// API routes
	api := router.Group("/api")
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
				files.POST("/upload-files", c.File.UploadFiles)
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
				settings.POST("/clean-logs", c.Settings.CleanLogs)
				settings.GET("/site", c.Settings.GetSiteSettings)
				settings.PUT("/site", c.Settings.UpdateSiteSettings)
				settings.GET("/about", c.Settings.GetAbout)
				settings.GET("/login-logs", c.Settings.GetLoginLogs)
			}
		}
	}

	return router
}
