package tasks

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/engigu/baihu-panel/internal/windows"
	"github.com/fsnotify/fsnotify"
)

// taskSubscription 单个同步任务的订阅配置与防抖状态
type taskSubscription struct {
	TaskID        string
	Task          *models.Task
	Matcher       *utils.GitIgnoreMatcher
	Prefixes      []string // 订阅的相对路径前缀集合 (如 "apps/jdpro" 或 "test.js")
	WatchedDirs   []string // 该任务当前精准占用的目录绝对路径列表
	DebounceDelay time.Duration
	Timer         *time.Timer
	Mu            sync.Mutex
}

// Trigger 重置防抖计时器并在窗口结束时触发执行回调
func (sub *taskSubscription) Trigger(onFire func(taskID string)) {
	sub.Mu.Lock()
	defer sub.Mu.Unlock()

	delay := sub.DebounceDelay
	if delay <= 0 {
		delay = 8 * time.Second
	}

	if sub.Timer != nil {
		sub.Timer.Stop()
	}

	taskID := sub.TaskID
	sub.Timer = time.AfterFunc(delay, func() {
		onFire(taskID)
	})
}

// Cancel 取消当前正在等待的防抖计时
func (sub *taskSubscription) Cancel() {
	sub.Mu.Lock()
	defer sub.Mu.Unlock()
	if sub.Timer != nil {
		sub.Timer.Stop()
		sub.Timer = nil
	}
}

// SyncWatcherService 全局单例文件监听与实时同步分发管理器
type SyncWatcherService struct {
	watcher         *fsnotify.Watcher
	executorService *ExecutorService
	subscriptions   map[string]*taskSubscription // TaskID -> taskSubscription
	subMu           sync.RWMutex
	watchedDirs     map[string]int               // 目录绝对路径 -> 订阅任务引用计数
	dirMu           sync.Mutex                   // 保护 watchedDirs 与 watcher 的动态增删
	scriptsRoot     string                       // scripts 目录规范绝对路径
	stopCh          chan struct{}
	running         bool
	lifecycleMu     sync.Mutex                   // 保护启动/休眠生命周期切换
}

var (
	globalSyncWatcherService *SyncWatcherService
	syncWatcherOnce          sync.Once
)

// GetSyncWatcherService 获取全局单例文件监听服务
func GetSyncWatcherService() *SyncWatcherService {
	syncWatcherOnce.Do(func() {
		globalSyncWatcherService = &SyncWatcherService{
			subscriptions: make(map[string]*taskSubscription),
			watchedDirs:   make(map[string]int),
		}
	})
	return globalSyncWatcherService
}

// SetExecutorService 绑定 ExecutorService 实例
func (s *SyncWatcherService) SetExecutorService(es *ExecutorService) {
	s.executorService = es
}

// Start 初始化文件监听服务基础环境并加载活跃任务（若无任务则保持休眠，零协程与零内存开销）
func (s *SyncWatcherService) Start() error {
	scriptsRoot, err := filepath.Abs(constant.ScriptsWorkDir)
	if err != nil {
		scriptsRoot = constant.ScriptsWorkDir
	}
	scriptsRoot = filepath.Clean(scriptsRoot)
	s.scriptsRoot = scriptsRoot

	if err := os.MkdirAll(scriptsRoot, 0755); err != nil {
		return fmt.Errorf("确保脚本根目录存在失败: %w", err)
	}

	// 从数据库加载启用了实时/混合同步的任务（若有任务则会自动按需唤醒 watcher，若无任务则保持完全休眠）
	s.loadActiveRealtimeTasks()

	logger.Infof("[SyncWatcher] 文件监听服务就绪 (惰性休眠模式，当前活跃实时任务: %d)", len(s.subscriptions))
	return nil
}

