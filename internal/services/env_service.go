package services

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"gorm.io/gorm"
)

type EnvService struct{}

func NewEnvService() *EnvService {
	return &EnvService{}
}

func (es *EnvService) CreateEnvVar(name, value, remark, envType string, hidden, enabled bool, userID string) *models.EnvironmentVariable {
	if envType == constant.EnvTypeSecret {
		if encValue, err := utils.Encrypt(value); err == nil {
			value = encValue
		}
	}

	env := &models.EnvironmentVariable{
		ID:        utils.GenerateID(),
		Name:      name,
		Value:     models.BigText(value),
		Remark:    remark,
		Type:      envType,
		Hidden:    &hidden,
		Enabled:   &enabled,
		UserID:    userID,
		CreatedAt: models.Now(),
		UpdatedAt: models.Now(),
	}
	database.DB.Select("*").Create(env)
	return env
}

func (es *EnvService) GetEnvVarsByUserID(userID string) []models.EnvironmentVariable {
	var envs []models.EnvironmentVariable
	database.DB.Where("user_id = ?", userID).Find(&envs)
	es.LoadEnvTags(envs)
	return envs
}

// GetFormattedEnvVarsByUserID 获取用户环境变量并格式化为 NAME=VALUE 格式（支持重名合并）
func (es *EnvService) GetFormattedEnvVarsByUserID(userID string) []string {
	envs := es.GetEnvVarsByUserID(userID)
	return es.formatEnvVars(envs)
}

func (es *EnvService) GetEnvVarsWithPagination(userID string, name string, envType string, tags string, page, pageSize int) ([]models.EnvironmentVariable, int64) {
	var envs []models.EnvironmentVariable
	var total int64

	query := database.DB.Model(&models.EnvironmentVariable{}).Where("user_id = ?", userID)
	if name != "" {
		query = query.Where("name LIKE ? OR remark LIKE ?", "%"+name+"%", "%"+name+"%")
	}
	if envType != "" {
		query = query.Where("type = ?", envType)
	}

	if tags != "" {
		tagList := strings.Split(tags, ",")
		var validTags []string
		for _, t := range tagList {
			t = strings.TrimSpace(t)
			if t != "" {
				validTags = append(validTags, t)
			}
		}
		if len(validTags) > 0 {
			var storageIDs []string
			database.DB.Model(&models.DataStorage{}).Where("type = ? AND name IN ?", "env_tag", validTags).Pluck("id", &storageIDs)
			
			var envIDs []string
			if len(storageIDs) > 0 {
				database.DB.Model(&models.DataRelation{}).Where("type = ? AND relate_id IN ?", "env_tag", storageIDs).Pluck("data_id", &envIDs)
			}
			
			if len(envIDs) > 0 {
				query = query.Where("id IN ?", envIDs)
			} else {
				query = query.Where("1 = 0")
			}
		}
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&envs)
	es.LoadEnvTags(envs)
	return envs, total
}

func (es *EnvService) GetEnvVarByID(id string) *models.EnvironmentVariable {
	var env models.EnvironmentVariable
	res := database.DB.Where("id = ?", id).Limit(1).Find(&env)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}
	envs := []models.EnvironmentVariable{env}
	es.LoadEnvTags(envs)
	return &envs[0]
}

func (es *EnvService) UpdateEnvVar(id string, name, value, remark, envType string, hidden, enabled bool) *models.EnvironmentVariable {
	var env models.EnvironmentVariable
	res := database.DB.Where("id = ?", id).Limit(1).Find(&env)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil
	}

	if envType == constant.EnvTypeSecret && value != "********" && value != "" {
		if encValue, err := utils.Encrypt(value); err == nil {
			value = encValue
		}
	} else if envType == constant.EnvTypeSecret && (value == "********" || value == "") {
		// Keep the original encrypted value
		value = string(env.Value)
	}

	updates := map[string]interface{}{
		"name":    name,
		"value":   models.BigText(value),
		"remark":  remark,
		"type":    envType,
		"hidden":  &hidden,
		"enabled": &enabled,
	}
	database.DB.Model(&env).Updates(updates)
	return &env
}

func (es *EnvService) GetAssociatedTasks(id string) []models.Task {
	var associatedTasks []models.Task
	query := "envs = ? OR envs LIKE ? OR envs LIKE ? OR envs LIKE ?"
	database.DB.Where(query, id, id+",%", "%,"+id, "%,"+id+",%").Find(&associatedTasks)
	return associatedTasks
}

