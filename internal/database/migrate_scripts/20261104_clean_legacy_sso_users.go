package migrate_scripts

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/rs/xid"
	"gorm.io/gorm"
)

// CleanLegacySSOUsers 自动清理存量历史 SSO 自动建号产生的多余账号，将其凭据、创建者、修改者及业务数据合并至本地主账号
func CleanLegacySSOUsers(db *gorm.DB, allModels []interface{}) {
	if HasMigrated(db, constant.KeyCleanLegacySSOUsersMigrated) {
		return
	}

	if !db.Migrator().HasTable(&models.User{}) {
		MarkMigrated(db, constant.KeyCleanLegacySSOUsersMigrated)
		return
	}

	var allUsers []models.User
	db.Order("created_at asc").Find(&allUsers)
	if len(allUsers) <= 1 {
		MarkMigrated(db, constant.KeyCleanLegacySSOUsersMigrated)
		return
	}

	// 1. 确定系统本地主账号 (Primary User)
	// 优先使用系统默认初始邮箱 (admin@local) 进行判断；若无，按默认用户名 admin 匹配；兜底取最早创建的账号
	var primaryUser *models.User
	for i := range allUsers {
		if allUsers[i].Email == "admin@local" {
			primaryUser = &allUsers[i]
			break
		}
	}
	if primaryUser == nil {
		for i := range allUsers {
			if allUsers[i].Username == "admin" {
				primaryUser = &allUsers[i]
				break
			}
		}
	}
	if primaryUser == nil {
		primaryUser = &allUsers[0]
	}

	// 2. 区分多余的 SSO 历史账号与候选本地账号
	var redundantUsers []models.User
	var candidateLocalUsers []models.User

	for _, u := range allUsers {
		if u.ID == primaryUser.ID {
			candidateLocalUsers = append(candidateLocalUsers, u)
			continue
		}
		// 历史 SSO 自动建号特征：拥有 oidc_subject，或者在 sys_user_sso_bindings 表中有记录
		isSSO := u.OidcSubject != ""
		if !isSSO && db.Migrator().HasTable(&models.UserSSOBinding{}) {
			var bindCount int64
			db.Model(&models.UserSSOBinding{}).Where("user_id = ?", u.ID).Count(&bindCount)
			if bindCount > 0 {
				isSSO = true
			}
		}
		if isSSO {
			redundantUsers = append(redundantUsers, u)
		} else {
			candidateLocalUsers = append(candidateLocalUsers, u)
		}
	}

	if len(redundantUsers) == 0 {
		MarkMigrated(db, constant.KeyCleanLegacySSOUsersMigrated)
		return
	}

	logger.Infof("[Database] 正在扫描并清理历史 SSO 遗留的多余账号（发现 %d 个候选账号）...", len(redundantUsers))

	for _, redundant := range redundantUsers {
		var targetUser *models.User

		// 规则 1：使用邮箱判断。如果候选本地账号中存在相同邮箱，则合并至该账号
		if redundant.Email != "" {
			for i := range candidateLocalUsers {
				if candidateLocalUsers[i].Email != "" && candidateLocalUsers[i].Email == redundant.Email {
					targetUser = &candidateLocalUsers[i]
					break
				}
			}
		}

		// 规则 2：如果邮箱未匹配，且候选本地账号只有唯一一个，就使用唯一的本地账号
		if targetUser == nil && len(candidateLocalUsers) == 1 {
			targetUser = &candidateLocalUsers[0]
		}

		// 兜底：若仍未确定，合并至主账号
		if targetUser == nil {
			targetUser = primaryUser
		}

		// 处理业务表：更新所有关联的 user_id、创建者 (created_by/creator) 与 修改者 (updated_by/modifier)
		transferUserReferences(db, allModels, redundant.ID, targetUser.ID)

		// 处理第三方 SSO 绑定关系表
		if db.Migrator().HasTable(&models.UserSSOBinding{}) {
			var bindings []models.UserSSOBinding
			db.Where("user_id = ?", redundant.ID).Find(&bindings)
			for _, b := range bindings {
				var count int64
				db.Model(&models.UserSSOBinding{}).
					Where("user_id = ? AND channel_type = ? AND subject = ?", targetUser.ID, b.ChannelType, b.Subject).
					Count(&count)
				if count > 0 {
					// 目标账号已有相同通道与 subject 的绑定，删除重复项
					db.Delete(&b)
				} else {
					// 将绑定过户给目标账号
					db.Model(&b).Updates(map[string]interface{}{
						"user_id":  targetUser.ID,
						"username": targetUser.Username,
					})
				}
			}

			// 若 redundant 用户的 oidc_subject 尚未在绑定表中记录，自动补齐到 targetUser
			if redundant.OidcSubject != "" {
				chType := "oidc"
				sub := redundant.OidcSubject
				if strings.HasPrefix(sub, "oauth2:") {
					chType = "oauth2"
					sub = strings.TrimPrefix(sub, "oauth2:")
				} else if strings.HasPrefix(sub, "oidc:") {
					sub = strings.TrimPrefix(sub, "oidc:")
				}
				var count int64
				db.Model(&models.UserSSOBinding{}).
					Where("user_id = ? AND channel_type = ? AND subject = ?", targetUser.ID, chType, sub).
					Count(&count)
				if count == 0 {
					db.Create(&models.UserSSOBinding{
						ID:           xid.New().String(),
						UserID:       targetUser.ID,
						Username:     targetUser.Username,
						ChannelType:  chType,
						Subject:      sub,
						IdentityName: redundant.Username,
						Email:        redundant.Email,
						BindType:     "oauth",
						CreatedAt:    models.Now(),
						UpdatedAt:    models.Now(),
					})
				}
			}
		}

		// 补齐/优化目标账号的凭据与邮箱信息
		if targetUser.OidcSubject == "" && redundant.OidcSubject != "" {
			db.Model(targetUser).Update("oidc_subject", redundant.OidcSubject)
			targetUser.OidcSubject = redundant.OidcSubject
		}
		if (targetUser.Email == "" || targetUser.Email == "admin@local") && redundant.Email != "" && redundant.Email != "admin@local" {
			db.Model(targetUser).Update("email", redundant.Email)
			targetUser.Email = redundant.Email
		}

		// 从 users 表删除多余账号
		db.Where("id = ?", redundant.ID).Delete(&models.User{})
		logger.Infof("[Database] 成功清理历史 SSO 多余账号 [%s](ID:%s)，凭据与关联业务已合并至本地账号 [%s](ID:%s)",
			redundant.Username, redundant.ID, targetUser.Username, targetUser.ID)
	}

	MarkMigrated(db, constant.KeyCleanLegacySSOUsersMigrated)
}

// transferUserReferences 遍历所有模型表，自动更新 user_id、创建者 (created_by/creator) 与修改者 (updated_by/modifier)
func transferUserReferences(db *gorm.DB, allModels []interface{}, oldUserID, newUserID string) {
	// 用户关联字段列表（包含用户ID、创建者、修改者）
	userFields := []string{
		"user_id",
		"created_by",
		"updated_by",
		"creator",
		"modifier",
		"create_by",
		"update_by",
	}

	for _, m := range allModels {
		// 跳过 User 本身和 UserSSOBinding（SSO 绑定有独立的去重与过户逻辑）
		if _, ok := m.(*models.User); ok {
			continue
		}
		if _, ok := m.(*models.UserSSOBinding); ok {
			continue
		}

		if !db.Migrator().HasTable(m) {
			continue
		}

		for _, field := range userFields {
			if db.Migrator().HasColumn(m, field) {
				db.Model(m).Where(field+" = ?", oldUserID).Update(field, newUserID)
			}
		}
	}
}

