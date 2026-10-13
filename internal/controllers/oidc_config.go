package controllers

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookie    = "BHOIDCState"
	oidcVerifierCookie = "BHOIDCVerifier"
	oidcStateTTL       = 10 * time.Minute
)

// oidcProviderCache 缓存发现文档与 JWKS，避免每次登录都远程拉取
type oidcProviderCache struct {
	mu       sync.Mutex
	issuer   string
	clientID string
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    *oauth2.Config
}

var oidcCache = &oidcProviderCache{}

type oidcConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	AutoCreate   bool
	DefaultRole  string
}

// oidcConfig 读取并校验 OIDC 设置；未启用或配置不全时返回 nil
func (ac *AuthController) oidcConfig() *oidcConfig {
	if ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcEnabled) != "true" {
		return nil
	}
	issuer := strings.TrimRight(ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcIssuer), "/")
	clientID := ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcClientID)
	clientSecret := utils.DecryptSecret(ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcClientSecret))
	redirect := ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcRedirectURL)
	if issuer == "" || clientID == "" || redirect == "" {
		return nil
	}

	// 允许管理员追加 scope（逗号或空格分隔），始终包含 openid
	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	for _, s := range strings.FieldsFunc(ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcScopes), func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n'
	}) {
		if s = strings.TrimSpace(s); s != "" && s != oidc.ScopeOpenID {
			scopes = append(scopes, s)
		}
	}

	role := ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcDefaultRole)
	if role != constant.AdminRole {
		role = constant.DefaultRole
	}

	return &oidcConfig{
		Issuer:       issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirect,
		Scopes:       scopes,
		AutoCreate:   ac.settingsService.Get(constant.SectionSecurity, constant.KeyOidcAutoCreate) == "true",
		DefaultRole:  role,
	}
}

// oidcProviderFor 返回可用的 provider / verifier / oauth2 配置（设置变更后自动重建）
func (ac *AuthController) oidcProviderFor(ctx context.Context, cfg *oidcConfig) (*oidc.IDTokenVerifier, *oauth2.Config, error) {
	oidcCache.mu.Lock()
	defer oidcCache.mu.Unlock()

	if oidcCache.provider != nil && oidcCache.issuer == cfg.Issuer && oidcCache.clientID == cfg.ClientID {
		return oidcCache.verifier, oidcCache.oauth, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, nil, err
	}

	oidcCache.provider = provider
	oidcCache.verifier = provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	oidcCache.oauth = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
	}
	oidcCache.issuer = cfg.Issuer
	oidcCache.clientID = cfg.ClientID

	return oidcCache.verifier, oidcCache.oauth, nil
}

type oidcClaims struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Groups            []string `json:"groups"`
}

// 确保 services 包被引用（settingsService 类型来自该包）
var _ = services.NewSettingsService
