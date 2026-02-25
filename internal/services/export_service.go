package services

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/systime"
	"github.com/engigu/baihu-panel/internal/utils"
)

const (
	ExportSection = "export"
	ExportFileKey = "export_file"
)

type ExportImportService struct {
	fileRecordService *FileRecordService
}

func NewExportImportService() *ExportImportService {
	return &ExportImportService{
		fileRecordService: NewFileRecordService(),
	}
}

// ExportParams 导出的参数，指定要导出的 ID
type ExportParams struct {
	ScriptPaths []string `json:"script_paths"`
	EnvIDs      []uint   `json:"env_ids"`
	TaskIDs     []uint   `json:"task_ids"`
	DepIDs      []uint   `json:"dep_ids"`
	Password    string   `json:"password"`
}

type ExportInfo struct {
	Version   string `json:"version"`
	ExportAt  string `json:"export_at"`
	HasScript bool   `json:"has_script"`
	HasEnv    bool   `json:"has_env"`
	HasTask   bool   `json:"has_task"`
	HasDep    bool   `json:"has_dep"`
}

func (s *ExportImportService) ExportData(params ExportParams) (string, error) {
	if err := os.MkdirAll(BackupDir, 0755); err != nil {
		return "", err
	}

	timestamp := systime.FormatDatetime(time.Now())
	timestampStr := strings.ReplaceAll(timestamp, ":", "")
	timestampStr = strings.ReplaceAll(timestampStr, " ", "_")
	zipPath := filepath.Join(BackupDir, fmt.Sprintf("baihushare_%s.zip", timestampStr))

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zw := utils.NewZipWriter(zipFile, params.Password)
	defer zw.Close()

	info := ExportInfo{
		Version:  "v1.0",
		ExportAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 1. Export Scripts
	if len(params.ScriptPaths) > 0 {
		info.HasScript = true
		scriptsDir := constant.ScriptsWorkDir
		for _, relPath := range params.ScriptPaths {
			if relPath == "" {
				continue
			}
			if strings.HasPrefix(relPath, "/") {
				relPath = relPath[1:]
			}
			fullPath := filepath.Join(scriptsDir, relPath)
			if _, err := os.Stat(fullPath); err == nil {
				w, err := zw.Create("export_scripts/" + relPath)
				if err == nil {
					if f, err := os.Open(fullPath); err == nil {
						io.Copy(w, f)
						f.Close()
					}
				}
			}
		}
	}

	// 2. Export Envs (mise format)
	if len(params.EnvIDs) > 0 {
		var envs []models.EnvironmentVariable
		if err := database.DB.Where("id IN ?", params.EnvIDs).Find(&envs).Error; err == nil && len(envs) > 0 {
			info.HasEnv = true
			w, err := zw.Create("env.yml")
			if err == nil {
				w.Write([]byte("env:\n"))
				for _, env := range envs {
					// 简单过滤包含双引号的情况
					val := strings.ReplaceAll(env.Value, "\"", "\\\"")
					w.Write([]byte(fmt.Sprintf("  %s: \"%s\"\n", env.Name, val)))
				}
			}
			// 同时导出一份 json 用来导入恢复原始信息
			w2, err := zw.Create("envs_meta.json")
			if err == nil {
				data, _ := json.MarshalIndent(envs, "", "  ")
				w2.Write(data)
			}
		}
	}

	// 3. Export Tasks
	if len(params.TaskIDs) > 0 {
		var tasks []models.Task
		if err := database.DB.Where("id IN ?", params.TaskIDs).Find(&tasks).Error; err == nil && len(tasks) > 0 {
			info.HasTask = true
			w, err := zw.Create("task.yml")
			if err == nil {
				w.Write([]byte("tasks:\n"))
				for _, task := range tasks {
					w.Write([]byte(fmt.Sprintf("  - name: \"%s\"\n", strings.ReplaceAll(task.Name, "\"", "\\\""))))
					w.Write([]byte(fmt.Sprintf("    command: \"%s\"\n", strings.ReplaceAll(task.Command, "\"", "\\\""))))
					w.Write([]byte(fmt.Sprintf("    schedule: \"%s\"\n", task.Schedule)))
				}
			}
			w2, err := zw.Create("tasks_meta.json")
			if err == nil {
				data, _ := json.MarshalIndent(tasks, "", "  ")
				w2.Write(data)
			}
		}
	}

	// 3.5 Export Dependencies
	if len(params.DepIDs) > 0 {
		var deps []models.Dependency
		if err := database.DB.Where("id IN ?", params.DepIDs).Find(&deps).Error; err == nil && len(deps) > 0 {
			info.HasDep = true
			w, err := zw.Create("deps.yml")
			if err == nil {
				w.Write([]byte("dependencies:\n"))
				for _, dep := range deps {
					w.Write([]byte(fmt.Sprintf("  - name: \"%s\"\n", strings.ReplaceAll(dep.Name, "\"", "\\\""))))
					if dep.Version != "" {
						w.Write([]byte(fmt.Sprintf("    version: \"%s\"\n", strings.ReplaceAll(dep.Version, "\"", "\\\""))))
					}
					w.Write([]byte(fmt.Sprintf("    language: \"%s\"\n", dep.Language)))
					if dep.LangVersion != "" {
						w.Write([]byte(fmt.Sprintf("    lang_version: \"%s\"\n", dep.LangVersion)))
					}
				}
			}
			w2, err := zw.Create("deps_meta.json")
			if err == nil {
				data, _ := json.MarshalIndent(deps, "", "  ")
				w2.Write(data)
			}
		}
	}

	// 4. Export Info
	wInfo, err := zw.Create("info.json")
	if err == nil {
		data, _ := json.MarshalIndent(info, "", "  ")
		wInfo.Write(data)
	}

	s.fileRecordService.SaveTempFileRecord(ExportSection, ExportFileKey, zipPath, 5*time.Minute)
	return zipPath, nil
}

func (s *ExportImportService) GetExportFile() string {
	return s.fileRecordService.GetFileRecord(ExportSection, ExportFileKey)
}
