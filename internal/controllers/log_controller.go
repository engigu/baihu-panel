package controllers

import (
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type LogController struct {
	taskLogService *tasks.TaskLogService
}

func NewLogController(taskLogService ...*tasks.TaskLogService) *LogController {
	var svc *tasks.TaskLogService
	if len(taskLogService) > 0 && taskLogService[0] != nil {
		svc = taskLogService[0]
	} else {
		svc = tasks.NewTaskLogService(services.NewSendStatsService())
	}
	return &LogController{taskLogService: svc}
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

	result, total := lc.taskLogService.GetLogsWithPagination(p.Page, p.PageSize, taskID, taskName, status, date)
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

	logVO, err := lc.taskLogService.GetLogDetailByID(id)
	if err != nil {
		utils.NotFound(c, err.Error())
		return
	}

	utils.Success(c, logVO)
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

	targetTaskID := ""
	if req.TaskID != nil {
		targetTaskID = *req.TaskID
	}

	_, err := lc.taskLogService.ClearLogs(targetTaskID, 0)
	if err != nil {
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
