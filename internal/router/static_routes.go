package router

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/engigu/baihu-panel/internal/static"

	"github.com/gin-gonic/gin"
)

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

	// 专门处理 /assets 目录下的资源
	root.GET("/assets/*filepath", cacheControl("public, max-age=31536000, immutable"), func(ctx *gin.Context) {
		// 获取相对路径，例如 "assets/chunk-123.js"
		fullPath := "assets" + ctx.Param("filepath")
		fullPath = strings.TrimPrefix(fullPath, "/")

		// 1. 检查浏览器是否支持 gzip
		isGzipSupported := strings.Contains(ctx.GetHeader("Accept-Encoding"), "gzip")

		// 2. 构造压缩路径
		gzPath := fullPath + ".gz"

		// 3. 确定 MIME 类型 (优先硬编码常用类型，防止 Windows 注册表错误)
		ext := filepath.Ext(fullPath)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			switch ext {
			case ".js":
				contentType = "application/javascript"
			case ".css":
				contentType = "text/css"
			case ".svg":
				contentType = "image/svg+xml"
			case ".json":
				contentType = "application/json"
			case ".wasm":
				contentType = "application/wasm"
			default:
				contentType = "application/octet-stream"
			}
		}

		// 4. 发送逻辑
		// 优先尝试发送 .gz 版本
		if gzData, err := fs.ReadFile(staticFS, gzPath); err == nil {
			ctx.Header("Content-Type", contentType)
			if isGzipSupported {
				ctx.Header("Content-Encoding", "gzip")
				ctx.Data(http.StatusOK, contentType, gzData)
			} else {
				// 客户端不支持 Gzip，解压后返回
				gr, _ := gzip.NewReader(bytes.NewReader(gzData))
				defer gr.Close()
				ctx.Status(http.StatusOK)
				io.Copy(ctx.Writer, gr)
			}
			return
		}

		// 如果没有 .gz，尝试发送原文件
		if data, err := fs.ReadFile(staticFS, fullPath); err == nil {
			ctx.Data(http.StatusOK, contentType, data)
			return
		}

		// 都没找到
		ctx.Status(http.StatusNotFound)
	})

	// logo.svg 处理
	root.GET("/logo.svg", func(ctx *gin.Context) {
		serveSingleFile(ctx, "logo.svg", "image/svg+xml", "public, max-age=86400")
	})
}

func serveSingleFile(ctx *gin.Context, filename string, contentType string, cache string) {
	staticFS := static.GetFS()
	if staticFS == nil {
		ctx.Status(404)
		return
	}

	if cache != "" {
		ctx.Header("Cache-Control", cache)
	}

	isGzipSupported := strings.Contains(ctx.GetHeader("Accept-Encoding"), "gzip")

	// 尝试压缩版
	if gzData, err := fs.ReadFile(staticFS, filename+".gz"); err == nil {
		ctx.Header("Content-Type", contentType)
		if isGzipSupported {
			ctx.Header("Content-Encoding", "gzip")
			ctx.Data(http.StatusOK, contentType, gzData)
		} else {
			gr, _ := gzip.NewReader(bytes.NewReader(gzData))
			defer gr.Close()
			ctx.Status(http.StatusOK)
			io.Copy(ctx.Writer, gr)
		}
		return
	}

	// 尝试原版
	if data, err := fs.ReadFile(staticFS, filename); err == nil {
		ctx.Data(http.StatusOK, contentType, data)
		return
	}

	ctx.Status(404)
}

// serveSPA 注入配置并返回 index.html 给前端渲染
func serveSPA(ctx *gin.Context, urlPrefix string, status int) {
	staticFS := static.GetFS()
	if staticFS == nil {
		serveFallback(ctx, urlPrefix, status)
		return
	}

	var data []byte
	// 尝试读取并解压为字符串以便注入配置
	if gzData, err := fs.ReadFile(staticFS, "index.html.gz"); err == nil {
		gr, _ := gzip.NewReader(bytes.NewReader(gzData))
		data, _ = io.ReadAll(gr)
		gr.Close()
	} else if rawData, err := fs.ReadFile(staticFS, "index.html"); err == nil {
		data = rawData
	}

	if data == nil {
		serveFallback(ctx, urlPrefix, status)
		return
	}

	html := string(data)
	baseHref := urlPrefix + "/"
	if urlPrefix == "" { baseHref = "/" }
	
	// 注入 Base 和 Config
	html = strings.Replace(html, "<head>", "<head>\n    <base href=\""+baseHref+"\">", 1)
	configScript := `<script>window.__BASE_URL__ = "` + urlPrefix + `"; window.__API_VERSION__ = "/api/v1";</script>`
	html = strings.Replace(html, "</head>", configScript+"</head>", 1)

	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	ctx.Data(status, "text/html; charset=utf-8", []byte(html))
}

func serveFallback(ctx *gin.Context, urlPrefix string, status int) {
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	ctx.String(status, "Frontend assets not found. Please run 'npm run build'.")
}
