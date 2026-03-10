package services

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/engigu/baihu-panel/internal/sdk/messenger"
)

// NotifyChannel 通知渠道配置
type NotifyChannel struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Enabled   bool              `json:"enabled"`
	CreatedAt models.LocalTime  `json:"created_at"`
	Config    map[string]string `json:"config"`
}

// NotifyMessage 通知消息
type NotifyMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// NotifyResult 发送结果
type NotifyResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// SupportedChannelTypes 支持的渠道类型
var SupportedChannelTypes = []map[string]string{
	{"type": messenger.ChannelTelegram, "label": "Telegram"},
	{"type": messenger.ChannelBark, "label": "Bark"},
	{"type": messenger.ChannelDtalk, "label": "钉钉"},
	{"type": messenger.ChannelQyWeiXin, "label": "企业微信"},
	{"type": messenger.ChannelFeishu, "label": "飞书"},
	{"type": messenger.ChannelEmail, "label": "邮件"},
	{"type": messenger.ChannelCustom, "label": "自定义Webhook"},
	{"type": messenger.ChannelNtfy, "label": "Ntfy"},
	{"type": messenger.ChannelGotify, "label": "Gotify"},
	{"type": messenger.ChannelPushMe, "label": "PushMe"},
	// {"type": messenger.ChannelWeChatOFAccount, "label": "微信公众号"},
	{"type": messenger.ChannelAliyunSMS, "label": "阿里云短信"},
	{"type": messenger.ChannelPushPlus, "label": "PushPlus"},
}

// SupportedEvents 支持的事件类型
var SupportedEvents = []map[string]string{
	{"type": constant.EventUserLogin, "label": "用户登录", "binding_type": constant.BindingTypeSystem},
	{"type": constant.EventBruteForceLogin, "label": "密码多次错误", "binding_type": constant.BindingTypeSystem},
	{"type": constant.EventPasswordChanged, "label": "密码修改", "binding_type": constant.BindingTypeSystem},
	{"type": constant.EventTaskSuccess, "label": "任务成功", "binding_type": constant.BindingTypeTask},
	{"type": constant.EventTaskFailed, "label": "任务失败", "binding_type": constant.BindingTypeTask},
	{"type": constant.EventTaskTimeout, "label": "任务超时", "binding_type": constant.BindingTypeTask},
}

type NotificationService struct {
	settingsService *SettingsService
	mu              sync.RWMutex
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		settingsService: NewSettingsService(),
	}
}

// GetChannels 获取所有渠道
func (s *NotificationService) GetChannels() []NotifyChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getChannelsInternal()
}

// SaveChannel 保存/更新渠道
func (s *NotificationService) SaveChannel(channel NotifyChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configJSON, err := json.Marshal(channel.Config)
	if err != nil {
		return err
	}

	if channel.ID == "" {
		// 新建
		channel.ID = utils.GenerateID()
		notifyWay := &models.NotifyWay{
			ID:      channel.ID,
			Name:    channel.Name,
			Type:    channel.Type,
			Config:  models.BigText(configJSON),
			Enabled: channel.Enabled,
		}
		return database.DB.Create(notifyWay).Error
	}

	// 更新
	updates := map[string]interface{}{
		"name":    channel.Name,
		"type":    channel.Type,
		"config":  models.BigText(configJSON),
		"enabled": channel.Enabled,
	}
	return database.DB.Model(&models.NotifyWay{}).Where("id = ?", channel.ID).Updates(updates).Error
}

// DeleteChannel 删除渠道
func (s *NotificationService) DeleteChannel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查渠道是否存在
	var count int64
	database.DB.Model(&models.NotifyWay{}).Where("id = ?", id).Count(&count)
	if count == 0 {
		return fmt.Errorf("渠道 %s 不存在", id)
	}

	// 删除渠道
	if err := database.DB.Unscoped().Where("id = ?", id).Delete(&models.NotifyWay{}).Error; err != nil {
		return err
	}

	// 同时清理事件绑定中引用此渠道的配置
	if err := database.DB.Unscoped().Where("way_id = ?", id).Delete(&models.NotifyBinding{}).Error; err != nil {
		logger.Errorf("[Notify] 清理事件绑定失败: %v", err)
	}

	return nil
}

