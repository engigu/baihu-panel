package app

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
)

// ==========================================================================
// 调和漂移闸门 (design/app-update-proposal.md §6，P2)
//
// 时机：应用主任务定时调和触发后、引擎代码同步前（先检测后同步）。
//   - 市场不可达 / 应用已下架 / URL 源失效 → 静默放行（挂起中维持拦截），不产生标签不报错
//   - 清单无变化 → 放行调和；若此前闸门挂起（上游已回退变更），自动解除
//   - 清单有变化且新必填变量缺值 → 拦截本次调和：代码停旧版本，旧契约继续生效，
//     子任务零运行时失败；落 pending_gate 标记（列表红色标签）+ 应用日志未读提示
//
// 与 409 安全阀同构（新必填变量缺值必拦截），检测为纯读操作，每应用每日节流一次。
// 挂起提示经通知中心（应用日志未读红点）与通知渠道双通道呈现（P3 §13，见 notify.go）。
// auto_update 开启的应用闸门自动放行，破坏性变更由 --refresh-manifest 的安全阀兜底（§10）：
// 拦截降级后调和冻结代码同步，保证「代码停旧版本」承诺不被回退调和绕开（AppAutoRefreshSkipError.FreezeSync）。
// ==========================================================================

// GateDecision 闸门决策结果
type GateDecision struct {
	Blocked     bool             // true = 拦截本次调和
	Reason      string           // missing_envs
	VersionFrom string           // 已装版本
	VersionTo   string           // 更新源最新版本
	MissingEnvs []MissingEnvItem // 拦截时缺失的必填环境变量
	Message     string           // 拦截时输出到调和日志与通知中心的说明
}

// gateCheckInterval 闸门检测节流间隔：每应用每日一次
const gateCheckInterval = 24 * time.Hour

var gateThrottle struct {
	sync.Mutex
	lastCheck map[string]time.Time
}

// gateCheckThrottled 节流窗口内返回 true（并顺带记录本次检测时间）
func gateCheckThrottled(appID string) bool {
	gateThrottle.Lock()
	defer gateThrottle.Unlock()
	if gateThrottle.lastCheck == nil {
		gateThrottle.lastCheck = make(map[string]time.Time)
	}
	if now := time.Now(); now.Sub(gateThrottle.lastCheck[appID]) < gateCheckInterval {
		return true
	}
	gateThrottle.lastCheck[appID] = time.Now()
	return false
}

// rearmGateCheck 解除检测节流：更新/回滚成功后调用，下一个调和周期立即重新评估上游漂移
func rearmGateCheck(manifestID string) {
	gateThrottle.Lock()
	defer gateThrottle.Unlock()
	if gateThrottle.lastCheck != nil {
		delete(gateThrottle.lastCheck, manifestID)
	}
}

// gateNoticeTitle 闸门通知在应用日志中的固定标题（upsert 去重键之一）
const gateNoticeTitle = "应用更新待补值"

