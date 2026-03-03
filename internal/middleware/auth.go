package middleware

import (
	"encoding/json"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthRequired 认证中间件
func AuthRequired() gin.HandlerFunc {
	settingsSvc := services.NewSettingsService()
	return func(c *gin.Context) {
		// 校验 API Token (实验特性)
		if checkApiToken(c, settingsSvc) {
			return
		}

		token, err := c.Cookie(constant.CookieName)
		if err != nil || token == "" {
			utils.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}

		// 验证 token
		userID, username, err := utils.ParseToken(token, constant.Secret)
		if err != nil {
			utils.Unauthorized(c, "登录已过期，请重新登录")
			c.Abort()
			return
		}

		// 安全增强：校验数据库中该用户的 ID 是否与 Token 一致
		// 防止迁移后旧 Token 中的数字 ID 污染新数据
		var user models.User
		if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil || user.ID != userID {
			utils.Unauthorized(c, "会话失效，请重新登录")
			ClearAuthCookie(c)
			c.Abort()
			return
		}

		// 将用户信息存入上下文 (必须使用数据库中的最新 ID)
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Next()
	}
}

// checkApiToken 校验 API Token (实验特性，后续可能移除或重构)
// 返回 true 表示校验通过并已放行请求
func checkApiToken(c *gin.Context, settingsSvc *services.SettingsService) bool {
	apiToken := c.GetHeader("X-API-Token")
	if apiToken == "" {
		return false
	}

	siteConfig := settingsSvc.GetSection(constant.SectionSite)
	tokenJson, ok := siteConfig[constant.KeyApiToken]
	if !ok || tokenJson == "" {
		return false
	}

	var tokenData map[string]string
	if err := json.Unmarshal([]byte(tokenJson), &tokenData); err != nil {
		return false
	}

	expectedToken, ok := tokenData["token"]
	if !ok || expectedToken == "" || apiToken != expectedToken {
		return false
	}

	// 检查过期时间
	if expireStr, ok := tokenData["expire_at"]; ok && expireStr != "" {
		// 前端传来的时间格式是 YYYY-MM-DD，使用 2006-01-02 解析
		expireDate, err := time.Parse("2006-01-02", expireStr)
		if err == nil {
			// 将过期时间设为当天的 23:59:59
			expireDate = expireDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			if time.Now().After(expireDate) {
				return false
			}
		}
	}

	// 模拟 Admin 角色，必须通过实际存在的 admin 用户 ID 来关联
	var adminUser models.User
	if err := database.DB.Where("role = ?", "admin").First(&adminUser).Error; err != nil {
		utils.Unauthorized(c, "未找到管理员账户，API Token 校验失败")
		c.Abort()
		return true // 返回 true 表示中间件已处理并截断了请求
	}
	
	c.Set("userID", adminUser.ID)
	c.Set("username", adminUser.Username)
	c.Next()
	return true
}

// SetAuthCookie 设置认证 Cookie，expireDays 为过期天数
func SetAuthCookie(c *gin.Context, token string, expireDays int) {
	maxAge := 86400 * expireDays
	c.SetCookie(constant.CookieName, token, maxAge, "/", "", false, true)
}

// ClearAuthCookie 清除认证 Cookie
func ClearAuthCookie(c *gin.Context) {
	c.SetCookie(constant.CookieName, "", -1, "/", "", false, true)
}
