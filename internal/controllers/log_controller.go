package controllers

import (
	"strconv"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type LogController struct{}

func NewLogController() *LogController {
	return &LogController{}
}

func (lc *LogController) GetLogs(c *gin.Context) {
	p := utils.ParsePagination(c)
	taskID, _ := strconv.Atoi(c.DefaultQuery("task_id", "0"))
	taskName := c.DefaultQuery("task_name", "")
	status := c.DefaultQuery("status", "")
	workflowID, _ := strconv.Atoi(c.DefaultQuery("workflow_id", "0"))
	workflowRunID := c.DefaultQuery("workflow_run_id", "")

	var logs []models.TaskLog
	var total int64

	query := database.DB.Model(&models.TaskLog{})
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if workflowID > 0 {
		query = query.Where("workflow_id = ?", workflowID)
	}
	if workflowRunID != "" {
		query = query.Where("workflow_run_id = ?", workflowRunID)
	}

	// 按任务名称过滤
	if taskName != "" {
		var taskIDs []uint
		database.DB.Model(&models.Task{}).Where("name LIKE ?", "%"+taskName+"%").Pluck("id", &taskIDs)
		if len(taskIDs) > 0 {
			query = query.Where("task_id IN ?", taskIDs)
		} else {
			utils.PaginatedResponse(c, []vo.TaskLogVO{}, 0, p)
			return
		}
	}

	query.Count(&total)
	query.Order("id DESC").Offset(p.Offset()).Limit(p.PageSize).Find(&logs)

	taskIDList := make([]uint, 0)
	for _, log := range logs {
		taskIDList = append(taskIDList, log.TaskID)
	}

	var tasks []models.Task
	database.DB.Where("id IN ?", taskIDList).Find(&tasks)
	taskMap := make(map[uint]models.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	result := make([]vo.TaskLogVO, len(logs))
	for i, log := range logs {
		task := taskMap[log.TaskID]
		taskType := task.Type
		if taskType == "" {
			taskType = "task"
		}
		result[i] = vo.TaskLogVO{
			ID:        log.ID,
			TaskID:    log.TaskID,
			TaskName:  task.Name,
			TaskType:  taskType,
			AgentID:   log.AgentID,
			Command:   log.Command,
			Status:    log.Status,
			Duration:  log.Duration,
			StartTime: log.StartTime,
			EndTime:   log.EndTime,
			CreatedAt: log.CreatedAt,
		}
	}

	utils.PaginatedResponse(c, result, total, p)
}

func (lc *LogController) GetLogDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "无效的日志ID")
		return
	}

	var log models.TaskLog
	if err := database.DB.First(&log, id).Error; err != nil {
		utils.NotFound(c, "日志不存在")
		return
	}

	utils.Success(c, vo.ToTaskLogVO(&log))
}

func (lc *LogController) ClearLogs(c *gin.Context) {
	var req struct {
		TaskID *int `json:"task_id"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	query := database.DB.Model(&models.TaskLog{})
	if req.TaskID != nil && *req.TaskID > 0 {
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

func (lc *LogController) DeleteLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "无效的日志ID")
		return
	}

	if err := database.DB.Delete(&models.TaskLog{}, id).Error; err != nil {
		utils.ServerError(c, "删除日志失败")
		return
	}

	utils.SuccessMsg(c, "日志已删除")
}
