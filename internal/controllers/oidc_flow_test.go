package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// fullFlow 走完 login → callback
func fullFlow(t *testing.T, idp *fakeIdP, autoCreate bool) (*httptest.ResponseRecorder, *AuthController) {
	t.Helper()
	ac := setupOidcTest(t, idp, autoCreate)
	router := gin.New()
	router.GET("/api/v1/auth/oidc/login", ac.OidcLogin)
	router.GET("/api/v1/auth/oidc/callback", ac.OidcCallback)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("login 期望 302，实际 %d (%s)", w.Code, w.Body.String())
	}
	state := extractQuery(t, w.Header().Get("Location"), "state")

	cbReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=test-code&state="+state, nil)
	for _, c := range w.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)
	return cbW, ac
}

func extractQuery(t *testing.T, rawURL, key string) string {
	t.Helper()
	i := strings.Index(rawURL, "?")
	if i < 0 {
		t.Fatalf("URL 无查询串: %s", rawURL)
	}
	for _, kv := range strings.Split(rawURL[i+1:], "&") {
		if p := strings.SplitN(kv, "=", 2); len(p) == 2 && p[0] == key {
			return p[1]
		}
	}
	t.Fatalf("URL 缺少参数 %s: %s", key, rawURL)
	return ""
}

func findCookie(w *httptest.ResponseRecorder, name string) string {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// 未开启自动创建时，陌生身份应被拒绝且不落库
func TestOidcLoginUnknownIdentityRejected(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-unknown", "nobody@example.com", "nobody")
	w, _ := fullFlow(t, idp, false)
	if w.Code != http.StatusForbidden {
		t.Fatalf("期望 403，实际 %d (%s)", w.Code, w.Body.String())
	}
	var n int64
	database.DB.Model(&models.User{}).Where("oidc_subject = ?", "sub-unknown").Count(&n)
	if n != 0 {
		t.Fatalf("被拒绝的登录不应创建用户，实际 %d 个", n)
	}
}

// 开启自动创建时，首次登录应建号并签发本系统 JWT cookie
func TestOidcLoginAutoCreateIssuesJwt(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-new", "new@example.com", "newuser")
	w, _ := fullFlow(t, idp, true)
	if w.Code != http.StatusFound {
		t.Fatalf("期望 302，实际 %d (%s)", w.Code, w.Body.String())
	}

	var user models.User
	if err := database.DB.Where("oidc_subject = ?", "sub-new").First(&user).Error; err != nil {
		t.Fatalf("未创建 OIDC 用户: %v", err)
	}
	if user.Username != "newuser" || user.Role != constant.DefaultRole {
		t.Fatalf("自动创建用户属性不符: %+v", user)
	}

	token := findCookie(w, constant.CookieName)
	if token == "" {
		t.Fatal("回调未下发认证 Cookie")
	}
	userID, username, version, err := utils.ParseToken(token, constant.Secret)
	if err != nil {
		t.Fatalf("签发的 token 无法解析: %v", err)
	}
	if userID != user.ID || username != user.Username || version != user.TokenVersion {
		t.Fatalf("token 内容与用户不符: %s/%s/%d", userID, username, version)
	}
}

// 第二次登录同一 subject 应命中已有账号，不重复建号
func TestOidcLoginBindsExistingSubject(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-re", "re@example.com", "reuser")
	w, ac := fullFlow(t, idp, true)
	if w.Code != http.StatusFound {
		t.Fatalf("首次登录失败: %d", w.Code)
	}

	router := gin.New()
	router.GET("/api/v1/auth/oidc/login", ac.OidcLogin)
	router.GET("/api/v1/auth/oidc/callback", ac.OidcCallback)
	loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, loginReq)
	state := extractQuery(t, lw.Header().Get("Location"), "state")
	cbReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=c2&state="+state, nil)
	for _, c := range lw.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)
	if cbW.Code != http.StatusFound {
		t.Fatalf("二次登录失败: %d (%s)", cbW.Code, cbW.Body.String())
	}

	var total int64
	database.DB.Model(&models.User{}).Where("oidc_subject = ?", "sub-re").Count(&total)
	if total != 1 {
		t.Fatalf("同一 subject 应只有一个账号，实际 %d", total)
	}
}