func (es *EnvService) DeleteEnvVar(id string, force bool) (bool, []models.Task) {
	associatedTasks := es.GetAssociatedTasks(id)

	if len(associatedTasks) > 0 && !force {
		return false, associatedTasks
	}

	if force {
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Update tasks to remove this env ID
			for _, task := range associatedTasks {
				ids := splitEnvIDs(string(task.Envs))
				var newIDs []string
				for _, eid := range ids {
					if eid != id {
						newIDs = append(newIDs, eid)
					}
				}
				newEnvs := strings.Join(newIDs, ",")
				if err := tx.Model(&task).Update("envs", newEnvs).Error; err != nil {
					return err
				}
			}
			// Delete the env var
			if err := tx.Where("id = ?", id).Delete(&models.EnvironmentVariable{}).Error; err != nil {
				return err
			}
			return nil
		})
		if err == nil {
			es.CleanEnvTags(id)
			return true, nil
		}
		return false, nil
	}

	result := database.DB.Where("id = ?", id).Delete(&models.EnvironmentVariable{})
	if result.RowsAffected > 0 {
		es.CleanEnvTags(id)
		return true, nil
	}
	return false, nil
}

// GetEnvVarsByIDs 根据逗号分隔的ID字符串获取环境变量列表，返回 NAME=VALUE 格式
// 如果存在重名变量，会类似青龙面板一样使用 & 拼接
func (es *EnvService) GetEnvVarsByIDs(envIDs string) []string {
	if envIDs == "" {
		return nil
	}

	ids := splitEnvIDs(envIDs)
	var envs []models.EnvironmentVariable
	for _, id := range ids {
		env := es.GetEnvVarByID(id)
		if env != nil {
			envs = append(envs, *env)
		}
	}

	return es.formatEnvVars(envs)
}

// GetEnvVarsAndSecretsByIDs 根据逗号分隔的ID字符串获取环境变量列表和安全机密值列表
func (es *EnvService) GetEnvVarsAndSecretsByIDs(envIDs string) ([]string, []string) {
	if envIDs == "" {
		return nil, nil
	}

	ids := splitEnvIDs(envIDs)
	var envs []models.EnvironmentVariable
	for _, id := range ids {
		env := es.GetEnvVarByID(id)
		if env != nil {
			envs = append(envs, *env)
		}
	}

	return es.formatEnvVarsAndSecrets(envs)
}

// GetAllEnvVars获取系统中所有的环境变量，并按 NAME=VALUE 格式返回
func (es *EnvService) GetAllEnvVars() []string {
	var envs []models.EnvironmentVariable
	if err := database.DB.Find(&envs).Error; err != nil {
		return nil
	}
	return es.formatEnvVars(envs)
}

// GetAllEnvVarsAndSecrets 获取系统中所有的环境变量和安全机密值列表
func (es *EnvService) GetAllEnvVarsAndSecrets() ([]string, []string) {
	var envs []models.EnvironmentVariable
	if err := database.DB.Find(&envs).Error; err != nil {
		return nil, nil
	}
	return es.formatEnvVarsAndSecrets(envs)
}

// formatEnvVars 将环境变量列表格式化为 NAME=VALUE 数组，并处理重名合并 (过滤掉所有的 Secret)
func (es *EnvService) formatEnvVars(envs []models.EnvironmentVariable) []string {
	if len(envs) == 0 {
		return nil
	}

	type mergedEnv struct {
		name   string
		values []string
	}
	var mergedList []mergedEnv
	nameToIndex := make(map[string]int)

	for _, env := range envs {
		// 非调度器入口，直接当做没有（跳过 Secret）
		if env.Type == constant.EnvTypeSecret {
			continue
		}

		value := string(env.Value)
		if !utils.DerefBool(env.Enabled, true) {
			value = ""
		}

		if idx, ok := nameToIndex[env.Name]; ok {
			mergedList[idx].values = append(mergedList[idx].values, value)
		} else {
			nameToIndex[env.Name] = len(mergedList)
			mergedList = append(mergedList, mergedEnv{
				name:   env.Name,
				values: []string{value},
			})
		}
	}

	var result []string
	for _, item := range mergedList {
		val := strings.Join(item.values, "&")
		result = append(result, item.name+"="+val)
	}
	return result
}

