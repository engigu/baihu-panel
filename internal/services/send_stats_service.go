package services

import (
	"time"

	"baihu/internal/database"
	"baihu/internal/models"
)

type SendStatsService struct{}

func NewSendStatsService() *SendStatsService {
	return &SendStatsService{}
}

// IncrementStats 增加任务执行统计
func (s *SendStatsService) IncrementStats(taskID uint, status string) error {
	today := time.Now().Format("2006-01-02")
	startOfDay, _ := time.ParseInLocation("2006-01-02", today, time.Local)

	var stats models.SendStats
	result := database.DB.Where("task_id = ? AND status = ? AND created_at >= ?", taskID, status, startOfDay).First(&stats)

	if result.Error != nil {
		// 不存在则创建
		stats = models.SendStats{
			TaskID:    taskID,
			Status:    status,
			Num:       1,
			CreatedAt: time.Now(),
		}
		return database.DB.Create(&stats).Error
	}

	// 存在则增加计数
	return database.DB.Model(&stats).Update("num", stats.Num+1).Error
}

// GetStatsByTaskID 获取任务的统计数据
func (s *SendStatsService) GetStatsByTaskID(taskID uint) []models.SendStats {
	var stats []models.SendStats
	database.DB.Where("task_id = ?", taskID).Order("created_at DESC").Find(&stats)
	return stats
}

// GetTodayStats 获取今日统计
func (s *SendStatsService) GetTodayStats() []models.SendStats {
	today := time.Now().Format("2006-01-02")
	startOfDay, _ := time.ParseInLocation("2006-01-02", today, time.Local)

	var stats []models.SendStats
	database.DB.Where("created_at >= ?", startOfDay).Find(&stats)
	return stats
}

// GetRecentStats 获取最近N天的统计
func (s *SendStatsService) GetRecentStats(days int) []models.SendStats {
	startDate := time.Now().AddDate(0, 0, -days)

	var stats []models.SendStats
	database.DB.Where("created_at >= ?", startDate).Order("created_at DESC").Find(&stats)
	return stats
}
