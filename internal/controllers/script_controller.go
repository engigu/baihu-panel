package controllers

import (

	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type ScriptController struct {
	scriptService *services.ScriptService
}

func NewScriptController(scriptService *services.ScriptService) *ScriptController {
	return &ScriptController{scriptService: scriptService}
}

// CreateScript 创建脚本
// @Summary 创建脚本
// @Description 创建一个新的脚本
// @Tags 脚本管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "脚本信息"
// @Success 200 {object} utils.Response{data=vo.ScriptVO}
// @Router /scripts [post]
func (sc *ScriptController) CreateScript(c *gin.Context) {
	userID := c.GetString("userID")

	var req struct {
		Name    string `json:"name" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	script := sc.scriptService.CreateScript(req.Name, req.Content, userID)
	utils.Success(c, vo.ToScriptVO(script))
}

// GetScripts 获取脚本列表
// @Summary 获取脚本列表
// @Description 获取当前用户的所有脚本（内容字段为空）
// @Tags 脚本管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=[]vo.ScriptVO}
// @Router /scripts [get]
func (sc *ScriptController) GetScripts(c *gin.Context) {
	userID := c.GetString("userID")
	scripts := sc.scriptService.GetScriptsByUserID(userID)
	vos := vo.ToScriptVOListFromModels(scripts)
	for i := range vos {
		vos[i].Content = "" // 列表不返回内容
	}
	utils.Success(c, vos)
}

// GetScript 获取脚本详情
// @Summary 获取脚本详情
// @Description 根据 ID 获取脚本详情
// @Tags 脚本管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "脚本ID"
// @Success 200 {object} utils.Response{data=vo.ScriptVO}
// @Failure 404 {object} utils.Response
// @Router /scripts/{id} [get]
func (sc *ScriptController) GetScript(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的脚本ID")
		return
	}

	script := sc.scriptService.GetScriptByID(id)
	if script == nil {
		utils.NotFound(c, "脚本不存在")
		return
	}

	utils.Success(c, vo.ToScriptVO(script))
}

// UpdateScript 更新脚本
// @Summary 更新脚本
// @Description 根据 ID 更新脚本信息
// @Tags 脚本管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "脚本ID"
// @Param body body object true "脚本更新信息"
// @Success 200 {object} utils.Response{data=vo.ScriptVO}
// @Failure 404 {object} utils.Response
// @Router /scripts/{id} [put]
func (sc *ScriptController) UpdateScript(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的脚本ID")
		return
	}

	var req struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	script := sc.scriptService.UpdateScript(id, req.Name, req.Content)
	if script == nil {
		utils.NotFound(c, "脚本不存在")
		return
	}

	utils.Success(c, vo.ToScriptVO(script))
}

// DeleteScript 删除脚本
// @Summary 删除脚本
// @Description 根据 ID 删除脚本
// @Tags 脚本管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "脚本ID"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /scripts/{id} [delete]
func (sc *ScriptController) DeleteScript(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的脚本ID")
		return
	}

	success := sc.scriptService.DeleteScript(id)
	if !success {
		utils.NotFound(c, "脚本不存在")
		return
	}

	utils.SuccessMsg(c, "删除成功")
}
