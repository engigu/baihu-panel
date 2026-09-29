package controllers

import (
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	DeletedTaskPrefix      = "[已删除] "
	DeletedTaskPlaceholder = "[已删除任务]"
)

type LogController struct{}

func NewLogController() *LogController {
	return &LogController{}
}

// GetLogs 获取任务日志列表
// @Summary 获取任务日志列表
// @Description 分页获取任务日志列表，支持按任务 ID、任务名称、状态、日期筛选
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param task_id query string false "任务 ID"
// @Param task_name query string false "任务名称"
// @Param status query string false "状态"
// @Param date query string false "日期 (today 或 YYYY-MM-DD)"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} utils.Response{data=utils.PaginationData{data=[]vo.TaskLogVO}}
// @Router /logs [get]
func (lc *LogController) GetLogs(c *gin.Context) {
	p := utils.ParsePagination(c)
	taskID := c.DefaultQuery("task_id", "")
	taskName := c.DefaultQuery("task_name", "")
	status := c.DefaultQuery("status", "")
	date := c.DefaultQuery("date", "")

	var logs []models.TaskLog
	var total int64

	query := database.DB.Model(&models.TaskLog{})
	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if date == "today" {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)
		query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
	} else if date != "" {
		if t, err := time.ParseInLocation("2006-01-02", date, time.Local); err == nil {
			startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
		}
	}

	// 按任务名称过滤（支持明确标记检索已删除任务，不使用空字符代表已删除）
	if taskName != "" {
		trimmedName := strings.TrimSpace(taskName)
		if trimmedName == "[已删除]" || trimmedName == "已删除" || trimmedName == ":deleted" {
			// 优化：提取当前存活任务 ID 列表，走 task_id 索引常数扫描，避免 NOT IN (subquery) 慢查询
			var activeIDs []string
			database.DB.Model(&models.Task{}).Pluck("id", &activeIDs)
			if len(activeIDs) > 0 {
				query = query.Where("task_id NOT IN ?", activeIDs)
			}
		} else if strings.HasPrefix(trimmedName, "[已删除]") {
			// 支持带 [已删除] 前缀的组合搜索
			keyword := strings.TrimSpace(strings.TrimPrefix(trimmedName, "[已删除]"))
			var activeIDs []string
			database.DB.Model(&models.Task{}).Pluck("id", &activeIDs)
			if len(activeIDs) > 0 {
				query = query.Where("task_id NOT IN ?", activeIDs)
			}
			if keyword != "" {
				query = query.Where("task_name LIKE ?", "%"+keyword+"%")
			}
		} else {
			var taskIDs []string
			database.DB.Model(&models.Task{}).Where("name LIKE ?", "%"+trimmedName+"%").Pluck("id", &taskIDs)
			if len(taskIDs) > 0 {
				query = query.Where("(task_id IN ? OR task_name LIKE ?)", taskIDs, "%"+trimmedName+"%")
			} else {
				query = query.Where("task_name LIKE ?", "%"+trimmedName+"%")
			}
		}
	}

	query.Count(&total)
	query.Omit("output", "error").Order("id DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&logs)

	taskIDList := make([]string, 0)
	for _, log := range logs {
		taskIDList = append(taskIDList, log.TaskID)
	}

	var tasks []models.Task
	if len(taskIDList) > 0 {
		database.DB.Select("id", "name", "type").Where("id IN ?", taskIDList).Find(&tasks)
	}
	taskMap := make(map[string]models.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	result := make([]vo.TaskLogVO, len(logs))
	for i, log := range logs {
		var taskPtr *models.Task
		if t, exists := taskMap[log.TaskID]; exists {
			taskPtr = &t
		}
		displayName, taskType, taskDeleted := resolveTaskLogInfo(taskPtr, &log)

		result[i] = vo.TaskLogVO{
			ID:          log.ID,
			TaskID:      log.TaskID,
			TaskName:    displayName,
			TaskDeleted: taskDeleted,
			TaskType:    taskType,
			AgentID:     log.AgentID,
			Command:     string(log.Command),
			Status:      log.Status,
			Duration:    log.Duration,
			StartTime:   log.StartTime,
			EndTime:     log.EndTime,
			CreatedAt:   log.CreatedAt,
		}
	}

	utils.PaginatedResponse(c, result, total, p)
}

// GetLogDetail 获取日志详情
// @Summary 获取日志详情
// @Description 根据 ID 获取任务日志详细内容（包含输出）
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "日志ID"
// @Success 200 {object} utils.Response{data=vo.TaskLogVO}
// @Failure 404 {object} utils.Response
// @Router /logs/{id} [get]
func (lc *LogController) GetLogDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的日志ID")
		return
	}

	var log models.TaskLog
	res := database.DB.Where("id = ?", id).Limit(1).Find(&log)
	if res.Error != nil || res.RowsAffected == 0 {
		utils.NotFound(c, "日志不存在")
		return
	}

	logVO := vo.ToTaskLogVO(&log)
	if logVO != nil {
		var task models.Task
		var taskPtr *models.Task
		if taskRes := database.DB.Where("id = ?", log.TaskID).Limit(1).Find(&task); taskRes.Error == nil && taskRes.RowsAffected > 0 {
			taskPtr = &task
		}
		displayName, taskType, taskDeleted := resolveTaskLogInfo(taskPtr, &log)
		logVO.TaskName = displayName
		logVO.TaskType = taskType
		logVO.TaskDeleted = taskDeleted
	}

	utils.Success(c, logVO)
}

// resolveTaskLogInfo 解析任务日志的显示名称、类型与已删除状态
func resolveTaskLogInfo(task *models.Task, log *models.TaskLog) (displayName string, taskType string, taskDeleted bool) {
	taskType = "task"
	if task != nil && task.ID != "" {
		if task.Type != "" {
			taskType = task.Type
		}
		if task.Name != "" {
			return task.Name, taskType, false
		}
		if log != nil && log.TaskName != "" {
			return log.TaskName, taskType, false
		}
		return "", taskType, false
	}

	// 任务记录不存在，判定为已删除
	taskDeleted = true
	if log != nil && log.TaskName != "" {
		displayName = DeletedTaskPrefix + log.TaskName
	} else {
		displayName = DeletedTaskPlaceholder
	}
	return displayName, taskType, taskDeleted
}

// ClearLogs 清空日志
func (lc *LogController) ClearLogs(c *gin.Context) {
	var req struct {
		TaskID *string `json:"task_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	query := database.DB.Model(&models.TaskLog{})
	if req.TaskID != nil && *req.TaskID != "" {
		query = query.Where("task_id = ?", *req.TaskID)
	} else {
		query = query.Where("1 = 1") // Allow delete all without GORM safety block
	}

	if err := query.Delete(&models.TaskLog{}).Error; err != nil {
		utils.ServerError(c, "清空日志失败")
		return
	}

	utils.SuccessMsg(c, "日志清空成功")
}

// DeleteLog 删除日志
func (lc *LogController) DeleteLog(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的日志ID")
		return
	}

	if err := database.DB.Where("id = ?", id).Delete(&models.TaskLog{}).Error; err != nil {
		utils.ServerError(c, "删除日志失败")
		return
	}

	utils.SuccessMsg(c, "日志已删除")
}
