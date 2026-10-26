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
	TaskID           string `json:"task_id"`     // 主应用任务 ID（前端路由键）
	ManifestID       string `json:"manifest_id"` // 清单 ID
	AppName          string `json:"app_name"`
	UpdateSource     string `json:"update_source"` // marketplace | url | manual
	InstalledVersion string `json:"installed_version"`
	LatestVersion    string `json:"latest_version,omitempty"`
	Updatable        bool   `json:"updatable"`
	Reason           string `json:"reason,omitempty"` // version_changed | manifest_changed | no_source | source_mismatch | fetch_failed
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

// manifestSignature 提取清单的结构签名（任务名与默认 Cron、环境变量契约（含 required 标志）、场景及场景预设 Cron），
// 用于同版本的清单变更检测：上游仅调整任务调度预设时也能被识别为「定义有更新」。
// required 必须参与签名：上游不升版本号把变量从可选改为必填时，闸门/自动预判/检测三处才能感知
// （否则旧契约放行调和、新代码要求该变量 → 运行时失败）。签名运行时两侧现算、从不落库，改算法无迁移负担
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

	envParts := make([]string, 0, len(m.EnvSchema))
	for _, e := range m.EnvSchema {
		if e.Required {
			envParts = append(envParts, e.Key+"(required)")
		} else {
			envParts = append(envParts, e.Key)
		}
	}
	sort.Strings(envParts)

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
	b.WriteString("|envs:" + strings.Join(envParts, ","))
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
	Auto       bool              // 自动更新路径（调度调和内触发）：每日节流 + 安全阀拦截降级为通知
	SkipLock   bool              // 内部选项：调和路径由调度器父进程持有 per-app 锁，子进程跳过加锁（否则父子互等自锁）
}

// AppAutoRefreshSkipError 自动刷新未执行（节流窗口内 / 已是最新 / 更新源不可达 / 安全阀拦截降级）。
// 调和路径收到该错误应回退为按当前清单快照正常调和（与漂移闸门「静默跳过」语义一致）。
// FreezeSync 为 true 表示闸门/安全阀挂起中（pending_gate 非空或本次降级落库）：
// 回退调和必须跳过代码同步，否则代码跟随分支漂移到新 HEAD 而清单停留旧版，
// 恰好复现闸门与失败启发式要防的「代码新、清单旧」运行时失败
type AppAutoRefreshSkipError struct {
	Reason     string
	FreezeSync bool
}

func (e *AppAutoRefreshSkipError) Error() string {
	return e.Reason
}

