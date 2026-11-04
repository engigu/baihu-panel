package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/oidc"
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
	router.GET(constant.PathAuthOidcLogin, ac.OidcLogin)
	router.GET(constant.PathAuthOidcCallback, ac.OidcCallback)

	req := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcLogin, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("login 期望 302，实际 %d (%s)", w.Code, w.Body.String())
	}
	state := extractQuery(t, w.Header().Get("Location"), "state")

	cbReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcCallback+"?code=test-code&state="+state, nil)
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

// 移除自动建号后，未绑定的陌生身份应被拒绝并跳回登录页带 error 提示，绝不创建新账号
func TestOidcLoginUnknownIdentityRejected(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-unknown", "nobody@example.com", "nobody")
	w, _ := fullFlow(t, idp, false)
	if w.Code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "/login?error=") {
		t.Fatalf("期望 302 跳回 /login?error=，实际 %d, Location: %s", w.Code, w.Header().Get("Location"))
	}
	var n int64
	database.DB.Model(&models.User{}).Where("oidc_subject = ?", "sub-unknown").Count(&n)
	if n != 0 {
		t.Fatalf("被拒绝的登录不应创建用户，实际 %d 个", n)
	}
}

// 已在绑定表中关联本地账号的身份，登录成功并签发本系统 JWT cookie
func TestOidcLoginBoundUserIssuesJwt(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-bound", "bound@example.com", "bounduser")
	ac := setupOidcTest(t, idp, false)

	userSvc := services.NewUserService()
	existingUser := userSvc.CreateUser("bounduser", "pass1234", "bound@example.com", constant.DefaultRole)

	// 预置绑定关系
	database.DB.Create(&models.UserSSOBinding{
		ID:           utils.GenerateID(),
		UserID:       existingUser.ID,
		Username:     existingUser.Username,
		ChannelType:  constant.SSOTypeOIDC,
		Subject:      "sub-bound",
		IdentityName: "bounduser",
		Email:        "bound@example.com",
		BindType:     "oauth",
		CreatedAt:    models.Now(),
		UpdatedAt:    models.Now(),
	})

	router := gin.New()
	router.GET(constant.PathAuthOidcLogin, ac.OidcLogin)
	router.GET(constant.PathAuthOidcCallback, ac.OidcCallback)

	loginReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcLogin, nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, loginReq)
	state := extractQuery(t, lw.Header().Get("Location"), "state")

	cbReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcCallback+"?code=test-code&state="+state, nil)
	for _, c := range lw.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)

	if cbW.Code != http.StatusFound {
		t.Fatalf("期望 302，实际 %d (%s)", cbW.Code, cbW.Body.String())
	}

	token := findCookie(cbW, constant.CookieName)
	if token == "" {
		t.Fatal("回调未下发认证 Cookie")
	}
	userID, username, version, err := utils.ParseToken(token, constant.Secret)
	if err != nil {
		t.Fatalf("签发的 token 无法解析: %v", err)
	}
	if userID != existingUser.ID || username != existingUser.Username || version != existingUser.TokenVersion {
		t.Fatalf("token 内容与用户不符: %s/%s/%d", userID, username, version)
	}
}

// 多次登录同一已绑定身份，应直接命中已有账号
func TestOidcLoginBindsExistingSubject(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-re", "re@example.com", "reuser")
	ac := setupOidcTest(t, idp, false)

	userSvc := services.NewUserService()
	existingUser := userSvc.CreateUser("reuser", "pass1234", "re@example.com", constant.DefaultRole)
	database.DB.Create(&models.UserSSOBinding{
		ID:           utils.GenerateID(),
		UserID:       existingUser.ID,
		Username:     existingUser.Username,
		ChannelType:  constant.SSOTypeOIDC,
		Subject:      "sub-re",
		IdentityName: "reuser",
		Email:        "re@example.com",
		BindType:     "oauth",
		CreatedAt:    models.Now(),
		UpdatedAt:    models.Now(),
	})

	router := gin.New()
	router.GET(constant.PathAuthOidcLogin, ac.OidcLogin)
	router.GET(constant.PathAuthOidcCallback, ac.OidcCallback)

	// 第一次登录
	loginReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcLogin, nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, loginReq)
	state := extractQuery(t, lw.Header().Get("Location"), "state")

	cbReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcCallback+"?code=c1&state="+state, nil)
	for _, c := range lw.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)
	if cbW.Code != http.StatusFound {
		t.Fatalf("首次登录失败: %d", cbW.Code)
	}

	// 第二次登录
	loginReq2 := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcLogin, nil)
	lw2 := httptest.NewRecorder()
	router.ServeHTTP(lw2, loginReq2)
	state2 := extractQuery(t, lw2.Header().Get("Location"), "state")

	cbReq2 := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcCallback+"?code=c2&state="+state2, nil)
	for _, c := range lw2.Result().Cookies() {
		cbReq2.AddCookie(c)
	}
	cbW2 := httptest.NewRecorder()
	router.ServeHTTP(cbW2, cbReq2)
	if cbW2.Code != http.StatusFound {
		t.Fatalf("二次登录失败: %d (%s)", cbW2.Code, cbW2.Body.String())
	}

	var total int64
	database.DB.Model(&models.User{}).Where("username = ?", "reuser").Count(&total)
	if total != 1 {
		t.Fatalf("同一用户应只有一个账号，实际 %d", total)
	}
}

