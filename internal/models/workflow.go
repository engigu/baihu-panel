package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Workflow 代表一个可视化任务编排工作流
type Workflow struct {
	ID          string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name        string         `json:"name" gorm:"size:255;not null"`
	Description string         `json:"description" gorm:"size:1024;default:''"`
	Schedule    string         `json:"schedule" gorm:"size:100"`       // 整体重跑的 Cron 表达式
	Enabled     bool           `json:"enabled" gorm:"default:true"`    // 总开关
	FlowData    string         `json:"flow_data" gorm:"type:longtext"` // 包含前端 VueFlow 节点与连线的 JSON: {nodes: [], edges: []}
	LastRun     *LocalTime     `json:"last_run"`
	NextRun     *LocalTime     `json:"next_run"`
	CreatedAt   LocalTime      `json:"created_at"`
	UpdatedAt   LocalTime      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Workflow) TableName() string {
	return constant.TablePrefix + "workflows"
}

func (w *Workflow) BeforeCreate(tx *gorm.DB) (err error) {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return
}

