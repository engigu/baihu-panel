package controllers

import (
	"bytes"
	"encoding/json"
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
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/gin-gonic/gin"
)

type AppController struct {
	appService *app.AppService
}

func NewAppController(appService *app.AppService) *AppController {
	if appService == nil {
		appService = app.DefaultAppService
	}
	return &AppController{
		appService: appService,
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
		for _, line := range lines {
			if line == "" {
				continue
			}
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
		c.JSON(http.StatusOK, utils.Response{
			Code: 400,
			Msg:  "部署应用失败: " + err.Error(),
			Data: gin.H{
				"log": buf.String(),
			},
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
		c.JSON(http.StatusOK, utils.Response{
			Code: 400,
			Msg:  "切换场景失败: " + err.Error(),
			Data: gin.H{"log": buf.String()},
		})
		return
	}

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
		c.JSON(http.StatusOK, utils.Response{
			Code: 400,
			Msg:  "重建失败: " + err.Error(),
			Data: gin.H{"log": buf.String()},
		})
		return
	}

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

	var buf bytes.Buffer
	err := ac.appService.RemoveApp(taskID, cleanData, &buf)
	if err != nil {
		utils.BadRequest(c, "卸载失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"log": buf.String(),
	})
}

// GetMarketplace 获取应用市场列表
func (ac *AppController) GetMarketplace(c *gin.Context) {
	// 2. 从官方远程源拉取应用市场索引（实时性优先 + 容灾加速矩阵）
	// 第1级: GitHub Pages 官方节点 (实时更新)
	// 第2级: 国内 GitHub 镜像代理加速 (ghproxy / ghp.ci)
	// 第3级: GitHub Raw 原生直连
	// 第4级: jsDelivr 备用 CDN
	remoteCandidates := []string{
		"https://engigu.github.io/baihu-appstore/apps.json",
		"https://ghproxy.net/https://raw.githubusercontent.com/engigu/baihu-appstore/main/apps.json",
		"https://ghp.ci/https://raw.githubusercontent.com/engigu/baihu-appstore/main/apps.json",
		"https://raw.githubusercontent.com/engigu/baihu-appstore/main/apps.json",
		"https://cdn.jsdelivr.net/gh/engigu/baihu-appstore@main/apps.json",
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var resp *http.Response
	var err error
	nowTs := time.Now().Unix()
	for _, targetURL := range remoteCandidates {
		fetchURL := fmt.Sprintf("%s?t=%d", targetURL, nowTs)
		resp, err = client.Get(fetchURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		body, rErr := io.ReadAll(resp.Body)
		if rErr == nil {
				var rawJSON interface{}
				if err := json.Unmarshal(body, &rawJSON); err == nil {
					var appList interface{} = rawJSON
					var rawMap map[string]interface{}
					if m, ok := rawJSON.(map[string]interface{}); ok {
						rawMap = m
						if arr, exists := m["apps"]; exists {
							appList = arr
						}
					}

					// 统一提取并注入结构化的 languages 契约列表
					enrichMarketplaceAppLanguages(appList)

					// 后台异步上报应用市场浏览 PV +1 (防阻塞)
					reportMarketplacePVTelemetry()

					respData := gin.H{
						"source": "remote",
						"apps":   appList,
					}
					if rawMap != nil {
						if v, ok := rawMap["generated_at_utc8"]; ok {
							respData["generated_at_utc8"] = v
						}
						if v, ok := rawMap["build_time"]; ok {
							respData["build_time"] = v
						}
						if v, ok := rawMap["generated_at"]; ok {
							respData["generated_at"] = v
						}
					}

					utils.Success(c, respData)
					return
				}
		}
	}

	// 3. 兜底返回空列表
	utils.Success(c, gin.H{
		"source": "none",
		"apps":   []interface{}{},
	})
}

// enrichMarketplaceAppLanguages 遍历应用列表，由 Go 后端统一提取并注入结构化的 languages 运行环境契约列表
func enrichMarketplaceAppLanguages(appList interface{}) {
	appsArr, ok := appList.([]interface{})
	if !ok {
		return
	}
	for _, rawItem := range appsArr {
		appMap, isMap := rawItem.(map[string]interface{})
		if !isMap {
			continue
		}
		langs, hasLangs := appMap["languages"].([]interface{})
		if hasLangs && len(langs) > 0 {
			continue
		}
		var manifest *app.AppManifest
		if rawYAML, hasYAML := appMap["manifest_raw"].(string); hasYAML && rawYAML != "" {
			manifest, _ = app.ParseManifestFromYAML([]byte(rawYAML))
		} else if tmpl, hasTmpl := appMap["template"]; hasTmpl {
			manifest = &app.AppManifest{Template: tmpl}
		}
		if manifest != nil {
			parsedLangs := manifest.GetLanguages()
			if len(parsedLangs) > 0 {
				appMap["languages"] = parsedLangs
			}
		}
	}
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