// UpdateApp 更新编排：解析最新清单 → 409 安全阀 → 组装已保存配置 → 复用 Apply 流水线。
// 手动路径（Auto=false）三端共用；自动路径（Auto=true，调度调和内触发）叠加每日节流与
// 安全阀拦截降级（design/app-update-proposal.md §10），可降级失败以 AppAutoRefreshSkipError 返回
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
	masterTask, appCfg, err := loadAppTask(taskID)
	if err != nil {
		return nil, err
	}

	// 2. 自动更新路径（P3，design/app-update-proposal.md §10）：先做节流与变更预判，
	//    无需刷新时以可跳过错误返回，由调和路径回退为按当前清单快照正常调和
	if opts.Auto {
		if skip := s.precheckAutoRefresh(masterTask, appCfg, out); skip != nil {
			return nil, skip
		}
	}

	// 3. 解析最新清单与原始 YAML
	var (
		newManifest *AppManifest
		rawYAML     []byte
		source      string
	)
	_, hasMarket, marketErr := s.marketHasApp(appCfg.ID)
	if marketErr != nil {
		return nil, s.autoSkipOr(appCfg, opts, fmt.Errorf("检查应用市场索引失败: %w", marketErr), "更新源不可达，本次按当前清单调和")
	}
	if hasMarket {
		m, appMap, err := s.GetMarketplaceApp(appCfg.ID)
		if err != nil {
			return nil, s.autoSkipOr(appCfg, opts, fmt.Errorf("拉取市场最新清单失败: %w", err), "更新源不可达，本次按当前清单调和")
		}
		newManifest = m
		rawYAML, _ = yaml.Marshal(appMap)
		source = "marketplace"
	} else if isHTTPURL(appCfg.ManifestPath) {
		m, raw, err := ParseManifestFromURL(appCfg.ManifestPath, "")
		if err != nil {
			return nil, s.autoSkipOr(appCfg, opts, fmt.Errorf("拉取远程最新清单失败: %w", err), "更新源不可达，本次按当前清单调和")
		}
		// 更新执行同样校验远程清单归属：URL 内容被替换为其他应用时拒绝覆盖，防止把已装应用原地更新成别的应用
		if appCfg.ID != "" && m.ID != "" && !strings.EqualFold(strings.TrimSpace(m.ID), strings.TrimSpace(appCfg.ID)) {
			return nil, fmt.Errorf("远程清单已不属于当前应用（已装 %s，远程 %s），为避免错误覆盖已停止更新，请卸载后重新导入", appCfg.ID, m.ID)
		}
		newManifest = m
		rawYAML = raw
		source = "url"
	} else {
		return nil, s.autoSkipOr(appCfg, opts, fmt.Errorf("该应用无自动更新源（手动导入或已从市场移除），请重新导入清单"), "该应用无自动更新源，本次按当前清单调和")
	}

	log("==================================================================")
	log("  白虎应用引擎: 开始更新应用 [%s] (v%s → v%s)", appCfg.Name, appCfg.Version, newManifest.Version)
	log("  更新源: %s", source)
	log("==================================================================")

	// 自动路径变更预判：版本与清单签名均无变化时跳过 Apply（调和回退为按当前快照空跑，避免每日重复全量流水线）；
	// 若此前闸门/安全阀挂起（上游已回退破坏性变更），一并解除挂起标记
	if opts.Auto && !versionChanged(appCfg.Version, newManifest.Version) {
		if oldManifest, pErr := ParseManifestFromYAML([]byte(appCfg.ManifestRaw)); pErr == nil &&
			manifestSignature(oldManifest) == manifestSignature(newManifest) {
			log("  ✓ 更新源已是最新版本 (v%s)，跳过自动更新", appCfg.Version)
			if appCfg.PendingGate != "" {
				s.clearPendingGate(masterTask.ID)
			}
			return nil, &AppAutoRefreshSkipError{Reason: "更新源已是最新版本，本次按当前清单调和"}
		}
	}

	// 4. 获取 per-app 互斥锁（与定时调和 / 手动执行互斥；调和路径父进程已持锁则跳过）
	if !opts.SkipLock {
		lock, err := LockAppUpdate(masterTask.ID, 10*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("应用更新锁获取失败: %w", err)
		}
		defer ReleaseAppLock(lock)
	}

	// 5. 409 安全阀：新必填变量缺值则拦截（永不静默执行）；
	//    自动路径拦截时降级为通知（闸门同构落库），调和回退为按当前清单继续，任务不失败
	if err := s.checkRequiredEnvs(newManifest, appCfg, opts.EnvValues); err != nil {
		var conflict *AppUpdateConflictError
		if errors.As(err, &conflict) {
			log("  ⛔ 安全阀拦截: %s", err.Error())
			for _, m := range conflict.MissingEnvs {
				log("    - %s (%s)", m.Key, m.Label)
			}
			if opts.Auto {
				s.degradeAutoUpdateConflict(masterTask, appCfg, newManifest, conflict, out)
				return nil, &AppAutoRefreshSkipError{Reason: "安全阀拦截（新增必填变量缺值），已降级为通知，本次按当前清单调和", FreezeSync: true}
			}
		}
		return nil, err
	}

	// 5. 组装 ApplyOptions（全部取自已保存配置，等价于编辑弹窗的回显）
	applyOpts := s.assembleUpdateApplyOptions(appCfg, masterTask, mergeEnvValues(appCfg.EnvValues, opts.EnvValues), opts, out)
	// 更新前快照：记录上一个成功版本，随本次 Apply 落库，供一键回滚（design/app-update-proposal.md §8）
	applyOpts.PreviousSnapshot = &models.AppPreviousSnapshot{
		Version:     appCfg.Version,
		ManifestRaw: appCfg.ManifestRaw,
		UpdatedAt:   models.Now().Time().Format(models.TimeFormat),
	}

	if appCfg.BuildOpts != nil && appCfg.BuildOpts.SkipSync {
		log("  ℹ 已按保存的构建选项跳过代码同步 (skip_sync=true)，仅刷新清单与任务编排")
	}

	// 6. 复用 Apply 五阶段流水线
	res, err := DefaultApplier.Apply(newManifest, rawYAML, applyOpts)
	if err != nil {
		// 更新中断：收敛快照不变式。仅清理「指向当前已装版本」的退化快照（回滚无意义），
		// 此前成功更新留下的有效快照保留，继续作为回滚目标
		s.clearDegeneratePreviousSnapshot(masterTask.ID)
		return nil, err
	}

	// 更新成功即闸门挂起原因消除（补值或更新后自动解除，design §6.2）
	s.clearPendingGate(masterTask.ID)
	// 解除检测节流：下一个调和周期立即重新评估上游漂移
	rearmGateCheck(appCfg.ID)

	log("  ✓ 更新完成: v%s（用户配置与定制项已保留）", res.Version)
	return res, nil
}