// 伪造/篡改的 state 必须被拒（CSRF 防护）
func TestOidcCallbackRejectsBadState(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-x", "x@example.com", "xuser")
	ac := setupOidcTest(t, idp, true)
	router := gin.New()
	router.GET("/api/v1/auth/oidc/login", ac.OidcLogin)
	router.GET("/api/v1/auth/oidc/callback", ac.OidcCallback)

	loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, loginReq)

	cbReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=c&state=forged-state", nil)
	for _, c := range lw.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)
	if cbW.Code != http.StatusUnauthorized {
		t.Fatalf("伪造 state 应返回 401，实际 %d (%s)", cbW.Code, cbW.Body.String())
	}
}

// 带 ?redirect= 登录时，IdP 回显的 state 必须与 Cookie 中的基础 state 一致（不含回跳路径），
// 否则回调会把回显值拿去比对基础值，直接登录被误判为「单点登录 state 校验失败」
func TestOidcLoginWithRedirectKeepsIdpStateOpaque(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-redir", "redir@example.com", "rediruser")
	ac := setupOidcTest(t, idp, true)
	router := gin.New()
	router.GET("/api/v1/auth/oidc/login", ac.OidcLogin)
	router.GET("/api/v1/auth/oidc/callback", ac.OidcCallback)

	loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login?redirect=%2Fdashboard", nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, loginReq)
	if lw.Code != http.StatusFound {
		t.Fatalf("login 期望 302，实际 %d (%s)", lw.Code, lw.Body.String())
	}
	state := extractQuery(t, lw.Header().Get("Location"), "state")
	if state == "" || strings.Contains(state, "dashboard") {
		t.Fatalf("发给 IdP 的 state 不应包含回跳路径: %q", state)
	}

	cbReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=redir-code&state="+state, nil)
	for _, c := range lw.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)
	if cbW.Code != http.StatusFound {
		t.Fatalf("带 redirect 的回调应成功，实际 %d (%s)", cbW.Code, cbW.Body.String())
	}
	if loc := cbW.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("登录后应回跳 /dashboard，实际 %q", loc)
	}
}

// 验证各种流行平台（GitHub, Gitee, 通用平台）返回的用户信息解析
func TestParseOAuth2Userinfo(t *testing.T) {
	// 1. GitHub 风格 (id 为数字，login 为用户名)
	ghJSON := []byte(`{"id": 583231, "login": "octocat", "name": "The Octocat", "email": "octocat@github.com"}`)
	claims, err := parseOAuth2Userinfo(ghJSON, "")
	if err != nil {
		t.Fatalf("GitHub JSON 解析失败: %v", err)
	}
	if claims.Sub != "583231" || claims.PreferredUsername != "octocat" || claims.Email != "octocat@github.com" {
		t.Errorf("GitHub 解析结果不符合预期: %+v", claims)
	}

	// 2. Gitee 风格 (无公开邮箱)
	giteeJSON := []byte(`{"id": 99999, "login": "gitee_user", "name": "Gitee User"}`)
	claimsGitee, err := parseOAuth2Userinfo(giteeJSON, "")
	if err != nil {
		t.Fatalf("Gitee JSON 解析失败: %v", err)
	}
	if claimsGitee.Sub != "99999" || claimsGitee.PreferredUsername != "gitee_user" {
		t.Errorf("Gitee 解析结果不符合预期: %+v", claimsGitee)
	}

	// 3. 自定义字段提取 (例如特定企业自建平台使用 uid)
	customJSON := []byte(`{"uid": "emp_10086", "user_name": "developer", "mail": "dev@corp.internal"}`)
	claimsCustom, err := parseOAuth2Userinfo(customJSON, "uid")
	if err != nil {
		t.Fatalf("自定义 JSON 解析失败: %v", err)
	}
	if claimsCustom.Sub != "emp_10086" || claimsCustom.Email != "dev@corp.internal" {
		t.Errorf("自定义解析结果不符合预期: %+v", claimsCustom)
	}
}

