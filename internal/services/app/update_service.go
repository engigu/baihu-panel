package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
)

// ==========================================================================
// 应用更新服务：市场索引缓存、更新检测、一键更新编排、409 安全阀与跨进程互斥锁
// 设计文档: design/app-update-proposal.md (P1)
// ==========================================================================

// --------------------------------------------------------------------------
// per-app 跨进程互斥锁（文件锁）：互斥「API 更新」与「定时调和 / 手动执行」
// --------------------------------------------------------------------------

func appLockPath(appID string) string {
	dir := filepath.Join(os.TempDir(), "baihu-app-locks")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, fmt.Sprintf("%s.lock", appID))
}

// LockAppUpdate 获取应用更新互斥锁（阻塞等待，带超时），更新编排全程持锁
func LockAppUpdate(appID string, timeout time.Duration) (*flock.Flock, error) {
	fl := flock.New(appLockPath(appID))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	locked, err := fl.TryLockContext(ctx, 250*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("应用锁获取失败 (app: %s): %w", appID, err)
	}
	if !locked {
		return nil, fmt.Errorf("应用锁获取超时 (app: %s, 已等待 %s)", appID, timeout)
	}
	return fl, nil
}

// TryLockAppRun 尝试获取应用运行互斥锁（非阻塞），供调和 / 手动执行路径调用
func TryLockAppRun(appID string) (*flock.Flock, bool) {
	fl := flock.New(appLockPath(appID))
	locked, err := fl.TryLock()
	if err != nil || !locked {
		return nil, false
	}
	return fl, true
}

// ReleaseAppLock 释放并关闭应用锁
func ReleaseAppLock(fl *flock.Flock) {
	if fl != nil {
		_ = fl.Unlock()
		_ = fl.Close()
	}
}

// --------------------------------------------------------------------------
// 市场索引 TTL 缓存：检测 API 高频调用，避免每次请求都远程拉取
// --------------------------------------------------------------------------

var marketCache struct {
	mu       sync.Mutex
	data     *MarketplaceResult
	expireAt time.Time
}

const marketCacheTTL = 5 * time.Minute

// marketDegradeBackoff 拉取失败降级复用旧缓存时的短退避时长,避免更新源宕机期间每次检测都打满网络超时
const marketDegradeBackoff = 30 * time.Second

// maxURLProbeConcurrency URL 源并发探测的并发上限
const maxURLProbeConcurrency = 4

// FetchMarketplaceCached 带缓存的市场索引拉取；force 为 true 时旁路缓存
func (s *AppService) FetchMarketplaceCached(force bool) (*MarketplaceResult, error) {
	marketCache.mu.Lock()
	defer marketCache.mu.Unlock()

	if !force && marketCache.data != nil && time.Now().Before(marketCache.expireAt) {
		return marketCache.data, nil
	}

	data, err := s.FetchMarketplace()
	// 网络不可达时 FetchMarketplace 返回 Source="none" 的空索引而非错误，同样视为拉取失败，
	// 避免空索引被当正常结果缓存 5 分钟，期间把市场应用误报为「无更新源」
	if err == nil && data.Source == "none" && len(data.Apps) == 0 {
		err = fmt.Errorf("应用市场索引拉取失败（更新源不可达）")
	}
	if err != nil {
		// 拉取失败时若存在旧缓存则降级复用，保证检测接口可用性;顺延短退避避免持续打满超时
		if marketCache.data != nil {
			marketCache.expireAt = time.Now().Add(marketDegradeBackoff)
			return marketCache.data, nil
		}
		return nil, err
	}

	marketCache.data = data
	marketCache.expireAt = time.Now().Add(marketCacheTTL)
	return data, nil
}

// --------------------------------------------------------------------------
// URL 清单探测结果缓存：仅作拉取失败时的降级数据源（与市场索引缓存「失败降级旧缓存」语义一致）。
// 检测主路径必须实时拉取：缓存优先会让上游刚发布的版本/清单变更在 TTL 窗口内不可见，
// 版本检测的正确性优先于出站请求成本（URL 源应用稀少 + 并发上限兜底）
// --------------------------------------------------------------------------

