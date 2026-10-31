package middleware

import (
	"fmt"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var demoWSUpgrader = websocket.Upgrader{
	CheckOrigin: utils.CheckWSOrigin,
}

// DemoGuard 演示模式切面拦截中间件
func DemoGuard(customMsg ...string) gin.HandlerFunc {
	msg := "演示模式下禁止此操作"
	if len(customMsg) > 0 && customMsg[0] != "" {
		msg = customMsg[0]
	}

	return func(c *gin.Context) {
		if !constant.DemoMode {
			c.Next()
			return
		}

		// 1. WebSocket 请求拦截
		if c.IsWebsocket() {
			conn, err := demoWSUpgrader.Upgrade(c.Writer, c.Request, nil)
			if err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n\033[1;33m[演示模式] "+msg+"\033[0m\r\n"))
				_ = conn.Close()
			}
			c.Abort()
			return
		}

		// 2. SSE 流式请求拦截
		if c.Query("stream") == "true" || strings.Contains(c.GetHeader("Accept"), "text/event-stream") {
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("X-Accel-Buffering", "no")
			c.Writer.Flush()
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", msg)
			c.Writer.Flush()
			c.Abort()
			return
		}

		// 3. 常规 HTTP 请求拦截
		utils.BadRequest(c, msg)
		c.Abort()
	}
}