// ensureWatcherRunningLocked 当存在活跃任务时动态唤醒底层 Watcher 与事件循环协程
func (s *SyncWatcherService) ensureWatcherRunningLocked() error {
	if s.running {
		return nil
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建 fsnotify Watcher 失败: %w", err)
	}
	s.watcher = w
	s.stopCh = make(chan struct{})

	// 监听 scripts 根目录自身（非递归，仅 1 个系统句柄）
	if err := w.Add(s.scriptsRoot); err != nil {
		logger.Warnf("[SyncWatcher] 监听 scripts 根目录失败: %v", err)
	} else {
		s.watchedDirs[s.scriptsRoot] = 1
	}

	s.running = true
	go s.eventLoop(s.scriptsRoot)

	logger.Infof("[SyncWatcher] 检测到实时同步任务，已动态唤醒文件系统监听与事件循环协程")
	return nil
}

// stopWatcherLocked 当所有实时任务注销或停用时，完全关闭底层 Watcher 与协程进入休眠状态
func (s *SyncWatcherService) stopWatcherLocked() {
	if !s.running {
		return
	}

	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
	if s.watcher != nil {
		_ = s.watcher.Close()
		s.watcher = nil
	}

	s.watchedDirs = make(map[string]int)
	s.running = false
	logger.Infof("[SyncWatcher] 当前无活跃实时同步任务，监听服务已进入完全休眠，释放所有协程与系统句柄")

	// 异步释放内存给 OS
	go utils.FreeMemory()
}

// Stop 彻底停止文件监听服务
func (s *SyncWatcherService) Stop() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	s.subMu.Lock()
	for _, sub := range s.subscriptions {
		sub.Cancel()
	}
	s.subscriptions = make(map[string]*taskSubscription)
	s.subMu.Unlock()

	s.dirMu.Lock()
	s.stopWatcherLocked()
	s.dirMu.Unlock()

	logger.Infof("[SyncWatcher] 全局文件监听服务已彻底关闭")
}

// DefaultIgnoredWatchDirs 默认文件监听递归遍历时跳过的系统及缓存目录
var DefaultIgnoredWatchDirs = []string{
	".git",
	"node_modules",
	"__pycache__",
	".idea",
	".vscode",
}

// isIgnoredWatchDir 判断给定目录名是否属于跳过监听的黑名单
func isIgnoredWatchDir(name string) bool {
	for _, dir := range DefaultIgnoredWatchDirs {
		if name == dir {
			return true
		}
	}
	return false
}

// watchDirRecursiveLocked 递归监听特定目录树，由调用方保证已获取 dirMu 锁
func (s *SyncWatcherService) watchDirRecursiveLocked(root string) []string {
	var addedDirs []string
	if s.watcher == nil {
		return addedDirs
	}
	if s.watchedDirs == nil {
		s.watchedDirs = make(map[string]int)
	}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		if isIgnoredWatchDir(d.Name()) {
			return filepath.SkipDir
		}

		cleanPath := filepath.Clean(path)
		if s.watchedDirs[cleanPath] == 0 {
			if err := s.watcher.Add(cleanPath); err == nil {
				s.watchedDirs[cleanPath] = 1
				addedDirs = append(addedDirs, cleanPath)
			}
		} else {
			s.watchedDirs[cleanPath]++
			addedDirs = append(addedDirs, cleanPath)
		}
		return nil
	})

	return addedDirs
}

// unwatchDirsLocked 减少目录引用计数，归零时释放系统底层监听句柄，由调用方保证已获取 dirMu 锁
func (s *SyncWatcherService) unwatchDirsLocked(dirs []string) {
	if s.watcher == nil || s.watchedDirs == nil {
		return
	}

	for _, dir := range dirs {
		cleanPath := filepath.Clean(dir)
		count := s.watchedDirs[cleanPath]
		if count <= 1 {
			delete(s.watchedDirs, cleanPath)
			// scriptsRoot 常驻保留，其他目录显式移出 fsnotify
			if cleanPath != s.scriptsRoot {
				_ = s.watcher.Remove(cleanPath)
			}
		} else {
			s.watchedDirs[cleanPath] = count - 1
		}
	}
}

