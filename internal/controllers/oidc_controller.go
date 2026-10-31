package controllers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/oidc"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookie    = "BHOIDCState"
	oidcVerifierCookie = "BHOIDCVerifier"
	oidcStateTTL       = 10 * time.Minute
)

// logSSOFailure 统一记录单点登录失败事件
func (ac *AuthController) logSSOFailure(c *gin.Context, username, message string) {
	eventbus.DefaultBus.Publish(eventbus.Event{
		Type: constant.EventUserLogin,
		Payload: map[string]interface{}{
			"ip":        c.ClientIP(),
			"username":  username,
			"userAgent": c.GetHeader("User-Agent"),
			"status":    "failed",
			"message":   "单点登录失败：" + message,
		},
	})
}

// getSSOConfig 获取当前生效的单点登录配置（按偏好通道加载，未开启时自动跨通道回退）
func (ac *AuthController) getSSOConfig(c *gin.Context, preferredType ...string) *oidc.Config {
	pref := ""
	if len(preferredType) > 0 {
		pref = preferredType[0]
	}
	return oidc.ResolveActiveConfig(ac.settingsService, pref, c)
}

// getOidcConfig 加载标准 OIDC 配置（未启用时自动回退至通用 OAuth 2.0）
func (ac *AuthController) getOidcConfig(c *gin.Context) *oidc.Config {
	return ac.getSSOConfig(c, constant.SSOTypeOIDC)
}

// getOAuth2Config 加载通用 OAuth 2.0 配置（未启用时自动回退至标准 OIDC）
func (ac *AuthController) getOAuth2Config(c *gin.Context) *oidc.Config {
	return ac.getSSOConfig(c, constant.SSOTypeOAuth2)
}

// startSSOLogin 发起单点登录跳转流程（统一支持 OIDC 与通用 OAuth 2.0）
func (ac *AuthController) startSSOLogin(c *gin.Context, cfg *oidc.Config) {
	if cfg == nil {
		utils.BadRequest(c, "该单点登录通道未启用或配置不完整")
		return
	}

	state := utils.RandomString(32)
	postLogin := "/"
	if to := c.Query("redirect"); strings.HasPrefix(to, "/") && !strings.HasPrefix(to, "//") {
		postLogin = to
	}
	verifier := oauth2.GenerateVerifier()

	c.SetSameSite(http.SameSiteLaxMode)
	// Cookie 保存「state.通道类型.回跳路径」，回调时精准识别通道
	c.SetCookie(oidcStateCookie, state+"."+cfg.Type+"."+postLogin, int(oidcStateTTL.Seconds()), "/", "", false, true)
	c.SetCookie(oidcVerifierCookie, verifier, int(oidcStateTTL.Seconds()), "/", "", false, true)

	authURL, err := oidc.BuildAuthURL(c.Request.Context(), cfg, state, verifier)
	if err != nil {
		logger.Errorf("[SSO] 无法连接单点登录提供方 (Type: %s): %v", cfg.Type, err)
		utils.ServerError(c, "无法连接单点登录提供方: "+err.Error())
		return
	}

	c.Redirect(302, authURL)
}

