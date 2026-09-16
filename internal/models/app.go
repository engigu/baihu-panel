package models
import (
	"github.com/engigu/baihu-panel/internal/constant"
)
// App 代表一个安装到白虎面板中的声明式应用实体
type App struct {
	ID              string    `json:"id" gorm:"primaryKey;size:64"`             // 应用全局唯一标识符
	Name            string    `json:"name" gorm:"size:255;not null"`            // 应用显示名称
	Version         string    `json:"version" gorm:"size:64"`                   // 应用语义化版本号
	Author          string    `json:"author" gorm:"size:128"`                   // 作者或组织
	Category        string    `json:"category" gorm:"size:64;index"`            // 分类标签
	Description     string    `json:"description" gorm:"size:1024"`             // 详细描述
	Icon            string    `json:"icon" gorm:"size:512"`                     // 图标地址或内置代号
	Homepage        string    `json:"homepage" gorm:"size:512"`                 // 开源项目主页
	ManifestPath    string    `json:"manifest_path" gorm:"size:512"`            // YAML 来源路径或 URL
	ManifestRaw     BigText   `json:"manifest_raw"`                             // 完整的原始 YAML 清单
	CurrentScenario string    `json:"current_scenario" gorm:"size:64"`          // 当前激活的场景 ID
	Status          string    `json:"status" gorm:"size:32;default:'installed'"` // 安装状态: installed, error, disabled
	Config          BigText   `json:"config"`                                   // 扩展运行配置（存储用户环境变量与选定参数）
	CreatedAt       LocalTime `json:"created_at"`                               // 安装时间
	UpdatedAt       LocalTime `json:"updated_at"`                               // 最近更新时间
}

func (App) TableName() string {
	return constant.TablePrefix + "apps"
}