package controllers

import (
	"strconv"

	"baihu/internal/database"
	"baihu/internal/models"
	"baihu/internal/utils"

	"github.com/gin-gonic/gin"
)

type SyncLogController struct{}

func NewSyncLogController() *SyncLogController {
	return &SyncLogController{}
}

// SyncLogListItem 同步日志列表项
type SyncLogListItem struct {
	ID           uint             `json:"id"`
	SyncTaskID   uint             `json:"sync_task_id"`
	SyncTaskName string           `json:"sync_task_name"`
	SourceURL    string           `json:"source_url"`
	TargetPath   string           `json:"target_path"`
	Status       string           `json:"status"`
	Duration     int64            `json:"duration"`
	CreatedAt    models.LocalTime `json:"created_at"`
}

// GetSyncLogs 获取同步日志列表
func (c *SyncLogController) GetSyncLogs(ctx *gin.Context) {
	p := utils.ParsePagination(ctx)
	syncTaskID := ctx.DefaultQuery("sync_task_id", "")
	taskName := ctx.DefaultQuery("task_name", "")

	var logs []models.SyncTaskLog
	var total int64

	query := database.DB.Model(&models.SyncTaskLog{})

	if syncTaskID != "" {
		if id, err := strconv.Atoi(syncTaskID); err == nil {
			query = query.Where("sync_task_id = ?", id)
		}
	}

	// 如果有任务名称过滤，需要关联查询
	if taskName != "" {
		var taskIDs []uint
		database.DB.Model(&models.SyncTask{}).Where("name LIKE ?", "%"+taskName+"%").Pluck("id", &taskIDs)
		if len(taskIDs) > 0 {
			query = query.Where("sync_task_id IN ?", taskIDs)
		} else {
			// 没有匹配的任务，返回空结果
			utils.PaginatedResponse(ctx, []SyncLogListItem{}, 0, p)
			return
		}
	}

	query.Count(&total)
	query.Order("id DESC").Offset((p.Page - 1) * p.PageSize).Limit(p.PageSize).Find(&logs)

	// 获取任务名称映射
	var taskIDs []uint
	for _, log := range logs {
		taskIDs = append(taskIDs, log.SyncTaskID)
	}

	taskNameMap := make(map[uint]string)
	if len(taskIDs) > 0 {
		var tasks []models.SyncTask
		database.DB.Where("id IN ?", taskIDs).Find(&tasks)
		for _, task := range tasks {
			taskNameMap[task.ID] = task.Name
		}
	}

	// 构建响应
	items := make([]SyncLogListItem, len(logs))
	for i, log := range logs {
		items[i] = SyncLogListItem{
			ID:           log.ID,
			SyncTaskID:   log.SyncTaskID,
			SyncTaskName: taskNameMap[log.SyncTaskID],
			SourceURL:    log.SourceURL,
			TargetPath:   log.TargetPath,
			Status:       log.Status,
			Duration:     log.Duration,
			CreatedAt:    log.CreatedAt,
		}
	}

	utils.PaginatedResponse(ctx, items, total, p)
}

// SyncLogDetail 同步日志详情
type SyncLogDetail struct {
	ID           uint             `json:"id"`
	SyncTaskID   uint             `json:"sync_task_id"`
	SyncTaskName string           `json:"sync_task_name"`
	SourceURL    string           `json:"source_url"`
	TargetPath   string           `json:"target_path"`
	Output       string           `json:"output"`
	Status       string           `json:"status"`
	Duration     int64            `json:"duration"`
	CreatedAt    models.LocalTime `json:"created_at"`
}

// GetSyncLogDetail 获取同步日志详情
func (c *SyncLogController) GetSyncLogDetail(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		utils.BadRequest(ctx, "无效的日志ID")
		return
	}

	var log models.SyncTaskLog
	if err := database.DB.First(&log, id).Error; err != nil {
		utils.NotFound(ctx, "日志不存在")
		return
	}

	// 获取任务名称
	var task models.SyncTask
	taskName := ""
	if err := database.DB.First(&task, log.SyncTaskID).Error; err == nil {
		taskName = task.Name
	}

	// 解压输出
	output, err := utils.DecompressFromBase64(log.Output)
	if err != nil {
		output = log.Output
	}

	detail := SyncLogDetail{
		ID:           log.ID,
		SyncTaskID:   log.SyncTaskID,
		SyncTaskName: taskName,
		SourceURL:    log.SourceURL,
		TargetPath:   log.TargetPath,
		Output:       output,
		Status:       log.Status,
		Duration:     log.Duration,
		CreatedAt:    log.CreatedAt,
	}

	utils.Success(ctx, detail)
}
