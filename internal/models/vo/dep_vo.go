package vo

import (
	"time"

	"github.com/engigu/baihu-panel/internal/models"
)

// DependencyVO 依赖包视图对象
type DependencyVO struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Language    string    `json:"language"`
	LangVersion string    `json:"lang_version"`
	Remark      string    `json:"remark"`
	Log         string    `json:"log,omitempty"` // 仅在需要时返回
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToDependencyVO 将 Dependency 模型转换为 DependencyVO
func ToDependencyVO(dep *models.Dependency) *DependencyVO {
	if dep == nil {
		return nil
	}
	return &DependencyVO{
		ID:          dep.ID,
		Name:        dep.Name,
		Version:     dep.Version,
		Language:    dep.Language,
		LangVersion: dep.LangVersion,
		Remark:      dep.Remark,
		Log:         dep.Log,
		CreatedAt:   dep.CreatedAt,
		UpdatedAt:   dep.UpdatedAt,
	}
}

// ToDependencyVOListFromModels 将 Dependency 模型列表转换为 DependencyVO 列表
func ToDependencyVOListFromModels(deps []models.Dependency) []*DependencyVO {
	vos := make([]*DependencyVO, len(deps))
	for i := range deps {
		vos[i] = ToDependencyVO(&deps[i])
	}
	return vos
}
