package services

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"gorm.io/gorm"
)

type EnvService struct{}

func NewEnvService() *EnvService {
	return &EnvService{}
}

func (es *EnvService) CreateEnvVar(name, value, remark string, hidden bool, userID string) *models.EnvironmentVariable {
	env := &models.EnvironmentVariable{
		ID:        utils.GenerateID(),
		Name:      name,
		Value:     models.BigText(value),
		Remark:    remark,
		Hidden:    hidden,
		UserID:    userID,
		CreatedAt: models.Now(),
		UpdatedAt: models.Now(),
	}
	database.DB.Create(env)
	return env
}

func (es *EnvService) GetEnvVarsByUserID(userID string) []models.EnvironmentVariable {
	var envs []models.EnvironmentVariable
	database.DB.Where("user_id = ?", userID).Find(&envs)
	return envs
}

// GetFormattedEnvVarsByUserID 获取用户环境变量并格式化为 NAME=VALUE 格式（支持重名合并）
func (es *EnvService) GetFormattedEnvVarsByUserID(userID string) []string {
	envs := es.GetEnvVarsByUserID(userID)
	return es.formatEnvVars(envs)
}

func (es *EnvService) GetEnvVarsWithPagination(userID string, name string, page, pageSize int) ([]models.EnvironmentVariable, int64) {
	var envs []models.EnvironmentVariable
	var total int64

	query := database.DB.Model(&models.EnvironmentVariable{}).Where("user_id = ?", userID)
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&envs)
	return envs, total
}

func (es *EnvService) GetEnvVarByID(id string) *models.EnvironmentVariable {
	var env models.EnvironmentVariable
	if err := database.DB.Where("id = ?", id).First(&env).Error; err != nil {
		return nil
	}
	return &env
}

func (es *EnvService) UpdateEnvVar(id string, name, value, remark string, hidden bool) *models.EnvironmentVariable {
	var env models.EnvironmentVariable
	if err := database.DB.Where("id = ?", id).First(&env).Error; err != nil {
		return nil
	}
	updates := map[string]interface{}{
		"name":   name,
		"value":  models.BigText(value),
		"remark": remark,
		"hidden": hidden,
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
		return err == nil, nil
	}

	result := database.DB.Where("id = ?", id).Delete(&models.EnvironmentVariable{})
	return result.RowsAffected > 0, nil
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

// GetAllEnvVars 获取系统中所有的环境变量，并按 NAME=VALUE 格式返回（支持重名合并）
func (es *EnvService) GetAllEnvVars() []string {
	var envs []models.EnvironmentVariable
	if err := database.DB.Find(&envs).Error; err != nil {
		return nil
	}
	return es.formatEnvVars(envs)
}

// formatEnvVars 将环境变量列表格式化为 NAME=VALUE 数组，并处理重名合并
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
		if idx, ok := nameToIndex[env.Name]; ok {
			mergedList[idx].values = append(mergedList[idx].values, string(env.Value))
		} else {
			nameToIndex[env.Name] = len(mergedList)
			mergedList = append(mergedList, mergedEnv{
				name:   env.Name,
				values: []string{string(env.Value)},
			})
		}
	}

	var result []string
	for _, item := range mergedList {
		// 多个值使用 & 拼接
		val := strings.Join(item.values, "&")
		result = append(result, item.name+"="+val)
	}
	return result
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