// eventLoop 文件系统变更监听主循环
func (s *SyncWatcherService) eventLoop(scriptsRoot string) {
	for {
		select {
		case <-s.stopCh:
			return

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			logger.Warnf("[SyncWatcher] 监听异常: %v", err)

		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// 忽略无实质写入的只读/权限访问事件
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			// 解析相对路径
			relPath, err := filepath.Rel(scriptsRoot, event.Name)
			if err != nil || strings.HasPrefix(relPath, "..") || relPath == "." {
				continue
			}
			relPath = filepath.ToSlash(relPath)

			// 若是新建目录，检查是否命中任何活跃任务关注的前缀范围
			if event.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
					if !isIgnoredWatchDir(filepath.Base(event.Name)) {
						s.handleDynamicDirCreated(relPath, event.Name)
					}
				}
			}

			s.dispatchFileEvent(relPath, event.Name)
		}
	}
}

// handleDynamicDirCreated 当检测到新建子目录时，仅在属于活跃任务监听前缀时动态增补监听
func (s *SyncWatcherService) handleDynamicDirCreated(relPath string, fullPath string) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	isWindows := windows.IsWindows()
	normRelPath := relPath
	if isWindows {
		normRelPath = strings.ToLower(relPath)
	}

	for _, sub := range s.subscriptions {
		for _, prefix := range sub.Prefixes {
			cleanPrefix := filepath.ToSlash(filepath.Clean(prefix))
			if cleanPrefix == "." || cleanPrefix == "/" || cleanPrefix == "" {
				s.dirMu.Lock()
				added := s.watchDirRecursiveLocked(fullPath)
				s.dirMu.Unlock()
				if len(added) > 0 {
					sub.Mu.Lock()
					sub.WatchedDirs = append(sub.WatchedDirs, added...)
					sub.Mu.Unlock()
				}
				return
			}

			cleanPrefix = strings.TrimPrefix(cleanPrefix, "/")
			targetPrefix := cleanPrefix
			compareRelPath := normRelPath
			if isWindows {
				targetPrefix = strings.ToLower(cleanPrefix)
			}

			// 如果新建目录是该前缀本身或其子孙目录
			if compareRelPath == targetPrefix || strings.HasPrefix(compareRelPath, targetPrefix+"/") {
				s.dirMu.Lock()
				added := s.watchDirRecursiveLocked(fullPath)
				s.dirMu.Unlock()
				if len(added) > 0 {
					sub.Mu.Lock()
					sub.WatchedDirs = append(sub.WatchedDirs, added...)
					sub.Mu.Unlock()
				}
				return
			}
		}
	}
}

// dispatchFileEvent 分发文件变动事件到各订阅任务
func (s *SyncWatcherService) dispatchFileEvent(relPath string, fullPath string) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	if len(s.subscriptions) == 0 {
		return
	}

	isDir := false
	if fi, err := os.Stat(fullPath); err == nil && fi.IsDir() {
		isDir = true
	}

	isWindows := windows.IsWindows()
	normRelPath := relPath
	if isWindows {
		normRelPath = strings.ToLower(relPath)
	}

	for _, sub := range s.subscriptions {
		// 1. 匹配该任务是否订阅了此路径前缀
		matchedPrefix := ""
		matched := false
		for _, prefix := range sub.Prefixes {
			cleanPrefix := filepath.ToSlash(filepath.Clean(prefix))
			if cleanPrefix == "." || cleanPrefix == "/" || cleanPrefix == "" {
				matched = true
				matchedPrefix = ""
				break
			}
			cleanPrefix = strings.TrimPrefix(cleanPrefix, "/")

			targetPrefix := cleanPrefix
			compareRelPath := normRelPath
			if isWindows {
				targetPrefix = strings.ToLower(cleanPrefix)
			}

			if compareRelPath == targetPrefix || strings.HasPrefix(compareRelPath, targetPrefix+"/") {
				matched = true
				matchedPrefix = cleanPrefix
				break
			}
		}

		if !matched {
			continue
		}

		// 2. 计算相对于映射根目录的子路径
		var subRel string
		if matchedPrefix == "" {
			subRel = relPath
		} else {
			if len(relPath) >= len(matchedPrefix) {
				subRel = relPath[len(matchedPrefix):]
			} else {
				subRel = relPath
			}
			subRel = strings.TrimPrefix(subRel, "/")
		}

		// 3. 使用该任务定制的 GitIgnore 规则过滤
		if sub.Matcher != nil && sub.Matcher.Match(subRel, isDir) {
			continue
		}

		// 4. 触发该任务专属的防抖队列
		logger.Infof("[SyncWatcher] 检测到文件变动: %s，命中任务 #%s 监听范围，进入防抖倒计时...", relPath, sub.TaskID)
		sub.Trigger(s.onTaskDebounceFired)
	}
}