// handleSSOCallback 处理单点登录授权回调（统一支持 OIDC 与通用 OAuth 2.0）
func (ac *AuthController) handleSSOCallback(c *gin.Context, defaultCfg *oidc.Config) {
	stateCookie, err := c.Cookie(oidcStateCookie)
	if err != nil || stateCookie == "" {
		utils.Unauthorized(c, "单点登录会话已失效，请重新登录")
		return
	}
	verifier, _ := c.Cookie(oidcVerifierCookie)
	middleware.ClearCookie(c, oidcStateCookie)
	middleware.ClearCookie(c, oidcVerifierCookie)

	// 解析 cookie: state.type.redirect
	parts := strings.SplitN(stateCookie, ".", 3)
	baseState := parts[0]
	channelType := ""
	postLogin := "/"
	if len(parts) == 3 {
		channelType = parts[1]
		postLogin = parts[2]
	} else if len(parts) == 2 {
		postLogin = parts[1]
	}

	if got := c.Query("state"); got == "" || got != baseState {
		utils.Unauthorized(c, "单点登录 state 校验失败")
		return
	}
	if e := c.Query("error"); e != "" {
		utils.Unauthorized(c, "第三方授权失败: "+e)
		return
	}
	code := c.Query("code")
	if code == "" {
		utils.Unauthorized(c, "缺少授权码")
		return
	}

	// 识别目标通道配置
	cfg := defaultCfg
	if channelType != "" {
		if matched := ac.getSSOConfig(c, channelType); matched != nil {
			cfg = matched
		}
	}
	if cfg == nil {
		cfg = ac.getSSOConfig(c)
	}
	if cfg == nil {
		utils.BadRequest(c, "单点登录通道未启用或配置不完整")
		return
	}

	user, created, err := oidc.ExchangeAndResolve(c.Request.Context(), cfg, code, verifier, ac.userService)
	if err != nil {
		ac.logSSOFailure(c, "", err.Error())
		if errors.Is(err, oidc.ErrUserNotLinked) {
			utils.Forbidden(c, err.Error())
			return
		}
		utils.Unauthorized(c, err.Error())
		return
	}

	expireDays := 7
	if d := utils.ParsePositiveInt(ac.settingsService.Get(constant.SectionSite, constant.KeyCookieDays)); d > 0 {
		expireDays = d
	}
	tokenStr, err := utils.GenerateToken(user.ID, user.Username, user.TokenVersion, expireDays, constant.Secret)
	if err != nil {
		utils.ServerError(c, "登录失败")
		return
	}
	middleware.SetAuthCookie(c, tokenStr, expireDays)

	msg := "单点登录成功 (" + cfg.Type + ")"
	if created {
		msg = "第三方身份首次登录，已自动创建账号 (" + cfg.Type + ")"
	}
	eventbus.DefaultBus.Publish(eventbus.Event{
		Type: constant.EventUserLogin,
		Payload: map[string]interface{}{
			"ip":        c.ClientIP(),
			"username":  user.Username,
			"userAgent": c.GetHeader("User-Agent"),
			"status":    "success",
			"message":   msg,
		},
	})

	if !strings.HasPrefix(postLogin, "/") || strings.HasPrefix(postLogin, "//") {
		postLogin = "/"
	}
	c.Redirect(302, postLogin)
}

// OidcLogin 重定向到 OIDC 授权端点
func (ac *AuthController) OidcLogin(c *gin.Context) {
	ac.startSSOLogin(c, ac.getSSOConfig(c, constant.SSOTypeOIDC))
}

// OidcCallback 处理 OIDC 授权回调
func (ac *AuthController) OidcCallback(c *gin.Context) {
	ac.handleSSOCallback(c, ac.getSSOConfig(c, constant.SSOTypeOIDC))
}

// OAuth2Login 重定向到通用 OAuth 2.0 授权端点
func (ac *AuthController) OAuth2Login(c *gin.Context) {
	ac.startSSOLogin(c, ac.getSSOConfig(c, constant.SSOTypeOAuth2))
}

// OAuth2Callback 处理通用 OAuth 2.0 授权回调
func (ac *AuthController) OAuth2Callback(c *gin.Context) {
	ac.handleSSOCallback(c, ac.getSSOConfig(c, constant.SSOTypeOAuth2))
}

// OidcTest 供设置界面测试单点登录提供方网络连通性与发现文档
func (ac *AuthController) OidcTest(c *gin.Context) {
	var req oidc.TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数格式错误")
		return
	}

	result, err := oidc.TestConnection(c.Request.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "连接提供方失败") || strings.Contains(err.Error(), "无法连接") {
			utils.ServerError(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "不能为空") {
			utils.BadRequest(c, err.Error())
			return
		}
		utils.ServerError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// OAuth2Test 供前端测试通用 OAuth 2.0 接口连通性
func (ac *AuthController) OAuth2Test(c *gin.Context) {
	ac.OidcTest(c)
}
