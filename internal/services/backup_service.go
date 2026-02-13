package services

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/systime"
	"gorm.io/gorm"
)

type BackupService struct {
	settingsService *SettingsService
}

func NewBackupService() *BackupService {
	return &BackupService{
		settingsService: NewSettingsService(),
	}
}

const (
	BackupSection = "backup"
	BackupFileKey = "backup_file"
	BackupDir     = "./data/backups"
)

// tableConfig 表备份配置
type tableConfig struct {
	filename string
	export   func(io.Writer) error
	restore  func([]byte) error
}

func (s *BackupService) getTableConfigs() []tableConfig {
	return []tableConfig{
		{"tasks.json", s.exportTable(&[]models.Task{}, true), s.restoreTable(&[]models.Task{}, true)},
		{"task_logs.json", s.exportTable(&[]models.TaskLog{}, false), s.restoreTable(&[]models.TaskLog{}, false)},
		{"envs.json", s.exportTable(&[]models.EnvironmentVariable{}, true), s.restoreTable(&[]models.EnvironmentVariable{}, true)},
		{"scripts.json", s.exportTable(&[]models.Script{}, true), s.restoreTable(&[]models.Script{}, true)},
		{"settings.json", s.exportSettings, s.restoreSettings},
		{"send_stats.json", s.exportTable(&[]models.SendStats{}, false), s.restoreTable(&[]models.SendStats{}, false)},
		{"login_logs.json", s.exportTable(&[]models.LoginLog{}, false), s.restoreTable(&[]models.LoginLog{}, false)},
		{"agents.json", s.exportTable(&[]models.Agent{}, true), s.restoreTable(&[]models.Agent{}, true)},
		{"tokens.json", s.exportTable(&[]models.AgentToken{}, true), s.restoreTable(&[]models.AgentToken{}, true)},
		{"languages.json", s.exportTable(&[]models.Language{}, true), s.restoreTable(&[]models.Language{}, true)},
		{"deps.json", s.exportTable(&[]models.Dependency{}, true), s.restoreTable(&[]models.Dependency{}, true)},
	}
}

func (s *BackupService) exportTable(modelPtr any, unscoped bool) func(io.Writer) error {
	return func(w io.Writer) error {
		db := database.DB
		if unscoped {
			db = db.Unscoped()
		}

		if _, err := w.Write([]byte("[\n")); err != nil {
			return err
		}

		first := true
		err := db.FindInBatches(modelPtr, 1000, func(tx *gorm.DB, batch int) error {
			val := reflect.ValueOf(modelPtr).Elem()
			count := val.Len()
			for i := 0; i < count; i++ {
				if !first {
					if _, err := w.Write([]byte(",\n")); err != nil {
						return err
					}
				}
				item := val.Index(i).Interface()
				jsonData, err := json.MarshalIndent(item, "  ", "  ")
				if err != nil {
					return err
				}
				if _, err := w.Write(jsonData); err != nil {
					return err
				}
				first = false
			}
			return nil
		}).Error

		if err != nil {
			return err
		}

		_, err = w.Write([]byte("\n]"))
		return err
	}
}

func (s *BackupService) restoreTable(dest any, unscoped bool) func([]byte) error {
	return func(data []byte) error {
		if err := json.Unmarshal(data, dest); err != nil {
			return err
		}
		return nil
	}
}

func (s *BackupService) exportSettings(w io.Writer) error {
	var data []models.Setting
	if err := database.DB.Where("section != ?", BackupSection).Find(&data).Error; err != nil {
		return err
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(jsonData)
	return err
}

func (s *BackupService) restoreSettings(data []byte) error {
	var settings []models.Setting
	return json.Unmarshal(data, &settings)
}

// CreateBackup 创建备份
func (s *BackupService) CreateBackup() (string, error) {
	if err := os.MkdirAll(BackupDir, 0755); err != nil {
		return "", err
	}

	timestamp := systime.FormatDatetime(time.Now())
	zipPath := filepath.Join(BackupDir, fmt.Sprintf("backup_%s.zip", timestamp))

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 导出各表
	for _, cfg := range s.getTableConfigs() {
		w, err := zipWriter.Create(cfg.filename)
		if err != nil {
			return "", err
		}
		if err := cfg.export(w); err != nil {
			return "", err
		}
	}

	// 写入元数据信息
	sysInfo := map[string]interface{}{
		"version": "v2",
		"ts":      time.Now().Format("2006-01-02 15:04:05"),
	}
	sysFile, err := zipWriter.Create("__sys__.json")
	if err != nil {
		return "", err
	}
	sysData, _ := json.MarshalIndent(sysInfo, "", "  ")
	if _, err := sysFile.Write(sysData); err != nil {
		return "", err
	}

	// 打包 scripts 文件夹
	scriptsDir := constant.ScriptsWorkDir
	if _, err := os.Stat(scriptsDir); err == nil {
		if err := s.addDirToZip(zipWriter, scriptsDir, "scripts"); err != nil {
			return "", err
		}
	}

	s.settingsService.Set(BackupSection, BackupFileKey, zipPath)
	return zipPath, nil
}

// Restore 恢复备份
func (s *BackupService) Restore(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// 构建文件名到配置的映射
	configs := s.getTableConfigs()
	fileMap := make(map[string]*zip.File)
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	// 开启全局事务
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 清空现有数据（物理删除）
		tx.Unscoped().Where("1=1").Delete(&models.Task{})
		tx.Unscoped().Where("1=1").Delete(&models.TaskLog{})
		tx.Unscoped().Where("1=1").Delete(&models.EnvironmentVariable{})
		tx.Unscoped().Where("1=1").Delete(&models.Script{})
		tx.Unscoped().Where("section != ?", BackupSection).Delete(&models.Setting{})
		tx.Unscoped().Where("1=1").Delete(&models.SendStats{})
		tx.Unscoped().Where("1=1").Delete(&models.LoginLog{})
		tx.Unscoped().Where("1=1").Delete(&models.Agent{})
		tx.Unscoped().Where("1=1").Delete(&models.AgentToken{})
		tx.Unscoped().Where("1=1").Delete(&models.Language{})
		tx.Unscoped().Where("1=1").Delete(&models.Dependency{})

		// 2. 依次恢复每个表
		for _, cfg := range configs {
			if f, ok := fileMap[cfg.filename]; ok {
				if err := s.restoreFromZipFile(tx, f, cfg.filename); err != nil {
					return err
				}
			}
		}

		// 3. 恢复 scripts 文件夹
		s.restoreScriptsDir(r)

		return nil
	})
}