// GetBindings 获取事件绑定列表（新接口，用于前端展示）
func (s *NotificationService) GetBindings() []models.NotifyBinding {
	var bindings []models.NotifyBinding
	database.DB.Find(&bindings)
	return bindings
}

// SaveBinding 保存事件绑定
func (s *NotificationService) SaveBinding(binding *models.NotifyBinding) error {
	if binding.ID == "" {
		// 检查是否已经存在相同的绑定（避免重复点击导致多个记录）
		var existing models.NotifyBinding
		err := database.DB.Where("type = ? AND event = ? AND way_id = ? AND data_id = ?",
			binding.Type, binding.Event, binding.WayID, binding.DataID).First(&existing).Error
		if err == nil {
			// 如果已存在且未删除，直接返回（或者更新它）
			*binding = existing
			return nil
		}

		binding.ID = utils.GenerateID()
		return database.DB.Create(binding).Error
	}
	return database.DB.Save(binding).Error
}

// DeleteBinding 删除事件绑定
func (s *NotificationService) DeleteBinding(id string) error {
	return database.DB.Unscoped().Where("id = ?", id).Delete(&models.NotifyBinding{}).Error
}

// GetBindingsByEvent 根据事件类型和数据ID获取绑定
func (s *NotificationService) GetBindingsByEvent(bindingType, event, dataID string) []models.NotifyBinding {
	var bindings []models.NotifyBinding

	// 如果是任务事件且带有 dataID，只获取特定任务的绑定（禁用全局任务配置）
	if bindingType == constant.BindingTypeTask && dataID != "" {
		database.DB.Where("type = ? AND event = ? AND data_id = ?", constant.BindingTypeTask, event, dataID).Find(&bindings)
		return bindings
	}

	// 对于系统事件或其他情况
	query := database.DB.Where("event = ?", event)
	if bindingType != "" {
		query = query.Where("type = ?", bindingType)
	}

	if dataID != "" {
		query = query.Where("data_id = ?", dataID)
	} else {
		query = query.Where("data_id = ? OR data_id IS NULL", "")
	}

	query.Find(&bindings)
	return bindings
}

// SendToChannel 使用 messenger SDK 发送通知到指定渠道
func (s *NotificationService) SendToChannel(channel NotifyChannel, msg *NotifyMessage) *NotifyResult {
	result, err := messenger.Send(channel.Type, messenger.ChannelConfig(channel.Config), &messenger.Message{
		Title: msg.Title,
		Text:  msg.Text,
	})

	payload := map[string]interface{}{
		"title":        msg.Title,
		"content":      msg.Text,
		"channel_id":   channel.ID,
		"channel_name": channel.Name,
		"success":      false,
		"error_msg":    "",
	}

	if err != nil {
		payload["error_msg"] = err.Error()
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type:    constant.EventNotifySent,
			Payload: payload,
		})
		return &NotifyResult{Success: false, Error: err.Error()}
	}

	if !result.Success {
		payload["error_msg"] = result.Error
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type:    constant.EventNotifySent,
			Payload: payload,
		})
		return &NotifyResult{Success: false, Error: result.Error}
	}

	payload["success"] = true
	eventbus.DefaultBus.Publish(eventbus.Event{
		Type:    constant.EventNotifySent,
		Payload: payload,
	})

	return &NotifyResult{Success: true}
}

// SendByChannelID 根据渠道ID发送通知
func (s *NotificationService) SendByChannelID(channelID string, msg *NotifyMessage) *NotifyResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var notifyWay models.NotifyWay
	if err := database.DB.Where("id = ?", channelID).First(&notifyWay).Error; err != nil {
		return &NotifyResult{Success: false, Error: "渠道不存在"}
	}

	if !notifyWay.Enabled {
		return &NotifyResult{Success: false, Error: "渠道已禁用"}
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(notifyWay.Config), &config); err != nil {
		return &NotifyResult{Success: false, Error: "渠道配置解析失败"}
	}

	ch := NotifyChannel{
		ID:      notifyWay.ID,
		Name:    notifyWay.Name,
		Type:    notifyWay.Type,
		Enabled: notifyWay.Enabled,
		Config:  config,
	}
	return s.SendToChannel(ch, msg)
}

