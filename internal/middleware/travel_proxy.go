package middleware

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/tunnel"
	"github.com/gin-gonic/gin"
)

var travelProxyClient = &http.Client{
	Timeout: 10 * time.Second,
}

func TravelProxyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 检查是否存在 active_interconnect_node_id Cookie
		nodeID, err := c.Cookie(constant.CookieActiveInterconnectNodeID)
		if err != nil || nodeID == "" {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		// 1.5 只拦截 API 请求，静态资源文件等必须从主节点本地加载
		if !strings.HasPrefix(path, "/api/v1/") {
			c.Next()
			return
		}

		// 2. 白名单放行：反向隧道建立连接端点必须直达本机，不能二次代理
		if strings.HasPrefix(path, "/api/v1/interconnect/tunnel") {
			c.Next()
			return
		}

		// 3. 查询数据库中节点信息
		var node models.InterconnectNode
		if err := database.DB.Where("id = ?", nodeID).First(&node).Error; err != nil {
			// 节点不存在，说明 Cookie 无效，清除并放行
			c.SetCookie(constant.CookieActiveInterconnectNodeID, "", -1, "/", "", false, false)
			c.Next()
			return
		}

		// 4. 准备代理的路径（若主节点配置了 URLPrefix，需要剥离）
		cfg := services.GetConfig()
		urlPrefix := strings.TrimSuffix(cfg.Server.URLPrefix, "/")
		targetPath := path
		if urlPrefix != "" {
			targetPath = strings.TrimPrefix(targetPath, urlPrefix)
		}
		if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}

		// 5. 执行代理转发
		if strings.HasPrefix(node.URL, "tunnel://") {
			// 走 WebSocket 逆向 Yamux 隧道
			err := tunnel.ProxyHTTP(nodeID, c, targetPath)
			if err != nil {
				// 如果是网页 HTML 导航请求，提供友好降级返回主节点
				if strings.Contains(c.GetHeader("Accept"), "text/html") {
					c.SetCookie(constant.CookieActiveInterconnectNodeID, "", -1, "/", "", false, false)
					c.Header("Content-Type", "text/html; charset=utf-8")
					c.String(200, `<p>与子节点连接失败，正在自动返回主节点...</p><script>document.cookie="` + constant.CookieActiveInterconnectNodeID + `=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;"; window.location.href="/";</script>`)
					c.Abort()
					return
				}
				c.JSON(502, gin.H{"code": 502, "msg": "与子节点逆向隧道通信异常: " + err.Error()})
				c.Abort()
				return
			}
			c.Abort()
			return
		}

		// 走普通 HTTP 直连代理
		targetURL := strings.TrimRight(node.URL, "/") + targetPath
		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.JSON(500, gin.H{"code": 500, "msg": "Failed to create proxy request"})
			c.Abort()
			return
		}

		// 复制请求头
		req.Header = c.Request.Header.Clone()
		
		// 覆盖认证授权 Header 确保子节点鉴权通过
		if node.Token != "" {
			req.Header.Set("Authorization", "Bearer "+node.Token)
		}
		// 移除 Cookie 头部防干扰
		req.Header.Del("Cookie")
		req.Header.Set("X-Tunnel-Proxy", "true")

		resp, err := travelProxyClient.Do(req)
		if err != nil {
			// 如果是客户端自己主动取消了请求（例如连续刷新、关闭网页等），直接退出，不应视为子节点离线而执行退回主节点的操作
			if c.Request.Context().Err() == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				c.Abort()
				return
			}
			if strings.Contains(c.GetHeader("Accept"), "text/html") {
				c.SetCookie(constant.CookieActiveInterconnectNodeID, "", -1, "/", "", false, false)
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.String(200, `<p>与子节点连接失败，正在自动返回主节点...</p><script>document.cookie="` + constant.CookieActiveInterconnectNodeID + `=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;"; window.location.href="/";</script>`)
				c.Abort()
				return
			}
			c.JSON(502, gin.H{"code": 502, "msg": "无法连接至目标子节点: " + err.Error()})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		// 复制响应头
		for k, v := range resp.Header {
			for _, vv := range v {
				c.Writer.Header().Add(k, vv)
			}
		}
		c.Status(resp.StatusCode)
		io.Copy(c.Writer, resp.Body)
		c.Abort()
	}
}
