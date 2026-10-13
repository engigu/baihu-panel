package controllers

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/systime"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// fakeIdP 起一个最小可用的 OIDC 提供方：发现文档 + JWKS + token 端点
type fakeIdP struct {
	server   *httptest.Server
	key      *rsa.PrivateKey
	kid      string
	clientID string
	sub      string
	email    string
	username string
}

func newFakeIdP(t *testing.T, clientID, sub, email, username string) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 密钥失败: %v", err)
	}
	f := &fakeIdP{key: key, kid: "test-key-1", clientID: clientID, sub: sub, email: email, username: username}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                                f.server.URL,
			"authorization_endpoint":                f.server.URL + "/authorize",
			"token_endpoint":                        f.server.URL + "/token",
			"jwks_uri":                              f.server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "kid": f.kid, "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(f.key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.key.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		raw := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss": f.server.URL, "aud": f.clientID, "sub": f.sub,
			"email": f.email, "preferred_username": f.username, "name": f.username,
			"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
		})
		raw.Header["kid"] = f.kid
		signed, err := raw.SignedString(f.key)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]any{
			"access_token": "fake-access-token", "token_type": "Bearer",
			"id_token": signed, "expires_in": 3600,
		})
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func setupOidcTest(t *testing.T, idp *fakeIdP, autoCreate bool) *AuthController {
	t.Helper()
	gin.SetMode(gin.TestMode)
	time.Local = systime.CST

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("开启内存库失败: %v", err)
	}
	database.DB = db
	if err := db.AutoMigrate(&models.Setting{}, &models.User{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	settings := services.NewSettingsService()
	for k, v := range map[string]string{
		constant.KeyOidcEnabled:      "true",
		constant.KeyOidcIssuer:       idp.server.URL,
		constant.KeyOidcClientID:     idp.clientID,
		constant.KeyOidcClientSecret: "test-secret",
		constant.KeyOidcRedirectURL:  "http://panel.local/api/v1/auth/oidc/callback",
		constant.KeyOidcAutoCreate:   map[bool]string{true: "true", false: "false"}[autoCreate],
		constant.KeyOidcDefaultRole:  constant.DefaultRole,
	} {
		if err := settings.Set(constant.SectionSecurity, k, v); err != nil {
			t.Fatalf("写入设置 %s 失败: %v", k, err)
		}
	}

	return NewAuthController(services.NewUserService(), settings, services.NewLoginLogService())
}
