package controllers

import (
	"strings"

	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/models/vo"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/deps"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type DependencyController struct {
	service *services.DependencyService
}

func NewDependencyController() *DependencyController {
	return &DependencyController{
		service: services.NewDependencyService(),
	}
}

// List 获取依赖列表
func (c *DependencyController) List(ctx *gin.Context) {
	language := ctx.Query("language")
	langVersion := ctx.Query("lang_version")
	deps, err := c.service.List(language, langVersion)
	if err != nil {
		utils.ServerError(ctx, "获取依赖列表失败")
		return
	}
	vos := vo.ToDependencyVOListFromModels(deps)
	utils.Success(ctx, vos)
}

// Create 添加依赖
func (c *DependencyController) Create(ctx *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Version     string `json:"version"`
		Language    string `json:"language" binding:"required"`
		LangVersion string `json:"lang_version"`
		Remark      string `json:"remark"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	dep := &models.Dependency{
		Name:        req.Name,
		Version:     req.Version,
		Language:    req.Language,
		LangVersion: req.LangVersion,
		Remark:      req.Remark,
	}

	if err := c.service.Create(dep); err != nil {
		utils.BadRequest(ctx, err.Error())
		return
	}

	utils.Success(ctx, vo.ToDependencyVO(dep))
}

// Delete 删除依赖
func (c *DependencyController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	if err := c.service.Delete(id); err != nil {
		utils.ServerError(ctx, "删除失败")
		return
	}

	utils.SuccessMsg(ctx, "删除成功")
}

func (c *DependencyController) Install(ctx *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Version     string `json:"version"`
		Language    string `json:"language"`
		LangVersion string `json:"lang_version"`
		Remark      string `json:"remark"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	language := req.Language
	if language == "" {
		language = ctx.Query("language")
	}
	langVersion := req.LangVersion
	if langVersion == "" {
		langVersion = ctx.Query("lang_version")
	}

	dep := &models.Dependency{
		Name:        req.Name,
		Version:     req.Version,
		Language:    language,
		LangVersion: langVersion,
		Remark:      req.Remark,
	}

	err := c.service.Install(dep)
	// 无论成功失败，都同步记录日志
	c.service.Create(dep)

	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}
	utils.SuccessMsg(ctx, "安装成功")
}

