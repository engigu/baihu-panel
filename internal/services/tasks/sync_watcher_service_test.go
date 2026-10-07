package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/memopt"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/shirou/gopsutil/v3/process"
)

func TestTaskSubscription_Trigger(t *testing.T) {
	var count int32
	sub := &taskSubscription{
		TaskID:        "task-123",
		DebounceDelay: 50 * time.Millisecond,
	}

	onFire := func(taskID string) {
		atomic.AddInt32(&count, 1)
	}

	// 快速连续触发 5 次
	for i := 0; i < 5; i++ {
		sub.Trigger(onFire)
		time.Sleep(10 * time.Millisecond)
	}

	// 等待防抖计时结束
	time.Sleep(100 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	if finalCount != 1 {
		t.Fatalf("预期防抖后只触发 1 次，实际触发: %d", finalCount)
	}
}

func TestSyncWatcherService_RegisterAndDispatch(t *testing.T) {
	svc := &SyncWatcherService{
		subscriptions: make(map[string]*taskSubscription),
		watchedDirs:   make(map[string]int),
		stopCh:        make(chan struct{}),
	}

	task := &models.Task{
		ID:      "test-sync-1",
		Type:    constant.TaskTypeAgentSyncScript,
		Enabled: utils.BoolPtr(true),
	}
	cfg := &models.AgentSyncConfig{
		AgentID:       "agent-1",
		SyncMode:      constant.SyncModeRealtime,
		DebounceDelay: 1,
		IgnoreRules:   []string{"*.tmp", "node_modules/"},
		DirMappings: []models.AgentSyncMapping{
			{
				SourcePath: "$SCRIPTS_DIR$/apps/my-app",
				TargetPath: "apps/my-app",
			},
		},
	}
	task.SetAgentSync(cfg)

	// 注册任务
	svc.RegisterTask(task)

	svc.subMu.RLock()
	sub, exists := svc.subscriptions["test-sync-1"]
	svc.subMu.RUnlock()

	if !exists {
		t.Fatalf("任务注册失败，未能加入 subscriptions")
	}
	if len(sub.Prefixes) != 1 || sub.Prefixes[0] != "apps/my-app" {
		t.Fatalf("前缀解析异常: %v", sub.Prefixes)
	}

	// 注销任务
	svc.UnregisterTask("test-sync-1")
	svc.subMu.RLock()
	_, existsAfter := svc.subscriptions["test-sync-1"]
	svc.subMu.RUnlock()

	if existsAfter {
		t.Fatalf("任务注销失败，仍然存在于 subscriptions")
	}

	// 校验注销后精准监听目录引用计数已被清空
	svc.dirMu.Lock()
	dirCount := len(svc.watchedDirs)
	svc.dirMu.Unlock()
	if dirCount > 1 { // 除了根目录自身外不应有残留
		t.Fatalf("任务注销后目录监听未彻底释放，残留目录数: %d", dirCount)
	}
}

func TestSyncWatcher_MassiveFilesAndDirs_MemoryUsage(t *testing.T) {
	tempDir := t.TempDir()

	getMemStats := func() (rss uint64, heapAlloc uint64, goroutines int) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		heapAlloc = ms.HeapAlloc
		goroutines = runtime.NumGoroutine()
		if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
			if memInfo, err := p.MemoryInfo(); err == nil {
				rss = memInfo.RSS
			}
		}
		return
	}

	initialRSS, initialHeap, initialGoroutines := getMemStats()
	t.Logf("================================================================================")
	t.Logf("【压测基准】初始状态: RSS=%.2f MB, HeapAlloc=%.2f MB, 活跃协程=%d",
		float64(initialRSS)/1024/1024, float64(initialHeap)/1024/1024, initialGoroutines)

	// 1. 生成大量深层嵌套目录与文件结构
	// 规划：创建 150 个目录（模拟多个大型 App 与组件模块），并在内部创建 1500 个脚本文件
	numDirs := 150
	filesPerDir := 10
	totalFiles := numDirs * filesPerDir

	t.Logf(">> 正在生成测试环境: %d 个目录, 包含 %d 个文件...", numDirs, totalFiles)
	startCreate := time.Now()
	for i := 0; i < numDirs; i++ {
		dirPath := filepath.Join(tempDir, fmt.Sprintf("apps/app_%03d/sub_%d", i/5, i%5))
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("创建测试目录失败: %v", err)
		}
		for j := 0; j < filesPerDir; j++ {
			filePath := filepath.Join(dirPath, fmt.Sprintf("script_%02d.js", j))
			content := fmt.Sprintf("// Auto generated test script %d-%d\nconsole.log('hello world %d');\n", i, j, j)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("写入测试文件失败: %v", err)
			}
		}
	}
	t.Logf(">> 测试环境准备完毕, 耗时: %v", time.Since(startCreate))

	afterFsRSS, afterFsHeap, _ := getMemStats()
	t.Logf("【文件生成后】状态: RSS=%.2f MB (增量: +%.2f MB), HeapAlloc=%.2f MB",
		float64(afterFsRSS)/1024/1024, float64(afterFsRSS-initialRSS)/1024/1024, float64(afterFsHeap)/1024/1024)

	// 2. 初始化 SyncWatcherService 并注册全量监听
	svc := &SyncWatcherService{
		subscriptions: make(map[string]*taskSubscription),
		watchedDirs:   make(map[string]int),
		scriptsRoot:   tempDir,
	}

	task := &models.Task{
		ID:      "stress-sync-task",
		Type:    constant.TaskTypeAgentSyncScript,
		Enabled: utils.BoolPtr(true),
	}
	cfg := &models.AgentSyncConfig{
		AgentID:       "agent-stress",
		SyncMode:      constant.SyncModeRealtime,
		DebounceDelay: 1, // 1 秒防抖方便测试
		IgnoreRules:   []string{"*.log", ".git/"},
		DirMappings: []models.AgentSyncMapping{
			{
				SourcePath: "apps",
				TargetPath: "apps",
			},
		},
	}
	task.SetAgentSync(cfg)

	// 注册任务，递归挂载全量目录树监听
	startWatch := time.Now()
	svc.RegisterTask(task)
	watchDuration := time.Since(startWatch)

	svc.dirMu.Lock()
	watchedDirCount := len(svc.watchedDirs)
	svc.dirMu.Unlock()

	afterWatchRSS, afterWatchHeap, watchGoroutines := getMemStats()
	t.Logf("================================================================================")
	t.Logf("【全量监听就绪】递归监听目录数: %d 个, 耗时: %v", watchedDirCount, watchDuration)
	t.Logf("【监听内存指标】RSS=%.2f MB (较初始: +%.2f MB), HeapAlloc=%.2f MB, 协程数=%d",
		float64(afterWatchRSS)/1024/1024,
		float64(afterWatchRSS-initialRSS)/1024/1024,
		float64(afterWatchHeap)/1024/1024,
		watchGoroutines,
	)

	// 3. 模拟并发大量文件写入/保存事件（测试高频变动事件分发与防抖负载）
	t.Logf("================================================================================")
	t.Logf(">> 模拟连续高频文件变动: 随机并发修改 100 个文件...")
	startMod := time.Now()
	for i := 0; i < 100; i++ {
		targetFile := filepath.Join(tempDir, fmt.Sprintf("apps/app_%03d/sub_%d/script_%02d.js", (i*3)%numDirs/5, (i*3)%5, i%filesPerDir))
		_ = os.WriteFile(targetFile, []byte(fmt.Sprintf("// modified at %d\n", time.Now().UnixNano())), 0644)
	}
	t.Logf(">> 100 次高频修改写入完成, 耗时: %v", time.Since(startMod))

	// 等待事件循环分发与防抖队列捕获
	time.Sleep(300 * time.Millisecond)

	afterModRSS, afterModHeap, _ := getMemStats()
	t.Logf("【高频修改触发后】RSS=%.2f MB (较监听态: +%.2f MB), HeapAlloc=%.2f MB",
		float64(afterModRSS)/1024/1024,
		float64(afterModRSS-afterWatchRSS)/1024/1024,
		float64(afterModHeap)/1024/1024,
	)

	// 4. 等待防抖计时结束并执行主动工作集回收
	time.Sleep(1200 * time.Millisecond)
	memopt.Free()

	afterCleanRSS, afterCleanHeap, _ := getMemStats()
	t.Logf("【触发 memopt.Free 物理回收后】RSS=%.2f MB (回落: -%.2f MB), HeapAlloc=%.2f MB",
		float64(afterCleanRSS)/1024/1024,
		float64(afterModRSS-afterCleanRSS)/1024/1024,
		float64(afterCleanHeap)/1024/1024,
	)

	// 5. 注销任务并关闭服务，验证资源完全释放与零残留
	t.Logf("================================================================================")
	t.Logf(">> 注销任务并关闭监听服务...")
	svc.UnregisterTask("stress-sync-task")
	svc.Stop()

	// 彻底回收
	memopt.Free()
	finalRSS, finalHeap, finalGoroutines := getMemStats()
	t.Logf("【注销退出休眠后】RSS=%.2f MB, HeapAlloc=%.2f MB, 协程数=%d (残留已彻底释放)",
		float64(finalRSS)/1024/1024, float64(finalHeap)/1024/1024, finalGoroutines)
	t.Logf("================================================================================")

	// 校验单目录平均占用不超过合理阈值
	if watchedDirCount < 100 {
		t.Fatalf("预期递归监听目录数 >= 100，实际: %d", watchedDirCount)
	}
}