// onTaskDebounceFired 防抖时间结束触发实际同步
func (s *SyncWatcherService) onTaskDebounceFired(taskID string) {
	if s.executorService == nil {
		logger.Warnf("[SyncWatcher] 未设置 ExecutorService，无法触发任务 #%s 执行", taskID)
		return
	}

	logger.Infof("[SyncWatcher] 任务 #%s 文件变动防抖窗口结束，触发实时热同步", taskID)
	go s.executorService.ExecuteTask(taskID, nil)
}

// RegisterTask 注册或更新一个同步任务的实时监听
func (s *SyncWatcherService) RegisterTask(task *models.Task) {
	if task == nil || task.Type != constant.TaskTypeAgentSyncScript || !utils.DerefBool(task.Enabled, true) {
		s.UnregisterTask(task.ID)
		return
	}

	syncCfg := task.GetAgentSync()
	if syncCfg == nil {
		s.UnregisterTask(task.ID)
		return
	}

	mode := syncCfg.SyncMode
	// 仅支持 realtime 或 hybrid 模式加入实时监听
	if mode != constant.SyncModeRealtime && mode != constant.SyncModeHybrid {
		s.UnregisterTask(task.ID)
		return
	}

	var prefixes []string
	for _, m := range syncCfg.DirMappings {
		src := strings.TrimSpace(m.SourcePath)
		src = strings.TrimPrefix(src, constant.ScriptsDirPlaceholder)
		src = filepath.ToSlash(filepath.Clean(src))
		src = strings.TrimPrefix(src, "/")
		prefixes = append(prefixes, src)
	}

	debounceSec := syncCfg.DebounceDelay
	if debounceSec <= 0 {
		debounceSec = 8
	}

	matcher := utils.CompileGitIgnore(syncCfg.IgnoreRules)

	scriptsRoot := s.scriptsRoot
	if scriptsRoot == "" {
		scriptsRoot, _ = filepath.Abs(constant.ScriptsWorkDir)
		scriptsRoot = filepath.Clean(scriptsRoot)
	}

	// 收集并精准递归监听该任务关联的目录树
	var targetDirs []string
	s.dirMu.Lock()
	if err := s.ensureWatcherRunningLocked(); err != nil {
		s.dirMu.Unlock()
		logger.Warnf("[SyncWatcher] 唤醒文件监听失败: %v", err)
		return
	}

	for _, prefix := range prefixes {
		if prefix == "" || prefix == "." {
			continue
		}
		targetPath := filepath.Join(scriptsRoot, filepath.FromSlash(prefix))
		if fi, err := os.Stat(targetPath); err == nil {
			if fi.IsDir() {
				dirs := s.watchDirRecursiveLocked(targetPath)
				targetDirs = append(targetDirs, dirs...)
			} else {
				parentDir := filepath.Dir(targetPath)
				dirs := s.watchDirRecursiveLocked(parentDir)
				targetDirs = append(targetDirs, dirs...)
			}
		}
	}
	s.dirMu.Unlock()

	s.subMu.Lock()
	// 若已有旧订阅，先取消定时器并释放旧目录引用
	if old, exists := s.subscriptions[task.ID]; exists {
		old.Cancel()
		s.dirMu.Lock()
		s.unwatchDirsLocked(old.WatchedDirs)
		s.dirMu.Unlock()
	}

	s.subscriptions[task.ID] = &taskSubscription{
		TaskID:        task.ID,
		Task:          task,
		Matcher:       matcher,
		Prefixes:      prefixes,
		WatchedDirs:   targetDirs,
		DebounceDelay: time.Duration(debounceSec) * time.Second,
	}
	s.subMu.Unlock()

	logger.Infof("[SyncWatcher] 成功注册任务 #%s 实时同步监听: 模式=%s, 前缀数=%d, 精准监听目录数=%d, 防抖=%d秒",
		task.ID, mode, len(prefixes), len(targetDirs), debounceSec)
}

