package router

import (
	// "fmt"

	// "github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/executor"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/services"
)

func setupEventHandlers(subscribers ...eventbus.Subscriber) {
	bus := eventbus.DefaultBus

	// 遍历并统一初始化所有订阅者的事件链路
	for _, s := range subscribers {
		s.SubscribeEvents(bus)
	}
}

func startAppLogCleanup(appLogSvc *services.AppLogService) {
	// 注册到内部系统定时器（并立即执行第一次）
	executor.GetSysCron().AddJobWithRun("@every 1h", func() {
		appLogSvc.CleanUp()
	})
}

// startAppUpdateDailyCheck 系统级每日更新检测（P3，design/app-update-proposal.md §10/§13）：
// 每日全量比对更新源，发现新可更新版本时落通知中心未读提示并推送通知渠道。
// 状态去重（应用+版本）见 services.RunDailyAppUpdateCheck；CLI baihu app check-updates --notify 共用同一编排
func startAppUpdateDailyCheck() {
	if _, err := executor.GetSysCron().AddJob("@every 24h", func() {
		services.RunDailyAppUpdateCheck()
	}); err != nil {
		logger.Errorf("[System] 注册应用更新每日检测任务失败: %v", err)
	} else {
		logger.Infof("[System] 已注册应用更新每日检测任务（发现新版本时通知）")
	}
}