// GetInstallCommand 获取安装命令
func (c *DependencyController) GetInstallCommand(ctx *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Version     string `json:"version"`
		Language    string `json:"language"`
		LangVersion string `json:"lang_version"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	language := req.Language
	if language == "" {
		language = ctx.Query("language")
	}
	langVersion := req.LangVersion
	if langVersion == "" {
		langVersion = ctx.Query("lang_version")
	}

	dep := &models.Dependency{
		Name:        req.Name,
		Version:     req.Version,
		Language:    language,
		LangVersion: langVersion,
	}

	cmd, err := c.service.GetInstallCommand(dep)
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"command": cmd})
}

// GetReinstallAllCommand 获取全部重装命令
func (c *DependencyController) GetReinstallAllCommand(ctx *gin.Context) {
	language := ctx.Query("language")
	langVersion := ctx.Query("lang_version")
	if language == "" {
		utils.BadRequest(ctx, "缺少 language 参数")
		return
	}

	cmd, err := c.service.GetReinstallAllCommand(language, langVersion)
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"command": cmd})
}

// Uninstall 卸载依赖
func (c *DependencyController) Uninstall(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	force := ctx.Query("force") == "true"

	// 获取依赖信息
	deps, _ := c.service.List("", "")
	var dep *models.Dependency
	for i := range deps {
		if deps[i].ID == id {
			dep = &deps[i]
			break
		}
	}

	if dep == nil {
		utils.NotFound(ctx, "依赖不存在")
		return
	}

	if err := c.service.Uninstall(dep); err != nil {
		if !force {
			utils.ServerError(ctx, err.Error())
			return
		}
	}

	// 卸载成功（或强制删除）后从数据库删除
	c.service.Delete(id)

	utils.SuccessMsg(ctx, "卸载成功")
}

// Reinstall 重新安装依赖
func (c *DependencyController) Reinstall(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	// 获取依赖信息
	deps, _ := c.service.List("", "")
	var dep *models.Dependency
	for i := range deps {
		if deps[i].ID == id {
			dep = &deps[i]
			break
		}
	}

	if dep == nil {
		utils.NotFound(ctx, "依赖不存在")
		return
	}

	err := c.service.Install(dep)
	// 无论成功失败，都同步记录日志
	c.service.Create(dep)

	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}
	utils.SuccessMsg(ctx, "重新安装成功")
}

// ReinstallAll 重新安装所有依赖
func (c *DependencyController) ReinstallAll(ctx *gin.Context) {
	language := ctx.Query("language")
	langVersion := ctx.Query("lang_version")
	if language == "" {
		utils.BadRequest(ctx, "缺少 language 参数")
		return
	}

	deps, err := c.service.List(language, langVersion)
	if err != nil {
		utils.ServerError(ctx, "获取依赖列表失败")
		return
	}

	var failed []string
	for i := range deps {
		d := &deps[i]
		err := c.service.Install(d)
		if err != nil {
			failed = append(failed, d.Name)
		}
		// 无论成功失败，都同步记录日志到数据库
		c.service.Create(d)
	}

	if len(failed) > 0 {
		utils.ServerError(ctx, "部分包安装失败: "+strings.Join(failed, ", "))
		return
	}

	utils.SuccessMsg(ctx, "全部重新安装成功")
}

// GetInstalled 获取已安装的包
func (c *DependencyController) GetInstalled(ctx *gin.Context) {
	language := ctx.Query("language")
	langVersion := ctx.Query("lang_version")
	if language == "" {
		utils.BadRequest(ctx, "缺少 language 参数")
		return
	}

	packages, err := c.service.GetInstalledPackages(language, langVersion)
	if err != nil {
		utils.ServerError(ctx, "获取已安装包失败: "+err.Error())
		return
	}

	utils.Success(ctx, packages)
}

// GetBatchInstallCommand 获取批量安装依赖包的命令
func (c *DependencyController) GetBatchInstallCommand(ctx *gin.Context) {
	var req struct {
		Items []struct {
			Name        string `json:"name" binding:"required"`
			Version     string `json:"version"`
			Language    string `json:"language" binding:"required"`
			LangVersion string `json:"lang_version"`
		} `json:"items" binding:"required,gt=0"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误: items 不能为空且必须包含 name 和 language")
		return
	}

	var depsList []models.Dependency
	for _, item := range req.Items {
		depsList = append(depsList, models.Dependency{
			Name:        item.Name,
			Version:     item.Version,
			Language:    item.Language,
			LangVersion: item.LangVersion,
		})
	}

	cmd, err := c.service.GetBatchInstallCommand(depsList)
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}

	utils.Success(ctx, gin.H{"command": cmd})
}

// ParseAndImport 解析上传/粘贴的清单文件内容并批量导入至数据库
func (c *DependencyController) ParseAndImport(ctx *gin.Context) {
	var req struct {
		Language    string `json:"language" binding:"required"`
		LangVersion string `json:"lang_version"`
		Content     string `json:"content" binding:"required"`
		ImportDB    bool   `json:"import_db"` // 是否持久化到数据库做可视化管理
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误: language 和 content 必填")
		return
	}

	// 1. 解析文本清单内容
	parsedDeps, err := deps.ParseManifest(req.Language, req.Content)
	if err != nil {
		utils.ServerError(ctx, "清单文件解析失败: "+err.Error())
		return
	}

	if len(parsedDeps) == 0 {
		utils.BadRequest(ctx, "未解析到任何有效依赖包")
		return
	}

	// 2. 补全语言和版本属性
	for i := range parsedDeps {
		parsedDeps[i].Language = req.Language
		parsedDeps[i].LangVersion = req.LangVersion
	}

	// 3. 根据需求决定是否导入数据库
	var finalDeps []models.Dependency
	if req.ImportDB {
		imported, err := c.service.ImportDependencies(parsedDeps)
		if err != nil {
			utils.ServerError(ctx, "导入依赖记录至数据库失败: "+err.Error())
			return
		}
		finalDeps = imported
	} else {
		finalDeps = parsedDeps
	}

	// 4. 为这一批包生成合并批量安装命令
	cmd, err := c.service.GetBatchInstallCommand(finalDeps)
	if err != nil {
		utils.ServerError(ctx, "生成安装命令失败: "+err.Error())
		return
	}

	utils.Success(ctx, gin.H{
		"dependencies": vo.ToDependencyVOListFromModels(finalDeps),
		"command":      cmd,
	})
}
