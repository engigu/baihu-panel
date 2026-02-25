package controllers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type ExportImportController struct {
	eiService *services.ExportImportService
}

func NewExportImportController() *ExportImportController {
	return &ExportImportController{
		eiService: services.NewExportImportService(),
	}
}

func (c *ExportImportController) ExportData(ctx *gin.Context) {
	var params services.ExportParams
	if err := ctx.ShouldBindJSON(&params); err != nil {
		utils.BadRequest(ctx, "无效的输入参数")
		return
	}

	zipPath, err := c.eiService.ExportData(params)
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"file_path": zipPath})
}

// DownloadExport 下载导出的文件
func (c *ExportImportController) DownloadExport(ctx *gin.Context) {
	zipPath := c.eiService.GetExportFile()
	if zipPath == "" {
		utils.NotFound(ctx, "暂无可用的导出文件")
		return
	}

	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		utils.NotFound(ctx, "服务器上未找到导出文件，可能已过期删除")
		return
	}

	// Use filepath.Base to get just the filename
	filename := filepath.Base(zipPath)
	ctx.FileAttachment(zipPath, filename)
}

func (c *ExportImportController) ImportData(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		utils.BadRequest(ctx, "文件上传失败")
		return
	}

	if err := os.MkdirAll(services.BackupDir, 0755); err != nil {
		utils.ServerError(ctx, "创建目录失败")
		return
	}

	zipPath := fmt.Sprintf("%s/import_%s", services.BackupDir, file.Filename)
	if err := ctx.SaveUploadedFile(file, zipPath); err != nil {
		utils.ServerError(ctx, "保存文件失败")
		return
	}
	defer os.Remove(zipPath)

	var params services.ImportParams
	params.ZipPath = zipPath
	params.Password = ctx.PostForm("password")
	params.ImportScript = ctx.PostForm("import_script") == "true"
	params.ImportEnv = ctx.PostForm("import_env") == "true"
	params.ImportTask = ctx.PostForm("import_task") == "true"

	if !params.ImportScript && !params.ImportEnv && !params.ImportTask {
		utils.BadRequest(ctx, "至少需要选择一项导入类别")
		return
	}

	if err := c.eiService.ImportData(params); err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.SuccessMsg(ctx, "导入成功")
}
