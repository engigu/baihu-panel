package controllers

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/middleware"
	"github.com/engigu/baihu-panel/internal/models"
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

// getOidcConfig 与 getOAuth2Config 暂无调用（已被 getSSOConfig 直接内联取代），注释保留备用
/*
func (ac *AuthController) getOidcConfig(c *gin.Context) *oidc.Config {
	return ac.getSSOConfig(c, constant.SSOTypeOIDC)
}

func (ac *AuthController) getOAuth2Config(c *gin.Context) *oidc.Config {
	return ac.getSSOConfig(c, constant.SSOTypeOAuth2)
}
*/

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
		c.Redirect(http.StatusFound, "/login?error="+url.QueryEscape("单点登录会话已失效，请重新登录"))
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
		c.Redirect(http.StatusFound, "/login?error="+url.QueryEscape("单点登录 state 校验失败"))
		return
	}
	if e := c.Query("error"); e != "" {
		c.Redirect(http.StatusFound, "/login?error="+url.QueryEscape("第三方授权失败: "+e))
		return
	}
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusFound, "/login?error="+url.QueryEscape("缺少授权码"))
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
		c.Redirect(http.StatusFound, "/login?error="+url.QueryEscape("单点登录通道未启用或配置不完整"))
		return
	}

	// 判断是否为绑定模式（已登录账号主动绑定第三方）
	if strings.HasPrefix(postLogin, "bind:") {
		bindUserID := strings.TrimPrefix(postLogin, "bind:")
		ac.handleSSOBindCallback(c, cfg, code, verifier, bindUserID)
		return
	}

	user, _, err := oidc.ExchangeAndResolve(c.Request.Context(), cfg, code, verifier, ac.userService)
	if err != nil {
		ac.logSSOFailure(c, "", err.Error())
		c.Redirect(http.StatusFound, "/login?error="+url.QueryEscape(err.Error()))
		return
	}

	expireDays := 7
	if d := utils.ParsePositiveInt(ac.settingsService.Get(constant.SectionSite, constant.KeyCookieDays)); d > 0 {
		expireDays = d
	}
	tokenStr, err := utils.GenerateToken(user.ID, user.Username, user.TokenVersion, expireDays, constant.Secret)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?error="+url.QueryEscape("生成用户凭证失败"))
		return
	}
	middleware.SetAuthCookie(c, tokenStr, expireDays)

	eventbus.DefaultBus.Publish(eventbus.Event{
		Type: constant.EventUserLogin,
		Payload: map[string]interface{}{
			"ip":        c.ClientIP(),
			"username":  user.Username,
			"userAgent": c.GetHeader("User-Agent"),
			"status":    "success",
			"message":   "单点登录成功 (" + cfg.Type + ")",
		},
	})

	if !strings.HasPrefix(postLogin, "/") || strings.HasPrefix(postLogin, "//") {
		postLogin = "/"
	}
	c.Redirect(302, postLogin)
}

// handleSSOBindCallback 处理主动授权绑定的回调流程
func (ac *AuthController) handleSSOBindCallback(c *gin.Context, cfg *oidc.Config, code, verifier, bindUserID string) {
	user, err := ac.userService.GetUserByID(bindUserID)
	if err != nil || user == nil {
		c.Redirect(302, "/settings?tab=security&sso_bind_error="+url.QueryEscape("用户会话不存在"))
		return
	}

	claims, err := oidc.ExchangeAndGetClaims(c.Request.Context(), cfg, code, verifier)
	if err != nil {
		c.Redirect(302, "/settings?tab=security&sso_bind_error="+url.QueryEscape("授权换取失败: "+err.Error()))
		return
	}

	// 检查是否已被其他用户绑定
	var existing models.UserSSOBinding
	res := database.DB.Where("(channel_type = ? AND subject = ?) OR subject = ?", cfg.Type, claims.Sub, claims.Sub).
		Limit(1).Find(&existing)
	if res.Error == nil && res.RowsAffected > 0 {
		if existing.UserID != user.ID {
			c.Redirect(302, "/settings?tab=security&sso_bind_error="+url.QueryEscape("该第三方身份已被账号 ["+existing.Username+"] 绑定"))
			return
		}
		// 若已绑定至当前用户，刷新信息
		database.DB.Model(&existing).Updates(map[string]interface{}{
			"identity_name": claims.PreferredUsername,
			"email":         claims.Email,
			"updated_at":    models.Now(),
		})
		c.Redirect(302, "/settings?tab=security&sso_bind_success=1")
		return
	}

	// 创建新的绑定映射
	newBinding := models.UserSSOBinding{
		ID:           utils.GenerateID(),
		UserID:       user.ID,
		Username:     user.Username,
		ChannelType:  cfg.Type,
		Subject:      claims.Sub,
		IdentityName: claims.PreferredUsername,
		Email:        claims.Email,
		BindType:     "oauth",
		CreatedAt:    models.Now(),
		UpdatedAt:    models.Now(),
	}
	if err := database.DB.Create(&newBinding).Error; err != nil {
		c.Redirect(302, "/settings?tab=security&sso_bind_error="+url.QueryEscape("保存绑定记录失败: "+err.Error()))
		return
	}

	if user.OidcSubject == "" {
		database.DB.Model(user).Update("oidc_subject", claims.Sub)
	}

	eventbus.DefaultBus.Publish(eventbus.Event{
		Type: constant.EventUserLogin,
		Payload: map[string]interface{}{
			"ip":        c.ClientIP(),
			"username":  user.Username,
			"userAgent": c.GetHeader("User-Agent"),
			"status":    "success",
			"message":   "账号成功绑定第三方身份 (" + cfg.Type + ": " + claims.Sub + ")",
		},
	})

	c.Redirect(302, "/settings?tab=security&sso_bind_success=1")
}

