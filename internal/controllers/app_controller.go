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
// @Summary 获取已安装应用列表
// @Description 获取所有已安装的声明式应用实体信息列表
// @Tags 应用管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=[]app.AppDTO}
// @Router /apps [get]
func (ac *AppController) GetApps(c *gin.Context) {
	apps, err := ac.appService.ListApps()
	if err != nil {
		utils.BadRequest(c, "获取应用列表失败: "+err.Error())
		return
	}

	utils.Success(c, apps)
}

// GetApp 获取指定应用详情（包含 Manifest、受控任务、环境配置）
// @Summary 获取指定应用详情
// @Description 获取指定应用的详细信息，包含 Manifest、受控任务列表及环境配置
// @Tags 应用管理
// @Produce json
// @Security BearerAuth
// @Param id path string true "应用 TaskID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /apps/{id} [get]
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
// @Summary 部署或安装应用
// @Description 根据本地/远程路径或直接提交的原始 YAML 清单部署应用，支持流式日志推送
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param stream query boolean false "是否开启 SSE 流式日志推送"
// @Param body body ApplyAppRequest true "应用部署参数"
// @Success 200 {object} utils.Response{data=app.ApplyResult}
// @Failure 400 {object} utils.Response
// @Router /apps/apply [post]
func (ac *AppController) ApplyApp(c *gin.Context) {
	var req ApplyAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数解析失败: "+err.Error())
		return
	}

	isStream := c.Query("stream") == "true" || strings.Contains(c.GetHeader("Accept"), "text/event-stream")

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
// @Summary 切换已安装应用的使用场景
// @Description 切换指定应用的使用场景，联动更新受控子任务的启停与定时规则
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "应用 TaskID"
// @Param body body object true "场景参数 (scenario_id: string)"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /apps/{id}/switch [post]
func (ac *AppController) SwitchScenario(c *gin.Context) {
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
// @Summary 强制重新预编译/安装依赖
// @Description 强制重新触发应用的 setup 依赖安装与编译流程
// @Tags 应用管理
// @Produce json
// @Security BearerAuth
// @Param id path string true "应用 TaskID"
// @Success 200 {object} utils.Response{data=app.ApplyResult}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /apps/{id}/rebuild [post]
func (ac *AppController) RebuildApp(c *gin.Context) {
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
// @Summary 卸载已安装应用
// @Description 卸载指定应用及其受控子任务，支持可选清理数据与环境变量
// @Tags 应用管理
// @Produce json
// @Security BearerAuth
// @Param id path string true "应用 TaskID"
// @Param clean_data query boolean false "是否清理应用工作目录 (默认 true)"
// @Param clean_envs query boolean false "是否清理应用绑定的环境变量 (默认 true)"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /apps/{id} [delete]
func (ac *AppController) RemoveApp(c *gin.Context) {
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
// @Summary 获取应用市场列表
// @Description 获取官方与镜像应用市场的应用清单索引
// @Tags 应用管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response
// @Router /apps/store [get]
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
// @Summary 检测已装应用更新
// @Description 检测已装应用与远端更新源的版本差异与可更新状态
// @Tags 应用管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /apps/updates [get]
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
// @Summary 一键更新应用
// @Description 一键更新已装应用至最新版本，支持环境变量安全阀拦截与流式日志推送
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "应用 TaskID"
// @Param stream query boolean false "是否开启 SSE 流式日志推送"
// @Param body body UpdateAppRequest false "更新参数"
// @Success 200 {object} utils.Response{data=app.ApplyResult}
// @Failure 400 {object} utils.Response
// @Failure 409 {object} utils.Response "存在未填写的必填环境变量"
// @Router /apps/{id}/update [post]
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
// @Summary 一键回滚应用
// @Description 一键回滚应用至更新前的版本快照
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "应用 TaskID"
// @Param stream query boolean false "是否开启 SSE 流式日志推送"
// @Param body body UpdateAppRequest false "回滚参数"
// @Success 200 {object} utils.Response{data=app.ApplyResult}
// @Failure 400 {object} utils.Response
// @Router /apps/{id}/rollback [post]
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

// SetAppAutoUpdateRequest 自动更新开关请求体
type SetAppAutoUpdateRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// SetAppAutoUpdate 开/关应用自动更新（P3，design/app-update-proposal.md §10）
// @Summary 开/关应用自动更新
// @Description 设置指定应用是否开启定时自动检查并静默更新
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "应用 TaskID"
// @Param body body SetAppAutoUpdateRequest true "自动更新开关设置"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /apps/{id}/auto-update [post]
func (ac *AppController) SetAppAutoUpdate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "应用 ID 不能为空")
		return
	}

	var req SetAppAutoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		utils.BadRequest(c, "参数解析失败: enabled 必填（布尔值）")
		return
	}

	dto, err := ac.appService.SetAppAutoUpdate(id, *req.Enabled)
	if err != nil {
		utils.BadRequest(c, "设置自动更新失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"app":         dto,
		"auto_update": dto.AutoUpdate,
	})
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