// formatEnvVarsAndSecrets 将环境变量列表格式化为 NAME=VALUE 数组，并提取明文安全机密列表
func (es *EnvService) formatEnvVarsAndSecrets(envs []models.EnvironmentVariable) ([]string, []string) {
	if len(envs) == 0 {
		return nil, nil
	}

	type mergedEnv struct {
		name   string
		values []string
	}
	var mergedList []mergedEnv
	var secrets []string
	nameToIndex := make(map[string]int)

	for _, env := range envs {
		value := string(env.Value)
		if env.Type == constant.EnvTypeSecret {
			if decValue, err := utils.Decrypt(value); err == nil {
				value = decValue
				if utils.DerefBool(env.Enabled, true) && value != "" {
					secrets = append(secrets, value)
				}
			}
		}

		if !utils.DerefBool(env.Enabled, true) {
			value = ""
		}

		if idx, ok := nameToIndex[env.Name]; ok {
			mergedList[idx].values = append(mergedList[idx].values, value)
		} else {
			nameToIndex[env.Name] = len(mergedList)
			mergedList = append(mergedList, mergedEnv{
				name:   env.Name,
				values: []string{value},
			})
		}
	}

	var result []string
	for _, item := range mergedList {
		// 多个值使用 & 拼接
		val := strings.Join(item.values, "&")
		result = append(result, item.name+"="+val)
	}
	return result, secrets
}

// splitEnvIDs 解析逗号分隔的ID字符串
func splitEnvIDs(envIDs string) []string {
	var ids []string
	for _, s := range strings.Split(envIDs, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			ids = append(ids, s)
		}
	}
	return ids
}

// SaveEnvTags 保存环境变量标签
func (es *EnvService) SaveEnvTags(envID string, tagsStr string) {
	database.DB.Where("data_id = ? AND type = ?", envID, "env_tag").Delete(&models.DataRelation{})
	if tagsStr == "" {
		return
	}
	tags := strings.Split(tagsStr, ",")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		var storage models.DataStorage
		res := database.DB.Where("type = ? AND name = ?", "env_tag", tag).Limit(1).Find(&storage)
		if res.RowsAffected == 0 {
			storage = models.DataStorage{
				ID:        utils.GenerateID(),
				Type:      "env_tag",
				Name:      tag,
				CreatedAt: models.Now(),
				UpdatedAt: models.Now(),
			}
			database.DB.Create(&storage)
		}
		relation := models.DataRelation{
			ID:        utils.GenerateID(),
			DataID:    envID,
			RelateID:  storage.ID,
			Type:      "env_tag",
			CreatedAt: models.Now(),
			UpdatedAt: models.Now(),
		}
		database.DB.Create(&relation)
	}
}

// LoadEnvTags 为环境变量列表加载标签
func (es *EnvService) LoadEnvTags(envs []models.EnvironmentVariable) {
	if len(envs) == 0 {
		return
	}
	envIDs := make([]string, len(envs))
	for i, e := range envs {
		envIDs[i] = e.ID
	}
	var relations []models.DataRelation
	database.DB.Where("data_id IN ? AND type = ?", envIDs, "env_tag").Find(&relations)
	if len(relations) == 0 {
		return
	}
	relateIDs := make([]string, len(relations))
	for i, r := range relations {
		relateIDs[i] = r.RelateID
	}
	var storages []models.DataStorage
	database.DB.Where("id IN ? AND type = ?", relateIDs, "env_tag").Find(&storages)

	storageMap := make(map[string]string)
	for _, s := range storages {
		storageMap[s.ID] = s.Name
	}

	envTagsMap := make(map[string][]string)
	for _, r := range relations {
		if name, ok := storageMap[r.RelateID]; ok {
			envTagsMap[r.DataID] = append(envTagsMap[r.DataID], name)
		}
	}

	for i, e := range envs {
		if tags, ok := envTagsMap[e.ID]; ok {
			envs[i].Tags = strings.Join(tags, ",")
		}
	}
}

// GetAllEnvTags 获取所有环境变量标签
func (es *EnvService) GetAllEnvTags() ([]string, error) {
	var tags []string
	err := database.DB.Model(&models.DataStorage{}).Where("type = ?", "env_tag").Pluck("name", &tags).Error
	return tags, err
}

// CleanEnvTags 删除环境变量时清理关联标签记录
func (es *EnvService) CleanEnvTags(id string) {
	database.DB.Where("data_id = ? AND type = ?", id, "env_tag").Delete(&models.DataRelation{})
}

