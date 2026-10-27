package app

import (
	"encoding/json"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/sdk/messenger"
	"github.com/engigu/baihu-panel/internal/utils"
)

// ==========================================================================
// 应用更新通知（design/app-update-proposal.md §13，P3）
//
// 通知中心落库 + 通知渠道推送的直发实现：不依赖事件订阅链路，面板进程与 CLI 子进程
// （调和路径 baihu app apply 的隔离进程）均可安全调用——CLI 子进程不装配事件订阅器，
// 经 eventbus 发布 EventSystemNotice 在子进程内无消费者，会静默丢失通知。
// 渠道语义与通知服务 handleEvent 对齐（系统通知绑定 + 全局前缀），不参与内容过滤规则
// （应用更新为系统级提示，非用户脚本输出触发的可过滤事件）
// ==========================================================================

// addSystemNotice 通知中心落一条未读系统提示，返回是否为新建（false = 命中未读同题提示仅刷新内容）。
// 同 refID（应用）同标题的未读提示 upsert 去重；落库语义与 AppLogService.Add 对齐：
// 应用层生成主键（AppLog 裸 Create 空主键首次可插入、此后主键冲突静默失败）并发布 EventAppLogAdded
func addSystemNotice(title, content, level, refID string) bool {
	if level == "" {
		level = constant.LogLevelInfo
	}
	query := database.DB.Where("category = ? AND title = ? AND status = ?",
		constant.LogCategorySystemNotice, title, constant.LogStatusUnread)
	if refID != "" {
		query = query.Where("ref_id = ?", refID)
	}
	var existing models.AppLog
	if err := query.Order("created_at desc").First(&existing).Error; err == nil {
		database.DB.Model(&existing).Update("content", models.BigText(content))
		return false
	}
	notice := &models.AppLog{
		ID:       utils.GenerateID(),
		Category: constant.LogCategorySystemNotice,
		Title:    title,
		Content:  models.BigText(content),
		Level:    level,
		Status:   constant.LogStatusUnread,
		RefID:    refID,
	}
	if err := database.DB.Create(notice).Error; err == nil {
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type:    constant.EventAppLogAdded,
			Payload: notice,
		})
		return true
	}
	return false
}

// pushNoticeToChannels 将应用更新通知按「系统通知」事件绑定直推各启用渠道（Telegram/飞书/钉钉等）。
// 无绑定或渠道全禁用时静默返回；发送结果发布 EventNotifySent 供推送日志留痕
func pushNoticeToChannels(title, content string) {
	var bindings []models.NotifyBinding
	database.DB.Where("type = ? AND event = ?", constant.BindingTypeSystem, constant.EventSystemNotice).Find(&bindings)
	if len(bindings) == 0 {
		return
	}

	// 渠道配置与启用状态
	var ways []models.NotifyWay
	database.DB.Find(&ways)
	wayMap := make(map[string]models.NotifyWay, len(ways))
	for _, w := range ways {
		wayMap[w.ID] = w
	}

	// 全局通知前缀（settings 表直读，避免引入 services 包循环依赖）
	prefix := readSettingValue(constant.SectionNotify, constant.KeyNotifyPrefix)

	sent := false
	for _, b := range bindings {
		way, ok := wayMap[b.WayID]
		if !ok || !utils.DerefBool(way.Enabled, true) {
			continue
		}
		var config map[string]string
		if err := json.Unmarshal([]byte(way.Config), &config); err != nil {
			logger.Warnf("[AppUpdate] 解析通知渠道 %s 配置失败: %v", way.ID, err)
			continue
		}

		msg := &messenger.Message{Title: prefix + title, Text: content}
		result, err := messenger.Send(way.Type, messenger.ChannelConfig(config), msg)

		payload := map[string]interface{}{
			"title":        title,
			"content":      content,
			"channel_id":   way.ID,
			"channel_name": way.Name,
			"success":      err == nil && result != nil && result.Success,
			"error_msg":    "",
		}
		if err != nil {
			payload["error_msg"] = err.Error()
		} else if result != nil && !result.Success {
			payload["error_msg"] = result.Error
		}
		eventbus.DefaultBus.Publish(eventbus.Event{
			Type:    constant.EventNotifySent,
			Payload: payload,
		})
		sent = true
	}
	if sent {
		logger.Infof("[AppUpdate] 已推送通知「%s」到绑定渠道", title)
	}
}

// readSettingValue 直读 settings 表单值（app 包内轻量实现，语义与 SettingsService.Get 一致）
func readSettingValue(section, key string) string {
	var setting models.Setting
	if err := database.DB.Where("section = ? AND key = ?", section, key).First(&setting).Error; err != nil {
		return ""
	}
	return string(setting.Value)
}

