package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/app"
	"github.com/engigu/baihu-panel/internal/services/tasks"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type AppController struct {
	appService      *app.AppService
	executorService *tasks.ExecutorService
}

func NewAppController(appService *app.AppService, executorService ...*tasks.ExecutorService) *AppController {
	if appService == nil {
		appService = app.DefaultAppService
	}
	var es *tasks.ExecutorService
	if len(executorService) > 0 {
		es = executorService[0]
	}
	return &AppController{
		appService:      appService,
		executorService: es,
	}
}

// syncAppCronTasks 同步指定主应用及其所有受控子任务在 CronManager 中的调度状态
func (ac *AppController) syncAppCronTasks(masterTaskID string) {
	if ac.executorService != nil {
		ac.executorService.SyncAppTasks(masterTaskID)
	}
}

// GetApps 获取已安装应用列表
func (ac *AppController) GetApps(c *gin.Context) {
	apps, err := ac.appService.ListApps()
	if err != nil {
		utils.BadRequest(c, "获取应用列表失败: "+err.Error())
		return
	}

	utils.Success(c, apps)
}

// GetApp 获取指定应用详情（包含 Manifest、受控任务、环境配置）
func (ac *AppController) GetApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "应用 ID 不能为空")
		return
	}

	appEntity, err := ac.appService.GetApp(id)
	if err != nil {
		utils.NotFound(c, "未找到该应用")
		return
	}

	// 解析 Manifest 结构
	var manifest *app.AppManifest
	if appEntity.ManifestRaw != "" {
		manifest, _ = app.ParseManifestFromYAML([]byte(appEntity.ManifestRaw))
	}

	// 查询该应用下的受控任务 (type = 'task')
	var tasks []models.Task
	database.DB.Where("source_id = ? AND type = ?", id, constant.TaskTypeNormal).Order("created_at asc").Find(&tasks)

	utils.Success(c, gin.H{
		"app":      appEntity,
		"manifest": manifest,
		"tasks":    tasks,
	})
}

// ApplyAppRequest 部署应用请求体
type ApplyAppRequest struct {
	PathOrURL     string            `json:"path_or_url"`
	RawYAML       string            `json:"raw_yaml"`
	ScenarioID    string            `json:"scenario_id"`
	EnvValues     map[string]string `json:"env_values"`
	SkipSetup       bool              `json:"skip_setup"`
	SkipSync        bool              `json:"skip_sync"`
	ForceSetup      bool              `json:"force_setup"`
	OverwriteEnv    bool              `json:"overwrite_env"`
	OverwriteTask   *bool             `json:"overwrite_task"`
	Schedule        string            `json:"schedule"`
	RandomRange   int               `json:"random_range"`
	Timeout       int               `json:"timeout"`
	RetryCount    int               `json:"retry_count"`
	RetryInterval int               `json:"retry_interval"`
	CleanConfig   string            `json:"clean_config"`
	UnifiedConfig string              `json:"unified_config"`
	Tag           string              `json:"tag"`
	Languages     []map[string]string `json:"languages"`
	EnableTelemetry *bool             `json:"enable_telemetry"`
}

type streamLogWriter struct {
	writer gin.ResponseWriter
	buf    *bytes.Buffer
	mu     sync.Mutex
	isSSE  bool
}

func (w *streamLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(p)

	if w.isSSE {
		lines := strings.Split(string(p), "\n")
		// 丢弃 Write 以换行结尾时 Split 产生的尾部空串，但保留日志中的真实空行
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = lines[:n-1]
		}
		for _, line := range lines {
			fmt.Fprintf(w.writer, "data: %s\n\n", line)
		}
	} else {
		w.writer.Write(p)
	}
	w.writer.Flush()
	return len(p), nil
}

