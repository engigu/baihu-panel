package controllers

import (
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type MiseController struct {
	service *services.MiseService
}

func NewMiseController(service *services.MiseService) *MiseController {
	return &MiseController{
		service: service,
	}
}

// List 获取语言列表
func (c *MiseController) List(ctx *gin.Context) {
	langs, err := c.service.List()
	if err != nil {
		utils.ServerError(ctx, "获取语言列表失败: "+err.Error())
		return
	}
	utils.Success(ctx, langs)
}

// Sync 同步本地环境到数据库
func (c *MiseController) Sync(ctx *gin.Context) {
	if err := c.service.Sync(); err != nil {
		utils.ServerError(ctx, "同步本地环境失败: "+err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// Plugins 获取可用插件列表
func (c *MiseController) Plugins(ctx *gin.Context) {
	plugins, err := c.service.Plugins()
	if err != nil {
		utils.ServerError(ctx, "获取插件列表失败: "+err.Error())
		return
	}
	utils.Success(ctx, plugins)
}

// Versions 获取指定插件的可用版本列表
func (c *MiseController) Versions(ctx *gin.Context) {
	plugin := ctx.Query("plugin")
	if plugin == "" {
		utils.BadRequest(ctx, "参数 plugin 不能为空")
		return
	}
	versions, err := c.service.Versions(plugin)
	if err != nil {
		utils.ServerError(ctx, "获取版本列表失败: "+err.Error())
		return
	}
	utils.Success(ctx, versions)
}

// VerifyCommand 获取验证命令
func (c *MiseController) VerifyCommand(ctx *gin.Context) {
	plugin := ctx.Query("plugin")
	version := ctx.Query("version")
	if plugin == "" {
		utils.BadRequest(ctx, "参数 plugin 不能为空")
		return
	}
	cmd, err := c.service.GetVerifyCommand(plugin, version)
	if err != nil {
		utils.ServerError(ctx, "获取验证命令失败: "+err.Error())
		return
	}
	utils.Success(ctx, gin.H{"command": cmd})
}
