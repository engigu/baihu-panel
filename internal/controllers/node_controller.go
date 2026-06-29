package controllers

import (
	"time"

	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type NodeController struct {
	nodeService  *services.NodeService
	agentService *services.AgentService
}

func NewNodeController() *NodeController {
	return &NodeController{
		nodeService:  services.NewNodeService(),
		agentService: services.NewAgentService(),
	}
}

// List 获取所有节点列表 (聚合了 Runner & Panel)
func (c *NodeController) List(ctx *gin.Context) {
	nodes, err := c.nodeService.List()
	if err != nil {
		utils.ServerError(ctx, err.Error())
		return
	}
	utils.Success(ctx, nodes)
}

// Create 创建节点（如果是 Panel 则新建互联节点，如果是 Runner 则新建 Token）
func (c *NodeController) Create(ctx *gin.Context) {
	var req struct {
		Type   string `json:"type" binding:"required"` // "runner" 或 "panel"
		Name   string `json:"name" binding:"required"`
		URL    string `json:"url"`
		Token  string `json:"token"`
		Remark string `json:"remark"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误: "+err.Error())
		return
	}

	res, err := c.nodeService.CreateNode(req.Type, req.Name, req.URL, req.Token, req.Remark)
	if err != nil {
		utils.ServerError(ctx, "创建节点失败: "+err.Error())
		return
	}
	utils.Success(ctx, res)
}

// Update 更新节点信息
func (c *NodeController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}

	var req struct {
		Type    string `json:"type" binding:"required"` // "runner" 或 "panel"
		Name    string `json:"name" binding:"required"`
		Remark  string `json:"remark"`
		Enabled bool   `json:"enabled"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误: "+err.Error())
		return
	}

	res, err := c.nodeService.UpdateNode(id, req.Type, req.Name, req.Remark, req.Enabled)
	if err != nil {
		utils.ServerError(ctx, "更新节点失败: "+err.Error())
		return
	}
	utils.Success(ctx, res)
}

// Delete 删除节点
func (c *NodeController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}
	nodeType := ctx.Query("type")
	if nodeType == "" {
		utils.BadRequest(ctx, "必须提供节点类型 type")
		return
	}

	err := c.nodeService.DeleteNode(id, nodeType)
	if err != nil {
		utils.ServerError(ctx, "删除节点失败: "+err.Error())
		return
	}
	utils.Success(ctx, nil)
}

// ListTokens 获取 Runner 的注册令牌列表
func (c *NodeController) ListTokens(ctx *gin.Context) {
	tokens := c.agentService.ListTokens()
	utils.Success(ctx, tokens)
}

// CreateToken 创建 Runner 注册令牌
func (c *NodeController) CreateToken(ctx *gin.Context) {
	var req struct {
		Remark    string `json:"remark"`
		MaxUses   int    `json:"max_uses"`
		ExpiresAt string `json:"expires_at"` // 格式: 2006-01-02 15:04:05
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "参数错误")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", req.ExpiresAt, time.Local)
		if err != nil {
			utils.BadRequest(ctx, "过期时间格式错误")
			return
		}
		expiresAt = &t
	}

	token, err := c.agentService.CreateToken(req.Remark, req.MaxUses, expiresAt)
	if err != nil {
		utils.ServerError(ctx, "创建令牌失败")
		return
	}
	utils.Success(ctx, token)
}

// DeleteToken 删除 Runner 注册令牌
func (c *NodeController) DeleteToken(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		utils.BadRequest(ctx, "无效的 ID")
		return
	}
	err := c.agentService.DeleteToken(id)
	if err != nil {
		utils.ServerError(ctx, "删除令牌失败")
		return
	}
	utils.Success(ctx, nil)
}
