package router

import (
	// "fmt"

	// "github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	// "github.com/engigu/baihu-panel/internal/logger"
	// "github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/executor"
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
