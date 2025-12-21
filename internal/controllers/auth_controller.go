package controllers

import (
	"strconv"

	"baihu/internal/constant"
	"baihu/internal/middleware"
	"baihu/internal/services"
	"baihu/internal/utils"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userService     *services.UserService
	settingsService *services.SettingsService
	loginLogService *services.LoginLogService
}

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

	user := ac.userService.GetUserByUsername(req.Username)
	if user == nil || !ac.userService.ValidatePassword(user, req.Password) {
		// 记录登录失败日志
		ac.loginLogService.Create(req.Username, ip, userAgent, "failed", "用户名或密码错误")
		utils.Unauthorized(c, "用户名或密码错误")
		return
	}

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
		ac.loginLogService.Create(req.Username, ip, userAgent, "failed", "Token生成失败")
		utils.ServerError(c, "登录失败")
		return
	}

	// 设置 Cookie
	middleware.SetAuthCookie(c, token, expireDays)

	// 记录登录成功日志
	ac.loginLogService.Create(req.Username, ip, userAgent, "success", "登录成功")

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
	utils.Success(c, user)
}