// SubscribeEvents 注册通知服务自身为事件流的订阅者
func (s *NotificationService) SubscribeEvents(bus *eventbus.EventBus) {
	// 系统事件
	systemEvents := []string{constant.EventUserLogin, constant.EventBruteForceLogin, constant.EventPasswordChanged}
	for _, evt := range systemEvents {
		bus.Subscribe(evt, s.handleEvent(constant.BindingTypeSystem))
	}

	// 任务事件
	taskEvents := []string{constant.EventTaskSuccess, constant.EventTaskFailed, constant.EventTaskTimeout}
	for _, evt := range taskEvents {
		bus.Subscribe(evt, s.handleEvent(constant.BindingTypeTask))
	}

	// 通用系统通知
	bus.Subscribe(constant.EventSystemNotice, s.handleEvent(constant.BindingTypeSystem))
}

func (s *NotificationService) handleEvent(bindingType string) eventbus.Handler {
	return func(e eventbus.Event) {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return
		}

		var dataID string
		if id, ok := payload["task_id"].(string); ok {
			dataID = id
		}

		var title, text string
		switch e.Type {
		case constant.EventUserLogin:
			title = "用户登录通知"
			text = fmt.Sprintf("用户 %v 在 IP %v 登录成功", payload["username"], payload["ip"])
		case constant.EventBruteForceLogin:
			title = "系统安全警告"
			text = fmt.Sprintf("检测到 IP %v 正在尝试暴力破解用户 %v", payload["ip"], payload["username"])
		case constant.EventPasswordChanged:
			title = "账户安全通知"
			text = fmt.Sprintf("用户 %v 刚刚修改了密码", payload["username"])
		case constant.EventTaskSuccess:
			title = fmt.Sprintf("任务[%v] 成功", payload["task_name"])
			text = fmt.Sprintf("任务 #%v %v\n状态: 成功\n耗时: %vms", payload["task_id"], payload["task_name"], payload["duration"])
		case constant.EventTaskFailed:
			title = fmt.Sprintf("任务[%v] 失败", payload["task_name"])
			if errStr, ok := payload["error"]; ok {
				text = fmt.Sprintf("任务 #%v %v\n执行失败\n错误: %v", payload["task_id"], payload["task_name"], errStr)
			} else {
				text = fmt.Sprintf("任务 #%v %v\n执行失败\n状态: %v\n耗时: %vms", payload["task_id"], payload["task_name"], payload["status"], payload["duration"])
			}
		case constant.EventTaskTimeout:
			title = fmt.Sprintf("任务[%v] 超时", payload["task_name"])
			text = fmt.Sprintf("任务 #%v %v\n执行超时\n耗时: %vms", payload["task_id"], payload["task_name"], payload["duration"])
		case constant.EventSystemNotice:
			title, _ = payload["title"].(string)
			text, _ = payload["content"].(string)
		default:
			return
		}

		msg := &NotifyMessage{Title: title, Text: text}
		bindings := s.GetBindingsByEvent(bindingType, e.Type, dataID)
		if len(bindings) == 0 {
			return
		}

		channels := s.GetChannels()
		channelMap := make(map[string]NotifyChannel)
		for _, ch := range channels {
			channelMap[ch.ID] = ch
		}

		for _, binding := range bindings {
			ch, ok := channelMap[binding.WayID]
			if !ok || !ch.Enabled {
				continue
			}
			go func(channel NotifyChannel) {
				result := s.SendToChannel(channel, msg)
				if !result.Success {
					logger.Warnf("[Notify] 发送事件 %s 到渠道 %s(%s) 失败: %s", e.Type, channel.Name, channel.Type, result.Error)
				}
			}(ch)
		}
	}
}

// --- 内部方法 ---

// getChannelsInternal 从 notify_ways 表中读取所有渠道配置
func (s *NotificationService) getChannelsInternal() []NotifyChannel {
	var notifyWays []models.NotifyWay
	database.DB.Find(&notifyWays)

	channels := make([]NotifyChannel, 0, len(notifyWays))
	for _, nw := range notifyWays {
		var config map[string]string
		if err := json.Unmarshal([]byte(nw.Config), &config); err != nil {
			logger.Warnf("[Notify] 解析渠道 %s 配置失败: %v", nw.ID, err)
			continue
		}
		channels = append(channels, NotifyChannel{
			ID:        nw.ID,
			Name:      nw.Name,
			Type:      nw.Type,
			Enabled:   nw.Enabled,
			CreatedAt: nw.CreatedAt,
			Config:    config,
		})
	}
	return channels
}
