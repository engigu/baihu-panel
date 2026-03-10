package router

import (
	// "fmt"
	"time"

	// "github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/eventbus"
	// "github.com/engigu/baihu-panel/internal/logger"
	// "github.com/engigu/baihu-panel/internal/models"
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
	// 初始化时执行一次清理
	appLogSvc.CleanUp()

	// 每天凌晨或者定期清理
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		appLogSvc.CleanUp()
	}
}
