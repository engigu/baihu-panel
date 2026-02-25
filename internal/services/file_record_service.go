package services

import (
	"os"
	"time"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
)

type FileRecordService struct {
	settingsService *SettingsService
}

func NewFileRecordService() *FileRecordService {
	return &FileRecordService{
		settingsService: NewSettingsService(),
	}
}

// SaveFileRecord 保存文件记录
func (s *FileRecordService) SaveFileRecord(section, key, path string) {
	s.ClearFileRecord(section, key)
	s.settingsService.Set(section, key, path)
}

// GetFileRecord 获取文件记录
func (s *FileRecordService) GetFileRecord(section, key string) string {
	var setting models.Setting
	if err := database.DB.Where("section = ? AND `key` = ?", section, key).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

// ClearFileRecord 清理旧文件记录
func (s *FileRecordService) ClearFileRecord(section, key string) {
	filePath := s.GetFileRecord(section, key)
	if filePath != "" {
		os.Remove(filePath)
		database.DB.Where("section = ? AND `key` = ?", section, key).Delete(&models.Setting{})
	}
}

// SaveTempFileRecord 保存临时文件记录，并在指定时间后自动删除
func (s *FileRecordService) SaveTempFileRecord(section, key, path string, duration time.Duration) {
	s.SaveFileRecord(section, key, path)

	go func() {
		time.Sleep(duration)
		// 确保此时的文件路径没变，如果没变则删除
		currentPath := s.GetFileRecord(section, key)
		if currentPath == path {
			s.ClearFileRecord(section, key)
		}
	}()
}