var urlProbeCache struct {
	mu    sync.Mutex
	items map[string]urlProbeEntry
}

type urlProbeEntry struct {
	manifest *AppManifest
	expireAt time.Time
}

const urlProbeCacheTTL = 5 * time.Minute

func getCachedURLManifest(rawURL string) (*AppManifest, bool) {
	urlProbeCache.mu.Lock()
	defer urlProbeCache.mu.Unlock()
	entry, ok := urlProbeCache.items[rawURL]
	if !ok || time.Now().After(entry.expireAt) {
		return nil, false
	}
	return entry.manifest, true
}

func cacheURLManifest(rawURL string, m *AppManifest) {
	if rawURL == "" || m == nil {
		return
	}
	urlProbeCache.mu.Lock()
	defer urlProbeCache.mu.Unlock()
	if urlProbeCache.items == nil {
		urlProbeCache.items = make(map[string]urlProbeEntry)
	}
	// 写入时顺手清理过期项，应用卸载后 URL 不会长期残留在内存
	now := time.Now()
	for k, e := range urlProbeCache.items {
		if now.After(e.expireAt) {
			delete(urlProbeCache.items, k)
		}
	}
	urlProbeCache.items[rawURL] = urlProbeEntry{manifest: m, expireAt: now.Add(urlProbeCacheTTL)}
}

// --------------------------------------------------------------------------
// 更新检测
// --------------------------------------------------------------------------

// AppUpdateInfo 单个已装应用的更新检测结果
type AppUpdateInfo struct {
	TaskID           string `json:"task_id"`           // 主应用任务 ID（前端路由键）
	ManifestID       string `json:"manifest_id"`       // 清单 ID
	AppName          string `json:"app_name"`
	UpdateSource     string `json:"update_source"`     // marketplace | url | manual
	InstalledVersion string `json:"installed_version"`
	LatestVersion    string `json:"latest_version,omitempty"`
	Updatable        bool   `json:"updatable"`
	Reason           string `json:"reason,omitempty"`  // version_changed | manifest_changed | no_source | source_mismatch | fetch_failed
	LastCommit       string `json:"last_commit,omitempty"`
}

