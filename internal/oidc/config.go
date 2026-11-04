package oidc

import (
	"fmt"
	"net/url"
	"strings"

	coreos_oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

const (
	// SecretMask 前端回显用的掩码；提交同名值时视为「未修改」，保留库中原值
	SecretMask = "********"

	// 默认单点登录端点常量
	DefaultOidcLoginPath      = constant.PathAuthOidcLogin
	DefaultOidcCallbackPath   = constant.PathAuthOidcCallback
	DefaultOAuth2LoginPath    = constant.PathAuthOAuth2Login
	DefaultOAuth2CallbackPath = constant.PathAuthOAuth2Callback
)

// MaskSecuritySection 读取 security 段时掩掉客户端密钥，避免密钥明文回显到浏览器
func MaskSecuritySection(section string, values map[string]string) map[string]string {
	if section != constant.SectionSecurity || values == nil {
		return values
	}
	if v, ok := values[constant.KeyOidcClientSecret]; ok && v != "" {
		values[constant.KeyOidcClientSecret] = SecretMask
	}
	if v, ok := values[constant.KeyOauth2ClientSecret]; ok && v != "" {
		values[constant.KeyOauth2ClientSecret] = SecretMask
	}
	return values
}

// PrepareSecuritySection 写入 security 段前处理客户端密钥：
// 掩码值 = 保留原值；明文 = 加密后落库；空串 = 清空
func PrepareSecuritySection(section string, values map[string]string, getter func(key string) string) map[string]string {
	if section != constant.SectionSecurity || values == nil {
		return values
	}
	for _, key := range []string{constant.KeyOidcClientSecret, constant.KeyOauth2ClientSecret} {
		raw, ok := values[key]
		if !ok {
			continue
		}
		switch raw {
		case SecretMask:
			// 未修改：原样写回当前值（可能是密文），避免把掩码存进库
			if getter != nil {
				values[key] = getter(key)
			}
		case "":
			values[key] = ""
		default:
			if encrypted, err := utils.Encrypt(raw); err == nil {
				values[key] = encrypted
			}
			// 未配置主密钥时 Encrypt 报错，此时按明文存储，DecryptSecret 会在读取时原样返回
		}
	}
	return values
}

// NormalizeIssuer 规范化 Issuer 地址，自动剔除用户误填的 /.well-known/openid-configuration 后缀与多余斜杠
func NormalizeIssuer(raw string) string {
	u := strings.TrimSpace(raw)
	u = strings.TrimRight(u, "/")
	for {
		trimmed := strings.TrimSuffix(u, "/.well-known/openid-configuration")
		trimmed = strings.TrimSuffix(trimmed, "/.well-known/openid-configuration/")
		trimmed = strings.TrimRight(trimmed, "/")
		if trimmed == u {
			break
		}
		u = trimmed
	}
	return u
}

// Config 单点登录（OIDC / 通用 OAuth 2.0）标准化配置模型
type Config struct {
	Type         string   `json:"type"` // "oidc" 或 "oauth2"
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`

	// 通用 OAuth 2.0 专用字段
	AuthURL     string `json:"auth_url"`
	TokenURL    string `json:"token_url"`
	UserinfoURL string `json:"userinfo_url"`
	DisplayName string `json:"display_name"`
	UserKey     string `json:"user_key"`
}

// SecurityReader 封装对 Security 配置段的高效安全读取、优先级回退与类型解析助手
type SecurityReader struct {
	ss *services.SettingsService
}

// NewSecurityReader 创建配置读取器
func NewSecurityReader(ss *services.SettingsService) SecurityReader {
	return SecurityReader{ss: ss}
}

func (r SecurityReader) Get(keys ...string) string {
	if r.ss == nil {
		return ""
	}
	for _, k := range keys {
		if v := strings.TrimSpace(r.ss.Get(constant.SectionSecurity, k)); v != "" {
			return v
		}
	}
	return ""
}

func (r SecurityReader) GetWithDefault(defaultVal string, keys ...string) string {
	if v := r.Get(keys...); v != "" {
		return v
	}
	return defaultVal
}

func (r SecurityReader) Bool(keys ...string) bool {
	return r.Get(keys...) == "true"
}

func (r SecurityReader) Secret(keys ...string) string {
	raw := r.Get(keys...)
	if raw == "" {
		return ""
	}
	return utils.DecryptSecret(raw)
}

func (r SecurityReader) Scopes(raw string, defaultScopes ...string) []string {
	seen := make(map[string]bool)
	var list []string
	for _, s := range defaultScopes {
		if s = strings.TrimSpace(s); s != "" && !seen[s] {
			seen[s] = true
			list = append(list, s)
		}
	}
	for _, s := range strings.FieldsFunc(raw, func(c rune) bool {
		return c == ',' || c == ' ' || c == '\n'
	}) {
		if s = strings.TrimSpace(s); s != "" && !seen[s] {
			seen[s] = true
			list = append(list, s)
		}
	}
	return list
}

// ResolveRedirectURL 解析回调地址：优先使用管理员配置；留空时根据当前请求动态推导标准回调地址
func ResolveRedirectURL(custom string, defaultPath string, c *gin.Context) string {
	if trimmed := strings.TrimSpace(custom); trimmed != "" {
		return trimmed
	}
	if defaultPath == "" {
		defaultPath = constant.PathAuthOidcCallback
	}
	if c != nil && c.Request != nil {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			// 自动适配前端开发调试模式 (如 Vite dev server 5173 转发) 或未设置头部的代理环境
			if ref := c.GetHeader("Referer"); ref != "" {
				if u, err := url.Parse(ref); err == nil && u.Host != "" {
					host = u.Host
					if u.Scheme != "" {
						scheme = u.Scheme
					}
				}
			} else if origin := c.GetHeader("Origin"); origin != "" {
				if u, err := url.Parse(origin); err == nil && u.Host != "" {
					host = u.Host
					if u.Scheme != "" {
						scheme = u.Scheme
					}
				}
			}
		}
		if host == "" {
			host = c.Request.Host
		}
		if host != "" {
			return fmt.Sprintf("%s://%s%s", scheme, host, defaultPath)
		}
	}
	return defaultPath
}

// ResolveSSOStatus 解析当前单点登录启用状态与展示信息（支持多身份源并行，用于 /public/settings）
func ResolveSSOStatus(settingsService *services.SettingsService) gin.H {
	if constant.DemoMode || settingsService == nil {
		return gin.H{"enabled": false, "providers": []gin.H{}}
	}

	r := NewSecurityReader(settingsService)
	var providers []gin.H

	// 1. 检查标准 OIDC 通道
	if r.Bool(constant.KeyOidcEnabled) {
		clientID := r.Get(constant.KeyOidcClientID)
		issuer := NormalizeIssuer(r.Get(constant.KeyOidcIssuer))
		if clientID != "" && issuer != "" {
			providers = append(providers, gin.H{
				"id":        "oidc",
				"type":      constant.SSOTypeOIDC,
				"name":      "OIDC",
				"login_url": constant.PathAuthOidcLogin,
			})
		}
	}

	// 2. 检查通用 OAuth 2.0 通道（支持独立开关或历史兼容开关）
	oauth2Enabled := r.Bool(constant.KeyOauth2Enabled) ||
		(r.Bool(constant.KeyOidcEnabled) && r.Get(constant.KeyOidcType) == constant.SSOTypeOAuth2)

	if oauth2Enabled {
		clientID := r.Get(constant.KeyOauth2ClientID, constant.KeyOidcClientID)
		authURL := r.Get(constant.KeyOauth2AuthURL)
		tokenURL := r.Get(constant.KeyOauth2TokenURL)
		userinfoURL := r.Get(constant.KeyOauth2UserinfoURL)
		if clientID != "" && authURL != "" && tokenURL != "" && userinfoURL != "" {
			providers = append(providers, gin.H{
				"id":        "oauth2",
				"type":      constant.SSOTypeOAuth2,
				"name":      r.GetWithDefault("OAuth 2.0", constant.KeyOauth2Name),
				"login_url": constant.PathAuthOAuth2Login,
			})
		}
	}

	if len(providers) == 0 {
		return gin.H{"enabled": false, "providers": providers}
	}

	// 兼容单通道旧字段回显
	return gin.H{
		"enabled":   true,
		"providers": providers,
		"type":      providers[0]["type"],
		"name":      providers[0]["name"],
	}
}

// LoadOidcConfig 读取标准 OIDC 配置
func LoadOidcConfig(ss *services.SettingsService, redirectURL string) *Config {
	r := NewSecurityReader(ss)
	if !r.Bool(constant.KeyOidcEnabled) {
		return nil
	}

	clientID := r.Get(constant.KeyOidcClientID)
	issuer := NormalizeIssuer(r.Get(constant.KeyOidcIssuer))
	if clientID == "" || issuer == "" {
		return nil
	}

	return &Config{
		Type:         constant.SSOTypeOIDC,
		Issuer:       issuer,
		ClientID:     clientID,
		ClientSecret: r.Secret(constant.KeyOidcClientSecret),
		RedirectURL:  redirectURL,
		Scopes:       r.Scopes(r.Get(constant.KeyOidcScopes), coreos_oidc.ScopeOpenID, "profile", "email"),
	}
}

// LoadOAuth2Config 读取通用 OAuth 2.0 配置
func LoadOAuth2Config(ss *services.SettingsService, redirectURL string) *Config {
	r := NewSecurityReader(ss)
	enabled := r.Bool(constant.KeyOauth2Enabled) ||
		(r.Bool(constant.KeyOidcEnabled) && r.Get(constant.KeyOidcType) == constant.SSOTypeOAuth2)
	if !enabled {
		return nil
	}

	clientID := r.Get(constant.KeyOauth2ClientID, constant.KeyOidcClientID)
	authURL := r.Get(constant.KeyOauth2AuthURL)
	tokenURL := r.Get(constant.KeyOauth2TokenURL)
	userinfoURL := r.Get(constant.KeyOauth2UserinfoURL)
	if clientID == "" || authURL == "" || tokenURL == "" || userinfoURL == "" {
		return nil
	}

	return &Config{
		Type:         constant.SSOTypeOAuth2,
		ClientID:     clientID,
		ClientSecret: r.Secret(constant.KeyOauth2ClientSecret, constant.KeyOidcClientSecret),
		RedirectURL:  redirectURL,
		Scopes:       r.Scopes(r.Get(constant.KeyOauth2Scopes, constant.KeyOidcScopes)),
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		UserinfoURL:  userinfoURL,
		DisplayName:  r.GetWithDefault("OAuth 2.0", constant.KeyOauth2Name),
		UserKey:      r.Get(constant.KeyOauth2UserKey),
	}
}

// ResolveActiveConfig 根据通道偏好（如 "oidc" 或 "oauth2"，留空时自动探测）及请求上下文动态解析当前生效的单点登录配置
func ResolveActiveConfig(ss *services.SettingsService, preferredType string, c *gin.Context) *Config {
	r := NewSecurityReader(ss)
	oidcRedirectURL := ResolveRedirectURL(r.Get(constant.KeyOidcRedirectURL), constant.PathAuthOidcCallback, c)
	oauth2RedirectURL := ResolveRedirectURL("", constant.PathAuthOAuth2Callback, c)

	if preferredType == constant.SSOTypeOAuth2 {
		if cfg := LoadOAuth2Config(ss, oauth2RedirectURL); cfg != nil {
			return cfg
		}
		return LoadOidcConfig(ss, oidcRedirectURL)
	}

	// 默认优先 OIDC，未开启时自动回退至 OAuth 2.0
	if cfg := LoadOidcConfig(ss, oidcRedirectURL); cfg != nil {
		return cfg
	}
	return LoadOAuth2Config(ss, oauth2RedirectURL)
}
