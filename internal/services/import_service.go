package services

import (
	"encoding/json"
	"fmt"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
	yeka_zip "github.com/yeka/zip"
	"gorm.io/gorm"
)

// ImportParams 导入参数
type ImportParams struct {
	ZipPath      string `json:"zip_path"`
	Password     string `json:"password"`
	ImportScript bool   `json:"import_script"`
	ImportEnv    bool   `json:"import_env"`
	ImportTask   bool   `json:"import_task"`
	ImportDep    bool   `json:"import_dep"`
}

func (s *ExportImportService) ImportData(params ImportParams) error {
	zrc, err := utils.OpenZipReader(params.ZipPath, params.Password)
	if err != nil {
		return err
	}
	defer zrc.Close()

	fileMap := make(map[string]*yeka_zip.File)
	for _, f := range zrc.GetFiles() {
		fileMap[f.Name] = f
	}

	if _, ok := fileMap["info.json"]; !ok {
		return fmt.Errorf("invalid export file format: info.json not found")
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		if params.ImportScript {
			// Extract scripts dir
			scriptsDir := constant.ScriptsWorkDir
			if err := zrc.ExtractDir("export_scripts/", scriptsDir); err != nil {
				return err
			}
		}

		// 2. Import Envs
		if params.ImportEnv {
			if f, ok := fileMap["envs_meta.json"]; ok {
				if err := s.importEnvsFromZip(tx, zrc, f); err != nil {
					return err
				}
			}
		}

		// 3. Import Tasks
		if params.ImportTask {
			if f, ok := fileMap["tasks_meta.json"]; ok {
				if err := s.importTasksFromZip(tx, zrc, f); err != nil {
					return err
				}
			}
		}

		// 4. Import Deps
		if params.ImportDep {
			if f, ok := fileMap["deps_meta.json"]; ok {
				if err := s.importDepsFromZip(tx, zrc, f); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *ExportImportService) importEnvsFromZip(tx *gorm.DB, zrc *utils.ZipReadCloser, f *yeka_zip.File) error {
	rc, err := zrc.OpenFile(f)
	if err != nil {
		return err
	}
	defer rc.Close()

	var envs []models.EnvironmentVariable
	if err := json.NewDecoder(rc).Decode(&envs); err != nil {
		return err
	}
	for i := range envs {
		env := envs[i]
		env.ID = 0
		if err := tx.Create(&env).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *ExportImportService) importDepsFromZip(tx *gorm.DB, zrc *utils.ZipReadCloser, f *yeka_zip.File) error {
	rc, err := zrc.OpenFile(f)
	if err != nil {
		return err
	}
	defer rc.Close()

	var deps []models.Dependency
	if err := json.NewDecoder(rc).Decode(&deps); err != nil {
		return err
	}
	for i := range deps {
		dep := deps[i]
		dep.ID = 0
		if err := tx.Create(&dep).Error; err != nil {
			return err
		}
	}
	return nil
}
func (s *ExportImportService) importTasksFromZip(tx *gorm.DB, zrc *utils.ZipReadCloser, f *yeka_zip.File) error {
	rc, err := zrc.OpenFile(f)
	if err != nil {
		return err
	}
	defer rc.Close()

	var tasks []models.Task
	if err := json.NewDecoder(rc).Decode(&tasks); err != nil {
		return err
	}
	for i := range tasks {
		task := tasks[i]
		task.ID = 0
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
	}
	return nil
}
