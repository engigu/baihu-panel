package router

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/openapi_docs"

	"github.com/gin-gonic/gin"
)

func initOpenAPIRoutes(root *gin.RouterGroup, urlPrefix string) {
	// OpenAPI documentation using Scalar UI (带 Basic Auth 认证)
	root.GET("/openapi/*any", func(c *gin.Context) {
		// 禁用整个 OpenAPI 路由的缓存
		c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		settingsSvc := services.NewSettingsService()
		siteConfig := settingsSvc.GetSection(constant.SectionSite)
		tokenJson := siteConfig[constant.KeyOpenapiToken]

		enabled := false
		if tokenJson != "" {
			var tokenConfig vo.TokenConfig
			if err := json.Unmarshal([]byte(tokenJson), &tokenConfig); err == nil {
				enabled = tokenConfig.Enabled
			}
		}

		// 如果未开启文档，直接返回 404 SPA 页面
		if !enabled {
			serveSPA(c, urlPrefix, 404)
			return
		}

		// 执行认证
		middleware.SwaggerAuth()(c)
		if c.IsAborted() {
			// 如果认证失败（且被中间件置为 404，如密码错误且我们想要隐藏它）
			if c.Writer.Status() == http.StatusNotFound {
				serveSPA(c, urlPrefix, 404)
			}
			return
		}

		// 获取内部路径并标准化（移除前后的所有斜杠）
		path := strings.Trim(c.Param("any"), "/")

		// 1. 根路径或空路径 -> 重定向到 index.html
		if path == "" {
			c.Redirect(http.StatusMovedPermanently, c.Request.URL.Path+"index.html")
			return
		}

		// 2. 提供 Scalar 渲染的 HTML 页面
		if path == "index.html" {
			scalarHTML := `<!doctype html>
<html>
  <head>
    <title>Baihu Panel API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script id="api-reference" data-url="` + urlPrefix + `/openapi/doc.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Status(http.StatusOK)
			c.Writer.Write([]byte(scalarHTML))
			c.Abort()
			return
		}

		// 3. 提供给 Scalar/Swagger 使用 host 环境变量后的 doc.json 内容
		if path == "doc.json" {
			// 动态设置当前请求的域名作为 Swagger Host
			openapi_docs.SwaggerInfo.Host = c.Request.Host
			scheme := "http"
			if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
				scheme = "https"
			}
			openapi_docs.SwaggerInfo.Schemes = []string{scheme}

			doc := openapi_docs.SwaggerInfo.ReadDoc()
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.String(http.StatusOK, doc)
			return
		}

		// 其他未匹配路径 -> 返回 404 SPA 页面
		serveSPA(c, urlPrefix, 404)
	})
}
