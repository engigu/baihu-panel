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
	"gorm.io/gorm"
	"regexp"
	"strings"
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
	{"type": messenger.ChannelVoceChat, "label": "VoceChat"},
	{"type": messenger.ChannelWxPusher, "label": "WxPusher"},
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
			Enabled: utils.BoolPtr(channel.Enabled),
		}
		return database.DB.Create(notifyWay).Error
	}

	// 更新
	updates := map[string]interface{}{
		"name":    channel.Name,
		"type":    channel.Type,
		"config":  models.BigText(configJSON),
		"enabled": &channel.Enabled,
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
	if err := database.DB.Where("id = ?", id).Delete(&models.NotifyWay{}).Error; err != nil {
		return err
	}

	// 同时清理事件绑定中引用此渠道的配置
	if err := database.DB.Where("way_id = ?", id).Delete(&models.NotifyBinding{}).Error; err != nil {
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
		res := database.DB.Where("type = ? AND event = ? AND way_id = ? AND data_id = ?",
			binding.Type, binding.Event, binding.WayID, binding.DataID).Limit(1).Find(&existing)
		if res.Error == nil && res.RowsAffected > 0 {
			// 如果已存在且未删除，更新现有记录（特别是 Extra 字段）
			existing.Extra = binding.Extra
			err := database.DB.Save(&existing).Error
			if err == nil {
				*binding = existing
			}
			return err
		}

		binding.ID = utils.GenerateID()
		return database.DB.Create(binding).Error
	}
	return database.DB.Save(binding).Error
}

