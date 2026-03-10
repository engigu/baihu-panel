package router

import (
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/gin-gonic/gin"
)

// initOpenAPIV1Routes 初始化 OpenAPI v1 路由
// 只注册有 @Tags OpenAPI 注释的接口
func initOpenAPIV1Routes(root *gin.RouterGroup, c *Controllers) {
	// OpenAPI v1 路由组 (使用 Bearer Token)
	open := root.Group("/open2api/v1")
	open.Use(middleware.OpenapiRequired())
	{
		// 任务相关接口
		registerOpenAPITaskRoutes(open, c)
		// 环境变量相关接口
		registerOpenAPIEnvRoutes(open, c)
		// 日志相关接口
		registerOpenAPILogRoutes(open, c)
		// 任务执行相关接口
		registerOpenAPIExecutorRoutes(open, c)
	}
}

// registerOpenAPITaskRoutes 注册 OpenAPI 任务路由（只包含有 @Tags OpenAPI 注释的接口）
func registerOpenAPITaskRoutes(g *gin.RouterGroup, c *Controllers) {
	tasks := g.Group("/tasks")
	{
		tasks.GET("", c.Task.GetTasks)
		tasks.GET("/:id", c.Task.GetTask)
		tasks.PUT("/:id", c.Task.UpdateTask)
		tasks.DELETE("/:id", c.Task.DeleteTask)
		tasks.POST("/stop/:logID", c.Task.StopTask)
	}
}

// registerOpenAPIEnvRoutes 注册 OpenAPI 环境变量路由（只包含有 @Tags OpenAPI 注释的接口）
func registerOpenAPIEnvRoutes(g *gin.RouterGroup, c *Controllers) {
	env := g.Group("/env")
	{
		env.GET("", c.Env.GetEnvVars)
		env.GET("/all", c.Env.GetAllEnvVars)
		env.GET("/:id", c.Env.GetEnvVar)
		env.GET("/:id/tasks", c.Env.GetAssociatedTasks)
	}
}

// registerOpenAPILogRoutes 注册 OpenAPI 日志路由（只包含有 @Tags OpenAPI 注释的接口）
func registerOpenAPILogRoutes(g *gin.RouterGroup, c *Controllers) {
	logs := g.Group("/logs")
	{
		logs.GET("", c.Log.GetLogs)
		logs.GET("/:id", c.Log.GetLogDetail)
	}
}

// registerOpenAPIExecutorRoutes 注册 OpenAPI 任务执行路由（只包含有 @Tags OpenAPI 注释的接口）
func registerOpenAPIExecutorRoutes(g *gin.RouterGroup, c *Controllers) {
	execution := g.Group("/execute")
	{
		execution.POST("/task/:id", c.Executor.ExecuteTask)
		execution.GET("/results", c.Executor.GetLastResults)
	}
}
