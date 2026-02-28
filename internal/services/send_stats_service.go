package services

import (
	"time"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/systime"
)

type SendStatsService struct{}

func NewSendStatsService() *SendStatsService {
	return &SendStatsService{}
}

// IncrementStats 增加任务执行统计
func (s *SendStatsService) IncrementStats(taskID string, status string) error {
	day := systime.FormatDate(time.Now())

	var stats models.SendStats
	result := database.DB.Where("task_id = ? AND day = ? AND status = ?", taskID, day, status).First(&stats)

	if result.Error != nil {
		// 不存在则创建
		stats = models.SendStats{
			TaskID: taskID,
			Day:    day,
			Status: status,
			Num:    1,
		}
		return database.DB.Create(&stats).Error
	}

	// 存在则增加计数
	return database.DB.Model(&stats).Update("num", stats.Num+1).Error
}

// GetStatsByTaskID 获取任务的统计数据
func (s *SendStatsService) GetStatsByTaskID(taskID string) []models.SendStats {
	var stats []models.SendStats
	database.DB.Where("task_id = ?", taskID).Order("day DESC").Find(&stats)
	return stats
}

// GetTodayStats 获取今日统计
func (s *SendStatsService) GetTodayStats() []models.SendStats {
	day := systime.FormatDate(time.Now())
	var stats []models.SendStats
	database.DB.Where("day = ?", day).Find(&stats)
	return stats
}

// GetRecentStats 获取最近N天的统计
func (s *SendStatsService) GetRecentStats(days int) []models.SendStats {
	startDay := systime.FormatDate(time.Now().AddDate(0, 0, -days))
	var stats []models.SendStats
	database.DB.Where("day >= ?", startDay).Order("day DESC").Find(&stats)
	return stats
}