// BatchSaveBindings 批量保存事件绑定
func (s *NotificationService) BatchSaveBindings(bindingType, dataID string, bindings []models.NotifyBinding) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// 如果指定了 dataID，先清理该对象的所有现有绑定
		if dataID != "" {
			if err := tx.Where("type = ? AND data_id = ?", bindingType, dataID).Delete(&models.NotifyBinding{}).Error; err != nil {
				return err
			}
		}

		// 批量插入新绑定
		for i := range bindings {
			bindings[i].ID = utils.GenerateID()
			bindings[i].Type = bindingType
			bindings[i].DataID = dataID
			if err := tx.Create(&bindings[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBinding 删除事件绑定
func (s *NotificationService) DeleteBinding(id string) error {
	return database.DB.Where("id = ?", id).Delete(&models.NotifyBinding{}).Error
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
	res := database.DB.Where("id = ?", channelID).Limit(1).Find(&notifyWay)
	if res.Error != nil || res.RowsAffected == 0 {
		return &NotifyResult{Success: false, Error: "渠道不存在"}
	}

	if !utils.DerefBool(notifyWay.Enabled, true) {
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
		Enabled: utils.DerefBool(notifyWay.Enabled, true),
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

var ansiRegexp = regexp.MustCompile(`[\x1b\x9b][\[()#;?]*([0-9]{1,4}(;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]`)

// stripAnsi 移除字符串中的 ANSI 转义码（如颜色代码）
func stripAnsi(str string) string {
	return ansiRegexp.ReplaceAllString(str, "")
}

// parseTemplate 简单的 {{key}} 模板替换
func (s *NotificationService) parseTemplate(tmpl string, payload map[string]interface{}) string {
	result := tmpl
	for k, v := range payload {
		placeholder := fmt.Sprintf("{{%s}}", k)
		valStr := fmt.Sprintf("%v", v)
		result = strings.ReplaceAll(result, placeholder, valStr)
	}
	return result
}

// getDefaultMessage 兜底默认消息内容
func (s *NotificationService) getDefaultMessage(eventType string, payload map[string]interface{}) (string, string) {
	var title, text string
	switch eventType {
	case constant.EventUserLogin:
		status, _ := payload["status"].(string)
		if status == "success" {
			title = "用户登录成功"
			text = fmt.Sprintf("用户 %v 在 IP %v 登录成功", payload["username"], payload["ip"])
		} else {
			title = "用户登录失败"
			reason, _ := payload["message"].(string)
			text = fmt.Sprintf("用户 %v 在 IP %v 登录失败\n原因: %v", payload["username"], payload["ip"], reason)
		}
	case constant.EventBruteForceLogin:
		title = "系统安全警告"
		text = fmt.Sprintf("检测到 IP %v 正在尝试暴力破解用户 %v", payload["ip"], payload["username"])
	case constant.EventPasswordChanged:
		title = "账户安全通知"
		text = fmt.Sprintf("用户 %v 刚刚修改了密码", payload["username"])
	case constant.EventTaskSuccess:
		title = fmt.Sprintf("任务[%v] 成功", payload["task_name"])
		text = fmt.Sprintf("任务 #%v %v\n状态: 成功\n执行时间: %v\n耗时: %vms", payload["task_id"], payload["task_name"], payload["start_time"], payload["duration"])
	case constant.EventTaskFailed:
		title = fmt.Sprintf("任务[%v] 失败", payload["task_name"])
		if errStr, ok := payload["error"]; ok {
			text = fmt.Sprintf("任务 #%v %v\n执行失败\n执行时间: %v\n错误: %v", payload["task_id"], payload["task_name"], payload["start_time"], errStr)
		} else {
			text = fmt.Sprintf("任务 #%v %v\n执行失败\n状态: %v\n执行时间: %v\n耗时: %vms", payload["task_id"], payload["task_name"], payload["status"], payload["start_time"], payload["duration"])
		}
	case constant.EventTaskTimeout:
		title = fmt.Sprintf("任务[%v] 超时", payload["task_name"])
		text = fmt.Sprintf("任务 #%v %v\n执行超时\n执行时间: %v\n耗时: %vms", payload["task_id"], payload["task_name"], payload["start_time"], payload["duration"])
	}
	return title, text
}

// handleEvent 处理事件订阅并发送通知
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

		// 获取全局前缀和模板配置
		prefix := s.settingsService.Get(constant.SectionNotify, constant.KeyNotifyPrefix)
		var tmplTitleKey, tmplTextKey string

		switch e.Type {
		case constant.EventUserLogin:
			tmplTitleKey = constant.KeyNotifyTemplateUserLoginTitle
			tmplTextKey = constant.KeyNotifyTemplateUserLoginText
			// 特殊处理登录状态
			status, _ := payload["status"].(string)
			if status == "success" {
				payload["status_label"] = "成功"
			} else {
				payload["status_label"] = "失败"
			}

		case constant.EventBruteForceLogin:
			tmplTitleKey = constant.KeyNotifyTemplateBruteForceLoginTitle
			tmplTextKey = constant.KeyNotifyTemplateBruteForceLoginText

		case constant.EventPasswordChanged:
			tmplTitleKey = constant.KeyNotifyTemplatePasswordChangedTitle
			tmplTextKey = constant.KeyNotifyTemplatePasswordChangedText

		case constant.EventTaskSuccess, constant.EventTaskFailed, constant.EventTaskTimeout:
			switch e.Type {
			case constant.EventTaskSuccess:
				tmplTitleKey = constant.KeyNotifyTemplateTaskSuccessTitle
				tmplTextKey = constant.KeyNotifyTemplateTaskSuccessText
			case constant.EventTaskFailed:
				tmplTitleKey = constant.KeyNotifyTemplateTaskFailedTitle
				tmplTextKey = constant.KeyNotifyTemplateTaskFailedText
			case constant.EventTaskTimeout:
				tmplTitleKey = constant.KeyNotifyTemplateTaskTimeoutTitle
				tmplTextKey = constant.KeyNotifyTemplateTaskTimeoutText
			}

			// 处理输出内容，避免过长
			if output, ok := payload["output"].(string); ok {
				// 如果输出包含了压缩后的 Base64 (以 "base64:" 开头)，由于是推送到通知，我们尽量不发大段 Base64
				// 这里简单处理：如果过长则截断，或者如果是压缩的则记录一下
				trimmed := utils.TrimLastRunes(output, 1000)
				if len(trimmed) < len(output) {
					payload["output"] = trimmed + "\n...(截断)"
				}
			}

		case constant.EventSystemNotice:
			title, _ = payload["title"].(string)
			text, _ = payload["content"].(string)
		default:
			return
		}

		if tmplTitleKey != "" {
			tmplTitle := s.settingsService.Get(constant.SectionNotify, tmplTitleKey)
			tmplText := s.settingsService.Get(constant.SectionNotify, tmplTextKey)

			if tmplTitle != "" {
				title = s.parseTemplate(tmplTitle, payload)
			}
			if tmplText != "" {
				text = s.parseTemplate(tmplText, payload)
			}

			// 如果模板为空，使用兜底默认逻辑（保持向上兼容）
			if title == "" || text == "" {
				title, text = s.getDefaultMessage(e.Type, payload)
			}
		}

		// 添加全局前缀
		if prefix != "" {
			title = fmt.Sprintf("%s %s", prefix, title)
		}

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

			// 克隆文本以便修改
			currentText := text

			// 解析额外配置
			var extra models.BindingExtra
			if binding.Extra != "" {
				_ = json.Unmarshal([]byte(binding.Extra), &extra)
			}
			// 默认日志限制为 1000
			if extra.LogLimit <= 0 {
				extra.LogLimit = 1000
			}

			// 如果开启了日志推送
			if extra.EnableLog {
				if output, ok := payload["output"].(string); ok && output != "" {
					// 仅保留指定字数的日志内容并移除 ANSI 颜色代码
					logSnippet := stripAnsi(output)
					
					trimmed := utils.TrimLastRunes(logSnippet, extra.LogLimit)
					if len(trimmed) < len(logSnippet) {
						logSnippet = "...\n" + trimmed
					} else {
						logSnippet = trimmed
					}
					
					currentText += "\n\n[执行日志]\n" + logSnippet
				}
			}

			go func(channel NotifyChannel, msgTitle, msgText string) {
				result := s.SendToChannel(channel, &NotifyMessage{Title: msgTitle, Text: msgText})
				if !result.Success {
					logger.Warnf("[Notify] 发送事件 %s 到渠道 %s(%s) 失败: %s", e.Type, channel.Name, channel.Type, result.Error)
				}
			}(ch, title, currentText)
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
			Enabled:   utils.DerefBool(nw.Enabled, true),
			CreatedAt: nw.CreatedAt,
			Config:    config,
		})
	}
	return channels
}