// 伪造/篡改的 state 必须被拒（CSRF 防护）
func TestOidcCallbackRejectsBadState(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-x", "x@example.com", "xuser")
	ac := setupOidcTest(t, idp, false)
	router := gin.New()
	router.GET(constant.PathAuthOidcLogin, ac.OidcLogin)
	router.GET(constant.PathAuthOidcCallback, ac.OidcCallback)

	loginReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcLogin, nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, loginReq)

	cbReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcCallback+"?code=c&state=forged-state", nil)
	for _, c := range lw.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)
	if cbW.Code != http.StatusFound || !strings.Contains(cbW.Header().Get("Location"), "/login?error=") {
		t.Fatalf("伪造 state 应重定向回登录页带 error，实际 %d (%s)", cbW.Code, cbW.Header().Get("Location"))
	}
}

// 带 ?redirect= 登录时，IdP 回显的 state 必须与 Cookie 中的基础 state 一致（不含回跳路径）
func TestOidcLoginWithRedirectKeepsIdpStateOpaque(t *testing.T) {
	idp := newFakeIdP(t, "baihu-panel", "sub-redir", "redir@example.com", "rediruser")
	ac := setupOidcTest(t, idp, false)

	userSvc := services.NewUserService()
	existingUser := userSvc.CreateUser("rediruser", "pass1234", "redir@example.com", constant.DefaultRole)
	database.DB.Create(&models.UserSSOBinding{
		ID:           utils.GenerateID(),
		UserID:       existingUser.ID,
		Username:     existingUser.Username,
		ChannelType:  constant.SSOTypeOIDC,
		Subject:      "sub-redir",
		IdentityName: "rediruser",
		Email:        "redir@example.com",
		BindType:     "oauth",
		CreatedAt:    models.Now(),
		UpdatedAt:    models.Now(),
	})

	router := gin.New()
	router.GET(constant.PathAuthOidcLogin, ac.OidcLogin)
	router.GET(constant.PathAuthOidcCallback, ac.OidcCallback)

	loginReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcLogin+"?redirect=%2Fdashboard", nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, loginReq)
	if lw.Code != http.StatusFound {
		t.Fatalf("login 期望 302，实际 %d (%s)", lw.Code, lw.Body.String())
	}
	state := extractQuery(t, lw.Header().Get("Location"), "state")
	if state == "" || strings.Contains(state, "dashboard") {
		t.Fatalf("发给 IdP 的 state 不应包含回跳路径: %q", state)
	}

	cbReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcCallback+"?code=redir-code&state="+state, nil)
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
	claims, err := oidc.ParseOAuth2Userinfo(ghJSON, "")
	if err != nil {
		t.Fatalf("GitHub JSON 解析失败: %v", err)
	}
	if claims.Sub != "583231" || claims.PreferredUsername != "octocat" || claims.Email != "octocat@github.com" {
		t.Errorf("GitHub 解析结果不符合预期: %+v", claims)
	}

	// 2. Gitee 风格 (无公开邮箱)
	giteeJSON := []byte(`{"id": 99999, "login": "gitee_user", "name": "Gitee User"}`)
	claimsGitee, err := oidc.ParseOAuth2Userinfo(giteeJSON, "")
	if err != nil {
		t.Fatalf("Gitee JSON 解析失败: %v", err)
	}
	if claimsGitee.Sub != "99999" || claimsGitee.PreferredUsername != "gitee_user" {
		t.Errorf("Gitee 解析结果不符合预期: %+v", claimsGitee)
	}

	// 3. 自定义字段提取 (例如特定企业自建平台使用 uid)
	customJSON := []byte(`{"uid": "emp_10086", "user_name": "developer", "mail": "dev@corp.internal"}`)
	claimsCustom, err := oidc.ParseOAuth2Userinfo(customJSON, "uid")
	if err != nil {
		t.Fatalf("自定义 JSON 解析失败: %v", err)
	}
	if claimsCustom.Sub != "emp_10086" || claimsCustom.Email != "dev@corp.internal" {
		t.Errorf("自定义解析结果不符合预期: %+v", claimsCustom)
	}
}