// UnregisterTask 注销一个任务的实时监听
func (s *SyncWatcherService) UnregisterTask(taskID string) {
	s.subMu.Lock()
	sub, exists := s.subscriptions[taskID]
	if exists {
		sub.Cancel()
		delete(s.subscriptions, taskID)
	}
	remaining := len(s.subscriptions)
	s.subMu.Unlock()

	if exists {
		s.dirMu.Lock()
		s.unwatchDirsLocked(sub.WatchedDirs)
		if remaining == 0 {
			s.stopWatcherLocked()
		}
		s.dirMu.Unlock()
		logger.Infof("[SyncWatcher] 已取消任务 #%s 实时同步监听，释放精准监听目录数=%d", taskID, len(sub.WatchedDirs))
	}
}

// OnAgentOnline 目标 Agent 重连上线时的补发逻辑
func (s *SyncWatcherService) OnAgentOnline(agentID string) {
	s.subMu.RLock()
	var toTrigger []string
	for _, sub := range s.subscriptions {
		if sub.Task == nil {
			continue
		}
		syncCfg := sub.Task.GetAgentSync()
		if syncCfg != nil && syncCfg.SyncOnOnline {
			targetAgent := syncCfg.AgentID
			if targetAgent == "" && sub.Task.AgentID != nil {
				targetAgent = *sub.Task.AgentID
			}
			if targetAgent == agentID {
				toTrigger = append(toTrigger, sub.TaskID)
			}
		}
	}
	s.subMu.RUnlock()

	for _, tid := range toTrigger {
		logger.Infof("[SyncWatcher] 目标 Agent #%s 上线，触发配置了'上线自动同步'的任务 #%s", agentID, tid)
		if s.executorService != nil {
			go s.executorService.ExecuteTask(tid, nil)
		}
	}
}

// loadActiveRealtimeTasks 启动时从数据库加载所有启用的实时同步任务
func (s *SyncWatcherService) loadActiveRealtimeTasks() {
	var tasksList []models.Task
	err := database.DB.Where("type = ? AND (enabled = 1 OR enabled IS NULL)", constant.TaskTypeAgentSyncScript).Find(&tasksList).Error
	if err != nil {
		logger.Warnf("[SyncWatcher] 从数据库加载同步任务失败: %v", err)
		return
	}

	count := 0
	for i := range tasksList {
		t := &tasksList[i]
		cfg := t.GetAgentSync()
		if cfg != nil && (cfg.SyncMode == constant.SyncModeRealtime || cfg.SyncMode == constant.SyncModeHybrid) {
			s.RegisterTask(t)
			count++
		}
	}

	logger.Infof("[SyncWatcher] 初始化完成，已加载 %d 个实时同步任务", count)

	// 启动加载完成后主动调用一次内存释放，释放初始化产生的大量瞬时堆分配
	utils.FreeMemory()
}

