package services

import (
	"baihu/internal/database"
	"baihu/internal/models"
)

type SyncTaskService struct{}

func NewSyncTaskService() *SyncTaskService {
	return &SyncTaskService{}
}

// CreateSyncTask 创建同步任务
func (s *SyncTaskService) CreateSyncTask(name, sourceType, sourceURL, branch, targetPath, schedule, proxy, proxyURL, authToken, cleanConfig string) *models.SyncTask {
	task := &models.SyncTask{
		Name:        name,
		SourceType:  sourceType,
		SourceURL:   sourceURL,
		Branch:      branch,
		TargetPath:  targetPath,
		Schedule:    schedule,
		Proxy:       proxy,
		ProxyURL:    proxyURL,
		AuthToken:   authToken,
		CleanConfig: cleanConfig,
		Enabled:     true,
	}
	database.DB.Create(task)
	return task
}

// GetSyncTasks 获取所有同步任务
func (s *SyncTaskService) GetSyncTasks() []models.SyncTask {
	var tasks []models.SyncTask
	database.DB.Find(&tasks)
	return tasks
}

// GetSyncTasksWithPagination 分页获取同步任务列表
func (s *SyncTaskService) GetSyncTasksWithPagination(page, pageSize int, name string) ([]models.SyncTask, int64) {
	var tasks []models.SyncTask
	var total int64

	query := database.DB.Model(&models.SyncTask{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks)

	return tasks, total
}

// GetSyncTaskByID 根据 ID 获取同步任务
func (s *SyncTaskService) GetSyncTaskByID(id int) *models.SyncTask {
	var task models.SyncTask
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil
	}
	return &task
}

// UpdateSyncTask 更新同步任务
func (s *SyncTaskService) UpdateSyncTask(id int, name, sourceType, sourceURL, branch, targetPath, schedule, proxy, proxyURL, authToken, cleanConfig string, enabled bool) *models.SyncTask {
	var task models.SyncTask
	if err := database.DB.First(&task, id).Error; err != nil {
		return nil
	}
	task.Name = name
	task.SourceType = sourceType
	task.SourceURL = sourceURL
	task.Branch = branch
	task.TargetPath = targetPath
	task.Schedule = schedule
	task.Proxy = proxy
	task.ProxyURL = proxyURL
	task.AuthToken = authToken
	task.CleanConfig = cleanConfig
	task.Enabled = enabled
	database.DB.Save(&task)
	return &task
}

// DeleteSyncTask 删除同步任务
func (s *SyncTaskService) DeleteSyncTask(id int) bool {
	result := database.DB.Delete(&models.SyncTask{}, id)
	return result.RowsAffected > 0
}

// UpdateLastSync 更新上次同步时间和状态
func (s *SyncTaskService) UpdateLastSync(id uint, status string) {
	database.DB.Model(&models.SyncTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_sync":   database.DB.NowFunc(),
		"last_status": status,
	})
}
