package vo

import (
	"github.com/engigu/baihu-panel/internal/models"
)

// ScriptVO 脚本视图对象
type ScriptVO struct {
	ID        uint             `json:"id"`
	UUID      string           `json:"uuid"`
	Name      string           `json:"name"`
	Content   string           `json:"content,omitempty"` // 仅在拉取详情时返回
	CreatedAt models.LocalTime `json:"created_at"`
	UpdatedAt models.LocalTime `json:"updated_at"`
}

// ToScriptVO 将 Script 模型转换为 ScriptVO
func ToScriptVO(script *models.Script) *ScriptVO {
	if script == nil {
		return nil
	}
	return &ScriptVO{
		ID:        script.ID,
		UUID:      script.UUID,
		Name:      script.Name,
		Content:   script.Content,
		CreatedAt: script.CreatedAt,
		UpdatedAt: script.UpdatedAt,
	}
}

// ToScriptVOListFromModels 将 Script 模型列表转换为 ScriptVO 列表
func ToScriptVOListFromModels(scripts []models.Script) []*ScriptVO {
	vos := make([]*ScriptVO, len(scripts))
	for i := range scripts {
		vos[i] = ToScriptVO(&scripts[i])
	}
	return vos
}
