package models

import "time"

// ExportData 全量或部分业务数据的导出/导入结构
type ExportData struct {
	Version  string                `json:"version"`
	ExportAt LocalTime             `json:"export_at"`
	Tasks    []Task                `json:"tasks"`
	Envs     []EnvironmentVariable `json:"envs"`
	Tags     []DataStorage         `json:"tags"`
	Bindings []NotifyBinding       `json:"bindings"`
}

// NewExportData 创建一个导出数据对象
func NewExportData() *ExportData {
	return &ExportData{
		Version:  "1.0",
		ExportAt: LocalTime(time.Now()),
		Tasks:    make([]Task, 0),
		Envs:     make([]EnvironmentVariable, 0),
		Tags:     make([]DataStorage, 0),
		Bindings: make([]NotifyBinding, 0),
	}
}
