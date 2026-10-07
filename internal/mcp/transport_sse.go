package mcp

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/engigu/baihu-panel/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// MaxActiveSSESessions 限制同时保持的 SSE 客户端长连接会话数上限（每个连接占用 1~2 个协程）
	MaxActiveSSESessions = 3
)

var (
	activeSSESessions int32
)

// GetActiveSSESessions 获取当前活跃的 SSE 连接会话数
func GetActiveSSESessions() int32 {
	return atomic.LoadInt32(&activeSSESessions)
}

// RegisterOpenAPIMCPRoutes 在 Gin 路由组上注册 MCP SSE 与交互端点
func RegisterOpenAPIMCPRoutes(g *gin.RouterGroup, deps *Deps) {
	mcpServer := NewBaihuMCPServer(deps)

	cfg := services.GetConfig()
	prefix := ""
	if cfg != nil && cfg.Server.URLPrefix != "" {
		prefix = strings.TrimSuffix(cfg.Server.URLPrefix, "/")
	}

	sseEndpoint := prefix + "/open2api/v1/mcp/sse"
	msgEndpoint := prefix + "/open2api/v1/mcp/messages"

	sseServer := server.NewSSEServer(
		mcpServer,
		server.WithSSEEndpoint(sseEndpoint),
		server.WithMessageEndpoint(msgEndpoint),
		server.WithAppendQueryToMessageEndpoint(),
	)

	mcpGroup := g.Group("/mcp")
	{
		// 挂载连接数限流中间件，防止大量并发 SSE 长连接耗尽协程
		mcpGroup.GET("/sse", sseSessionLimiter(), gin.WrapH(sseServer.SSEHandler()))
		mcpGroup.POST("/messages", gin.WrapH(sseServer.MessageHandler()))
	}
}

// sseSessionLimiter 限制同时保持的 SSE 挂起连接协程数
func sseSessionLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		current := atomic.AddInt32(&activeSSESessions, 1)
		if current > MaxActiveSSESessions {
			atomic.AddInt32(&activeSSESessions, -1)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": http.StatusTooManyRequests,
				"msg":  fmt.Sprintf("当前活跃的 MCP SSE 连接数已达上限 (%d)，已拒绝新连接以保护系统协程资源", MaxActiveSSESessions),
			})
			c.Abort()
			return
		}

		defer func() {
			atomic.AddInt32(&activeSSESessions, -1)
		}()

		c.Next()
	}
}
