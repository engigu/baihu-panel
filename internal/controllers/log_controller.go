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

// GetLogs 获取任务日志列表
// @Summary 获取任务日志列表
// @Description 分页获取任务日志列表，支持按任务 ID、任务名称、状态筛选
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param task_id query string false "任务 ID"
// @Param task_name query string false "任务名称"
// @Param status query string false "状态"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} utils.Response{data=utils.PaginationData{data=[]vo.TaskLogVO}}
// @Router /logs [get]
func (lc *LogController) GetLogs(c *gin.Context) {
	p := utils.ParsePagination(c)
	taskID := c.DefaultQuery("task_id", "")
	taskName := c.DefaultQuery("task_name", "")
	status := c.DefaultQuery("status", "")
	workflowID, _ := strconv.Atoi(c.DefaultQuery("workflow_id", "0"))
	workflowRunID := c.DefaultQuery("workflow_run_id", "")

	var logs []models.TaskLog
	var total int64

	query := database.DB.Model(&models.TaskLog{})
	if taskID != "" {
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
		var taskIDs []string
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

	taskIDList := make([]string, 0)
	for _, log := range logs {
		taskIDList = append(taskIDList, log.TaskID)
	}

	var tasks []models.Task
	database.DB.Where("id IN ?", taskIDList).Find(&tasks)
	taskMap := make(map[string]models.Task)
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
			Command:   string(log.Command),
			Status:    log.Status,
			Duration:  log.Duration,
			StartTime: log.StartTime,
			EndTime:   log.EndTime,
			CreatedAt: log.CreatedAt,
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
	if err := database.DB.Where("id = ?", id).First(&log).Error; err != nil {
		utils.NotFound(c, "日志不存在")
		return
	}

	utils.Success(c, vo.ToTaskLogVO(&log))
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
