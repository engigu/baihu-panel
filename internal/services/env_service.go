package services

import (
	"strconv"
	"strings"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
)

type EnvService struct{}

func NewEnvService() *EnvService {
	return &EnvService{}
}

func (es *EnvService) CreateEnvVar(name, value, remark string, hidden bool, userUUID string) *models.EnvironmentVariable {
	env := &models.EnvironmentVariable{
		Name:   name,
		Value:  value,
		Remark: remark,
		Hidden: hidden,
		UserID: userUUID,
	}
	data := map[string]interface{}{
		"name":    name,
		"value":   value,
		"remark":  remark,
		"hidden":  hidden,
		"user_id": userUUID,
	}
	database.DB.Model(&models.EnvironmentVariable{}).Create(data)

	// 将自动生成的 ID 赋值回对象以便返回
	if id, ok := data["id"].(uint); ok {
		env.ID = id
	} else if id, ok := data["id"].(int64); ok {
		env.ID = uint(id)
	} else {
		// 如果 ID 没有自动回填到 map，尝试通过刚才的数据查出来
		database.DB.Where("name = ? AND user_id = ?", name, userUUID).Order("id DESC").First(env)
	}
	return env
}

func (es *EnvService) GetEnvVarsByUserID(userUUID string) []models.EnvironmentVariable {
	var envs []models.EnvironmentVariable
	database.DB.Where("user_id = ?", userUUID).Find(&envs)
	return envs
}

func (es *EnvService) GetEnvVarsWithPagination(userUUID string, name string, page, pageSize int) ([]models.EnvironmentVariable, int64) {
	var envs []models.EnvironmentVariable
	var total int64

	query := database.DB.Model(&models.EnvironmentVariable{}).Where("user_id = ?", userUUID)
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&envs)
	return envs, total
}

func (es *EnvService) GetEnvVarByID(id int) *models.EnvironmentVariable {
	var env models.EnvironmentVariable
	if err := database.DB.First(&env, id).Error; err != nil {
		return nil
	}
	return &env
}

func (es *EnvService) UpdateEnvVar(id int, name, value, remark string, hidden bool) *models.EnvironmentVariable {
	var env models.EnvironmentVariable
	if err := database.DB.First(&env, id).Error; err != nil {
		return nil
	}
	env.Name = name
	env.Value = value
	env.Remark = remark
	env.Hidden = hidden
	database.DB.Save(&env)
	return &env
}

func (es *EnvService) DeleteEnvVar(id int) bool {
	result := database.DB.Delete(&models.EnvironmentVariable{}, id)
	return result.RowsAffected > 0
}

// GetEnvVarsByIDs 根据逗号分隔的ID字符串获取环境变量列表，返回 NAME=VALUE 格式
func (es *EnvService) GetEnvVarsByIDs(envIDs string) []string {
	if envIDs == "" {
		return nil
	}

	var envVars []string
	ids := splitEnvIDs(envIDs)
	for _, id := range ids {
		env := es.GetEnvVarByID(id)
		if env != nil {
			envVars = append(envVars, env.Name+"="+env.Value)
		}
	}
	return envVars
}

// splitEnvIDs 解析逗号分隔的ID字符串
func splitEnvIDs(envIDs string) []int {
	var ids []int
	for _, s := range strings.Split(envIDs, ",") {
		s = strings.TrimSpace(s)
		if id, err := strconv.Atoi(s); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
