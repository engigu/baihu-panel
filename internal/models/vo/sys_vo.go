package vo

import (
	"github.com/engigu/baihu-panel/internal/models"
)

// UserVO 用户视图对象
type UserVO struct {
	ID        uint             `json:"id"`
	UUID      string           `json:"uuid"`
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
		UUID:      user.UUID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// EnvVO 环境变量视图对象
type EnvVO struct {
	ID        uint             `json:"id"`
	UUID      string           `json:"uuid"`
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
		UUID:      env.UUID,
		Name:      env.Name,
		Value:     env.Value,
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
	ID        uint             `json:"id"`
	Username  string           `json:"username"`
	IP        string           `json:"ip"`
	UserAgent string           `json:"user_agent"`
	Status    string           `json:"status"`
	Message   string           `json:"message"`
	CreatedAt models.LocalTime `json:"created_at"`
}

// ToLoginLogVO 将 LoginLog 模型转换为 LoginLogVO
func ToLoginLogVO(log *models.LoginLog) *LoginLogVO {
	if log == nil {
		return nil
	}
	return &LoginLogVO{
		ID:        log.ID,
		Username:  log.Username,
		IP:        log.IP,
		UserAgent: log.UserAgent,
		Status:    log.Status,
		Message:   log.Message,
		CreatedAt: log.CreatedAt,
	}
}

// ToLoginLogVOList 将 LoginLog 模型列表转换为 LoginLogVO 列表
func ToLoginLogVOList(logs []*models.LoginLog) []*LoginLogVO {
	if logs == nil {
		return nil
	}
	vos := make([]*LoginLogVO, len(logs))
	for i, l := range logs {
		vos[i] = ToLoginLogVO(l)
	}
	return vos
}

// ToLoginLogVOListFromModels 将 LoginLog 模型列表转换为 LoginLogVO 列表
func ToLoginLogVOListFromModels(logs []models.LoginLog) []*LoginLogVO {
	vos := make([]*LoginLogVO, len(logs))
	for i := range logs {
		vos[i] = ToLoginLogVO(&logs[i])
	}
	return vos
}
