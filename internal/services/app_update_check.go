package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/services/app"
)

// ==========================================================================
// 系统级每日更新检测（P3 §10「检测提示」档，§13）：
// 面板 SysCron 每日调度 + CLI baihu app check-updates --notify 手动触发共用。
// 去重状态存 settings（应用 + 已提醒版本）：同一应用同一版本只提醒一次，
// 安装/更新后不再重复打扰；无新发现时静默返回
// ==========================================================================

const (
	appUpdateSettingsSection = "app_update"
	appUpdateStateKey        = "daily_check_state"
	appUpdateNoticeTitle     = "应用可更新提醒"
)

// RunDailyAppUpdateCheck 系统级每日更新检测：全量比对更新源，发现新可更新版本时
// 落通知中心未读提示并推送通知渠道；检测失败仅记日志（每日任务的静默降级语义）
func RunDailyAppUpdateCheck() {
	RunDailyAppUpdateCheckWith(nil, nil, nil, nil)
}

// RunDailyAppUpdateCheckWith 支持自定义注入依赖项的每日检测实现，便于单测与特定上下文调用
func RunDailyAppUpdateCheckWith(appSvc *app.AppService, settingsSvc *SettingsService, notifySvc *NotificationService, logSvc *AppLogService) {
	if appSvc == nil {
		appSvc = app.DefaultAppService
	}

	infos, err := appSvc.CheckUpdates()
	if err != nil {
		logger.Warnf("[AppUpdate] 每日更新检测失败: %v", err)
		return
	}

	NotifyAppUpdateResults(infos, settingsSvc, notifySvc, logSvc)
}

// NotifyAppUpdateResults 根据给定的更新检测结果执行去重、通知中心落库与通知渠道直推，返回本次实际发送提醒的应用数
func NotifyAppUpdateResults(infos []app.AppUpdateInfo, settingsSvc *SettingsService, notifySvc *NotificationService, logSvc *AppLogService) int {
	if settingsSvc == nil {
		settingsSvc = NewSettingsService()
	}
	if notifySvc == nil {
		notifySvc = NewNotificationService()
	}
	if logSvc == nil {
		logSvc = NewAppLogService()
	}

	state := loadDailyCheckState(settingsSvc)
	next := make(map[string]string, len(infos))
	var fresh []app.AppUpdateInfo
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
		logSvc.AddSystemNoticeUpsert(appUpdateNoticeTitle, content, constant.LogLevelInfo, "")
		notifySvc.SendSystemNotice(appUpdateNoticeTitle, content)
		logger.Infof("[AppUpdate] 每日检测发现 %d 个应用可更新，已落通知中心并推送渠道", len(fresh))
	}

	saveDailyCheckState(settingsSvc, next)
	return len(fresh)
}

// formatAppVersion 规范化应用版本号展示（兼容带或不带 v 前缀，统一输出形如 v1.0.0）
func formatAppVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	clean := strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	return "v" + clean
}

// buildUpdateNoticeContent 组装每日检测通知正文
func buildUpdateNoticeContent(items []app.AppUpdateInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "检测到 %d 个已装应用可更新：\n", len(items))
	for _, it := range items {
		reason := "版本更新"
		if it.Reason == "manifest_changed" {
			reason = "应用定义有更新"
		}
		fmt.Fprintf(&b, "  - %s（%s）：%s → %s（%s）\n",
			it.AppName, it.UpdateSource, formatAppVersion(it.InstalledVersion), formatAppVersion(it.LatestVersion), reason)
	}
	b.WriteString("\n请到「调度实体 → 应用行详情与更新」查看并一键更新。")
	return b.String()
}

// loadDailyCheckState 读取已提醒状态（map[taskID]已提醒的最新版本）
func loadDailyCheckState(settingsSvc *SettingsService) map[string]string {
	state := map[string]string{}
	if settingsSvc == nil {
		return state
	}
	val := settingsSvc.Get(appUpdateSettingsSection, appUpdateStateKey)
	if val == "" {
		return state
	}
	_ = json.Unmarshal([]byte(val), &state)
	return state
}

// saveDailyCheckState 写回已提醒状态（仅保留当前仍可更新的应用，已处理项自然清除）
func saveDailyCheckState(settingsSvc *SettingsService, state map[string]string) {
	if settingsSvc == nil {
		return
	}
	raw, _ := json.Marshal(state)
	_ = settingsSvc.Set(appUpdateSettingsSection, appUpdateStateKey, string(raw))
}