// --------------------------------------------------------------------------
// 自动更新路径（P3，design/app-update-proposal.md §10）
// 调度调和触发（CLI apply --refresh-manifest）：每应用节流窗口内至多尝试一次。
// 编排前置失败（更新源不可达 / 节流 / 已是最新 / 安全阀拦截降级）静默回退为按当前清单调和
// （挂起中冻结代码同步）；刷新流水线真实中断（依赖安装失败等）与更新源归属异常则如实失败
// 留痕、令本次调和失败——半更新状态下静默回退调和比让本次调和失败更危险
// --------------------------------------------------------------------------

// autoRefreshInterval 自动更新节流窗口（每应用默认每日一次）；
// BAIHU_APP_AUTO_REFRESH_INTERVAL 可覆盖（供 E2E / 本机验证缩短窗口，如 3s）
var autoRefreshInterval = func() time.Duration {
	if v := strings.TrimSpace(os.Getenv("BAIHU_APP_AUTO_REFRESH_INTERVAL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 24 * time.Hour
}()

// precheckAutoRefresh 自动路径节流预判：窗口内已尝试过则跳过；否则记录本次尝试时间
// （无论后续成功失败均计数，更新源宕机期间不重复打满网络超时）。返回非 nil 表示跳过刷新
func (s *AppService) precheckAutoRefresh(masterTask *models.Task, appCfg *models.AppTaskConfig, out io.Writer) error {
	if appCfg.AutoUpdatedAt != "" {
		if last, pErr := time.ParseInLocation(models.TimeFormat, appCfg.AutoUpdatedAt, time.Local); pErr == nil {
			if elapsed := time.Since(last); elapsed < autoRefreshInterval {
				fmt.Fprintf(out, "  ℹ 距上次自动更新检测仅 %s（节流窗口 %s），跳过刷新，按当前清单调和\n", elapsed.Round(time.Second), autoRefreshInterval)
				// 挂起中（此前安全阀/闸门拦截未解除）节流跳过同样冻结代码同步，避免回退调和把代码拉到新 HEAD
				return s.autoSkipError(appCfg, "自动更新节流窗口内，本次按当前清单调和")
			}
		}
	}
	touchAutoRefreshAt(masterTask.ID, models.Now().Time().Format(models.TimeFormat))
	return nil
}

// touchAutoRefreshAt 写入自动刷新尝试时间戳（JSON 内扩展免迁移，仅更新 unified_config 单字段语义）
func touchAutoRefreshAt(taskID, at string) {
	var task models.Task
	if err := database.DB.Where("id = ? AND type = ?", taskID, constant.TaskTypeApp).First(&task).Error; err != nil {
		return
	}
	cfg := task.GetAppConfig()
	if cfg == nil {
		return
	}
	cfg.AutoUpdatedAt = at
	unified := task.GetUnifiedConfig()
	unified.App = cfg
	database.DB.Model(&task).Update("unified_config", unified.ToJSON())
}

// autoSkipOr 自动路径把可降级错误收敛为跳过（调和回退按当前清单），手动路径原样返回真实错误。
// 挂起中回退时冻结代码同步（autoSkipError 按 pending_gate 判定）
func (s *AppService) autoSkipOr(appCfg *models.AppTaskConfig, opts AppUpdateOptions, err error, autoReason string) error {
	if opts.Auto {
		return s.autoSkipError(appCfg, autoReason)
	}
	return err
}

// autoSkipError 组装自动路径的可跳过错误：闸门/安全阀挂起中（pending_gate 非空）携带 FreezeSync，
// 调和路径回退时跳过代码同步，兑现「代码停旧版本」——否则上游分支新代码会在挂起期间经回退调和照常落地
func (s *AppService) autoSkipError(appCfg *models.AppTaskConfig, reason string) *AppAutoRefreshSkipError {
	return &AppAutoRefreshSkipError{Reason: reason, FreezeSync: appCfg != nil && appCfg.PendingGate != ""}
}

// degradeAutoUpdateConflict 自动更新被安全阀拦截时的降级落库：
// 与漂移闸门同构（pending_gate 标记 + 通知中心未读提示 + 通知渠道推送），
// 用户补值或下次成功更新后自动解除；调和回退为按当前清单继续且冻结代码同步（FreezeSync），任务不失败
func (s *AppService) degradeAutoUpdateConflict(masterTask *models.Task, appCfg *models.AppTaskConfig, newManifest *AppManifest, conflict *AppUpdateConflictError, out io.Writer) {
	decision := &GateDecision{
		Blocked:     true,
		Reason:      "missing_envs",
		VersionFrom: appCfg.Version,
		VersionTo:   newManifest.Version,
		MissingEnvs: conflict.MissingEnvs,
	}
	decision.Message = buildGateBlockMessage(appCfg, decision)
	// CLI 子进程同步推送渠道：仅首次挂起触发（upsert 去重）；子进程随即退出，异步发送会被截断
	if s.applyGateBlock(masterTask, decision) {
		pushNoticeToChannels(gateNoticeTitle, decision.Message)
	}
	fmt.Fprintf(out, "  ℹ 已降级为通知（通知中心 + 通知渠道），本次调和按当前清单继续且跳过代码同步，应用停留 v%s\n", appCfg.Version)
}

// RollbackApp 一键回滚到上一成功版本（单快照方案，design/app-update-proposal.md §8）：
// 用快照清单重跑 Apply 流水线（同样走配置保留与 409 安全阀），成功后清空快照；
// 回滚失败保留快照（当前版本未成功退回，快照仍是唯一回滚目标，可修复后重试）
func (s *AppService) RollbackApp(taskID string, opts AppUpdateOptions, out io.Writer) (*ApplyResult, error) {
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

	// 1. 加载主应用任务
	masterTask, appCfg, err := loadAppTask(taskID)
	if err != nil {
		return nil, err
	}
	if appCfg.Previous == nil || strings.TrimSpace(appCfg.Previous.ManifestRaw) == "" {
		return nil, fmt.Errorf("暂无可回滚的版本快照（首次成功更新后自动生成）")
	}

	// 2. 解析快照清单并校验归属：防止配置被异常篡改后把别的应用覆盖进来
	rollbackManifest, err := ParseManifestFromYAML([]byte(appCfg.Previous.ManifestRaw))
	if err != nil {
		return nil, fmt.Errorf("回滚快照清单解析失败: %w", err)
	}
	if appCfg.ID != "" && rollbackManifest.ID != "" && !strings.EqualFold(strings.TrimSpace(rollbackManifest.ID), strings.TrimSpace(appCfg.ID)) {
		return nil, fmt.Errorf("回滚快照已不属于当前应用（快照 %s，已装 %s），已拒绝执行", rollbackManifest.ID, appCfg.ID)
	}

	log("==================================================================")
	log("  白虎应用引擎: 开始回滚应用 [%s] (v%s → v%s)", appCfg.Name, appCfg.Version, appCfg.Previous.Version)
	log("==================================================================")

	// 3. per-app 互斥锁
	lock, err := LockAppUpdate(masterTask.ID, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("应用回滚锁获取失败: %w", err)
	}
	defer ReleaseAppLock(lock)

	// 4. 409 安全阀同样生效：用户在更新后删除过必填变量值时拦截，绝不静默回滚
	if err := s.checkRequiredEnvs(rollbackManifest, appCfg, opts.EnvValues); err != nil {
		var conflict *AppUpdateConflictError
		if errors.As(err, &conflict) {
			log("  ⛔ 安全阀拦截: %s", err.Error())
			for _, m := range conflict.MissingEnvs {
				log("    - %s (%s)", m.Key, m.Label)
			}
		}
		return nil, err
	}

	// 5. 组装配置保留（与更新路径一致，定制识别基线取当前已装清单）
	applyOpts := s.assembleUpdateApplyOptions(appCfg, masterTask, mergeEnvValues(appCfg.EnvValues, opts.EnvValues), opts, out)
	// 回滚成功即「上一成功版本」成为当前版本，快照自指失去意义，显式清空
	applyOpts.ClearPreviousSnapshot = true

	// 6. 复用 Apply 五阶段流水线
	res, err := DefaultApplier.Apply(rollbackManifest, []byte(appCfg.Previous.ManifestRaw), applyOpts)
	if err != nil {
		return nil, err
	}

	// 回滚成功后应用停留旧版本，闸门挂起原因随之上一个更新循环消失
	s.clearPendingGate(masterTask.ID)
	rearmGateCheck(appCfg.ID)

	log("  ✓ 回滚完成: v%s（用户配置与定制项已保留）", res.Version)
	return res, nil
}

// loadAppTask 严格按主应用 TaskID 加载应用主任务及其 AppTaskConfig
func loadAppTask(taskID string) (*models.Task, *models.AppTaskConfig, error) {
	masterTask, err := GetMasterTask(taskID)
	if err != nil {
		return nil, nil, err
	}
	appCfg := masterTask.GetAppConfig()
	if appCfg == nil || appCfg.ID == "" {
		return nil, nil, fmt.Errorf("应用缺少清单配置，无法执行该操作")
	}
	return masterTask, appCfg, nil
}

// mergeEnvValues 合并已保存环境变量值与补值（409 重放），补值优先且跳过空串
func mergeEnvValues(saved map[string]string, extra map[string]string) map[string]string {
	merged := make(map[string]string, len(saved)+len(extra))
	for k, v := range saved {
		merged[k] = v
	}
	for k, v := range extra {
		if strings.TrimSpace(v) != "" {
			merged[k] = v
		}
	}
	return merged
}

// assembleUpdateApplyOptions 按已保存配置组装更新/回滚共用的 ApplyOptions（配置保留核心，三端一致由服务端组装保证）
func (s *AppService) assembleUpdateApplyOptions(appCfg *models.AppTaskConfig, masterTask *models.Task, mergedEnvs map[string]string, opts AppUpdateOptions, out io.Writer) ApplyOptions {
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

	return ApplyOptions{
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
		BaseManifestRaw: appCfg.ManifestRaw, // 定制识别基线：更新/回滚前的已装清单
		BaseScenarioID:  appCfg.CurrentScenario,
		LogWriter:       out,
	}
}

// clearDegeneratePreviousSnapshot 清理「指向当前已装版本」的退化快照（回滚无意义）。
// Apply 阶段 3 落库先于依赖安装与任务编排，更新中断可能把本次快照留在库中，
// 需收敛「Previous 永不指向当前版本」不变式；有效快照（上个成功更新留下）保持原样
func (s *AppService) clearDegeneratePreviousSnapshot(taskID string) {
	var task models.Task
	if err := database.DB.Where("id = ? AND type = ?", taskID, constant.TaskTypeApp).First(&task).Error; err != nil {
		return
	}
	appCfg := task.GetAppConfig()
	if appCfg == nil || appCfg.Previous == nil {
		return
	}
	if appCfg.Previous.Version != appCfg.Version || appCfg.Previous.ManifestRaw != appCfg.ManifestRaw {
		return
	}
	appCfg.Previous = nil
	unified := task.GetUnifiedConfig()
	unified.App = appCfg
	database.DB.Model(&task).Update("unified_config", unified.ToJSON())
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
	// 1. SQL 层标准粗筛（ANSI SQL 兼容，不依赖各数据库方言内置的 TRIM 特殊行为）
	database.DB.Select("name", "value").
		Where("enabled IS NULL OR enabled = ?", true).
		Where("value IS NOT NULL AND value <> ''").
		Find(&dbEnvs)
	// 2. Go 应用层严密清洗（统一消除 \t \n \r 空格全量字符集差异）
	for _, e := range dbEnvs {
		if strings.TrimSpace(string(e.Value)) != "" {
			existingEnvNames[e.Name] = true
		}
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
