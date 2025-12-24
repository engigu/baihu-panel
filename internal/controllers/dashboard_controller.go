package controllers

import (
	"sort"
	"time"

	"baihu/internal/constant"
	"baihu/internal/database"
	"baihu/internal/models"
	"baihu/internal/services"
	"baihu/internal/utils"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	cronService     *services.CronService
	executorService *services.ExecutorService
}

func NewDashboardController(cronService *services.CronService, executorService *services.ExecutorService) *DashboardController {
	return &DashboardController{
		cronService:     cronService,
		executorService: executorService,
	}
}

type StatsResponse struct {
	Tasks      int64 `json:"tasks"`
	TodayExecs int64 `json:"today_execs"`
	Envs       int64 `json:"envs"`
	Logs       int64 `json:"logs"`
	Scheduled  int   `json:"scheduled"`
	Running    int   `json:"running"`
}

func (dc *DashboardController) GetStats(c *gin.Context) {
	var taskCount, envCount, logCount, todayExecs int64

	database.DB.Model(&models.Task{}).Count(&taskCount)
	database.DB.Model(&models.EnvironmentVariable{}).Count(&envCount)
	database.DB.Model(&models.TaskLog{}).Count(&logCount)

	// 今日执行总数
	today := time.Now().Format("2006-01-02")
	database.DB.Model(&models.SendStats{}).Where("day = ?", today).Select("COALESCE(SUM(num), 0)").Scan(&todayExecs)

	stats := StatsResponse{
		Tasks:      taskCount,
		TodayExecs: todayExecs,
		Envs:       envCount,
		Logs:       logCount,
		Scheduled:  dc.cronService.GetScheduledCount(),
		Running:    dc.executorService.GetRunningCount(),
	}

	utils.Success(c, stats)
}

// GetSentence 获取随机古诗词
func (dc *DashboardController) GetSentence(c *gin.Context) {
	utils.Success(c, gin.H{
		"sentence": constant.GetRandomSentence(),
	})
}

// DailyStats 每日统计数据
type DailyStats struct {
	Day     string `json:"day"`
	Total   int    `json:"total"`
	Success int    `json:"success"`
	Failed  int    `json:"failed"`
}

// GetSendStats 获取最近30天发送统计
func (dc *DashboardController) GetSendStats(c *gin.Context) {
	// 获取最近30天的日期范围
	now := time.Now()
	startDay := now.AddDate(0, 0, -29).Format("2006-01-02")

	var stats []models.SendStats
	database.DB.Where("day >= ?", startDay).Find(&stats)

	// 按日期聚合
	dayMap := make(map[string]*DailyStats)
	for _, s := range stats {
		if _, ok := dayMap[s.Day]; !ok {
			dayMap[s.Day] = &DailyStats{Day: s.Day}
		}
		ds := dayMap[s.Day]
		ds.Total += s.Num
		if s.Status == "success" {
			ds.Success += s.Num
		} else {
			ds.Failed += s.Num
		}
	}

	// 填充缺失的日期
	result := make([]DailyStats, 0, 30)
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		if ds, ok := dayMap[day]; ok {
			result = append(result, *ds)
		} else {
			result = append(result, DailyStats{Day: day})
		}
	}

	// 按日期排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Day < result[j].Day
	})

	utils.Success(c, result)
}

// TaskStats 任务执行统计
type TaskStats struct {
	TaskID   uint   `json:"task_id"`
	TaskName string `json:"task_name"`
	Count    int    `json:"count"`
}

// GetTaskStats 获取最近30天任务执行占比
func (dc *DashboardController) GetTaskStats(c *gin.Context) {
	now := time.Now()
	startDay := now.AddDate(0, 0, -29).Format("2006-01-02")

	// 按 task_id 聚合统计
	var results []struct {
		TaskID uint
		Total  int
	}
	database.DB.Model(&models.SendStats{}).
		Select("task_id, SUM(num) as total").
		Where("day >= ?", startDay).
		Group("task_id").
		Order("total DESC").
		Find(&results)

	// 获取任务名称
	taskIDs := make([]uint, 0, len(results))
	for _, r := range results {
		taskIDs = append(taskIDs, r.TaskID)
	}

	var tasks []models.Task
	if len(taskIDs) > 0 {
		database.DB.Where("id IN ?", taskIDs).Find(&tasks)
	}
	taskNameMap := make(map[uint]string)
	for _, t := range tasks {
		taskNameMap[t.ID] = t.Name
	}

	// 构建结果
	stats := make([]TaskStats, 0, len(results))
	for _, r := range results {
		name := taskNameMap[r.TaskID]
		if name == "" {
			name = "未知任务"
		}
		stats = append(stats, TaskStats{
			TaskID:   r.TaskID,
			TaskName: name,
			Count:    r.Total,
		})
	}

	utils.Success(c, stats)
}
