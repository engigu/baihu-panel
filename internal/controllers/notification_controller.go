package controllers

import (
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	notifyService *services.NotificationService
}

func NewNotificationController() *NotificationController {
	return &NotificationController{
		notifyService: services.NewNotificationService(),
	}
}

// GetChannelTypes 获取支持的渠道类型
func (nc *NotificationController) GetChannelTypes(c *gin.Context) {
	utils.Success(c, gin.H{
		"channel_types": services.SupportedChannelTypes,
		"event_types":   services.SupportedEvents,
	})
}

// GetChannels 获取所有渠道
func (nc *NotificationController) GetChannels(c *gin.Context) {
	channels := nc.notifyService.GetChannels()
	utils.Success(c, channels)
}

// SaveChannel 保存/更新渠道
func (nc *NotificationController) SaveChannel(c *gin.Context) {
	var req services.NotifyChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	if req.Name == "" || req.Type == "" {
		utils.BadRequest(c, "渠道名称和类型不能为空")
		return
	}

	if err := nc.notifyService.SaveChannel(req); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.SuccessMsg(c, "保存成功")
}

// DeleteChannel 删除渠道
func (nc *NotificationController) DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "缺少渠道ID")
		return
	}

	if err := nc.notifyService.DeleteChannel(id); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.SuccessMsg(c, "删除成功")
}

// TestChannel 测试渠道
func (nc *NotificationController) TestChannel(c *gin.Context) {
	var req services.NotifyChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	result := nc.notifyService.SendToChannel(req, &services.NotifyMessage{
		Title: "🔔 白虎面板测试通知",
		Text:  "如果你看到这条消息，说明通知渠道配置正确！",
	})

	utils.Success(c, result)
}


// GetBindings 获取事件绑定列表
func (nc *NotificationController) GetBindings(c *gin.Context) {
	bindings := nc.notifyService.GetBindings()
	utils.Success(c, bindings)
}

// SaveBinding 保存事件绑定
func (nc *NotificationController) SaveBinding(c *gin.Context) {
	var req struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Event  string `json:"event"`
		WayID  string `json:"way_id"`
		DataID string `json:"data_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	if req.Type == "" || req.Event == "" || req.WayID == "" {
		utils.BadRequest(c, "类型、事件和渠道ID不能为空")
		return
	}

	binding := &models.NotifyBinding{
		ID:     req.ID,
		Type:   req.Type,
		Event:  req.Event,
		WayID:  req.WayID,
		DataID: req.DataID,
	}

	if err := nc.notifyService.SaveBinding(binding); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.Success(c, binding)
}

// DeleteBinding 删除事件绑定
func (nc *NotificationController) DeleteBinding(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "缺少绑定ID")
		return
	}

	if err := nc.notifyService.DeleteBinding(id); err != nil {
		utils.ServerError(c, err.Error())
		return
	}

	utils.SuccessMsg(c, "删除成功")
}

// SendNotification API 发送通知（供脚本调用）
func (nc *NotificationController) SendNotification(c *gin.Context) {
	var req struct {
		ChannelID string `json:"channel_id"`
		Title     string `json:"title"`
		Text      string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数错误")
		return
	}

	if req.ChannelID == "" || req.Title == "" {
		utils.BadRequest(c, "channel_id 和 title 不能为空")
		return
	}

	result := nc.notifyService.SendByChannelID(req.ChannelID, &services.NotifyMessage{
		Title: req.Title,
		Text:  req.Text,
	})

	utils.Success(c, result)
}
