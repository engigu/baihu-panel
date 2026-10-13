package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
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
