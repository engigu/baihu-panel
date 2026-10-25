package app

import (
	"encoding/json"
	"fmt"
	"strings"

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

// 应用更新通知的设置存储（settings 表 JSON 内扩展，免迁移）
const (
	appUpdateSettingsSection = "app_update"
	appUpdateStateKey        = "daily_check_state"
)

// appUpdateNoticeTitle 系统级每日检测通知的固定标题
const appUpdateNoticeTitle = "应用可更新提醒"

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

// --------------------------------------------------------------------------
// 系统级每日更新检测（P3 §10「检测提示」档，§13）：面板 SysCron 每日调度 +
// CLI baihu app check-updates --notify 手动触发共用。
// 去重状态存 settings（应用 + 已提醒版本）：同一应用同一版本只提醒一次，
// 安装/更新后不再重复打扰；无新发现时静默返回
// --------------------------------------------------------------------------

// RunDailyUpdateCheck 系统级每日更新检测：全量比对更新源，发现新可更新版本时
// 落通知中心未读提示并推送通知渠道；检测失败仅记日志（每日任务的静默降级语义）
func RunDailyUpdateCheck() {
	infos, err := DefaultAppService.CheckUpdates()
	if err != nil {
		logger.Warnf("[AppUpdate] 每日更新检测失败: %v", err)
		return
	}

	state := loadDailyCheckState()
	next := make(map[string]string, len(infos))
	var fresh []AppUpdateInfo
	for _, it := range infos {
		if !it.Updatable || it.LatestVersion == "" {
			continue
		}
		next[it.TaskID] = it.LatestVersion
		// 每应用每版本只提醒一次：上次提醒后仍未处理的应用不重复通知（列表琥珀徽标持续可见）
		if prev, ok := state[it.TaskID]; !ok || prev != it.LatestVersion {
			fresh = append(fresh, it)
		}
	}

	if len(fresh) > 0 {
		content := buildUpdateNoticeContent(fresh)
		addSystemNotice(appUpdateNoticeTitle, content, constant.LogLevelInfo, "")
		pushNoticeToChannels(appUpdateNoticeTitle, content)
		logger.Infof("[AppUpdate] 每日检测发现 %d 个应用可更新，已落通知中心并推送渠道", len(fresh))
	}

	saveDailyCheckState(next)
}

// buildUpdateNoticeContent 组装每日检测通知正文
func buildUpdateNoticeContent(items []AppUpdateInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "检测到 %d 个已装应用可更新：\n", len(items))
	for _, it := range items {
		reason := "版本更新"
		if it.Reason == "manifest_changed" {
			reason = "应用定义有更新"
		}
		fmt.Fprintf(&b, "  - %s（%s）：v%s → v%s（%s）\n",
			it.AppName, it.UpdateSource, it.InstalledVersion, it.LatestVersion, reason)
	}
	b.WriteString("\n请到「调度实体 → 应用行详情与更新」查看并一键更新。")
	return b.String()
}

// loadDailyCheckState 读取已提醒状态（map[taskID]已提醒的最新版本）
func loadDailyCheckState() map[string]string {
	state := map[string]string{}
	var setting models.Setting
	if err := database.DB.Where("section = ? AND key = ?", appUpdateSettingsSection, appUpdateStateKey).First(&setting).Error; err != nil {
		return state
	}
	_ = json.Unmarshal([]byte(string(setting.Value)), &state)
	return state
}

// saveDailyCheckState 写回已提醒状态（仅保留当前仍可更新的应用，已处理项自然清除）
func saveDailyCheckState(state map[string]string) {
	raw, _ := json.Marshal(state)
	writeSettingValue(appUpdateSettingsSection, appUpdateStateKey, string(raw))
}

// readSettingValue 直读 settings 表单值（app 包内轻量实现，语义与 SettingsService.Get 一致）
func readSettingValue(section, key string) string {
	var setting models.Setting
	if err := database.DB.Where("section = ? AND key = ?", section, key).First(&setting).Error; err != nil {
		return ""
	}
	return string(setting.Value)
}

// writeSettingValue settings 表单值 upsert
func writeSettingValue(section, key, value string) {
	var setting models.Setting
	if err := database.DB.Where("section = ? AND key = ?", section, key).First(&setting).Error; err == nil {
		database.DB.Model(&setting).Update("value", models.BigText(value))
		return
	}
	database.DB.Create(&models.Setting{
		ID:      utils.GenerateID(),
		Section: section,
		Key:     key,
		Value:   models.BigText(value),
	})
}
