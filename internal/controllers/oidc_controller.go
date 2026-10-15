package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/utils"
	"golang.org/x/oauth2"

	"github.com/gin-gonic/gin"
)

// OidcStatus 供登录页判断是否展示 SSO (OIDC / OAuth 2.0) 登录按钮及显示名称
func (ac *AuthController) OidcStatus(c *gin.Context) {
	if constant.DemoMode {
		utils.Success(c, gin.H{"enabled": false})
		return
	}
	cfg := ac.oidcConfig()
	if cfg == nil {
		utils.Success(c, gin.H{"enabled": false})
		return
	}

	name := "OIDC"
	if cfg.Type == constant.SSOTypeOAuth2 {
		if cfg.DisplayName != "" {
			name = cfg.DisplayName
		} else {
			name = "OAuth 2.0"
		}
	}

	utils.Success(c, gin.H{
		"enabled": true,
		"type":    cfg.Type,
		"name":    name,
	})
}

// OidcLogin 重定向到 SSO 授权端点（授权码 + PKCE）
func (ac *AuthController) OidcLogin(c *gin.Context) {
	if constant.DemoMode {
		utils.BadRequest(c, "演示模式下禁止使用单点登录")
		return
	}

	cfg := ac.oidcConfig()
	if cfg == nil {
		utils.BadRequest(c, "单点登录未启用或配置不完整")
		return
	}

	_, oauthCfg, err := ac.oidcProviderFor(c.Request.Context(), cfg)
	if err != nil {
		utils.ServerError(c, "无法连接单点登录提供方，请检查配置")
		return
	}

	state := utils.RandomString(32)
	// 允许站内相对路径回跳；仅接受 "/" 开头且非 "//" 的值，防开放重定向
	postLogin := "/"
	if to := c.Query("redirect"); strings.HasPrefix(to, "/") && !strings.HasPrefix(to, "//") {
		postLogin = to
	}
	verifier := oauth2.GenerateVerifier()

	c.SetSameSite(http.SameSiteLaxMode)
	// Cookie 保存「state.回跳路径」，回调时按首个 "." 拆开
	c.SetCookie(oidcStateCookie, state+"."+postLogin, int(oidcStateTTL.Seconds()), "/", "", false, true)
	c.SetCookie(oidcVerifierCookie, verifier, int(oidcStateTTL.Seconds()), "/", "", false, true)

	// 只把不透明的 state 交给 IdP：回跳路径仅留在 Cookie 中。
	// 否则 IdP 回显的 state 会带上「.回跳路径」，与回调里比对的基础 state 不相等，
	// 直接登录（带 ?redirect=）会被误判为「单点登录 state 校验失败」。
	authURL := oauthCfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	c.Redirect(302, authURL)
}

// OidcCallback 校验 state、换取令牌/身份、绑定或创建本地账号并签发本系统 JWT
func (ac *AuthController) OidcCallback(c *gin.Context) {
	if constant.DemoMode {
		utils.BadRequest(c, "演示模式下禁止使用单点登录")
		return
	}

	cfg := ac.oidcConfig()
	if cfg == nil {
		utils.BadRequest(c, "单点登录未启用")
		return
	}

	stateCookie, err := c.Cookie(oidcStateCookie)
	if err != nil || stateCookie == "" {
		utils.Unauthorized(c, "单点登录会话已失效，请重新登录")
		return
	}
	verifier, _ := c.Cookie(oidcVerifierCookie)
	middleware.ClearCookie(c, oidcStateCookie)
	middleware.ClearCookie(c, oidcVerifierCookie)

	baseState, postLogin := stateCookie, "/"
	if i := strings.Index(stateCookie, "."); i > 0 {
		baseState, postLogin = stateCookie[:i], stateCookie[i+1:]
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	verifierObj, oauthCfg, err := ac.oidcProviderFor(ctx, cfg)
	if err != nil {
		utils.ServerError(c, "无法连接单点登录提供方")
		return
	}

	// 换取 Token：优先尝试带 PKCE verifier，若第三方不支持则降级重试普通 exchange
	token, err := oauthCfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		token, err = oauthCfg.Exchange(ctx, code)
	}
	if err != nil {
		ac.logOidcFailure(c, "", "授权码换取令牌失败: "+err.Error())
		utils.Unauthorized(c, "授权码换取令牌失败")
		return
	}

	var claims oidcClaims
	if cfg.Type == constant.SSOTypeOAuth2 {
		// 通用 OAuth 2.0 模式：通过 Access Token 调用 Userinfo 接口获取用户信息
		cObj, err := ac.fetchOAuth2Userinfo(ctx, cfg, token.AccessToken)
		if err != nil {
			ac.logOidcFailure(c, "", "获取用户信息失败: "+err.Error())
			utils.Unauthorized(c, "获取用户信息失败: "+err.Error())
			return
		}
		claims = *cObj
	} else {
		// 标准 OIDC 模式：验证 id_token 签名与 Claims
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			utils.Unauthorized(c, "OIDC 未返回 id_token")
			return
		}

		// 验签 + 校验 iss/aud/exp，JWKS 由 go-oidc 内部缓存
		idToken, err := verifierObj.Verify(ctx, rawIDToken)
		if err != nil {
			ac.logOidcFailure(c, "", "id_token 校验失败")
			utils.Unauthorized(c, "id_token 校验失败")
			return
		}
		if err := idToken.Claims(&claims); err != nil {
			utils.Unauthorized(c, "无法解析 OIDC 身份信息")
			return
		}
	}

	user, created := ac.resolveOidcUser(cfg, &claims)
	if user == nil {
		ac.logOidcFailure(c, claims.PreferredUsername, "未绑定账号且未开启自动创建")
		utils.Forbidden(c, "当前 OIDC 身份未绑定任何账号，请联系管理员")
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

	msg := "OIDC 登录成功"
	if created {
		msg = "OIDC 首次登录，已自动创建账号"
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
