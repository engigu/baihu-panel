package controllers

import (

	"strconv"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userService     *services.UserService
	settingsService *services.SettingsService
	loginLogService *services.LoginLogService
}

type loginAttempt struct {
	Count       int
	LastAttempt time.Time
}

var loginAttempts sync.Map

func NewAuthController(userService *services.UserService, settingsService *services.SettingsService, loginLogService *services.LoginLogService) *AuthController {
	return &AuthController{
		userService:     userService,
		settingsService: settingsService,
		loginLogService: loginLogService,
	}
}

func (ac *AuthController) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 暴力破解防御
	if val, ok := loginAttempts.Load(ip); ok {
		attempt := val.(*loginAttempt)
		if attempt.Count >= 5 && time.Since(attempt.LastAttempt) < time.Minute {
			eventbus.DefaultBus.Publish(eventbus.Event{
				Type: constant.EventBruteForceLogin,
				Payload: map[string]interface{}{
					"ip":        ip,
					"username":  req.Username,
					"userAgent": userAgent,
				},
			})
			utils.TooManyRequests(c, "尝试次数过多，请一分钟后再试")
			return
		}
		// 如果距离上次尝试已超过一分钟，重置计数
		if time.Since(attempt.LastAttempt) >= time.Minute {
			loginAttempts.Delete(ip)
		}
	}

	user := ac.userService.GetUserByUsername(req.Username)
	if user == nil || !ac.userService.ValidatePassword(user, req.Password) {
		// 记录失败尝试
		val, _ := loginAttempts.LoadOrStore(ip, &loginAttempt{Count: 0, LastAttempt: time.Now()})
		attempt := val.(*loginAttempt)
		attempt.Count++
		attempt.LastAttempt = time.Now()

		// 记录登录失败日志
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type: constant.EventUserLogin,
			Payload: map[string]interface{}{
				"ip":        ip,
				"username":  req.Username,
				"userAgent": userAgent,
				"status":    "failed",
				"message":   "用户名或密码错误",
			},
		})
		utils.Unauthorized(c, "用户名或密码错误")
		return
	}

	// 登录成功，清除尝试记录
	loginAttempts.Delete(ip)

	// 获取 cookie 过期天数
	expireDays := 7
	if days := ac.settingsService.Get(constant.SectionSite, constant.KeyCookieDays); days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			expireDays = d
		}
	}

	// 生成 token
	token, err := utils.GenerateToken(user.ID, user.Username, expireDays, constant.Secret)
	if err != nil {
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type: constant.EventUserLogin,
			Payload: map[string]interface{}{
				"ip":        ip,
				"username":  req.Username,
				"userAgent": userAgent,
				"status":    "failed",
				"message":   "Token生成失败",
			},
		})
		utils.ServerError(c, "登录失败")
		return
	}

	// 设置 Cookie
	middleware.SetAuthCookie(c, token, expireDays)

	// 记录登录成功日志
	eventbus.DefaultBus.Publish(eventbus.Event{
		Type: constant.EventUserLogin,
		Payload: map[string]interface{}{
			"ip":        ip,
			"username":  req.Username,
			"userAgent": userAgent,
			"status":    "success",
			"message":   "登录成功",
		},
	})

	utils.Success(c, gin.H{
		"user": user.Username,
	})
}

func (ac *AuthController) Logout(c *gin.Context) {
	middleware.ClearAuthCookie(c)
	utils.SuccessMsg(c, "退出成功")
}

func (ac *AuthController) GetCurrentUser(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		utils.Unauthorized(c, "未登录")
		return
	}
	utils.Success(c, gin.H{
		"username": username,
	})
}

func (ac *AuthController) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	user := ac.userService.CreateUser(req.Username, req.Email, req.Password, "user")
	utils.Success(c, vo.ToUserVO(user))
}
