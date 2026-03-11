package controllers

import (

	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type EnvController struct {
	envService *services.EnvService
}

func NewEnvController(envService *services.EnvService) *EnvController {
	return &EnvController{envService: envService}
}

// CreateEnvVar 创建环境变量
// @Summary 创建环境变量
// @Description 创建一个新的环境变量
// @Tags 环境变量
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "环境变量信息"
// @Success 200 {object} utils.Response{data=vo.EnvVO}
// @Router /env [post]
func (ec *EnvController) CreateEnvVar(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		Name   string `json:"name" binding:"required"`
		Value  string `json:"value" binding:"required"`
		Remark string `json:"remark"`
		Hidden *bool  `json:"hidden"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	hidden := true
	if req.Hidden != nil {
		hidden = *req.Hidden
	}

	envVar := ec.envService.CreateEnvVar(req.Name, req.Value, req.Remark, hidden, userID)
	utils.Success(c, vo.ToEnvVO(envVar))
}

// GetEnvVars 获取环境变量列表
// @Summary 获取环境变量列表
// @Description 分页获取环境变量列表，支持按名称筛选
// @Tags 环境变量
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name query string false "按名称模糊查询"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} utils.Response{data=utils.PaginationData{data=[]vo.EnvVO}}
// @Router /env [get]
func (ec *EnvController) GetEnvVars(c *gin.Context) {
	userID := c.GetString("userID")
	p := utils.ParsePagination(c)
	name := c.DefaultQuery("name", "")
	envVars, total := ec.envService.GetEnvVarsWithPagination(userID, name, p.Page, p.PageSize)
	utils.PaginatedResponse(c, vo.ToEnvVOListFromModels(envVars), total, p)
}

// GetAllEnvVars 获取所有环境变量
// @Summary 获取所有环境变量
// @Description 获取当前用户的所有环境变量（不分页）
// @Tags 环境变量
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=[]vo.EnvVO}
// @Router /env/all [get]
func (ec *EnvController) GetAllEnvVars(c *gin.Context) {
	userID := c.GetString("userID")
	envVars := ec.envService.GetEnvVarsByUserID(userID)
	utils.Success(c, vo.ToEnvVOListFromModels(envVars))
}

// GetEnvVar 获取环境变量详情
// @Summary 获取环境变量详情
// @Description 根据 ID 获取环境变量详情
// @Tags 环境变量
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "环境变量ID"
// @Success 200 {object} utils.Response{data=vo.EnvVO}
// @Failure 404 {object} utils.Response
// @Router /env/{id} [get]
func (ec *EnvController) GetEnvVar(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的环境变量ID")
		return
	}

	envVar := ec.envService.GetEnvVarByID(id)
	if envVar == nil {
		utils.NotFound(c, "环境变量不存在")
		return
	}

	utils.Success(c, vo.ToEnvVO(envVar))
}

// UpdateEnvVar 更新环境变量
// @Summary 更新环境变量
// @Description 根据 ID 更新环境变量信息
// @Tags 环境变量
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "环境变量ID"
// @Param body body object true "环境变量更新信息"
// @Success 200 {object} utils.Response{data=vo.EnvVO}
// @Failure 404 {object} utils.Response
// @Router /env/{id} [put]
func (ec *EnvController) UpdateEnvVar(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的环境变量ID")
		return
	}

	var req struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Remark string `json:"remark"`
		Hidden *bool  `json:"hidden"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 对于更新，获取现有数据
	existing := ec.envService.GetEnvVarByID(id)
	if existing == nil {
		utils.NotFound(c, "环境变量不存在")
		return
	}

	hidden := existing.Hidden
	if req.Hidden != nil {
		hidden = *req.Hidden
	}
	envVar := ec.envService.UpdateEnvVar(id, req.Name, req.Value, req.Remark, hidden)
	if envVar == nil {
		utils.NotFound(c, "环境变量不存在")
		return
	}

	utils.Success(c, vo.ToEnvVO(envVar))
}

// DeleteEnvVar 删除环境变量
// @Summary 删除环境变量
// @Description 根据 ID 删除环境变量
// @Tags 环境变量
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "环境变量ID"
// @Param force query boolean false "强制删除（忽略任务关联）"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 409 {object} utils.Response{data=[]vo.TaskVO}
// @Router /env/{id} [delete]
func (ec *EnvController) DeleteEnvVar(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的环境变量ID")
		return
	}

	force := c.Query("force") == "true"
	success, associatedTasks := ec.envService.DeleteEnvVar(id, force)

	if len(associatedTasks) > 0 {
		c.JSON(200, utils.Response{
			Code: 409,
			Msg:  "该环境变量已被任务引用，请先在任务中删除引用或选择强制删除",
			Data: vo.ToTaskVOListFromModels(associatedTasks),
		})
		return
	}

	if !success {
		utils.NotFound(c, "环境变量不存在或删除失败")
		return
	}

	utils.SuccessMsg(c, "删除成功")
}

// GetAssociatedTasks 获取关联任务
// @Summary 获取关联任务
// @Description 获取引用了该环境变量的任务列表
// @Tags 环境变量
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "环境变量ID"
// @Success 200 {object} utils.Response{data=[]vo.TaskVO}
// @Router /env/{id}/tasks [get]
func (ec *EnvController) GetAssociatedTasks(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的环境变量ID")
		return
	}
	tasks := ec.envService.GetAssociatedTasks(id)
	utils.Success(c, vo.ToTaskVOListFromModels(tasks))
}