// MissingEnvItem 409 安全阀返回的缺失必填环境变量
type MissingEnvItem struct {
	Key         string `json:"key"`
	Label       string `json:"label,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// AppUpdateConflictError 更新被安全阀拦截（必填变量缺值，永不静默执行）
type AppUpdateConflictError struct {
	MissingEnvs []MissingEnvItem
}

func (e *AppUpdateConflictError) Error() string {
	return fmt.Sprintf("新版本存在 %d 个必填环境变量且当前无可用值，已暂停更新（补值后可继续）", len(e.MissingEnvs))
}

// ErrAppUpToDate 已是最新版本
var ErrAppUpToDate = errors.New("已是最新版本，无需更新")

// CheckUpdates 全量检测已装应用与更新源（市场索引 / URL）的版本与清单差异
func (s *AppService) CheckUpdates() ([]AppUpdateInfo, error) {
	apps, err := s.ListApps()
	if err != nil {
		return nil, err
	}

	market, _ := s.FetchMarketplaceCached(false)
	marketMap := make(map[string]map[string]interface{})
	if market != nil {
		for _, am := range market.Apps {
			if id, _ := am["id"].(string); strings.TrimSpace(id) != "" {
				marketMap[strings.ToLower(strings.TrimSpace(id))] = am
			}
		}
	}

	infos := make([]AppUpdateInfo, 0, len(apps))
	type urlJob struct {
		idx      int
		url      string
		expectID string
	}
	var urlJobs []urlJob
	fetchedNew := make(map[int]*AppManifest)

	for i := range apps {
		dto := &apps[i]
		info := AppUpdateInfo{
			TaskID:           dto.ID,
			ManifestID:       dto.ManifestID,
			AppName:          dto.Name,
			InstalledVersion: dto.Version,
			LastCommit:       dto.LastCommit,
		}

		switch {
		case dto.ManifestID != "" && marketMap[strings.ToLower(dto.ManifestID)] != nil:
			info.UpdateSource = "marketplace"
			info.LatestVersion, _ = marketMap[strings.ToLower(dto.ManifestID)]["version"].(string)
		case isHTTPURL(dto.ManifestPath):
			info.UpdateSource = "url"
			urlJobs = append(urlJobs, urlJob{idx: i, url: dto.ManifestPath, expectID: dto.ManifestID})
		case dto.ManifestID != "" && market == nil:
			// 市场索引拉取失败且无旧缓存可降级：标记检测失败，而非误报「无更新源」
			info.UpdateSource = "marketplace"
			info.Reason = "fetch_failed"
		default:
			info.UpdateSource = "manual"
			info.Reason = "no_source"
		}
		infos = append(infos, info)
	}

	// URL 源并发拉取最新清单（仅读取版本号，不落库）,限制并发数避免应用较多时瞬时打满连接
	if len(urlJobs) > 0 {
		var wg sync.WaitGroup
		var mu sync.Mutex
		sem := make(chan struct{}, maxURLProbeConcurrency)
		for _, job := range urlJobs {
			wg.Add(1)
			go func(j urlJob) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				m, _, mErr := ParseManifestFromURL(j.url, "")
				if mErr != nil {
					// 实时拉取失败时降级复用旧缓存（若有），避免源站瞬时抖动误报「检测失败」
					if cm, ok := getCachedURLManifest(j.url); ok {
						m = cm
					}
				}
				if m == nil {
					return
				}
				// URL 源必须校验远程清单 id 与已装应用一致：URL 内容可能已被替换为其他应用，
				// 不校验会把别人的清单误报为本应用的新版本，一键更新将原地覆盖出另一个应用
				if j.expectID != "" && m.ID != "" && !strings.EqualFold(strings.TrimSpace(m.ID), strings.TrimSpace(j.expectID)) {
					mu.Lock()
					infos[j.idx].Reason = "source_mismatch"
					mu.Unlock()
					return
				}
				cacheURLManifest(j.url, m)
				mu.Lock()
				fetchedNew[j.idx] = m
				mu.Unlock()
			}(job)
		}
		wg.Wait()

		for _, job := range urlJobs {
			if m := fetchedNew[job.idx]; m != nil {
				infos[job.idx].LatestVersion = m.Version
			} else if infos[job.idx].Reason == "" {
				infos[job.idx].Reason = "fetch_failed"
			}
		}
	}

	// 逐个判定可更新状态
	for i := range infos {
		info := &infos[i]
		if info.Reason == "fetch_failed" || info.Reason == "no_source" || info.Reason == "source_mismatch" {
			continue
		}
		if info.LatestVersion == "" {
			continue
		}

		if versionChanged(info.InstalledVersion, info.LatestVersion) {
			info.Updatable = true
			info.Reason = "version_changed"
			continue
		}

		// 版本号相同但清单结构有差异 → “应用定义有更新”（降级提示）
		var newManifest *AppManifest
		if info.UpdateSource == "marketplace" {
			if m := marketMap[strings.ToLower(info.ManifestID)]; m != nil {
				if raw, _ := m["manifest_raw"].(string); strings.TrimSpace(raw) != "" {
					newManifest, _ = ParseManifestFromYAML([]byte(raw))
				}
			}
		} else if info.UpdateSource == "url" {
			newManifest = fetchedNew[i]
		}

		if newManifest != nil {
			idx := findAppIdx(apps, info.TaskID)
			if idx >= 0 && strings.TrimSpace(apps[idx].ManifestRaw) != "" {
				if oldManifest, oErr := ParseManifestFromYAML([]byte(apps[idx].ManifestRaw)); oErr == nil {
					if manifestSignature(oldManifest) != manifestSignature(newManifest) {
						info.Updatable = true
						info.Reason = "manifest_changed"
					}
				}
			}
		}
	}

	return infos, nil
}

func findAppIdx(apps []AppDTO, taskID string) int {
	for i := range apps {
		if apps[i].ID == taskID {
			return i
		}
	}
	return -1
}

// manifestSignature 提取清单的结构签名（任务名与默认 Cron、环境变量契约、场景及场景预设 Cron），
// 用于同版本的清单变更检测：上游仅调整任务调度预设时也能被识别为「定义有更新」
func manifestSignature(m *AppManifest) string {
	taskParts := make([]string, 0)
	for _, t := range m.GetTasks() {
		cron := t.Cron
		if cron == "" {
			cron = t.DefaultCron
		}
		taskParts = append(taskParts, t.Name+"="+cron)
	}
	sort.Strings(taskParts)

	envKeys := make([]string, 0, len(m.EnvSchema))
	for _, e := range m.EnvSchema {
		envKeys = append(envKeys, e.Key)
	}
	sort.Strings(envKeys)

	scenarioParts := make([]string, 0, len(m.Scenarios))
	for _, sc := range m.Scenarios {
		presetParts := make([]string, 0, len(sc.TaskPresets))
		for taskID, p := range sc.TaskPresets {
			if p.Cron != "" {
				presetParts = append(presetParts, taskID+"="+p.Cron)
			}
		}
		sort.Strings(presetParts)
		scenarioParts = append(scenarioParts, sc.ID+"("+strings.Join(presetParts, ",")+")")
	}
	sort.Strings(scenarioParts)

	var b strings.Builder
	b.WriteString("tasks:" + strings.Join(taskParts, ","))
	b.WriteString("|envs:" + strings.Join(envKeys, ","))
	b.WriteString("|scenarios:" + strings.Join(scenarioParts, ","))
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

// versionChanged 版本号比较：数值段补齐长度后逐位比较（1.2 与 1.2.0 视为相同），解析失败退化为字符串不等
func versionChanged(installed, latest string) bool {
	installed = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(installed), "v"))
	latest = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(latest), "v"))
	if installed == latest {
		return false
	}

	a := strings.Split(installed, ".")
	b := strings.Split(latest, ".")
	allNumeric := len(a) > 0 && len(b) > 0
	for _, seg := range a {
		if !isNumeric(seg) {
			allNumeric = false
			break
		}
	}
	if allNumeric {
		for _, seg := range b {
			if !isNumeric(seg) {
				allNumeric = false
				break
			}
		}
	}
	if allNumeric {
		// 段数不齐按 0 补齐再逐位比较，避免 1.2 与 1.2.0 这类等价版本被误判为有更新
		n := len(a)
		if len(b) > n {
			n = len(b)
		}
		for i := 0; i < n; i++ {
			ai, bi := 0, 0
			if i < len(a) {
				ai, _ = strconv.Atoi(a[i])
			}
			if i < len(b) {
				bi, _ = strconv.Atoi(b[i])
			}
			if ai != bi {
				return true
			}
		}
		return false
	}
	return true
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// --------------------------------------------------------------------------
// 更新编排
// --------------------------------------------------------------------------

// AppUpdateOptions 更新请求选项
type AppUpdateOptions struct {
	EnvValues  map[string]string // 补值（409 重放时传入，优先级高于已保存值）
	SkipSetup  bool              // 跳过依赖安装（默认继承已保存 build_opts）
	ForceSetup bool              // 强制重建
}

// UpdateApp 一键更新编排：解析最新清单 → 409 安全阀 → 组装已保存配置 → 复用 Apply 流水线
func (s *AppService) UpdateApp(taskID string, opts AppUpdateOptions, out io.Writer) (*ApplyResult, error) {
	if out == nil {
		out = os.Stdout
	}
	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		_, _ = out.Write([]byte(msg))
	}

	// 1. 加载主应用任务（兼容传入 manifestID 的场景）
	var masterTask models.Task
	if err := database.DB.Where("id = ? AND type = ?", taskID, constant.TaskTypeApp).First(&masterTask).Error; err != nil {
		if err2 := database.DB.Where("source_id = ? AND type = ?", "app:"+taskID, constant.TaskTypeApp).First(&masterTask).Error; err2 != nil {
			return nil, fmt.Errorf("未找到应用记录: %s", taskID)
		}
	}

	appCfg := masterTask.GetAppConfig()
	if appCfg == nil || appCfg.ID == "" {
		return nil, fmt.Errorf("应用缺少清单配置，无法执行更新")
	}

	// 2. 解析最新清单与原始 YAML
	var (
		newManifest *AppManifest
		rawYAML     []byte
		source      string
	)
	_, hasMarket, marketErr := s.marketHasApp(appCfg.ID)
	if marketErr != nil {
		return nil, fmt.Errorf("检查应用市场索引失败: %w", marketErr)
	}
	if hasMarket {
		m, appMap, err := s.GetMarketplaceApp(appCfg.ID)
		if err != nil {
			return nil, fmt.Errorf("拉取市场最新清单失败: %w", err)
		}
		newManifest = m
		rawYAML, _ = yaml.Marshal(appMap)
		source = "marketplace"
	} else if isHTTPURL(appCfg.ManifestPath) {
		m, raw, err := ParseManifestFromURL(appCfg.ManifestPath, "")
		if err != nil {
			return nil, fmt.Errorf("拉取远程最新清单失败: %w", err)
		}
		// 更新执行同样校验远程清单归属：URL 内容被替换为其他应用时拒绝覆盖，防止把已装应用原地更新成别的应用
		if appCfg.ID != "" && m.ID != "" && !strings.EqualFold(strings.TrimSpace(m.ID), strings.TrimSpace(appCfg.ID)) {
			return nil, fmt.Errorf("远程清单已不属于当前应用（已装 %s，远程 %s），为避免错误覆盖已停止更新，请卸载后重新导入", appCfg.ID, m.ID)
		}
		newManifest = m
		rawYAML = raw
		source = "url"
	} else {
		return nil, fmt.Errorf("该应用无自动更新源（手动导入或已从市场移除），请重新导入清单")
	}

	log("==================================================================")
	log("  白虎应用引擎: 开始更新应用 [%s] (v%s → v%s)", appCfg.Name, appCfg.Version, newManifest.Version)
	log("  更新源: %s", source)
	log("==================================================================")

	// 3. 获取 per-app 互斥锁（与定时调和 / 手动执行互斥）
	lock, err := LockAppUpdate(masterTask.ID, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("应用更新锁获取失败: %w", err)
	}
	defer ReleaseAppLock(lock)

	// 4. 409 安全阀：新必填变量缺值则拦截（永不静默执行）
	if err := s.checkRequiredEnvs(newManifest, appCfg, opts.EnvValues); err != nil {
		var conflict *AppUpdateConflictError
		if errors.As(err, &conflict) {
			log("  ⛔ 安全阀拦截: %s", err.Error())
			for _, m := range conflict.MissingEnvs {
				log("    - %s (%s)", m.Key, m.Label)
			}
		}
		return nil, err
	}

	// 5. 组装 ApplyOptions（全部取自已保存配置，等价于编辑弹窗的回显）
	mergedEnvs := make(map[string]string, len(appCfg.EnvValues)+len(opts.EnvValues))
	for k, v := range appCfg.EnvValues {
		mergedEnvs[k] = v
	}
	for k, v := range opts.EnvValues {
		if strings.TrimSpace(v) != "" {
			mergedEnvs[k] = v
		}
	}

	var overwriteTask *bool
	skipSetup := opts.SkipSetup
	forceSetup := opts.ForceSetup
	skipSync := false
	if appCfg.BuildOpts != nil {
		overwriteTask = appCfg.BuildOpts.OverwriteTask
		if overwriteTask == nil {
			def := true
			overwriteTask = &def
		}
		if !opts.SkipSetup {
			skipSetup = appCfg.BuildOpts.SkipSetup
		}
		if !opts.ForceSetup {
			forceSetup = appCfg.BuildOpts.ForceSetup
		}
		skipSync = appCfg.BuildOpts.SkipSync
	}

	appTag := ""
	if appCfg.Template != nil {
		appTag = appCfg.Template.Tag
	}

	applyOpts := ApplyOptions{
		ManifestPath:    appCfg.ManifestPath,
		ScenarioID:      appCfg.CurrentScenario,
		Schedule:        masterTask.Schedule,
		RandomRange:     masterTask.RandomRange,
		Timeout:         masterTask.Timeout,
		RetryCount:      masterTask.RetryCount,
		RetryInterval:   masterTask.RetryInterval,
		CleanConfig:     masterTask.CleanConfig,
		EnvValues:       mergedEnvs,
		OverwriteEnv:    false,
		KeepEnvValues:   true, // 更新路径：绝不覆盖用户已填环境变量值，清单级 overwrite_env 声明一并忽略
		OverwriteTask:   overwriteTask,
		SkipSetup:       skipSetup,
		ForceSetup:      forceSetup,
		SkipSync:        skipSync,
		Tag:             appTag,
		Languages:       templateLanguagesToMaps(appCfg.Template),
		BaseManifestRaw: appCfg.ManifestRaw, // 定制识别基线：旧清单快照
		BaseScenarioID:  appCfg.CurrentScenario,
		LogWriter:       out,
	}

	if appCfg.BuildOpts != nil && appCfg.BuildOpts.SkipSync {
		log("  ℹ 已按保存的构建选项跳过代码同步 (skip_sync=true)，仅刷新清单与任务编排")
	}

	// 6. 复用 Apply 五阶段流水线
	res, err := DefaultApplier.Apply(newManifest, rawYAML, applyOpts)
	if err != nil {
		return nil, err
	}

	log("  ✓ 更新完成: v%s（用户配置与定制项已保留）", res.Version)
	return res, nil
}

// marketHasApp 检查市场索引中是否存在指定应用（使用缓存）
func (s *AppService) marketHasApp(manifestID string) (map[string]interface{}, bool, error) {
	market, err := s.FetchMarketplaceCached(false)
	if err != nil || market == nil {
		return nil, false, err
	}
	for _, am := range market.Apps {
		if id, _ := am["id"].(string); strings.EqualFold(strings.TrimSpace(id), manifestID) {
			return am, true, nil
		}
	}
	return nil, false, nil
}

// checkRequiredEnvs 409 安全阀：校验新清单必填环境变量是否全部有可用值
func (s *AppService) checkRequiredEnvs(newManifest *AppManifest, appCfg *models.AppTaskConfig, extraValues map[string]string) error {
	if newManifest == nil || len(newManifest.EnvSchema) == 0 {
		return nil
	}

	// 环境变量库中「启用且值非空」的同名项视为已满足（用户手动建过变量）。
	// 与执行注入逻辑对齐：禁用变量在注入时会被跳过，不能算作有可用值，否则安全阀放行后任务仍拿不到值
	existingEnvNames := make(map[string]bool)
	var dbEnvs []models.EnvironmentVariable
	database.DB.Select("name").
		Where("enabled IS NULL OR enabled = ?", true).
		Where("TRIM(COALESCE(value, '')) <> ''").
		Find(&dbEnvs)
	for _, e := range dbEnvs {
		existingEnvNames[e.Name] = true
	}

	var missing []MissingEnvItem
	for _, item := range newManifest.EnvSchema {
		if !item.Required {
			continue
		}
		if hasValue(item.Key, extraValues) || hasValue(item.Key, appCfg.EnvValues) || existingEnvNames[item.Key] {
			continue
		}
		missing = append(missing, MissingEnvItem{
			Key:         item.Key,
			Label:       item.Label,
			Type:        item.Type,
			Description: item.Description,
		})
	}

	if len(missing) > 0 {
		return &AppUpdateConflictError{MissingEnvs: missing}
	}
	return nil
}

func hasValue(key string, m map[string]string) bool {
	if m == nil {
		return false
	}
	return strings.TrimSpace(m[key]) != ""
}

// templateLanguagesToMaps 将已保存的模板语言契约转为 ApplyOptions 所需格式
func templateLanguagesToMaps(tpl *models.AppTemplateConfig) []map[string]string {
	if tpl == nil || len(tpl.Languages) == 0 {
		return nil
	}
	langs := make([]map[string]string, 0, len(tpl.Languages))
	for _, l := range tpl.Languages {
		if l.Name == "" {
			continue
		}
		langs = append(langs, map[string]string{"name": l.Name, "version": l.Version})
	}
	return langs
}
