package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/engigu/baihu-panel/internal/static"

	"github.com/gin-gonic/gin"
)

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

func initStaticRoutes(root *gin.RouterGroup) {
	staticFS := static.GetFS()
	if staticFS == nil {
		return
	}

	// 静态资源服务（Vue SPA），带缓存头部
	assetsGroup := root.Group("/assets")
	assetsGroup.Use(cacheControl("public, max-age=31536000, immutable")) // 带哈希的资源缓存
	assetsGroup.StaticFS("/", http.FS(mustSubFS(staticFS, "assets")))

	// logo.svg 短缓存实现
	root.GET("/logo.svg", func(ctx *gin.Context) {
		data, err := static.ReadFile("logo.svg")
		if err != nil {
			ctx.Status(404)
			return
		}
		ctx.Header("Cache-Control", "public, max-age=86400") // 缓存1天
		ctx.Data(200, "image/svg+xml", data)
	})
}

// serveSPA 注入配置并返回 index.html 给前端渲染
func serveSPA(ctx *gin.Context, urlPrefix string, status int) {
	data, err := static.ReadFile("index.html")
	if err != nil {
		// 如果读不到 index.html (如 dev 模式未 build)，返回基础 HTML
		// 如果已经是 /404 路径，则不再重定向以免死循环
		path := ctx.Request.URL.Path
		if strings.HasSuffix(path, "/404") {
			ctx.Data(status, "text/html; charset=utf-8", []byte("<!DOCTYPE html><html><body><h1>404 Not Found</h1><p>Frontend assets not found. Please run 'npm run build' or check dev server.</p><a href='/'>Go Home</a></body></html>"))
			ctx.Abort()
			return
		}

		fallback := `<!DOCTYPE html><html><head><meta charset="utf-8"/><title>404 Not Found</title></head><body>
			<script>
				const baseUrl = window.__BASE_URL__ || "/";
				if (!window.location.pathname.endsWith("/404")) {
					window.location.href = baseUrl + (baseUrl.endsWith("/") ? "" : "/") + "404";
				}
			</script>
			<p>Not Found. Redirecting...</p>
			</body></html>`
		ctx.Header("Content-Type", "text/html; charset=utf-8")
		ctx.Data(status, "text/html", []byte(fallback))
		ctx.Abort()
		return
	}

	html := string(data)

	// 注入配置变量供前端使用（API 调用和路由）
	baseHref := urlPrefix + "/"
	if urlPrefix == "" {
		baseHref = "/"
	}

	// 注入 base tag 为了让深度路由的相对路径资源加载正常
	html = strings.Replace(html, "<head>", "<head>\n    <base href=\""+baseHref+"\">", 1)

	configScript := `<script>window.__BASE_URL__ = "` + urlPrefix + `"; window.__API_VERSION__ = "/api/v1";</script>`
	html = strings.Replace(html, "</head>", configScript+"</head>", 1)

	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	ctx.Data(status, "text/html; charset=utf-8", []byte(html))
	ctx.Abort()
}
