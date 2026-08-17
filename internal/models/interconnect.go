package models

import (
	"github.com/engigu/baihu-panel/internal/constant"
)

// NodeMetrics 表示互联节点的性能指标，使用 JSON 存储
type NodeMetrics struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
	DiskPercent float64 `json:"disk_percent"`
	TxBytes     uint64  `json:"tx_bytes,omitempty"`
	RxBytes     uint64  `json:"rx_bytes,omitempty"`
}

// InterconnectNode represents a connected remote panel
type InterconnectNode struct {
	ID              string      `json:"id" gorm:"primaryKey;size:20"`
	Name            string      `json:"name" gorm:"size:255;not null"`
	URL             string      `json:"url" gorm:"size:255;not null"`
	Token           string      `json:"token" gorm:"size:255"`
	Remark          string      `json:"remark" gorm:"size:500"`
	CreatedAt       LocalTime   `json:"created_at"`
	UpdatedAt       LocalTime   `json:"updated_at"`
	Status          string      `json:"status" gorm:"size:50"` // online / offline
	Metrics         NodeMetrics `json:"metrics" gorm:"serializer:json"`
	LastHeartbeatAt *LocalTime  `json:"last_heartbeat_at"`
}

func (InterconnectNode) TableName() string {
	return constant.TablePrefix + "interconnect_nodes"
}
