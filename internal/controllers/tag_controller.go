package controllers

import (
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type TagController struct {
	tagService *services.TagService
}

func NewTagController(tagService *services.TagService) *TagController {
	return &TagController{
		tagService: tagService,
	}
}

// GetTags 获取标签列表 (带关联统计)
// @Summary 获取标签列表
// @Description 获取标签列表，支持分页、模糊搜索、类型过滤以及统计关联的实体数
// @Tags 标签管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码"
// @Param page_size query int false "每页大小"
// @Param name query string false "标签名模糊搜索"
// @Param type query string false "标签类型 (task_tag, env_tag)"
// @Success 200 {object} utils.Response{data=[]services.TagWithCount}
// @Router /tags [get]
func (tc *TagController) GetTags(c *gin.Context) {
	p := utils.ParsePagination(c)
	name := c.Query("name")
	relType := c.Query("type")

	tags, total, err := tc.tagService.GetTagsWithPagination(p.Page, p.PageSize, name, relType)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.PaginatedResponse(c, tags, total, p)
}

// CreateTag 手动新建标签
// @Summary 新建标签
// @Description 手动新建标签
// @Tags 标签管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=models.DataStorage}
// @Router /tags [post]
func (tc *TagController) CreateTag(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	tag, err := tc.tagService.CreateTag(req.Name, req.Type)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.Success(c, tag)
}

// UpdateTag 重命名标签
// @Summary 修改标签
// @Description 重命名标签名称
// @Tags 标签管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "标签ID"
// @Success 200 {object} utils.Response
// @Router /tags/{id} [put]
func (tc *TagController) UpdateTag(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if err := tc.tagService.RenameTag(id, req.Name); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.SuccessMsg(c, "更新成功")
}

// DeleteTag 删除标签
// @Summary 删除标签
// @Description 删除标签，并清理其所有对应绑定关系
// @Tags 标签管理
// @Produce json
// @Security BearerAuth
// @Param id path string true "标签ID"
// @Success 200 {object} utils.Response
// @Router /tags/{id} [delete]
func (tc *TagController) DeleteTag(c *gin.Context) {
	id := c.Param("id")
	if err := tc.tagService.DeleteTag(id); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}
	utils.SuccessMsg(c, "删除成功")
}
