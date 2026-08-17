package controllers

import (
	"os"
	"path/filepath"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type WebUIController struct {
	webuiService *services.WebUIService
}

func NewWebUIController(webuiService *services.WebUIService) *WebUIController {
	return &WebUIController{
		webuiService: webuiService,
	}
}

// GetWebUIs 获取所有WebUI
func (c *WebUIController) GetWebUIs(ctx *gin.Context) {
	webuis, err := c.webuiService.GetWebUIs()
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}
	utils.Success(ctx, webuis)
}

// UploadWebUI 上传新WebUI
func (c *WebUIController) UploadWebUI(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		utils.BadRequest(ctx, "获取上传文件失败")
		return
	}

	// 临时保存上传的文件到挂载目录，避免 /tmp 跨分区移动或权限问题
	tmpDir := filepath.Join(constant.DataDir, "tmp")
	os.MkdirAll(tmpDir, 0755)
	tmpFile := filepath.Join(tmpDir, file.Filename)
	
	if err := ctx.SaveUploadedFile(file, tmpFile); err != nil {
		utils.ServerError(ctx, "保存临时文件失败")
		return
	}
	defer os.Remove(tmpFile) // 自动清理临时文件

	webuiName, err := c.webuiService.ExtractWebUI(tmpFile)
	if err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"message": "WebUI上传成功", "webui": webuiName})
}

// SetActiveWebUI 切换活动WebUI
func (c *WebUIController) SetActiveWebUI(ctx *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "无效的请求参数")
		return
	}

	if err := c.webuiService.SetActiveWebUI(req.Name); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"message": "WebUI已切换成功，部分页面可能需要刷新"})
}

// DeleteWebUI 删除自定义WebUI
func (c *WebUIController) DeleteWebUI(ctx *gin.Context) {
	name := ctx.Param("name")
	if name == "" {
		utils.BadRequest(ctx, "未提供WebUI名称")
		return
	}

	if err := c.webuiService.DeleteWebUI(name); err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"message": "WebUI已删除"})
}