// EvaluateReconcileGate 调和漂移闸门检测（仅应用主任务定时触发路径调用）。
// 返回 nil 表示未挂起且本次无需评估或静默放行：节流窗口内 / 无自动更新源 / 更新源不可达；
// 挂起中（PendingGate 非空）永不返回 nil —— 解除仅经每日检测确认上游回退或补值/更新成功（design §6.2）
func (s *AppService) EvaluateReconcileGate(task *models.Task, appCfg *models.AppTaskConfig) *GateDecision {
	if task == nil || appCfg == nil || appCfg.ID == "" {
		return nil
	}

	// 0.1 自动更新开启的应用闸门自动放行（design §6.2/§10）：破坏性变更由 --refresh-manifest
	//     路径的 409 安全阀兜底（拦截时降级为通知并回退调和，挂起期间冻结代码同步），闸门不再重复拦截
	if appCfg.AutoUpdate {
		return nil
	}

	// 0.2 挂起中的闸门拦截本次调和（保证「代码停旧版本」），但挂起 ≠ 停止评估：
	//    节流窗口内的调和直接拦截（不消耗每日节流）；窗口外的调和照常执行每日检测，
	//    用于发现漂移与确认上游是否已回退变更（此前提前 return 会让解除路径不可达）
	pendingReason := appCfg.PendingGate

	// 1. 每应用每日节流；窗口内无需评估：挂起中拦截，未挂起静默放行
	if gateCheckThrottled(appCfg.ID) {
		if pendingReason != "" {
			return pendingGateDecision(appCfg, pendingReason)
		}
		return nil
	}

	// 2. 拉取最新清单：市场源走缓存降级；URL 源失败复用旧缓存；
	//    均不可用时未挂起静默放行，挂起中维持拦截（拿不到可靠清单不解除、不放行）
	newManifest := s.fetchGateLatestManifest(appCfg)
	if newManifest == nil {
		if pendingReason != "" {
			return pendingGateDecision(appCfg, pendingReason)
		}
		return nil
	}

	// 3. 清单无变化 → 放行；若此前闸门挂起（上游已回退变更），自动解除挂起标记
	if !versionChanged(appCfg.Version, newManifest.Version) {
		if old, pErr := ParseManifestFromYAML([]byte(appCfg.ManifestRaw)); pErr == nil &&
			manifestSignature(old) == manifestSignature(newManifest) {
			s.clearPendingGate(task.ID)
			return &GateDecision{VersionFrom: appCfg.Version, VersionTo: newManifest.Version}
		}
	}

	// 4. 破坏性变更判定：新必填变量缺值 → 拦截（永不静默放行破坏性变更）；
	//    挂起中重复拦截不会重复落库（applyGateBlock 对同原因 no-op，仅刷新提示内容）
	if err := s.checkRequiredEnvs(newManifest, appCfg, nil); err != nil {
		var conflict *AppUpdateConflictError
		if errors.As(err, &conflict) {
			decision := &GateDecision{
				Blocked:     true,
				Reason:      "missing_envs",
				VersionFrom: appCfg.Version,
				VersionTo:   newManifest.Version,
				MissingEnvs: conflict.MissingEnvs,
			}
			decision.Message = buildGateBlockMessage(appCfg, decision)
			// 面板进程常驻：异步推送渠道，慢渠道不拖长 per-app 锁窗口（调和持有 TryLockAppRun 期间走到这里）
			if s.applyGateBlock(task, decision) {
				go pushNoticeToChannels(gateNoticeTitle, decision.Message)
			}
			return decision
		}
		return nil
	}

	// 5. 有变化但无破坏性 → 放行调和；若此前挂起（上游已将变量改为可选/移除，或用户已补值），一并解除。
	//    琥珀色「可更新」徽标由 P1 检测接口在列表呈现
	if pendingReason != "" {
		s.clearPendingGate(task.ID)
	}
	return &GateDecision{VersionFrom: appCfg.Version, VersionTo: newManifest.Version}
}

// pendingGateDecision 挂起中且本轮无需评估（节流窗口内/更新源不可达）时返回的拦截决策，
// 维持既有挂起原因与提示文案
func pendingGateDecision(appCfg *models.AppTaskConfig, reason string) *GateDecision {
	return &GateDecision{
		Blocked:     true,
		Reason:      reason,
		VersionFrom: appCfg.Version,
		Message:     fmt.Sprintf("应用 [%s] 此前检测到上游清单新增必填环境变量且无可用值（见通知中心「%s」），闸门挂起期间暂停跟随上游更新，应用停留旧版本；请到「详情与更新」补值或一键更新后自动解除", appCfg.Name, gateNoticeTitle),
	}
}

// fetchGateLatestManifest 闸门检测的最新清单获取；拿不到可靠清单时返回 nil（静默跳过，不误报）。
// 源判定与 CheckUpdates 保持一致：清单 ID 能在市场索引匹配 → 市场源；否则 URL 源
func (s *AppService) fetchGateLatestManifest(appCfg *models.AppTaskConfig) *AppManifest {
	// 市场源：可达且收录该应用时使用市场清单；不可达或未收录时继续尝试 URL（URL 源应用不依赖市场可用性）
	if appCfg.ID != "" {
		if market, err := s.FetchMarketplaceCached(false); err == nil && market != nil {
			for _, am := range market.Apps {
				if id, _ := am["id"].(string); strings.EqualFold(strings.TrimSpace(id), appCfg.ID) {
					if raw, _ := am["manifest_raw"].(string); strings.TrimSpace(raw) != "" {
						if m, pErr := ParseManifestFromYAML([]byte(raw)); pErr == nil {
							return m
						}
					}
					return nil
				}
			}
		}
	}

	if isHTTPURL(appCfg.ManifestPath) {
		m, _, err := ParseManifestFromURL(appCfg.ManifestPath, "")
		if err != nil {
			if cm, ok := getCachedURLManifest(appCfg.ManifestPath); ok {
				return cm
			}
			return nil
		}
		return m
	}

	// manual（手动导入 / 市场已下架且无 URL）无自动更新源，闸门不适用
	return nil
}

