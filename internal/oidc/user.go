package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
)

// ErrUserNotLinked 提示第三方身份未绑定任何系统账号
var ErrUserNotLinked = errors.New("当前第三方身份未绑定任何系统账号，请先使用系统账号登录并在设置中授权绑定")

// Claims OIDC / OAuth 2.0 统一身份声明模型
type Claims struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Groups            []string `json:"groups"`
}

// ResolveUser 按 绑定映射表 → subject 精确匹配 → 邮箱绑定 查找本地系统账号（不自动创建新账号）
func ResolveUser(cfg *Config, claims *Claims, userSvc *services.UserService) (*models.User, bool, error) {
	if claims == nil || claims.Sub == "" {
		return nil, false, fmt.Errorf("身份标识 (sub) 不能为空")
	}

	qualifiedSub := fmt.Sprintf("%s:%s", cfg.Type, claims.Sub)

	var user models.User

	// 1. 优先从专属绑定映射表中匹配 (支持按 channel_type:sub 或纯 sub 匹配)
	var binding models.UserSSOBinding
	bindRes := database.DB.Where("(channel_type = ? AND subject = ?) OR subject = ? OR subject = ?", cfg.Type, claims.Sub, claims.Sub, qualifiedSub).
		Order("created_at desc").Limit(1).Find(&binding)
	if bindRes.Error == nil && bindRes.RowsAffected > 0 {
		if err := database.DB.Where("id = ?", binding.UserID).First(&user).Error; err == nil {
			return &user, false, nil
		}
	}

	// 2. 兼容历史主表的 oidc_subject（兼容纯 sub 或带命名空间前缀的记录）
	res := database.DB.Where("oidc_subject = ? OR oidc_subject = ?", claims.Sub, qualifiedSub).Limit(1).Find(&user)
	if res.Error == nil && res.RowsAffected > 0 {
		// 自动补齐绑定映射记录
		var count int64
		database.DB.Model(&models.UserSSOBinding{}).Where("user_id = ? AND subject = ?", user.ID, claims.Sub).Count(&count)
		if count == 0 {
			database.DB.Create(&models.UserSSOBinding{
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
			})
		}
		return &user, false, nil
	}

	// 3. 兼容已存在账号：邮箱相同且尚未绑定过任何第三方 subject，则自动关联
	if claims.Email != "" {
		res := database.DB.Where("email = ? AND (oidc_subject = '' OR oidc_subject IS NULL)", claims.Email).Limit(1).Find(&user)
		if res.Error == nil && res.RowsAffected > 0 {
			database.DB.Model(&user).Update("oidc_subject", claims.Sub)
			database.DB.Create(&models.UserSSOBinding{
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
			})
			return &user, false, nil
		}
	}

	// 未匹配到已绑定的本地账号，直接拒绝登录（不自动建号）
	return nil, false, ErrUserNotLinked
}

// FetchOAuth2Userinfo 通过 Access Token 调用第三方 Userinfo 端点获取用户信息
func FetchOAuth2Userinfo(ctx context.Context, cfg *Config, accessToken string) (*Claims, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.UserinfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建用户信息请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Baihu-Panel-SSO/1.0")

	client := NewSSOHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求用户信息接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("用户信息接口响应状态异常 (%d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, fmt.Errorf("读取用户信息失败: %w", err)
	}

	claims, err := ParseOAuth2Userinfo(body, cfg.UserKey)
	if err != nil {
		return nil, err
	}

	// 针对 GitHub 等平台：若未公开邮箱，尝试请求 /user/emails 补齐主邮箱
	if claims.Email == "" && strings.Contains(cfg.UserinfoURL, "api.github.com") {
		if ghEmail := FetchGitHubPrimaryEmail(ctx, accessToken); ghEmail != "" {
			claims.Email = ghEmail
		}
	}

	return claims, nil
}

// ParseOAuth2Userinfo 从任意 OAuth 2.0 用户信息 JSON 响应中智能提取身份字段
func ParseOAuth2Userinfo(raw []byte, customUserKey string) (*Claims, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("解析用户信息 JSON 失败: %w", err)
	}

	claims := &Claims{}

	// 1. 提取唯一标识 Sub
	if customUserKey != "" {
		if val, ok := data[customUserKey]; ok && val != nil {
			claims.Sub = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
	}
	if claims.Sub == "" {
		candidates := []string{"sub", "id", "user_id", "uid", "openid", "login", "username"}
		for _, key := range candidates {
			if val, ok := data[key]; ok && val != nil {
				str := strings.TrimSpace(fmt.Sprintf("%v", val))
				if str != "" && str != "<nil>" {
					claims.Sub = str
					break
				}
			}
		}
	}
	if claims.Sub == "" {
		return nil, fmt.Errorf("未能从用户信息中识别出唯一身份标识 (ID)")
	}

	// 2. 提取用户名
	userCandidates := []string{"login", "username", "preferred_username", "name", "nickname"}
	for _, key := range userCandidates {
		if val, ok := data[key]; ok && val != nil {
			str := strings.TrimSpace(fmt.Sprintf("%v", val))
			if str != "" && str != "<nil>" {
				claims.PreferredUsername = str
				break
			}
		}
	}
	if claims.PreferredUsername == "" {
		claims.PreferredUsername = claims.Sub
	}

	// 3. 提取邮箱
	emailCandidates := []string{"email", "mail", "user_email"}
	for _, key := range emailCandidates {
		if val, ok := data[key]; ok && val != nil {
			str := strings.TrimSpace(fmt.Sprintf("%v", val))
			if str != "" && strings.Contains(str, "@") {
				claims.Email = str
				break
			}
		}
	}

	// 4. 提取显示名称
	if val, ok := data["name"]; ok && val != nil {
		claims.Name = fmt.Sprintf("%v", val)
	}

	return claims, nil
}

// FetchGitHubPrimaryEmail 通过 GitHub /user/emails API 补充已验证的主邮箱
func FetchGitHubPrimaryEmail(ctx context.Context, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Baihu-Panel-SSO/1.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified && e.Email != "" {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}
