package services

import (
	"sync"
	"testing"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/eventbus"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/sdk/messenger"
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

	t.Run("Verify ServerChan channel registration", func(t *testing.T) {
		ch, err := messenger.GetChannel(messenger.ChannelServerChan)
		if err != nil {
			t.Fatalf("无法获取 ServerChan 渠道: %v", err)
		}
		if ch == nil {
			t.Fatalf("ServerChan 渠道实例为空")
		}

		found := false
		for _, ct := range SupportedChannelTypes {
			if ct["type"] == messenger.ChannelServerChan {
				found = true
				if ct["label"] != "Server酱" {
					t.Errorf("ServerChan 标签不匹配: got %s, want Server酱", ct["label"])
				}
				break
			}
		}
		if !found {
			t.Errorf("SupportedChannelTypes 中未找到 ServerChan 渠道类型")
		}
	})
}

func TestExtractCustomVars(t *testing.T) {
	output := `
初始化任务开始...
[BHP_VAR] coins=100
[OUTPUT] count=42
::set-output name=branch::feature/notify
[BHP_JSON] {"summary":"签到成功","level":"VIP6"}
[System] 本次签到获得积分: 888 点
[System] 备份耗时: 12.5s
任务结束
`

	rules := []models.CustomVarRule{
		{
			Name:  "points",
			Regex: `本次签到获得积分:\s*(\d+)`,
		},
		{
			Name:  "backup_time",
			Regex: `备份耗时:\s*([0-9.]+s)`,
		},
		{
			Name:    "missing_var",
			Regex:   `未出现的匹配:\s*(\w+)`,
			Default: "默认统计",
		},
	}

	vars := ExtractCustomVars(output, rules)

	expected := map[string]string{
		"coins":       "100",
		"count":       "42",
		"branch":      "feature/notify",
		"summary":     "签到成功",
		"level":       "VIP6",
		"points":      "888",
		"backup_time": "12.5s",
		"missing_var": "默认统计",
	}

	for k, want := range expected {
		if got := vars[k]; got != want {
			t.Errorf("变量 %s 提取失败: got %q, want %q", k, got, want)
		}
	}

	// 验证模板替换效果
	svc := NewNotificationService()
	payload := map[string]interface{}{
		"task_name": "日常签到",
	}
	for k, v := range vars {
		payload[k] = v
	}

	tmpl := "任务[{{task_name}}] 结果: {{summary}}, 获得: {{points}}分, 金币: {{coins}}, 等级: {{level}}, 耗时: {{backup_time}}, 兜底: {{missing_var}}"
	rendered := svc.parseTemplate(tmpl, payload)
	expectedRendered := "任务[日常签到] 结果: 签到成功, 获得: 888分, 金币: 100, 等级: VIP6, 耗时: 12.5s, 兜底: 默认统计"
	if rendered != expectedRendered {
		t.Errorf("模板渲染不匹配:\n got  %q\n want %q", rendered, expectedRendered)
	}
}

func TestExtractCustomVars_OverwriteAndPrecedence(t *testing.T) {
	// 场景：
	// 1. 先输出 coins=10，后输出 coins=50（应该得到 50）
	// 2. 先输出 JSON {"exp": 20, "summary": "step1"}，后输出 JSON {"exp": 100, "status": "done"}（exp 变为 100，summary 和 status 均保留）
	// 3. 协议输出了 override_by_regex=proto_val，但 rules 正则命中 override_by_regex=regex_val（正则应该覆盖协议值）
	// 4. 协议输出了 keep_proto=proto_val，rules 配置了正则但未命中且有 Default="default_val"（应该保留协议值 proto_val，不被 default 冲掉）
	// 5. 协议未输出 missing_val，rules 正则未命中但有 Default="default_val"（应该得到 default_val）
	output := `
[BHP_VAR] coins=10
[BHP_VAR] coins=50
[BHP_JSON] {"exp": 20, "summary": "step1"}
[BHP_JSON] {"exp": 100, "status": "done"}
[BHP_VAR] override_by_regex=proto_val
[System] 正则匹配目标: regex_val
[BHP_VAR] keep_proto=proto_val
`

	rules := []models.CustomVarRule{
		{
			Name:  "override_by_regex",
			Regex: `正则匹配目标:\s*(\w+)`,
		},
		{
			Name:    "keep_proto",
			Regex:   `不存在的目标:\s*(\w+)`,
			Default: "default_val",
		},
		{
			Name:    "missing_val",
			Regex:   `不存在的目标:\s*(\w+)`,
			Default: "fallback_val",
		},
	}

	vars := ExtractCustomVars(output, rules)

	expected := map[string]string{
		"coins":             "50",
		"exp":               "100",
		"summary":           "step1",
		"status":            "done",
		"override_by_regex": "regex_val",
		"keep_proto":        "proto_val",
		"missing_val":       "fallback_val",
	}

	for k, want := range expected {
		if got := vars[k]; got != want {
			t.Errorf("优先级测试失败 - key: %s, got: %q, want: %q", k, got, want)
		}
	}
}


func TestOutputLooksLikeMissingEnv(t *testing.T) {
	hits := []string{
		"Traceback (most recent call last):\nKeyError: 'API_KEY'",
		"config = os.environ['JWT_SECRET']",
		"sh: 1: BARK_URL: parameter not set",
		"bash: line 1: WS_KEY: unbound variable",
		"Error: environment variable TG_TOKEN is not set",
		"缺少配置：环境变量 XX_TOKEN 不能为空",
	}
	for _, out := range hits {
		if !outputLooksLikeMissingEnv(out) {
			t.Errorf("应命中缺失变量特征但未命中: %q", out)
		}
	}

	misses := []string{
		"",
		"任务执行完成，共处理 100 条记录",
		"Connected to database successfully",
		"2026-10-24 12:00:00 INFO job done",
	}
	for _, out := range misses {
		if outputLooksLikeMissingEnv(out) {
			t.Errorf("不应命中缺失变量特征但命中: %q", out)
		}
	}
}
