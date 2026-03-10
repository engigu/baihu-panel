package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthRequired 认证中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
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

// OpenapiRequired OpenAPI 认证中间件
func OpenapiRequired() gin.HandlerFunc {
	settingsSvc := services.NewSettingsService()
	return func(c *gin.Context) {
		if checkOpenapiToken(c, settingsSvc) {
			return
		}
		utils.Unauthorized(c, "无效的 OpenAPI 令牌")
		c.Abort()
	}
}

// checkOpenapiToken 校验 OpenAPI Token
// 返回 true 表示校验通过并已放行请求
func checkOpenapiToken(c *gin.Context, settingsSvc *services.SettingsService) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	// 提取 token：支持 "Bearer <token>" 和直接 "<token>" 两种格式
	var openapiToken string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		// 标准格式：Bearer <token>
		openapiToken = authHeader[7:]
	} else {
		// 直接使用 token
		openapiToken = authHeader
	}

	// Token 不能为空
	if openapiToken == "" {
		return false
	}

	siteConfig := settingsSvc.GetSection(constant.SectionSite)
	tokenJson, ok := siteConfig[constant.KeyOpenapiToken]
	if !ok || tokenJson == "" {
		return false
	}

	var tokenConfig vo.TokenConfig
	if err := json.Unmarshal([]byte(tokenJson), &tokenConfig); err != nil {
		return false
	}

	// 校验开启状态
	if !tokenConfig.Enabled {
		return false
	}

	if tokenConfig.Token == "" || openapiToken != tokenConfig.Token {
		return false
	}

	// 检查过期时间
	if tokenConfig.ExpireAt != "" {
		expireDate, err := time.Parse("2006-01-02", tokenConfig.ExpireAt)
		if err == nil {
			expireDate = expireDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			if time.Now().After(expireDate) {
				return false
			}
		}
	}

	// 模拟 Admin 角色
	var adminUser models.User
	if err := database.DB.Where("role = ?", "admin").First(&adminUser).Error; err != nil {
		utils.Unauthorized(c, "未找到管理员账户，OpenAPI Token 校验失败")
		c.Abort()
		return true
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

// SwaggerAuth Swagger 认证中间件 (Basic Auth)
func SwaggerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		settingsSvc := services.NewSettingsService()
		siteConfig := settingsSvc.GetSection(constant.SectionSite)
		tokenJson := siteConfig[constant.KeyOpenapiToken]

		if tokenJson == "" {
			c.Status(http.StatusNotFound)
			c.Abort()
			return
		}

		var tokenConfig vo.TokenConfig
		if err := json.Unmarshal([]byte(tokenJson), &tokenConfig); err != nil {
			c.Status(http.StatusNotFound)
			c.Abort()
			return
		}

		// 必须开启鉴权开关
		if !tokenConfig.Enabled {
			c.Status(http.StatusNotFound)
			c.Abort()
			return
		}

		// 检查过期时间
		if tokenConfig.ExpireAt != "" {
			expire, err := time.ParseInLocation("2006/01/02", tokenConfig.ExpireAt, time.Local)
			if err == nil {
				// 包含当天，所以设置到当天 23:59:59
				expire = expire.Add(24*time.Hour - time.Second)
				if time.Now().After(expire) {
					c.Status(http.StatusNotFound)
					c.Abort()
					return
				}
			}
		}

		// 获取请求中携带的凭证
		// 1. URL 参数 token
		// 2. Cookie 中的 openapi_token
		// 3. HTTP Basic Auth
		tokenQuery := c.Query("token")
		tokenCookie, _ := c.Cookie("openapi_token")
		_, password, hasAuth := c.Request.BasicAuth()

		var providedToken string
		if tokenQuery != "" {
			providedToken = tokenQuery
		} else if tokenCookie != "" {
			providedToken = tokenCookie
		} else if hasAuth {
			providedToken = password
		}

		// 检查提供的 token 是否匹配
		if providedToken != "" && providedToken == tokenConfig.Token {
			// 如果是通过 url 参数进来的，自动将其种入 Cookie，便于后续加载静态资源 (如 json)
			if tokenQuery != "" {
				c.SetCookie("openapi_token", providedToken, 86400, "/openapi", "", false, false)
			}
			c.Next()
			return
		}

		// 验证失败，不再返回 WWW-Authenticate 头触发浏览器反人类原生弹窗
		// 我们返回 401 的 JSON 或纯文本结构，以便由调用方自行接管鉴权逻辑
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "OpenAPI 访问未授权或 Token 错误",
		})
		c.Abort()
	}
}