// applyGateBlock 落闸门挂起标记 + 应用日志未读提示（同应用同标题未读提示 upsert 去重）。
// 返回是否新建了未读提示（false = 命中同题未读提示仅刷新内容），调用方据此决定是否推送通知渠道：
// 面板闸门路径异步推送；CLI 安全阀降级路径同步推送（子进程随即退出，异步发送会被截断）
func (s *AppService) applyGateBlock(task *models.Task, decision *GateDecision) bool {
	// 1. pending_gate 标记落库：列表红色「待补值」标签的展示依据，补值/更新成功后自动清除
	var fresh models.Task
	if err := database.DB.Where("id = ?", task.ID).First(&fresh).Error; err == nil {
		if cfg := fresh.GetAppConfig(); cfg != nil && cfg.PendingGate != decision.Reason {
			cfg.PendingGate = decision.Reason
			unified := fresh.GetUnifiedConfig()
			unified.App = cfg
			database.DB.Model(&fresh).Update("unified_config", unified.ToJSON())
		}
	}

	// 2. 应用日志未读提示：已存在同应用未读提示则仅刷新内容，避免闸门挂起期间每日重复堆叠
	//    （addSystemNotice 内对齐 AppLogService.Add 的落库语义：应用层生成主键 + 事件发布）
	return addSystemNotice(gateNoticeTitle, decision.Message, constant.LogLevelWarning, task.ID)
}

// clearPendingGate 清除闸门挂起标记：调和确认上游变更已回退/不再破坏、补值或更新/回滚成功后调用。
// 同时把未读的闸门提示置为已读，兑现提示中「补值或更新成功后本提示自动解除」的承诺
func (s *AppService) clearPendingGate(taskID string) {
	var task models.Task
	if err := database.DB.Where("id = ? AND type = ?", taskID, constant.TaskTypeApp).First(&task).Error; err != nil {
		return
	}
	cfg := task.GetAppConfig()
	if cfg == nil || cfg.PendingGate == "" {
		return
	}
	cfg.PendingGate = ""
	unified := task.GetUnifiedConfig()
	unified.App = cfg
	database.DB.Model(&task).Update("unified_config", unified.ToJSON())

	database.DB.Model(&models.AppLog{}).
		Where("category = ? AND ref_id = ? AND title = ? AND status = ?",
			constant.LogCategorySystemNotice, taskID, gateNoticeTitle, constant.LogStatusUnread).
		Update("status", constant.LogStatusRead)
}

// buildGateBlockMessage 组装闸门拦截说明（调和日志与通知中心共用）
func buildGateBlockMessage(appCfg *models.AppTaskConfig, d *GateDecision) string {
	keys := make([]string, 0, len(d.MissingEnvs))
	for _, m := range d.MissingEnvs {
		if m.Label != "" {
			keys = append(keys, fmt.Sprintf("%s（%s）", m.Key, m.Label))
		} else {
			keys = append(keys, m.Key)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "应用 [%s] 检测到上游清单更新（v%s → v%s）新增必填环境变量且当前无可用值：\n  - %s\n",
		appCfg.Name, d.VersionFrom, d.VersionTo, strings.Join(keys, "\n  - "))
	b.WriteString("\n为避免更新后子任务静默失败，已暂停跟随上游更新：本次调和跳过代码同步，应用停留旧版本，旧契约继续生效。\n")
	b.WriteString("请到「调度实体 → 详情与更新」补值后一键更新；补值或更新成功后本提示自动解除。")
	return b.String()
}
