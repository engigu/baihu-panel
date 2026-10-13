package controllers

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

func (ac *AuthController) logOidcFailure(c *gin.Context, username, message string) {
	eventbus.DefaultBus.Publish(eventbus.Event{
		Type: constant.EventUserLogin,
		Payload: map[string]interface{}{
			"ip":        c.ClientIP(),
			"username":  username,
			"userAgent": c.GetHeader("User-Agent"),
			"status":    "failed",
			"message":   "OIDC 登录失败：" + message,
		},
	})
}

// resolveOidcUser 按 subject 精确匹配 → 邮箱绑定 → 按配置自动创建
func (ac *AuthController) resolveOidcUser(cfg *oidcConfig, claims *oidcClaims) (*models.User, bool) {
	if claims.Sub == "" {
		return nil, false
	}

	var user models.User
	res := database.DB.Where("oidc_subject = ?", claims.Sub).Limit(1).Find(&user)
	if res.Error == nil && res.RowsAffected > 0 {
		return &user, false
	}

	// 兼容已存在账号：邮箱相同且尚未绑定过任何 subject，则自动绑定
	if claims.Email != "" {
		res := database.DB.Where("email = ? AND (oidc_subject = '' OR oidc_subject IS NULL)", claims.Email).Limit(1).Find(&user)
		if res.Error == nil && res.RowsAffected > 0 {
			database.DB.Model(&user).Update("oidc_subject", claims.Sub)
			return &user, false
		}
	}

	if !cfg.AutoCreate {
		return nil, false
	}

	username := claims.PreferredUsername
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		username = "oidc_" + claims.Sub
	}

	created := ac.userService.CreateOidcUser(ac.uniqueUsername(username), claims.Email, cfg.DefaultRole, claims.Sub)
	if created == nil {
		return nil, false
	}
	return created, true
}

func (ac *AuthController) uniqueUsername(base string) string {
	candidate := base
	for i := 0; i < 20; i++ {
		var count int64
		database.DB.Model(&models.User{}).Where("username = ?", candidate).Count(&count)
		if count == 0 {
			return candidate
		}
		candidate = base + "_" + utils.RandomString(4)
	}
	return base + "_" + utils.RandomString(8)
}