// 模拟纯 OAuth 2.0 服务端（无 OIDC / 无 id_token），验证端到端单点登录全流程
func TestOAuth2LoginFlowBoundUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	database.DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{}, &models.UserSSOBinding{})

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
		constant.KeyOidcEnabled:       "true",
		constant.KeyOidcType:          constant.SSOTypeOAuth2,
		constant.KeyOidcClientID:      "mock-client-id",
		constant.KeyOidcClientSecret:  "mock-client-secret",
		constant.KeyOidcRedirectURL:   "http://panel.local" + constant.PathAuthOidcCallback,
		constant.KeyOauth2AuthURL:     oauthServer.URL + "/oauth/authorize",
		constant.KeyOauth2TokenURL:    oauthServer.URL + "/oauth/token",
		constant.KeyOauth2UserinfoURL: oauthServer.URL + "/api/user",
		constant.KeyOauth2Name:        "GitHub",
	} {
		_ = settings.Set(constant.SectionSecurity, k, v)
	}

	userSvc := services.NewUserService()
	existingUser := userSvc.CreateUser("github_coder", "pass1234", "coder@example.com", constant.DefaultRole)
	database.DB.Create(&models.UserSSOBinding{
		ID:           utils.GenerateID(),
		UserID:       existingUser.ID,
		Username:     existingUser.Username,
		ChannelType:  constant.SSOTypeOAuth2,
		Subject:      "1001",
		IdentityName: "github_coder",
		Email:        "coder@example.com",
		BindType:     "oauth",
		CreatedAt:    models.Now(),
		UpdatedAt:    models.Now(),
	})

	ac := NewAuthController(userSvc, settings, services.NewLoginLogService())
	pc := NewPublicController(settings)

	router := gin.New()
	router.GET("/api/v1/public/settings", pc.GetPublicSettings)
	router.GET(constant.PathAuthOidcLogin, ac.OidcLogin)
	router.GET(constant.PathAuthOidcCallback, ac.OidcCallback)

	// 3. 验证 public/settings 接口返回正确 sso 名称与类型
	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/settings", nil)
	statusW := httptest.NewRecorder()
	router.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK || !strings.Contains(statusW.Body.String(), "GitHub") {
		t.Fatalf("GetPublicSettings SSO 响应不符合预期: %s", statusW.Body.String())
	}

	// 4. 发起登录
	loginReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcLogin, nil)
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)
	if loginW.Code != http.StatusFound {
		t.Fatalf("OAuth2 login 期望 302，实际 %d (%s)", loginW.Code, loginW.Body.String())
	}
	state := extractQuery(t, loginW.Header().Get("Location"), "state")

	// 5. 模拟回调
	cbReq := httptest.NewRequest(http.MethodGet, constant.PathAuthOidcCallback+"?code=mock_code&state="+state, nil)
	for _, c := range loginW.Result().Cookies() {
		cbReq.AddCookie(c)
	}
	cbW := httptest.NewRecorder()
	router.ServeHTTP(cbW, cbReq)

	if cbW.Code != http.StatusFound {
		t.Fatalf("OAuth2 callback 期望 302 成功跳回，实际 %d (%s)", cbW.Code, cbW.Body.String())
	}

	// 6. 验证是否成功下发 JWT 会话 Cookie
	token := findCookie(cbW, constant.CookieName)
	if token == "" {
		t.Fatal("OAuth2 登录成功后未下发认证 Cookie")
	}
}

// 验证当管理员启用单点登录时，/settings/public 能自动推导并在 sso 字段中正常返回 enabled: true
func TestPublicSiteSettingsSSOFallback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("初始化测试内存数据库失败: %v", err)
	}
	database.DB = db
	_ = db.AutoMigrate(&models.Setting{}, &models.User{})

	settings := services.NewSettingsService()
	for k, v := range map[string]string{
		constant.KeyOidcEnabled:     "true",
		constant.KeyOidcType:        "oidc",
		constant.KeyOidcIssuer:      "https://id.example.com",
		constant.KeyOidcClientID:    "my-client",
		constant.KeyOidcRedirectURL: "",
	} {
		_ = settings.Set(constant.SectionSecurity, k, v)
	}

	pc := NewPublicController(settings)
	router := gin.New()
	router.GET("/api/v1/public/settings", pc.GetPublicSettings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/settings", nil)
	req.Host = "panel.test.com"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际: %d (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"enabled":true`) {
		t.Fatalf("public/settings 未能正确包含 sso enabled: true，响应: %s", w.Body.String())
	}
}

// 验证 OidcTest 接口测试
func TestOidcTestEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService()
	ac := NewAuthController(services.NewUserService(), settings, services.NewLoginLogService())

	router := gin.New()
	router.POST("/api/v1/auth/oidc/test", ac.OidcTest)

	// 1. 测试空 Issuer 参数校验
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oidc/test", strings.NewReader(`{"type":"oidc","issuer":""}`))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("空 Issuer 应返回 400，实际: %d", w1.Code)
	}

	// 2. 测试不可达域名并返回具体错误明细
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oidc/test", strings.NewReader(`{"type":"oidc","issuer":"https://127.0.0.1:54321"}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("不可达地址应返回 500，实际: %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "连接提供方失败") {
		t.Fatalf("期望返回清晰的连接提供方失败信息，实际: %s", w2.Body.String())
	}
}
