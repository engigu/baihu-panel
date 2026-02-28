package controllers

import (
	"strconv"

	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
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

func (ec *EnvController) CreateEnvVar(c *gin.Context) {
	var user models.User
	database.DB.First(&user)
	userUUID := user.UUID

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

	envVar := ec.envService.CreateEnvVar(req.Name, req.Value, req.Remark, hidden, userUUID)
	utils.Success(c, vo.ToEnvVO(envVar))
}

func (ec *EnvController) GetEnvVars(c *gin.Context) {
	var user models.User
	database.DB.First(&user)
	userUUID := user.UUID
	p := utils.ParsePagination(c)
	name := c.DefaultQuery("name", "")
	envVars, total := ec.envService.GetEnvVarsWithPagination(userUUID, name, p.Page, p.PageSize)
	utils.PaginatedResponse(c, vo.ToEnvVOListFromModels(envVars), total, p)
}

func (ec *EnvController) GetAllEnvVars(c *gin.Context) {
	var user models.User
	database.DB.First(&user)
	userUUID := user.UUID
	envVars := ec.envService.GetEnvVarsByUserID(userUUID)
	utils.Success(c, vo.ToEnvVOListFromModels(envVars))
}

func (ec *EnvController) GetEnvVar(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
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

func (ec *EnvController) UpdateEnvVar(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
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

func (ec *EnvController) DeleteEnvVar(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "无效的环境变量ID")
		return
	}

	success := ec.envService.DeleteEnvVar(id)
	if !success {
		utils.NotFound(c, "环境变量不存在")
		return
	}

	utils.SuccessMsg(c, "删除成功")
}
