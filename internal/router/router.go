package router

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/controllers"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/services"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Task         *controllers.TaskController
	Auth         *controllers.AuthController
	Env          *controllers.EnvController
	Script       *controllers.ScriptController
	Executor     *controllers.ExecutorController
	File         *controllers.FileController
	Dashboard    *controllers.DashboardController
	Log          *controllers.LogController
	LogWS        *controllers.LogWSController
	Terminal     *controllers.TerminalController
	Settings     *controllers.SettingsController
	Dependency   *controllers.DependencyController
	Agent        *controllers.AgentController
	Mise         *controllers.MiseController
	Notification *controllers.NotificationController
	AppLog       *controllers.AppLogController
}

func Setup(c *Controllers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.GinLogger(), middleware.GinRecovery())

	// 获取 URL 前缀
	cfg := services.GetConfig()
	urlPrefix := strings.TrimSuffix(cfg.Server.URLPrefix, "/")

	// 按需绑定 Pprof 调试路由
	if cfg.Server.PprofEnabled {
		pprof.Register(router)
	}

	// 创建一个路由组，如果有前缀则使用前缀，否则使用根路径
	var root *gin.RouterGroup
	if urlPrefix != "" {
		root = router.Group(urlPrefix)
	} else {
		root = router.Group("")
	}

	// =========================================================================
	// 路由分类组装 (对应 Nginx 的 location 块分发)
	// =========================================================================

	// 1. [ location /assets ] 静态资源路由
	initStaticRoutes(root)

	// 3. [ location /api ] 内部 API 路由组
	apiV1 := root.Group("/api/v1")
	initPublicAPIRoutes(apiV1, c)     // 公开接口 (无需认证)
	initAuthorizedAPIRoutes(apiV1, c) // 授权接口 (需 JWT)

	// 4. [ location /api/agent ] Agent 相关 API 路由组
	initAgentAPIRoutes(root, c)
	initOpenAPIV1Routes(root, c)

	// =========================================================================
	// [ location / ] 全局 404 兜底与 SPA 渲染
	// 对应 Nginx: try_files $uri $uri/ /index.html;
	// =========================================================================
	router.NoRoute(func(ctx *gin.Context) {
		path := ctx.Request.URL.Path

		// 如果配置了前缀，只处理带前缀的路径
		if urlPrefix != "" && !strings.HasPrefix(path, urlPrefix) {
			ctx.Status(404)
			return
		}

		// 解析实际的相对路径
		relPath := strings.TrimPrefix(path, urlPrefix)
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}

		// 拦截器：不该返回 index.html 的情况
		// 如果该请求被识别为 API 请求、静态资源请求，或者是带有明确文件后缀（如 .js / .css / .png）的物理文件请求
		// 都不应该返回 SPA 页面（会报前端 MIME 类型错误），而是直接掐断返回 404
		hasAnyExt := false
		if idx := strings.LastIndex(relPath, "."); idx > 0 && len(relPath)-idx < 6 {
			// 简单判断是否有后缀（如 .js, .css）
			hasAnyExt = true
		}

		if strings.HasPrefix(relPath, "/api/") || strings.HasPrefix(relPath, "/assets/") || hasAnyExt {
			ctx.String(404, "404 Not Found")
			return
		}

		// 其他所有有效的前端页面路径（如 /tasks, /settings），都返回 index.html 交给 vue-router 处理
		serveSPA(ctx, urlPrefix, 200)
	})

	return router
}
