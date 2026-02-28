package controllers

import (
	"strconv"

	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/gin-gonic/gin"
)

type WorkflowController struct {
	workflowService *services.WorkflowService
	executorService *tasks.ExecutorService
}

func NewWorkflowController(workflowService *services.WorkflowService, executorService *tasks.ExecutorService) *WorkflowController {
	return &WorkflowController{
		workflowService: workflowService,
		executorService: executorService,
	}
}

func (ctrl *WorkflowController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	name := c.Query("name")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	workflows, total, err := ctrl.workflowService.List(page, pageSize, name)
	if err != nil {
		utils.ServerError(c, "获取列表失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"data":      workflows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (ctrl *WorkflowController) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的ID")
		return
	}

	workflow, err := ctrl.workflowService.GetByID(id)
	if err != nil {
		utils.NotFound(c, "未找到该工作流")
		return
	}

	utils.Success(c, workflow)
}

func (ctrl *WorkflowController) Create(c *gin.Context) {
	var req models.Workflow
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求格式错误")
		return
	}

	if req.Name == "" {
		utils.BadRequest(c, "名称不能为空")
		return
	}

	if err := ctrl.workflowService.Create(&req); err != nil {
		utils.ServerError(c, "保存失败")
		return
	}

	utils.Success(c, req)
}

func (ctrl *WorkflowController) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的ID")
		return
	}

	var req models.Workflow
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求格式错误")
		return
	}

	req.ID = id
	if err := ctrl.workflowService.Update(&req); err != nil {
		utils.ServerError(c, "修改失败: "+err.Error())
		return
	}

	utils.SuccessMsg(c, "修改成功")
}

func (ctrl *WorkflowController) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的ID")
		return
	}

	if err := ctrl.workflowService.Delete(id); err != nil {
		utils.ServerError(c, "删除失败: "+err.Error())
		return
	}

	utils.SuccessMsg(c, "删除成功")
}

func (ctrl *WorkflowController) Run(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的ID")
		return
	}

	var req struct {
		Envs []string `json:"envs"`
	}
	c.ShouldBindJSON(&req)

	if err := ctrl.executorService.TriggerWorkflow(id, req.Envs); err != nil {
		utils.ServerError(c, "工作流触发失败: "+err.Error())
		return
	}

	utils.SuccessMsg(c, "工作流已成功启动后台运行")
}
