package controllers

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
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

func init() {
	// 定期清理过期的登录尝试统计，防止内存溢出
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			loginAttempts.Range(func(key, value any) bool {
				attempt := value.(*loginAttempt)
				if time.Since(attempt.LastAttempt) > 10*time.Minute {
					loginAttempts.Delete(key)
				}
				return true
			})
		}
	}()
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

	// 检查是否开启了两步验证
	if user.OtpEnabled {
		// 生成临时待验证 OTP 的 token，有效期 5 分钟
		pendingToken, err := utils.GenerateOtpPendingToken(user.ID, constant.Secret)
		if err != nil {
			utils.ServerError(c, "生成临时凭证失败")
			return
		}
		utils.Success(c, gin.H{
			"require_otp":       true,
			"otp_pending_token": pendingToken,
		})
		return
	}

	expireDays := 7
	if days := ac.settingsService.Get(constant.SectionSite, constant.KeyCookieDays); days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			expireDays = d
		}
	}

	// 生成 token
	token, err := utils.GenerateToken(user.ID, user.Username, user.TokenVersion, expireDays, constant.Secret)
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
	if userID, exists := c.Get("userID"); exists {
		ac.userService.InvalidateUserTokens(userID.(string))
	}
	middleware.ClearAuthCookie(c)
	utils.SuccessMsg(c, "退出成功")
}

func (ac *AuthController) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := ac.userService.GetUserByID(userID)
	if err != nil {
		utils.Unauthorized(c, "会话无效")
		return
	}
	utils.Success(c, gin.H{
		"username": user.Username,
		"role":     user.Role,
	})
}

func (ac *AuthController) Register(c *gin.Context) {
	/*
		var req struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			utils.BadRequest(c, err.Error())
			return
		}

		// 安全性：强制设定角色为 user，防止注册时篡改角色为 admin
		user := ac.userService.CreateUser(req.Username, req.Password, req.Email, constant.DefaultRole)
		utils.Success(c, vo.ToUserVO(user))
	*/
	utils.BadRequest(c, "注册功能已关闭")
}

func (ac *AuthController) VerifyOTP(c *gin.Context) {
	var req struct {
		OtpPendingToken string `json:"otp_pending_token" binding:"required"`
		Code            string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	userID, err := utils.ParseOtpPendingToken(req.OtpPendingToken, constant.Secret)
	if err != nil {
		utils.Unauthorized(c, "临时凭证无效或已过期")
		return
	}

	user, err := ac.userService.GetUserByID(userID)
	if err != nil || user == nil {
		utils.Unauthorized(c, "用户不存在")
		return
	}

	if !user.OtpEnabled || user.OtpSecret == "" {
		utils.BadRequest(c, "未开启两步验证")
		return
	}

	// 验证 OTP 验证码
	if !totp.Validate(req.Code, user.OtpSecret) {
		utils.Unauthorized(c, "验证码错误")
		return
	}

	// 校验通过，生成正式 Token 并登录
	expireDays := 7
	if days := ac.settingsService.Get(constant.SectionSite, constant.KeyCookieDays); days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			expireDays = d
		}
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.TokenVersion, expireDays, constant.Secret)
	if err != nil {
		utils.ServerError(c, "登录失败")
		return
	}

	middleware.SetAuthCookie(c, token, expireDays)

	// 记录登录成功日志
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	eventbus.DefaultBus.Publish(eventbus.Event{
		Type: constant.EventUserLogin,
		Payload: map[string]interface{}{
			"ip":        ip,
			"username":  user.Username,
			"userAgent": userAgent,
			"status":    "success",
			"message":   "两步验证登录成功",
		},
	})

	utils.Success(c, gin.H{
		"user": user.Username,
	})
}

func (ac *AuthController) GetOTPStatus(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := ac.userService.GetUserByID(userID)
	if err != nil {
		utils.Unauthorized(c, "用户不存在")
		return
	}
	utils.Success(c, gin.H{
		"otp_enabled": user.OtpEnabled,
	})
}

func (ac *AuthController) GenerateOTP(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := ac.userService.GetUserByID(userID)
	if err != nil {
		utils.Unauthorized(c, "用户不存在")
		return
	}

	// 从设置服务获取站点标题作为 App 中显示的 Issuer
	// 从设置服务获取站点标题作为账号名称后缀，以便于区分多环境
	siteTitle := ac.settingsService.Get(constant.SectionSite, constant.KeyTitle)
	accountName := user.Username
	if siteTitle != "" {
		accountName = fmt.Sprintf("%s@%s", user.Username, siteTitle)
	}

	// 生成新的 TOTP 密钥。固定 Issuer 为 "BaihuPanel" 以保留 Authenticator App 内置的 Logo 识别
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "BaihuPanel",
		AccountName: accountName,
	})
	if err != nil {
		utils.ServerError(c, "生成密钥失败")
		return
	}

	utils.Success(c, gin.H{
		"secret": key.Secret(),
		"url":    key.URL(),
	})
}

func (ac *AuthController) EnableOTP(c *gin.Context) {
	userID := c.GetString("userID")
	var req struct {
		Secret string `json:"secret" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 验证验证码是否与此 secret 匹配，防止错误绑定导致锁在外面
	if !totp.Validate(req.Code, req.Secret) {
		utils.BadRequest(c, "验证码校验失败")
		return
	}

	// 绑定并保存
	if err := ac.userService.UpdateOTP(userID, req.Secret, true); err != nil {
		utils.ServerError(c, "开启两步验证失败")
		return
	}

	utils.SuccessMsg(c, "开启两步验证成功")
}

func (ac *AuthController) DisableOTP(c *gin.Context) {
	userID := c.GetString("userID")
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	user, err := ac.userService.GetUserByID(userID)
	if err != nil {
		utils.Unauthorized(c, "用户不存在")
		return
	}

	if !user.OtpEnabled {
		utils.BadRequest(c, "两步验证尚未开启")
		return
	}

	// 验证验证码
	if !totp.Validate(req.Code, user.OtpSecret) {
		utils.BadRequest(c, "验证码错误")
		return
	}

	// 禁用并清除 secret
	if err := ac.userService.UpdateOTP(userID, "", false); err != nil {
		utils.ServerError(c, "关闭两步验证失败")
		return
	}

	utils.SuccessMsg(c, "关闭两步验证成功")
}
