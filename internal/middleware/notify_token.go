package middleware

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

// NotifyTokenAuth 通知 Token 认证中间件
func NotifyTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("notify-token")
		if token == "" {
			utils.Unauthorized(c, "缺少通知 Token")
			c.Abort()
			return
		}

		// 从 settings 表读取配置的通知 Token
		settingsService := services.NewSettingsService()
		savedToken := settingsService.Get(constant.SectionNotify, constant.KeyNotifyToken)

		if savedToken == "" {
			utils.Unauthorized(c, "通知 Token 未配置")
			c.Abort()
			return
		}

		if strings.ToLower(token) != strings.ToLower(savedToken) {
			utils.Unauthorized(c, "通知 Token 无效")
			c.Abort()
			return
		}

		c.Next()
	}
}
