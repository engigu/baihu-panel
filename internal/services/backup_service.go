package services

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/cache"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/systime"
	"github.com/rs/xid"
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
		{"users.json", s.exportTable(&[]models.User{}), s.restoreTable(&[]models.User{})},
		{"tasks.json", s.exportTable(&[]models.Task{}), s.restoreTable(&[]models.Task{})},
		{"task_logs.json", s.exportTable(&[]models.TaskLog{}), s.restoreTable(&[]models.TaskLog{})},
		{"envs.json", s.exportTable(&[]models.EnvironmentVariable{}), s.restoreTable(&[]models.EnvironmentVariable{})},
		{"scripts.json", s.exportTable(&[]models.Script{}), s.restoreTable(&[]models.Script{})},
		{"settings.json", s.exportSettings, s.restoreSettings},
		{"send_stats.json", s.exportTable(&[]models.SendStats{}), s.restoreTable(&[]models.SendStats{})},

		{"agents.json", s.exportTable(&[]models.Agent{}), s.restoreTable(&[]models.Agent{})},
		{"tokens.json", s.exportTable(&[]models.AgentToken{}), s.restoreTable(&[]models.AgentToken{})},
		{"languages.json", s.exportTable(&[]models.Language{}), s.restoreTable(&[]models.Language{})},
		{"deps.json", s.exportTable(&[]models.Dependency{}), s.restoreTable(&[]models.Dependency{})},
		{"notify_ways.json", s.exportTable(&[]models.NotifyWay{}), s.restoreTable(&[]models.NotifyWay{})},
		{"notify_bindings.json", s.exportTable(&[]models.NotifyBinding{}), s.restoreTable(&[]models.NotifyBinding{})},
		{"app_logs.json", s.exportTable(&[]models.AppLog{}), s.restoreTable(&[]models.AppLog{})},
		{"data_storage.json", s.exportTable(&[]models.DataStorage{}), s.restoreTable(&[]models.DataStorage{})},
		{"data_relations.json", s.exportTable(&[]models.DataRelation{}), s.restoreTable(&[]models.DataRelation{})},
	}
}