// 模拟纯 OAuth 2.0 服务端（无 OIDC / 无 id_token），验证端到端单点登录全流程
func TestOAuth2LoginFlowAutoCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	database.DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{})

	// 1. 搭建一个模拟的纯 OAuth 2.0 授权服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"access_token": "oauth2-mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer oauth2-mock-access-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{
			"id":    1001,
			"login": "github_coder",
			"name":  "Coder",
			"email": "coder@example.com",
		})
	})
	oauthServer := httptest.NewServer(mux)
	defer oauthServer.Close()

	// 2. 配置通用 OAuth 2.0 参数
	settings := services.NewSettingsService()
	for k, v := range map[string]string{
		constant.KeyOidcEnabled:      "true",
		constant.KeyOidcType:         constant.SSOTypeOAuth2,
		constant.KeyOidcClientID:     "mock-client-id",
		constant.KeyOidcClientSecret: "mock-client-secret",
		constant.KeyOidcRedirectURL:  "http://panel.local/api/v1/auth/oidc/callback",
		constant.KeyOidcAutoCreate:   "true",
		constant.KeyOidcDefaultRole:  constant.DefaultRole,
		constant.KeyOauth2AuthURL:    oauthServer.URL + "/oauth/authorize",
		constant.KeyOauth2TokenURL:   oauthServer.URL + "/oauth/token",
		constant.KeyOauth2UserinfoURL: oauthServer.URL + "/api/user",
		constant.KeyOauth2Name:       "GitHub",
	} {
		_ = settings.Set(constant.SectionSecurity, k, v)
	}

	ac := NewAuthController(services.NewUserService(), settings, services.NewLoginLogService())

	router := gin.New()
	router.GET("/api/v1/auth/oidc/status", ac.OidcStatus)
	router.GET("/api/v1/auth/oidc/login", ac.OidcLogin)
	router.GET("/api/v1/auth/oidc/callback", ac.OidcCallback)

	// 3. 验证 status 接口返回正确名称与类型
	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/status", nil)
	statusW := httptest.NewRecorder()
	router.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK || !strings.Contains(statusW.Body.String(), "GitHub") {
		t.Fatalf("OidcStatus 响应不符合预期: %s", statusW.Body.String())
	}

	// 4. 发起登录
	loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusFound {
		t.Fatalf("OAuth2 login 期望 302，实际 %d (%s)", loginW.Code, loginW.Body.String())
	}
	state := extractQuery(t, loginW.Header().Get("Location"), "state")

	// 5. 模拟回调
	cbReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=mock_code&state="+state, nil)
	for _, c := range loginW.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)

	if cbW.Code != http.StatusFound {
		t.Fatalf("OAuth2 callback 期望 302 成功跳回，实际 %d (%s)", cbW.Code, cbW.Body.String())
	}

	// 6. 验证数据库中用户是否被正确建号
	var user models.User
	if err := database.DB.Where("oidc_subject = ?", "1001").First(&user).Error; err != nil {
		t.Fatalf("未成功通过 OAuth2 创建用户: %v", err)
	}
	if user.Username != "github_coder" || user.Email != "coder@example.com" {
		t.Errorf("创建的用户属性不符合预期: %+v", user)
	}

	// 7. 验证是否成功下发 JWT 会话 Cookie
	token := findCookie(cbW, constant.CookieName)
	if token == "" {
		t.Fatal("OAuth2 登录成功后未下发认证 Cookie")
	}
}