// ApplyApp 部署应用（解析本地/远程/Raw YAML 并应用，支持 SSE 实时日志流推送）
func (ac *AppController) ApplyApp(c *gin.Context) {
	var req ApplyAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数解析失败: "+err.Error())
		return
	}

	isStream := c.Query("stream") == "true" || strings.Contains(c.GetHeader("Accept"), "text/event-stream")

	// 演示模式拦截
	if constant.DemoMode {
		if isStream {
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("X-Accel-Buffering", "no")
			c.Writer.Flush()
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", "演示模式下禁止安装或部署应用")
			c.Writer.Flush()
			return
		}
		utils.BadRequest(c, "演示模式下禁止安装或部署应用")
		return
	}

	var buf bytes.Buffer
	var multiLogWriter io.Writer = &buf

	if isStream {
		c.Header("Content-Type", "text/event-stream; charset=utf-8")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Writer.Flush()

		multiLogWriter = &streamLogWriter{
			writer: c.Writer,
			buf:    &buf,
			isSSE:  true,
		}
	}

	userID := c.GetString("userID")
	if userID == "" {
		userID = "0"
	}

	opts := app.ApplyOptions{
		ScenarioID:    req.ScenarioID,
		EnvValues:     req.EnvValues,
		SkipSetup:     req.SkipSetup,
		SkipSync:      req.SkipSync,
		ForceSetup:    req.ForceSetup,
		OverwriteEnv:  req.OverwriteEnv,
		OverwriteTask: req.OverwriteTask,
		Schedule:      req.Schedule,
		RandomRange:   req.RandomRange,
		Timeout:       req.Timeout,
		RetryCount:    req.RetryCount,
		RetryInterval: req.RetryInterval,
		CleanConfig:   req.CleanConfig,
		UnifiedConfig: req.UnifiedConfig,
		Tag:           req.Tag,
		UserID:        userID,
		Languages:     req.Languages,
		LogWriter:     multiLogWriter,
	}

	var res *app.ApplyResult
	var err error

	if strings.TrimSpace(req.RawYAML) != "" {
		// 直接通过传入的 Raw YAML 进行部署
		rawBytes := []byte(req.RawYAML)
		processedBytes, pErr := app.PreprocessYAMLTemplate(rawBytes)
		if pErr == nil {
			rawBytes = processedBytes
		}
		manifest, pErr := app.ParseManifestFromYAML(rawBytes)
		if pErr != nil {
			if isStream {
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", "解析 YAML 失败: "+pErr.Error())
				c.Writer.Flush()
				return
			}
			utils.BadRequest(c, "解析 YAML 失败: "+pErr.Error())
			return
		}
		opts.ManifestPath = req.PathOrURL
		res, err = app.DefaultApplier.Apply(manifest, rawBytes, opts)
	} else if strings.TrimSpace(req.PathOrURL) != "" {
		res, err = ac.appService.ApplyApp(strings.TrimSpace(req.PathOrURL), opts)
	} else {
		if isStream {
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", "必须提供 path_or_url 或 raw_yaml")
			c.Writer.Flush()
			return
		}
		utils.BadRequest(c, "必须提供 path_or_url 或 raw_yaml")
		return
	}

	if err == nil && res != nil {
		ac.syncAppCronTasks(res.ID)
	}

	if isStream {
		if err != nil {
			errObj, _ := json.Marshal(gin.H{"error": err.Error(), "log": buf.String()})
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", string(errObj))
		} else {
			// 后台异步上报该应用的安装下载量 +1（防阻塞，单次生效）
			reportAppDownloadTelemetry(req.EnableTelemetry, res)
			resObj, _ := json.Marshal(gin.H{"result": res, "log": buf.String()})
			fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", string(resObj))
		}
		c.Writer.Flush()
		return
	}

	if err != nil {
		// 部署失败按双轨契约返回 HTTP 400，日志经 data 回传
		utils.ErrorData(c, 400, "部署应用失败: "+err.Error(), gin.H{
			"log": buf.String(),
		})
		return
	}

	// 非 SSE 流式部署时的计数上报
	reportAppDownloadTelemetry(req.EnableTelemetry, res)

	utils.Success(c, gin.H{
		"result": res,
		"log":    buf.String(),
	})
}