// StartSSOBind 当前已登录用户发起单点登录授权绑定流程
func (ac *AuthController) StartSSOBind(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		utils.Unauthorized(c, "未登录或会话无效")
		return
	}
	channelType := c.DefaultQuery("type", constant.SSOTypeOIDC)
	cfg := ac.getSSOConfig(c, channelType)
	if cfg == nil {
		utils.BadRequest(c, "该单点登录通道未启用或配置不完整")
		return
	}

	state := utils.RandomString(32)
	verifier := oauth2.GenerateVerifier()

	c.SetSameSite(http.SameSiteLaxMode)
	// Cookie 记录 bind 标记与当前用户 ID
	c.SetCookie(oidcStateCookie, state+"."+cfg.Type+".bind:"+userID, int(oidcStateTTL.Seconds()), "/", "", false, true)
	c.SetCookie(oidcVerifierCookie, verifier, int(oidcStateTTL.Seconds()), "/", "", false, true)

	authURL, err := oidc.BuildAuthURL(c.Request.Context(), cfg, state, verifier)
	if err != nil {
		logger.Errorf("[SSO] 无法连接单点登录提供方 (Type: %s): %v", cfg.Type, err)
		utils.ServerError(c, "无法连接单点登录提供方: "+err.Error())
		return
	}

	c.Redirect(302, authURL)
}

// GetSSOBindings 获取全量 SSO 账号绑定列表
func (ac *AuthController) GetSSOBindings(c *gin.Context) {
	var bindings []models.UserSSOBinding
	if err := database.DB.Order("created_at desc").Find(&bindings).Error; err != nil {
		utils.ServerError(c, "获取绑定列表失败")
		return
	}
	utils.Success(c, bindings)
}

// DeleteSSOBinding 按 ID 解除一条身份绑定
func (ac *AuthController) DeleteSSOBinding(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "缺少绑定ID")
		return
	}

	var binding models.UserSSOBinding
	res := database.DB.Where("id = ?", id).Limit(1).Find(&binding)
	if res.Error != nil || res.RowsAffected == 0 {
		utils.NotFound(c, "未找到该绑定记录")
		return
	}

	currentUserID := c.GetString("userID")
	currentUser, _ := ac.userService.GetUserByID(currentUserID)
	if currentUser != nil && currentUser.Role != "admin" && binding.UserID != currentUserID {
		utils.Forbidden(c, "无权解绑他人账号")
		return
	}

	// 核心安全校验：若用户没有设置本地密码且仅剩该唯一 SSO 绑定，禁止解绑
	targetUser, _ := ac.userService.GetUserByID(binding.UserID)
	if targetUser != nil && targetUser.Password == "" {
		var remainCount int64
		database.DB.Model(&models.UserSSOBinding{}).Where("user_id = ?", binding.UserID).Count(&remainCount)
		if remainCount <= 1 {
			utils.BadRequest(c, "当前账号未设置本地密码，解除唯一第三方绑定将导致无法登录系统，请先设置密码后再解绑")
			return
		}
	}

	if err := database.DB.Delete(&binding).Error; err != nil {
		utils.ServerError(c, "解绑失败")
		return
	}

	// 维护主表兼容字段 oidc_subject
	var remaining models.UserSSOBinding
	nextRes := database.DB.Where("user_id = ?", binding.UserID).Order("created_at desc").Limit(1).Find(&remaining)
	newSub := ""
	if nextRes.Error == nil && nextRes.RowsAffected > 0 {
		newSub = remaining.Subject
	}
	database.DB.Model(&models.User{}).Where("id = ? AND oidc_subject = ?", binding.UserID, binding.Subject).
		Update("oidc_subject", newSub)

	utils.SuccessMsg(c, "解绑成功")
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
