package middleware

import (
	"fmt"
	"time"

	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

// GinLogger 返回使用 logrus 的 Gin 日志中间件
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		// 格式化延迟时间
		var latencyStr string
		if latency < time.Millisecond {
			latencyStr = fmt.Sprintf("%dµs", latency.Microseconds())
		} else if latency < time.Second {
			latencyStr = fmt.Sprintf("%dms", latency.Milliseconds())
		} else {
			latencyStr = fmt.Sprintf("%.2fs", latency.Seconds())
		}

		msg := fmt.Sprintf("[HTTP] %d %s %s %s [%s]", status, method, path, latencyStr, clientIP)

		if status >= 500 {
			logger.Error(msg)
		} else if status >= 400 {
			logger.Warn(msg)
		} else {
			logger.Info(msg)
		}
	}
}

// GinRecovery 返回使用 logrus 的 Gin 恢复中间件
func GinRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("[HTTP] Panic: %v | %s", err, c.Request.URL.Path)
				// 返回真实 500 + JSON 信封，避免空 body 导致前端 res.json() 抛异常
				utils.ServerError(c, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}