// SwitchScenario 切换已安装应用的使用场景
func (ac *AppController) SwitchScenario(c *gin.Context) {
	if constant.DemoMode {
		utils.BadRequest(c, "演示模式下禁止切换应用场景")
		return
	}

	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "应用 ID 不能为空")
		return
	}

	var req struct {
		ScenarioID string `json:"scenario_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "场景 ID 不能为空")
		return
	}

	var buf bytes.Buffer
	err := ac.appService.SwitchScenario(id, req.ScenarioID, &buf)
	if err != nil {
		// 切换失败按双轨契约返回 HTTP 400，日志经 data 回传
		utils.ErrorData(c, 400, "切换场景失败: "+err.Error(), gin.H{"log": buf.String()})
		return
	}

	ac.syncAppCronTasks(id)

	utils.Success(c, gin.H{
		"id":          id,
		"scenario_id": req.ScenarioID,
		"log":         buf.String(),
	})
}

// RebuildApp 强制重新预编译/安装依赖（自愈修复）
func (ac *AppController) RebuildApp(c *gin.Context) {
	if constant.DemoMode {
		utils.BadRequest(c, "演示模式下禁止重新构建应用")
		return
	}

	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "应用 ID 不能为空")
		return
	}

	appEntity, err := ac.appService.GetApp(id)
	if err != nil {
		utils.NotFound(c, "未找到该应用")
		return
	}

	if appEntity.ManifestRaw == "" {
		utils.BadRequest(c, "应用缺少原始清单数据，无法重新编译")
		return
	}

	manifest, err := app.ParseManifestFromYAML([]byte(appEntity.ManifestRaw))
	if err != nil {
		utils.BadRequest(c, "解析应用定义失败: "+err.Error())
		return
	}

	// per-app 互斥：重建与更新/执行流程互斥
	lock, lockErr := app.LockAppUpdate(appEntity.ID, 10*time.Minute)
	if lockErr != nil {
		utils.BadRequest(c, "应用正在更新或执行中，重建请稍后再试")
		return
	}
	defer app.ReleaseAppLock(lock)

	var buf bytes.Buffer
	opts := app.ApplyOptions{
		ManifestPath: appEntity.ManifestPath,
		ScenarioID:   appEntity.CurrentScenario,
		SkipSetup:    false,
		SkipSync:     true, // 重建无需重新拉取代码
		ForceSetup:   true, // 强制重新构建，跳过 check
		LogWriter:    &buf,
	}

	res, err := app.DefaultApplier.Apply(manifest, []byte(appEntity.ManifestRaw), opts)
	if err != nil {
		// 重建失败按双轨契约返回 HTTP 400，日志经 data 回传
		utils.ErrorData(c, 400, "重建失败: "+err.Error(), gin.H{"log": buf.String()})
		return
	}

	ac.syncAppCronTasks(id)

	utils.Success(c, gin.H{
		"result": res,
		"log":    buf.String(),
	})
}

// RemoveApp 卸载已安装应用
func (ac *AppController) RemoveApp(c *gin.Context) {
	if constant.DemoMode {
		utils.BadRequest(c, "演示模式下禁止卸载应用")
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		utils.BadRequest(c, "应用 TaskID 不能为空")
		return
	}

	cleanData := c.Query("clean_data") != "false" && c.Query("clean_data") != "0"
	cleanEnvs := c.Query("clean_envs") != "false" && c.Query("clean_envs") != "0"

	var buf bytes.Buffer
	deletedTaskIDs, err := ac.appService.RemoveAppWithOptions(taskID, app.AppRemoveOptions{
		CleanData: cleanData,
		CleanEnvs: cleanEnvs,
	}, &buf)
	if err != nil {
		utils.BadRequest(c, "卸载失败: "+err.Error())
		return
	}

	if ac.executorService != nil {
		for _, tid := range deletedTaskIDs {
			ac.executorService.RemoveCronTask(tid)
			ac.executorService.GetScheduler().StopTask(tid)
		}
	}

	utils.Success(c, gin.H{
		"log": buf.String(),
	})
}

// GetMarketplace 获取应用市场列表
func (ac *AppController) GetMarketplace(c *gin.Context) {
	market, err := ac.appService.FetchMarketplace()
	if err != nil || market == nil || len(market.Apps) == 0 {
		utils.Success(c, gin.H{
			"source": "none",
			"apps":   []interface{}{},
		})
		return
	}

	// 后台异步上报应用市场浏览 PV +1 (防阻塞)
	reportMarketplacePVTelemetry()

	respData := gin.H{
		"source": market.Source,
		"apps":   market.Apps,
	}
	if market.GeneratedAtUTC8 != "" {
		respData["generated_at_utc8"] = market.GeneratedAtUTC8
	}
	if market.BuildTime != "" {
		respData["build_time"] = market.BuildTime
	}
	if market.GeneratedAt != "" {
		respData["generated_at"] = market.GeneratedAt
	}

	utils.Success(c, respData)
}

// GetAppUpdates 检测已装应用与更新源（市场索引 / 远程 URL）的版本差异
func (ac *AppController) GetAppUpdates(c *gin.Context) {
	infos, err := ac.appService.CheckUpdates()
	if err != nil {
		utils.BadRequest(c, "检测应用更新失败: "+err.Error())
		return
	}

	updatable := 0
	for _, info := range infos {
		if info.Updatable {
			updatable++
		}
	}

	utils.Success(c, gin.H{
		"updates":         infos,
		"updatable_count": updatable,
		"checked_at":      models.Now().Time().Format(models.TimeFormat),
	})
}

// UpdateAppRequest 应用更新请求体
type UpdateAppRequest struct {
	EnvValues  map[string]string `json:"env_values"` // 补值（409 安全阀重放时传入）
	SkipSetup  bool              `json:"skip_setup"`
	ForceSetup bool              `json:"force_setup"`
}

// UpdateApp 一键更新已装应用（解析最新清单并按已保存配置重放 Apply 流水线，支持 SSE 实时日志流）
func (ac *AppController) UpdateApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "应用 ID 不能为空")
		return
	}

	var req UpdateAppRequest
	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.BadRequest(c, "参数解析失败: "+err.Error())
			return
		}
	}

	isStream := c.Query("stream") == "true" || strings.Contains(c.GetHeader("Accept"), "text/event-stream")

	// 演示模式拦截
	if constant.DemoMode {
		if isStream {
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("X-Accel-Buffering", "no")
			c.Writer.Flush()
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", "演示模式下禁止更新应用")
			c.Writer.Flush()
			return
		}
		utils.BadRequest(c, "演示模式下禁止更新应用")
		return
	}

	var buf bytes.Buffer
	var multiLogWriter io.Writer = &buf

	if isStream {
		c.Header("Content-Type", "text/event-stream; charset=utf-8")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Writer.Flush()

		multiLogWriter = &streamLogWriter{
			writer: c.Writer,
			buf:    &buf,
			isSSE:  true,
		}
	}

	res, err := ac.appService.UpdateApp(id, app.AppUpdateOptions{
		EnvValues:  req.EnvValues,
		SkipSetup:  req.SkipSetup,
		ForceSetup: req.ForceSetup,
	}, multiLogWriter)

	if isStream {
		// 流式模式下日志已由 streamLogWriter 逐行推送，事件负载不再重复携带全量日志快照（避免前端重复渲染）
		if err != nil {
			var conflict *app.AppUpdateConflictError
			if errors.As(err, &conflict) {
				obj, _ := json.Marshal(gin.H{"error": err.Error(), "missing_envs": conflict.MissingEnvs})
				fmt.Fprintf(c.Writer, "event: conflict\ndata: %s\n\n", string(obj))
			} else {
				errObj, _ := json.Marshal(gin.H{"error": err.Error()})
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", string(errObj))
			}
		} else {
			resObj, _ := json.Marshal(gin.H{"result": res})
			fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", string(resObj))
		}
		c.Writer.Flush()
		return
	}

	if err != nil {
		var conflict *app.AppUpdateConflictError
		if errors.As(err, &conflict) {
			// 安全阀拦截按双轨契约返回 HTTP 409，缺失清单与日志经 data 回传
			utils.ErrorData(c, 409, err.Error(), gin.H{"missing_envs": conflict.MissingEnvs, "log": buf.String()})
			return
		}
		utils.ErrorData(c, 400, "更新应用失败: "+err.Error(), gin.H{"log": buf.String()})
		return
	}

	// 更新可能调整受控子任务编排（新增任务/保留定制后的 Cron），同步调度器注册状态
	ac.syncAppCronTasks(id)

	utils.Success(c, gin.H{"result": res, "log": buf.String()})
}

// RollbackApp 一键回滚到上一成功版本（更新前快照重跑 Apply 流水线，支持 SSE 实时日志流）
func (ac *AppController) RollbackApp(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "应用 ID 不能为空")
		return
	}

	var req UpdateAppRequest
	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.BadRequest(c, "参数解析失败: "+err.Error())
			return
		}
	}

	isStream := c.Query("stream") == "true" || strings.Contains(c.GetHeader("Accept"), "text/event-stream")

	// 演示模式拦截
	if constant.DemoMode {
		if isStream {
			c.Header("Content-Type", "text/event-stream; charset=utf-8")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("X-Accel-Buffering", "no")
			c.Writer.Flush()
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", "演示模式下禁止回滚应用")
			c.Writer.Flush()
			return
		}
		utils.BadRequest(c, "演示模式下禁止回滚应用")
		return
	}

	var buf bytes.Buffer
	var multiLogWriter io.Writer = &buf

	if isStream {
		c.Header("Content-Type", "text/event-stream; charset=utf-8")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Writer.Flush()

		multiLogWriter = &streamLogWriter{
			writer: c.Writer,
			buf:    &buf,
			isSSE:  true,
		}
	}

	res, err := ac.appService.RollbackApp(id, app.AppUpdateOptions{
		EnvValues:  req.EnvValues,
		SkipSetup:  req.SkipSetup,
		ForceSetup: req.ForceSetup,
	}, multiLogWriter)

	if isStream {
		if err != nil {
			var conflict *app.AppUpdateConflictError
			if errors.As(err, &conflict) {
				obj, _ := json.Marshal(gin.H{"error": err.Error(), "missing_envs": conflict.MissingEnvs})
				fmt.Fprintf(c.Writer, "event: conflict\ndata: %s\n\n", string(obj))
			} else {
				errObj, _ := json.Marshal(gin.H{"error": err.Error()})
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", string(errObj))
			}
		} else {
			resObj, _ := json.Marshal(gin.H{"result": res})
			fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", string(resObj))
		}
		c.Writer.Flush()
		return
	}

	if err != nil {
		var conflict *app.AppUpdateConflictError
		if errors.As(err, &conflict) {
			utils.ErrorData(c, 409, err.Error(), gin.H{"missing_envs": conflict.MissingEnvs, "log": buf.String()})
			return
		}
		utils.ErrorData(c, 400, "回滚应用失败: "+err.Error(), gin.H{"log": buf.String()})
		return
	}

	ac.syncAppCronTasks(id)

	utils.Success(c, gin.H{"result": res, "log": buf.String()})
}

// reportMarketplacePVTelemetry 异步上报应用市场浏览 PV (+1)
func reportMarketplacePVTelemetry() {
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		_, _ = client.Get(constant.AppStoreStatsPVURL)
	}()
}

// reportAppDownloadTelemetry 异步上报应用下载量 (+1)
func reportAppDownloadTelemetry(optEnable *bool, res *app.ApplyResult) {
	enableTelemetry := optEnable != nil && *optEnable
	if !enableTelemetry || res == nil || res.ManifestID == "" {
		return
	}

	manifestID := res.ManifestID
	go func(mID string) {
		client := &http.Client{Timeout: 5 * time.Second}
		_, _ = client.Get(constant.AppStoreStatsCountURL + mID)
	}(manifestID)
}