func (s *BackupService) exportTable(modelPtr any) func(io.Writer) error {
	return func(w io.Writer) error {
		db := database.DB

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

func (s *BackupService) restoreTable(dest any) func([]byte) error {
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
		"version": "v3",
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

	// 校验版本
	if f, ok := fileMap["__sys__.json"]; ok {
		rc, err := f.Open()
		if err == nil {
			var sysInfo map[string]interface{}
			json.NewDecoder(rc).Decode(&sysInfo)
			rc.Close()
			if v, ok := sysInfo["version"]; ok {
				vs, _ := v.(string)
				if vs < "v3" {
					return fmt.Errorf("只能数据随版本升级上来，当前备份版本为 %s，限制 v3 以下的不能导入", vs)
				}
			}
		}
		// } else {
		// 	return fmt.Errorf("非法备份包：缺失版本标记")
	}

	// 开启全局事务
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 清空现有数据（物理删除）
		tx.Where("1=1").Delete(&models.User{})
		tx.Where("1=1").Delete(&models.Task{})
		tx.Where("1=1").Delete(&models.TaskLog{})
		tx.Where("1=1").Delete(&models.EnvironmentVariable{})
		tx.Where("1=1").Delete(&models.Script{})
		tx.Where("section != ?", BackupSection).Delete(&models.Setting{})
		tx.Where("1=1").Delete(&models.SendStats{})

		tx.Where("1=1").Delete(&models.Agent{})
		tx.Where("1=1").Delete(&models.AgentToken{})
		tx.Where("1=1").Delete(&models.Language{})
		tx.Where("1=1").Delete(&models.Dependency{})
		tx.Where("1=1").Delete(&models.NotifyWay{})
		tx.Where("1=1").Delete(&models.NotifyBinding{})
		tx.Where("1=1").Delete(&models.AppLog{})
		tx.Where("1=1").Delete(&models.DataStorage{})
		tx.Where("1=1").Delete(&models.DataRelation{})

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

	if err == nil {
		// 备份恢复成功后，需要同时刷新内存中的配置缓存以免数据不一致导致异常
		constant.Secret = s.settingsService.Get(constant.SectionSecurity, constant.KeySecret)
		cache.LoadSiteCache()
	}

	return err
}

func restoreStreamBatch[T any](tx *gorm.DB, decoder *json.Decoder) error {
	batchSize := 1000
	var batch []*T

	for decoder.More() {
		var m T
		if err := decoder.Decode(&m); err != nil {
			return err
		}
		batch = append(batch, &m)

		if len(batch) >= batchSize {
			if err := tx.Select("*").CreateInBatches(batch, batchSize).Error; err != nil {
				return err
			}
			batch = nil // reset batch
		}
	}

	if len(batch) > 0 {
		return tx.Select("*").CreateInBatches(batch, len(batch)).Error
	}

	return nil
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
				return tx.Select("*").Create(&settings).Error
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

	switch filename {
	case "users.json":
		return restoreStreamBatch[models.User](tx, decoder)
	case "tasks.json":
		return s.restoreTasks(tx, decoder)
	case "task_logs.json":
		return restoreStreamBatch[models.TaskLog](tx, decoder)
	case "envs.json":
		return restoreStreamBatch[models.EnvironmentVariable](tx, decoder)
	case "scripts.json":
		return restoreStreamBatch[models.Script](tx, decoder)
	case "send_stats.json":
		return restoreStreamBatch[models.SendStats](tx, decoder)

	case "agents.json":
		return restoreStreamBatch[models.Agent](tx, decoder)
	case "tokens.json":
		return restoreStreamBatch[models.AgentToken](tx, decoder)
	case "languages.json":
		return restoreStreamBatch[models.Language](tx, decoder)
	case "deps.json":
		return restoreStreamBatch[models.Dependency](tx, decoder)
	case "notify_ways.json":
		return restoreStreamBatch[models.NotifyWay](tx, decoder)
	case "notify_bindings.json":
		return restoreStreamBatch[models.NotifyBinding](tx, decoder)
	case "app_logs.json":
		return restoreStreamBatch[models.AppLog](tx, decoder)
	case "data_storage.json":
		return restoreStreamBatch[models.DataStorage](tx, decoder)
	case "data_relations.json":
		return restoreStreamBatch[models.DataRelation](tx, decoder)
	default:
		return nil
	}
}

func (s *BackupService) restoreTasks(tx *gorm.DB, decoder *json.Decoder) error {
	batchSize := 1000
	var batch []*models.Task

	for decoder.More() {
		var m models.Task
		if err := decoder.Decode(&m); err != nil {
			return err
		}
		batch = append(batch, &m)

		if len(batch) >= batchSize {
			if err := s.insertTasksBatch(tx, batch); err != nil {
				return err
			}
			batch = nil // reset
		}
	}

	if len(batch) > 0 {
		return s.insertTasksBatch(tx, batch)
	}

	return nil
}

func (s *BackupService) insertTasksBatch(tx *gorm.DB, batch []*models.Task) error {
	if err := tx.Select("*").CreateInBatches(batch, len(batch)).Error; err != nil {
		return err
	}

	// [COMPATIBILITY CODE - TEMPORARY]
	// 为旧版备份数据做向下兼容（在旧版本中 tags/envs 还是直接存储在 tasks.json 中）。
	// 此段代码将旧数据迁移到新的 data_relations 和 data_storages 中，确保旧备份包能正常恢复。
	// 由于新版本的备份中 tasks.json 已不再包含这些字段，未来某次大版本更新后，这段兼容代码将被移除。
	for _, t := range batch {
		if t.Tags != "" {
			tags := strings.Split(t.Tags, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				var storage models.DataStorage
				res := tx.Where("type = ? AND name = ?", constant.RelationTypeTaskTag, tag).Limit(1).Find(&storage)
				if res.RowsAffected == 0 {
					storage = models.DataStorage{
						ID:        xid.New().String(),
						Type:      constant.RelationTypeTaskTag,
						Name:      tag,
						CreatedAt: models.Now(),
						UpdatedAt: models.Now(),
					}
					tx.Create(&storage)
				}
				
				var count int64
				tx.Model(&models.DataRelation{}).Where("data_id = ? AND relate_id = ? AND type = ?", t.ID, storage.ID, constant.RelationTypeTaskTag).Count(&count)
				if count == 0 {
					relation := models.DataRelation{
						ID:        xid.New().String(),
						DataID:    t.ID,
						RelateID:  storage.ID,
						Type:      constant.RelationTypeTaskTag,
						CreatedAt: models.Now(),
						UpdatedAt: models.Now(),
					}
					tx.Create(&relation)
				}
			}
		}

		if string(t.Envs) != "" {
			envs := strings.Split(string(t.Envs), ",")
			for _, envID := range envs {
				envID = strings.TrimSpace(envID)
				if envID == "" {
					continue
				}
				var count int64
				tx.Model(&models.DataRelation{}).Where("data_id = ? AND relate_id = ? AND type = ?", t.ID, envID, constant.RelationTypeTaskEnv).Count(&count)
				if count == 0 {
					relation := models.DataRelation{
						ID:        xid.New().String(),
						DataID:    t.ID,
						RelateID:  envID,
						Type:      constant.RelationTypeTaskEnv,
						CreatedAt: models.Now(),
						UpdatedAt: models.Now(),
					}
					tx.Create(&relation)
				}
			}
		}
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
	res := database.DB.Where(&models.Setting{Section: BackupSection, Key: BackupFileKey}).Limit(1).Find(&setting)
	if res.Error != nil || res.RowsAffected == 0 {
		return ""
	}
	return string(setting.Value)
}

func (s *BackupService) ClearBackup() error {
	filePath := s.GetBackupFile()
	if filePath != "" {
		os.Remove(filePath)
		database.DB.Where(&models.Setting{Section: BackupSection, Key: BackupFileKey}).Delete(&models.Setting{})
	}
	return nil
}