func (s *BackupService) restoreFromZipFile(tx *gorm.DB, f *zip.File, filename string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 特殊处理设置表（设置表通常很小，直接反序列化）
	if filename == "settings.json" {
		data, _ := io.ReadAll(rc)
		var settings []models.Setting
		if err := json.Unmarshal(data, &settings); err == nil {
			if len(settings) > 0 {
				return tx.Create(&settings).Error
			}
		}
		return nil
	}

	// 流式解析 JSON 数组
	decoder := json.NewDecoder(rc)

	// 找到数组开始 [
	if t, err := decoder.Token(); err != nil || t != json.Delim('[') {
		return fmt.Errorf("invalid json format: expected %s", filename)
	}

	batchSize := 1000
	var batch []any

	// 根据文件名确定模型类型
	getModel := func() any {
		switch filename {
		case "tasks.json":
			return &models.Task{}
		case "task_logs.json":
			return &models.TaskLog{}
		case "envs.json":
			return &models.EnvironmentVariable{}
		case "scripts.json":
			return &models.Script{}
		case "send_stats.json":
			return &models.SendStats{}
		case "login_logs.json":
			return &models.LoginLog{}
		case "agents.json":
			return &models.Agent{}
		case "tokens.json":
			return &models.AgentToken{}
		case "languages.json":
			return &models.Language{}
		case "deps.json":
			return &models.Dependency{}
		default:
			return nil
		}
	}

	for decoder.More() {
		m := getModel()
		if m == nil {
			break
		}
		if err := decoder.Decode(m); err != nil {
			return err
		}
		batch = append(batch, m)

		if len(batch) >= batchSize {
			if err := tx.CreateInBatches(batch, batchSize).Error; err != nil {
				return err
			}
			batch = nil
		}
	}

	if len(batch) > 0 {
		return tx.CreateInBatches(batch, batchSize).Error
	}

	return nil
}

// insertRecords, restoreFromData 方法已合并入 restoreFromZipFile，此处删除冗余方法

func (s *BackupService) restoreScriptsDir(r *zip.ReadCloser) {
	scriptsDir := constant.ScriptsWorkDir
	for _, f := range r.File {
		if len(f.Name) > 8 && f.Name[:8] == "scripts/" {
			relPath := f.Name[8:]
			if relPath == "" {
				continue
			}
			fpath := filepath.Join(scriptsDir, relPath)
			if f.FileInfo().IsDir() {
				os.MkdirAll(fpath, 0755)
				continue
			}
			os.MkdirAll(filepath.Dir(fpath), 0755)
			if outFile, err := os.Create(fpath); err == nil {
				if rc, err := f.Open(); err == nil {
					io.Copy(outFile, rc)
					rc.Close()
				}
				outFile.Close()
			}
		}
	}
}

func (s *BackupService) readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (s *BackupService) addDirToZip(zipWriter *zip.Writer, srcDir, prefix string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		zipPath := filepath.ToSlash(filepath.Join(prefix, relPath))
		if info.IsDir() {
			if relPath != "." {
				_, err := zipWriter.Create(zipPath + "/")
				return err
			}
			return nil
		}
		w, err := zipWriter.Create(zipPath)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(w, file)
		return err
	})
}

func (s *BackupService) GetBackupFile() string {
	var setting models.Setting
	if err := database.DB.Where("section = ? AND `key` = ?", BackupSection, BackupFileKey).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

func (s *BackupService) ClearBackup() error {
	filePath := s.GetBackupFile()
	if filePath != "" {
		os.Remove(filePath)
		database.DB.Where("section = ? AND `key` = ?", BackupSection, BackupFileKey).Delete(&models.Setting{})
	}
	return nil
}
