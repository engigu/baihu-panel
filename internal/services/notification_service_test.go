package services

import (
	"sync"
	"testing"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/systime"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
	time.Local = systime.CST
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法开启 SQLite 内存测试库: %v", err)
	}

	database.DB = db

	err = db.AutoMigrate(
		&models.Setting{},
		&models.NotifyWay{},
		&models.NotifyBinding{},
		&models.NotifyFilter{},
		&models.AppLog{},
	)
	if err != nil {
		t.Fatalf("测试数据迁移失败: %v", err)
	}
}

func TestNotificationFilterRules(t *testing.T) {
	setupTestDB(t)

	db := database.DB
	
	enabledVal := true
	db.Create(&models.NotifyWay{
		ID:      "ch_test",
		Name:    "测试渠道",
		Type:    "dingtalk",
		Config:  `{"webhook":"http://localhost"}`,
		Enabled: &enabledVal,
	})

	db.Create(&models.NotifyBinding{
		ID:     "b_test",
		Type:   "task",
		Event:  "task_failed",
		WayID:  "ch_test",
		DataID: "t_verify",
	})

	// 注入静默规则
	db.Create(&models.NotifyFilter{
		ID:         "f_silent",
		Name:       "静默规则",
		Event:      "task_failed",
		Keyword:    "BAD_ERROR_SILENT",
		MatchField: "content",
		Action:     "silent",
		Enabled:    true,
	})

	// 注入替换规则
	db.Create(&models.NotifyFilter{
		ID:          "f_custom",
		Name:        "替换规则",
		Event:       "task_failed",
		Keyword:     "BAD_ERROR_CUSTOM",
		MatchField:  "content",
		Action:      "custom",
		CustomTitle: "【已拦截】{{task_name}} 运行失败",
		CustomText:  "测试警告: 原始日志中出现了敏感错误，输出摘要: {{output}}",
		Enabled:     true,
	})

	svc := NewNotificationService()

	// 订阅 Eventbus 日志发布以断言
	var mu sync.Mutex
	var loggedEvents []eventbus.Event
	eventbus.DefaultBus.Subscribe(constant.EventSchedulerLog, func(event eventbus.Event) {
		mu.Lock()
		loggedEvents = append(loggedEvents, event)
		mu.Unlock()
	})

	// 1. 验证静默拦截 (silent) 触发后不会投递，且能触发 filter 过滤日志生成
	t.Run("Verify Silent Notification Rule", func(t *testing.T) {
		mu.Lock()
		loggedEvents = nil
		mu.Unlock()
		
		eventPayload := eventbus.Event{
			Type: constant.EventTaskFailed,
			Payload: map[string]interface{}{
				"task_id":   "t_verify",
				"task_name": "测试任务",
				"output":    "出错了，日志中带有 BAD_ERROR_SILENT 敏感关键字",
				"error":     "exit status 1",
			},
		}

		handler := svc.handleEvent("task")
		handler(eventPayload)

		// 由于 Eventbus Publish 启动 goroutine 异步发送，这里加 50ms 等待
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(loggedEvents) == 0 {
			t.Errorf("静默规则命中时，应该通过 eventbus 抛出 filter 记录日志事件，但当前并未捕获")
		} else {
			payload := loggedEvents[0].Payload.(map[string]interface{})
			if payload["action"] != "silent" || payload["type"] != "filter" {
				t.Errorf("捕获的拦截事件属性不正确，期望 action: silent, 实际: %v", payload["action"])
			}
		}
	})

	// 2. 验证内容替换 (custom) 触发后改写模板并捕获日志发布
	t.Run("Verify Custom Content Replacement Rule", func(t *testing.T) {
		mu.Lock()
		loggedEvents = nil
		mu.Unlock()

		eventPayload := eventbus.Event{
			Type: constant.EventTaskFailed,
			Payload: map[string]interface{}{
				"task_id":   "t_verify",
				"task_name": "测试任务",
				"output":    "出错了，日志中带有 BAD_ERROR_CUSTOM 替换关键字",
				"error":     "exit status 1",
			},
		}

		handler := svc.handleEvent("task")
		handler(eventPayload)

		// 同样加 50ms 异步等待
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(loggedEvents) == 0 {
			t.Errorf("替换规则命中时，没有发出对应的过滤落库日志事件")
		} else {
			payload := loggedEvents[0].Payload.(map[string]interface{})
			if payload["action"] != "custom" || payload["type"] != "filter" {
				t.Errorf("捕获的替换事件属性不正确，期望 action: custom, 实际: %v", payload["action"])
			}
			expectedTitle := "【已拦截】测试任务 运行失败"
			if payload["title"] != expectedTitle {
				t.Errorf("自定义标题解析模板错误，期望: %s, 实际: %s", expectedTitle, payload["title"])
			}
		}
	})
}
