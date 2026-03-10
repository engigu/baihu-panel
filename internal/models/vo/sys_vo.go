package vo

import (
	"github.com/engigu/baihu-panel/internal/models"
)

// UserVO 用户视图对象
type UserVO struct {
	ID        string           `json:"id"`
	Username  string           `json:"username"`
	Email     string           `json:"email"`
	Role      string           `json:"role"`
	CreatedAt models.LocalTime `json:"created_at"`
	UpdatedAt models.LocalTime `json:"updated_at"`
}

// ToUserVO 将 User 模型转换为 UserVO
func ToUserVO(user *models.User) *UserVO {
	if user == nil {
		return nil
	}
	return &UserVO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// EnvVO 环境变量视图对象
type EnvVO struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Value     string           `json:"value"`
	Remark    string           `json:"remark"`
	Hidden    bool             `json:"hidden"`
	CreatedAt models.LocalTime `json:"created_at"`
	UpdatedAt models.LocalTime `json:"updated_at"`
}

// ToEnvVO 将 Env 模型转换为 EnvVO
func ToEnvVO(env *models.EnvironmentVariable) *EnvVO {
	if env == nil {
		return nil
	}
	return &EnvVO{
		ID:        env.ID,
		Name:      env.Name,
		Value:     string(env.Value),
		Remark:    env.Remark,
		Hidden:    env.Hidden,
		CreatedAt: env.CreatedAt,
		UpdatedAt: env.UpdatedAt,
	}
}

// ToEnvVOList 将 Env 模型列表转换为 EnvVO 列表
func ToEnvVOList(envs []*models.EnvironmentVariable) []*EnvVO {
	if envs == nil {
		return nil
	}
	vos := make([]*EnvVO, len(envs))
	for i, e := range envs {
		vos[i] = ToEnvVO(e)
	}
	return vos
}

// ToEnvVOListFromModels 将 Env 模型列表转换为 EnvVO 列表
func ToEnvVOListFromModels(envs []models.EnvironmentVariable) []*EnvVO {
	vos := make([]*EnvVO, len(envs))
	for i := range envs {
		vos[i] = ToEnvVO(&envs[i])
	}
	return vos
}

// LoginLogVO 登录日志视图对象
type LoginLogVO struct {
	ID        string           `json:"id"`
	Username  string           `json:"username"`
	IP        string           `json:"ip"`
	UserAgent string           `json:"user_agent"`
	Status    string           `json:"status"`
	Message   string           `json:"message"`
	CreatedAt models.LocalTime `json:"created_at"`
}

// TokenConfig Token 配置结构体
type TokenConfig struct {
	Enabled  bool   `json:"enabled"`
	Token    string `json:"token"`
	ExpireAt string `json:"expire_at"`
}
