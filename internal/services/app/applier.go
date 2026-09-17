package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services/relation"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/engigu/baihu-panel/internal/windows"
	"github.com/rs/xid"
)

// LogFunc 日志输出函数签名
type LogFunc func(format string, args ...interface{})

// ApplyOptions 应用执行选项
type ApplyOptions struct {
	ManifestPath  string            // Manifest 来源路径或 URL
	ScenarioID    string            // 指定激活的场景 ID（留空默认选择 default 场景）
	EnvValues     map[string]string // 用户指定的环境变量覆盖值
	SkipSetup     bool              // 是否跳过依赖安装
	SkipSync      bool              // 是否跳过代码源拉取
	ForceSetup    bool              // 强制执行依赖安装与编译（跳过 check 探活）
	OverwriteEnv  bool              // 是否覆盖已有环境变量（默认 false 不覆盖）
	Schedule      string            // 定时规则 Cron 表达式
	RandomRange   int               // 随机延迟范围(秒)
	Timeout       int               // 执行超时时间(分钟)
	RetryCount    int               // 失败重试次数
	RetryInterval int               // 失败重试间隔(秒)
	CleanConfig   string            // 日志清理配置
	UnifiedConfig string            // 统一配置 JSON
	Tag           string            // 统一绑定的分类 Tag
	UserID        string            // 当前安装用户的 UserID
	Languages     []map[string]string // 关联绑定的运行语言环境列表
	LogWriter     io.Writer         // 执行日志输出流
}

// ApplyResult 应用执行结果
type ApplyResult struct {
	ID             string   `json:"id"`
	ManifestID     string   `json:"manifest_id"`
	AppName        string   `json:"app_name"`
	Version        string   `json:"version"`
	ActiveScenario string   `json:"active_scenario"`
	AppDir         string   `json:"app_dir"`
	CreatedTasks   []string `json:"created_tasks"`
	UpdatedTasks   []string `json:"updated_tasks"`
	CreatedEnvs    []string `json:"created_envs"`
	SkippedSetup   bool     `json:"skipped_setup"`
	InstalledAt    string   `json:"installed_at"`
}

// AppApplier 应用规范核心应用引擎
type AppApplier struct{}

var DefaultApplier = &AppApplier{}

// Apply 应用主流程编排：调用各个独立的过程函数完成全生命周期落地
func (a *AppApplier) Apply(manifest *AppManifest, rawYAML []byte, opts ApplyOptions) (*ApplyResult, error) {
	if manifest == nil {
		return nil, fmt.Errorf("Manifest 不能为空")
	}

	out := opts.LogWriter
	if out == nil {
		out = os.Stdout
	}

	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		out.Write([]byte(msg))
	}

	log("==================================================================")
	log("  白虎应用引擎: 开始部署应用 [%s] (v%s)", manifest.Name, manifest.Version)
	log("==================================================================")

	// 阶段 1: 确定应用存储路径与宏变量
	appDir, err := a.prepareAppDir(manifest, log)
	if err != nil {
		return nil, err
	}

	// 阶段 2: 同步代码与资源源 (直接调用 cmd/reposync)
	if err := a.syncSources(manifest, appDir, opts.SkipSync, log); err != nil {
		return nil, err
	}

	// 阶段 3: 执行前置环境探测与依赖安装 (Setup)
	skippedSetup, err := a.runSetup(manifest, appDir, opts.SkipSetup, opts.ForceSetup, out, log)
	if err != nil {
		return nil, err
	}

	// 阶段 4: 环境变量契约写入与 Tag 绑定 (Env Schema)
	createdEnvs, err := a.applyEnvSchema(manifest, opts, log)
	if err != nil {
		return nil, err
	}

	// 阶段 5: 先持久化主应用记录到 tasks 表 (Type = app)，获取主应用任务实体 masterTask.ID
	masterTask, activeScenarioID, err := a.saveMasterAppTask(manifest, rawYAML, appDir, opts.ManifestPath, opts.ScenarioID, opts.EnvValues, opts)
	if err != nil {
		return nil, err
	}

	// 阶段 6: 场景编排与受控任务批量生成/热更新 (把 masterTask.ID 传入作为受控子任务的 source_id)
	createdTasks, updatedTasks, err := a.orchestrateTasks(manifest, appDir, masterTask.ID, activeScenarioID, opts, log)
	if err != nil {
		return nil, err
	}

	log("\n==================================================================")
	log("  ✓ 应用 [%s] (v%s) 部署成功！", manifest.Name, manifest.Version)
	log("  任务编排: 新增 %d 个, 更新 %d 个 | 环境变量: 注册 %d 项", len(createdTasks), len(updatedTasks), len(createdEnvs))
	log("==================================================================")

	return &ApplyResult{
		ID:             masterTask.ID,
		ManifestID:     manifest.ID,
		AppName:        manifest.Name,
		Version:        manifest.Version,
		ActiveScenario: activeScenarioID,
		AppDir:         appDir,
		CreatedTasks:   createdTasks,
		UpdatedTasks:   updatedTasks,
		CreatedEnvs:    createdEnvs,
		SkippedSetup:   skippedSetup,
		InstalledAt:    models.Now().Time().Format(models.TimeFormat),
	}, nil
}

// --------------------------------------------------------------------------
// 过程函数 1: 准备应用存储目录与宏变量
// --------------------------------------------------------------------------
func (a *AppApplier) prepareAppDir(manifest *AppManifest, log LogFunc) (string, error) {
	absScriptsDir := utils.ResolveAbsScriptsDir()
	appDir := manifest.GetAppDir(absScriptsDir)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", fmt.Errorf("创建应用根目录失败 (%s): %w", appDir, err)
	}

	log("[1/5] 应用根目录就绪: %s", appDir)
	return appDir, nil
}

// --------------------------------------------------------------------------
// 过程函数 2: 同步代码与资源源 (直接调用 cmd/reposync)
// --------------------------------------------------------------------------
func (a *AppApplier) syncSources(manifest *AppManifest, appDir string, skipSync bool, log LogFunc) error {
	if skipSync || len(manifest.Sources) == 0 {
		log("[2/5] 跳过代码源同步阶段")
		return nil
	}

	log("[2/5] 正在同步代码与资源源 (共 %d 个源)...", len(manifest.Sources))
	for idx, src := range manifest.Sources {
		if err := a.syncSingleSource(idx, len(manifest.Sources), src, appDir, log); err != nil {
			return err
		}
	}

	return nil
}

// syncSingleSource 根据 source_type 路由调用不同的处理逻辑函数
func (a *AppApplier) syncSingleSource(idx int, total int, src AppSource, appDir string, log LogFunc) error {
	srcType := strings.ToLower(strings.TrimSpace(src.SourceType))

	switch srcType {
	case "null", "none", "":
		return a.handleNullSource(idx, total, src, log)
	case "git":
		return a.handleGitSource(idx, total, src, appDir, log)
	case "url":
		return a.handleURLSource(idx, total, src, appDir, log)
	default:
		return fmt.Errorf("不支持的代码源类型 '%s' (源 ID: %s)", src.SourceType, src.ID)
	}
}

// handleNullSource 处理空/纯二进制免同步模式 (null/none)
func (a *AppApplier) handleNullSource(idx int, total int, src AppSource, log LogFunc) error {
	log("  ✓ [%d/%d] 代码源 '%s' 为纯二进制免同步模式 (source_type: %s)，直接跳过代码同步", idx+1, total, src.ID, src.SourceType)
	return nil
}

// handleGitSource 处理 Git 仓库代码同步
func (a *AppApplier) handleGitSource(idx int, total int, src AppSource, appDir string, log LogFunc) error {
	return a.execRepoSync(idx, total, src, appDir, log)
}

// handleURLSource 处理单文件/URL直链下载同步
func (a *AppApplier) handleURLSource(idx int, total int, src AppSource, appDir string, log LogFunc) error {
	return a.execRepoSync(idx, total, src, appDir, log)
}

// execRepoSync 调度底层 reposync 工具拉取 Git 或 URL 代码源
func (a *AppApplier) execRepoSync(idx int, total int, src AppSource, appDir string, log LogFunc) error {
	targetSubPath := src.TargetPath
	if targetSubPath == "" {
		targetSubPath = src.ID
	}
	targetAbsDir := filepath.Join(appDir, targetSubPath)
	gitDir := filepath.Join(targetAbsDir, ".git")
	if fi, err := os.Stat(targetAbsDir); err == nil && fi.IsDir() {
		if _, gitErr := os.Stat(gitDir); os.IsNotExist(gitErr) {
			_ = os.RemoveAll(targetAbsDir)
		}
	}
	_ = os.MkdirAll(targetAbsDir, 0755)

	syncArgs := []string{
		"--source-type", src.SourceType,
		"--source-url", src.SourceURL,
		"--target-path", targetAbsDir,
		"--repo-name", ".",
	}
	if src.Branch != "" {
		syncArgs = append(syncArgs, "--branch", src.Branch)
	}
	if src.Path != "" {
		syncArgs = append(syncArgs, "--path", src.Path)
	}
	if src.SingleFile {
		syncArgs = append(syncArgs, "--single-file")
	}
	if src.Proxy != "" && src.Proxy != "none" {
		syncArgs = append(syncArgs, "--proxy", src.Proxy)
		if src.Proxy == "custom" && src.ProxyURL != "" {
			syncArgs = append(syncArgs, "--proxy-url", src.ProxyURL)
		}
	}
	if src.AuthToken != "" {
		syncArgs = append(syncArgs, "--auth-token", src.AuthToken)
	}
	if src.HttpProxy != "" {
		syncArgs = append(syncArgs, "--http-proxy", src.HttpProxy)
	}
	if src.WhitelistPaths != "" {
		syncArgs = append(syncArgs, "--whitelist-paths", src.WhitelistPaths)
	}
	if src.Blacklist != "" {
		syncArgs = append(syncArgs, "--blacklist", src.Blacklist)
	}

	var syncOutput string
	exe := utils.GetBaihuExecutable()
	syncCmd := exec.Command(exe, append([]string{"reposync"}, syncArgs...)...)
	syncCmd.Env = append(os.Environ(), utils.BuildRuntimeProcessEnv()...)
	if outBytes, err := syncCmd.CombinedOutput(); err == nil || len(outBytes) > 0 {
		syncOutput = string(outBytes)
	}

	if fi, err := os.Stat(targetAbsDir); err != nil || !fi.IsDir() {
		log("  ✗ 代码源 '%s' 同步失败: 目标目录缺失 (%s)", src.ID, targetAbsDir)
		if syncOutput != "" {
			log("    [底层同步错误日志]\n%s", syncOutput)
		}
		return fmt.Errorf("代码源 '%s' 同步失败: 目标目录缺失 (%s)", src.ID, targetAbsDir)
	}
	entries, err := os.ReadDir(targetAbsDir)
	if err != nil || len(entries) == 0 {
		log("  ✗ 代码源 '%s' 同步失败: 目标目录为空 (%s)", src.ID, targetAbsDir)
		if syncOutput != "" {
			log("    [底层同步错误日志]\n%s", syncOutput)
		}
		return fmt.Errorf("代码源 '%s' 同步失败: 目标目录为空 (%s)", src.ID, targetAbsDir)
	}

	filterNote := ""
	reFilter := regexp.MustCompile(`共删除 (\d+) 个`)
	if m := reFilter.FindStringSubmatch(syncOutput); len(m) > 1 {
		filterNote = fmt.Sprintf(" | 清理冗余文件 %s 项", m[1])
	}

	branchDesc := src.Branch
	if branchDesc == "" {
		branchDesc = "默认分支"
	}
	log("  ✓ [%d/%d] 代码源 '%s' 同步就绪 (类型: %s, 分支: %s%s)", idx+1, total, src.ID, src.SourceType, branchDesc, filterNote)
	return nil
}

// --------------------------------------------------------------------------
// 过程函数 3: 执行前置环境探测与依赖安装 (Setup)
// --------------------------------------------------------------------------
func (a *AppApplier) runSetup(manifest *AppManifest, appDir string, skipSetup bool, forceSetup bool, out io.Writer, log LogFunc) (bool, error) {
	if skipSetup || (manifest.Setup.Check == "" && manifest.Setup.Install == "") {
		log("[3/5] 跳过环境与依赖编排阶段")
		return true, nil
	}

	log("[3/5] 检查与准备应用运行环境...")

	needInstall := true
	if forceSetup {
		log("  ℹ 已开启强制重构模式 (--rebuild/force-setup)，跳过环境探活，直接执行依赖安装与构建...")
	} else if manifest.Setup.Check != "" {
		checkScript := strings.ReplaceAll(manifest.Setup.Check, "{app_dir}", appDir)
		log("  -> 执行先验环境探活检查: %s", checkScript)
		checkCmd := utils.NewShellCommandCmd(checkScript)
		checkCmd.Dir = appDir
		checkCmd.Env = append(os.Environ(), "APP_DIR="+appDir, "CURR_APP_DIR="+appDir)
		var checkErrBuf bytes.Buffer
		checkCmd.Stderr = &checkErrBuf

		if err := checkCmd.Run(); err == nil {
			log("  ✓ 环境探活检测通过，依赖条件完全满足，秒级跳过安装！")
			return true, nil
		}
		log("  ℹ 环境探测未通过，准备执行依赖安装...")
	}

	if needInstall && manifest.Setup.Install != "" {
		installScript := strings.ReplaceAll(manifest.Setup.Install, "{app_dir}", appDir)
		log("  -> 正在执行依赖安装与构建 (严格 Fail-Fast 模式)...")
		installCmd := utils.NewFailFastShellCommandCmd(installScript)
		installCmd.Dir = appDir
		env := append(os.Environ(), "APP_DIR="+appDir, "CURR_APP_DIR="+appDir)
		installCmd.Env = windows.FixPathEnv(env)

		var fullLogBuf bytes.Buffer
		writer := &smartLogWriter{
			rawBuf: &fullLogBuf,
			onHighlight: func(line string) {
				log("    %s", line)
			},
		}
		installCmd.Stdout = writer
		installCmd.Stderr = writer

		if err := installCmd.Run(); err != nil {
			log("  ✗ 依赖安装构建失败: %v", err)
			log("  [构建错误上下文输出]\n%s", fullLogBuf.String())
			return false, fmt.Errorf("应用依赖安装失败: %w", err)
		}

		// 产物健全性强校验：若任务配置了 bin 工作目录，验证预编译产物是否真实生成
		for _, t := range manifest.GetTasks() {
			wd := t.WorkDir
			if wd == "" && manifest.SyncRules != nil {
				wd = manifest.SyncRules.Defaults.WorkDir
			}
			resolvedWd := strings.ReplaceAll(wd, "{app_dir}", appDir)
			if strings.HasSuffix(resolvedWd, "bin") || strings.Contains(resolvedWd, "/bin") || strings.Contains(resolvedWd, "\\bin") {
				if fi, statErr := os.Stat(resolvedWd); statErr != nil || !fi.IsDir() {
					log("  ✗ 产物断言校验失败: 目标工作目录未生成 (%s)", resolvedWd)
					return false, fmt.Errorf("应用依赖安装失败: 预期产物目录缺失 (%s)", resolvedWd)
				}
				entries, readErr := os.ReadDir(resolvedWd)
				if readErr != nil || len(entries) == 0 {
					log("  ✗ 产物断言校验失败: 目标工作目录为空，预编译产物未生成 (%s)", resolvedWd)
					return false, fmt.Errorf("应用依赖安装失败: 预期产物目录为空，构建产物未生成 (%s)", resolvedWd)
				}
				break
			}
		}

		log("  ✓ 依赖安装与产物校验完成")
	}

	// 额外后置步骤: 安装/构建完成后执行后置 Shell 初始化脚本 (post_install)
	if manifest.Setup.PostInstall != "" {
		postScript := strings.ReplaceAll(manifest.Setup.PostInstall, "{app_dir}", appDir)
		log("  -> 正在执行安装后置命令 (post_install)...")
		postCmd := utils.NewFailFastShellCommandCmd(postScript)
		postCmd.Dir = appDir
		postEnv := append(os.Environ(), "APP_DIR="+appDir, "CURR_APP_DIR="+appDir)
		postCmd.Env = windows.FixPathEnv(postEnv)

		var fullLogBuf bytes.Buffer
		writer := &smartLogWriter{
			rawBuf: &fullLogBuf,
			onHighlight: func(line string) {
				log("    %s", line)
			},
		}
		postCmd.Stdout = writer
		postCmd.Stderr = writer

		if err := postCmd.Run(); err != nil {
			log("  ✗ 安装后置脚本 (post_install) 执行失败: %v", err)
			log("  [执行错误上下文输出]\n%s", fullLogBuf.String())
			return false, fmt.Errorf("安装后置脚本执行失败: %w", err)
		}
		log("  ✓ 后置安装命令 (post_install) 执行完成")
	}

	return false, nil
}

// --------------------------------------------------------------------------
// 过程函数 4: 环境变量契约写入与 Tag 绑定 (Env Schema)
// --------------------------------------------------------------------------
func (a *AppApplier) applyEnvSchema(manifest *AppManifest, opts ApplyOptions, log LogFunc) ([]string, error) {
	userID := opts.UserID
	if userID == "" {
		userID = "0"
	}
	log("[4/5] 配置应用环境变量契约 (共 %d 项)...", len(manifest.EnvSchema))

	templateTag := manifest.GetTemplateTag()
	var createdEnvs []string
	for _, envItem := range manifest.EnvSchema {
		tag := resolveEnvTag(envItem.Tag, templateTag, opts.Tag, manifest.ID)
		created, err := a.upsertSingleEnv(envItem, opts.EnvValues, tag, userID, opts.OverwriteEnv, log)
		if err != nil {
			return nil, err
		}
		if created {
			createdEnvs = append(createdEnvs, envItem.Key)
		}
	}

	return createdEnvs, nil
}

// resolveEnvTag 计算 Tag 映射优先级 (envItem.Tag > templateTag > appTag > manifestID)
func resolveEnvTag(itemTag, templateTag, appTag, manifestID string) string {
	if tag := strings.TrimSpace(itemTag); tag != "" {
		return tag
	}
	if tag := strings.TrimSpace(templateTag); tag != "" {
		return tag
	}
	if tag := strings.TrimSpace(appTag); tag != "" {
		return tag
	}
	return manifestID
}

// upsertSingleEnv 处理单条环境变量的创建、更新与 Tag 绑定
func (a *AppApplier) upsertSingleEnv(envItem AppEnvItem, envValues map[string]string, tag, userID string, overwrite bool, log LogFunc) (bool, error) {
	val := ""
	if envValues != nil {
		if v, ok := envValues[envItem.Key]; ok {
			val = v
		}
	}
	if val == "" && envItem.Default != nil {
		val = fmt.Sprintf("%v", envItem.Default)
	}

	envType := constant.EnvTypeNormal
	if envItem.Type == "secret" {
		envType = constant.EnvTypeSecret
		if val != "" {
			if encValue, err := utils.Encrypt(val); err == nil {
				val = encValue
			}
		}
	}

	remark := envItem.Description
	if remark == "" {
		remark = envItem.Label
	}

	var existing models.EnvironmentVariable
	res := database.DB.Where("name = ?", envItem.Key).Limit(1).Find(&existing)
	if res.RowsAffected > 0 {
		// 若已存在且未开启覆盖，则只绑定 Tag，保持原变量值不变
		if !overwrite {
			relation.DataRelation.SaveTags(existing.ID, constant.RelationTypeEnvTag, tag)
			log("  ~ 环境变量已存在, 跳过覆盖: %s (Tag: %s)", envItem.Key, tag)
			return false, nil
		}

		updates := map[string]interface{}{
			"type": envType,
		}
		if remark != "" {
			updates["remark"] = remark
		}
		if val != "" {
			updates["value"] = models.BigText(val)
		} else if existing.Type == constant.EnvTypeSecret && envType == constant.EnvTypeNormal && string(existing.Value) != "" {
			// 若原来为 secret 密文，现改为普通 string 且未传新值，自动解密恢复明文
			if decValue, err := utils.Decrypt(string(existing.Value)); err == nil {
				updates["value"] = models.BigText(decValue)
			}
		}
		if existing.UserID == "" {
			updates["user_id"] = userID
		}
		if len(updates) > 0 {
			database.DB.Model(&existing).Updates(updates)
		}
		relation.DataRelation.SaveTags(existing.ID, constant.RelationTypeEnvTag, tag)
		log("  + 覆盖更新环境变量: %s (Tag: %s)", envItem.Key, tag)
		return false, nil
	}

	newEnv := models.EnvironmentVariable{
		ID:        xid.New().String(),
		Name:      envItem.Key,
		Value:     models.BigText(val),
		Remark:    remark,
		Type:      envType,
		Enabled:   utils.BoolPtr(true),
		UserID:    userID,
		CreatedAt: models.Now(),
		UpdatedAt: models.Now(),
	}

	if err := database.DB.Create(&newEnv).Error; err != nil {
		return false, fmt.Errorf("注册环境变量 '%s' 失败: %w", envItem.Key, err)
	}

	relation.DataRelation.SaveTags(newEnv.ID, constant.RelationTypeEnvTag, tag)
	log("  + 注册环境变量: %s (Tag: %s, 必填: %v)", envItem.Key, tag, envItem.Required)
	return true, nil
}

// --------------------------------------------------------------------------
// 过程函数 5: 场景编排与受控任务批量生成/热更新 (Tasks Orchestration)
// --------------------------------------------------------------------------
func (a *AppApplier) orchestrateTasks(manifest *AppManifest, appDir string, masterTaskID string, targetScenarioID string, opts ApplyOptions, log LogFunc) ([]string, []string, error) {
	activeScenarioID := targetScenarioID
	var activeScenario *AppScenarioItem

	if len(manifest.Scenarios) > 0 {
		for i := range manifest.Scenarios {
			sc := &manifest.Scenarios[i]
			if activeScenarioID != "" && sc.ID == activeScenarioID {
				activeScenario = sc
				break
			} else if activeScenarioID == "" && sc.Default {
				activeScenario = sc
				activeScenarioID = sc.ID
				break
			}
		}
		if activeScenario == nil {
			activeScenario = &manifest.Scenarios[0]
			activeScenarioID = activeScenario.ID
		}
	}

	log("[5/5] 编排应用定时任务 (激活场景: '%s')...", activeScenarioID)

	tasksList := manifest.GetTasks()
	var createdTasks []string
	var updatedTasks []string
	var enabledSummary []string
	var disabledSummary []string
	foundTaskNames := make(map[string]bool)

	defTimeout := 30
	defWorkDir := appDir
	defTag := ""
	if manifest.SyncRules != nil {
		if manifest.SyncRules.Defaults.Timeout > 0 {
			defTimeout = manifest.SyncRules.Defaults.Timeout
		}
		if manifest.SyncRules.Defaults.WorkDir != "" {
			defWorkDir = strings.ReplaceAll(manifest.SyncRules.Defaults.WorkDir, "{app_dir}", appDir)
		}
	}
	for _, t := range tasksList {
		foundTaskNames[t.Name] = true

		isEnabled := t.Enabled
		scheduleCron := t.Cron
		if scheduleCron == "" {
			scheduleCron = t.DefaultCron
		}

		if activeScenario != nil && activeScenario.TaskPresets != nil {
			if preset, ok := activeScenario.TaskPresets[t.ID]; ok {
				if preset.Enabled != nil {
					isEnabled = *preset.Enabled
				}
				if preset.Cron != "" {
					scheduleCron = preset.Cron
				}
			}
		}

		// 确保将 5 位 Cron 自动规范化为白虎面板所需的 6 位秒级表达式
		scheduleCron = normalizeCron(scheduleCron)

		cmdStr := t.Command
		if cmdStr == "" && t.File != "" {
			cmdStr = t.File
		}
		cmdStr = strings.ReplaceAll(cmdStr, "{app_dir}", appDir)

		taskWorkDir := defWorkDir
		if t.WorkDir != "" {
			taskWorkDir = strings.ReplaceAll(t.WorkDir, "{app_dir}", appDir)
		}

		timeout := defTimeout
		if t.Timeout > 0 {
			timeout = t.Timeout
		}

		templateTag := manifest.GetTemplateTag()
		taskTag := t.Tag
		if taskTag == "" {
			taskTag = defTag
		}
		if taskTag == "" {
			taskTag = templateTag
		}
		if taskTag == "" && opts.Tag != "" {
			taskTag = opts.Tag
		}
		if taskTag == "" {
			taskTag = manifest.ID
		}

		taskLangs := t.GetParsedLanguages()
		if len(taskLangs) == 0 && len(opts.Languages) > 0 {
			taskLangs = opts.Languages
		}
		if len(taskLangs) == 0 {
			taskLangs = manifest.GetLanguages()
		}

		var existingTask models.Task
		tx := database.DB.Where("source_id = ? AND type = ? AND name = ?", masterTaskID, constant.TaskTypeNormal, t.Name).Limit(1).Find(&existingTask)
		if tx.RowsAffected > 0 {
			existingTask.Name = t.Name
			existingTask.Command = models.BigText(cmdStr)
			existingTask.Schedule = scheduleCron
			existingTask.WorkDir = taskWorkDir
			existingTask.Timeout = timeout
			existingTask.Languages = taskLangs
			existingTask.Type = constant.TaskTypeNormal
			existingTask.SourceID = masterTaskID
			if err := database.DB.Model(&existingTask).Select("Name", "Command", "Schedule", "WorkDir", "Timeout", "Languages", "Type", "SourceID").Updates(&existingTask).Error; err != nil {
				return nil, nil, fmt.Errorf("更新任务 '%s' 失败: %w", t.Name, err)
			}
			relation.DataRelation.SaveTags(existingTask.ID, constant.RelationTypeTaskTag, taskTag)
			updatedTasks = append(updatedTasks, t.Name)
		} else {
			newTask := models.Task{
				ID:          utils.GenerateID(),
				Name:        t.Name,
				Command:     models.BigText(cmdStr),
				Schedule:    scheduleCron,
				Type:        constant.TaskTypeNormal,
				TriggerType: constant.TriggerTypeCron,
				Enabled:     utils.BoolPtr(isEnabled),
				Timeout:     timeout,
				WorkDir:     taskWorkDir,
				Languages:   taskLangs,
				SourceID:    masterTaskID,
				CleanConfig: `{"type":"count","keep":30}`,
			}
			if err := database.DB.Create(&newTask).Error; err != nil {
				return nil, nil, fmt.Errorf("创建任务 '%s' 失败: %w", t.Name, err)
			}
			relation.DataRelation.SaveTags(newTask.ID, constant.RelationTypeTaskTag, taskTag)
			createdTasks = append(createdTasks, t.Name)
		}

		if isEnabled {
			enabledSummary = append(enabledSummary, fmt.Sprintf("%s (Cron: %s)", t.Name, scheduleCron))
		} else {
			disabledSummary = append(disabledSummary, t.Name)
		}
	}

	// 树状视觉呈现：已启用清晰列出，休眠任务整洁折叠
	for i, taskStr := range enabledSummary {
		prefix := "  ├─"
		if i == len(enabledSummary)-1 && len(disabledSummary) == 0 {
			prefix = "  └─"
		}
		log("%s [✓ 已启用] %s", prefix, taskStr)
	}
	if len(disabledSummary) > 0 {
		if len(disabledSummary) <= 3 {
			for i, name := range disabledSummary {
				prefix := "  ├─"
				if i == len(disabledSummary)-1 {
					prefix = "  └─"
				}
				log("%s [○ 已休眠] %s", prefix, name)
			}
		} else {
			log("  └─ [○ 已休眠] 其余 %d 个可选任务保持休眠状态 (可在面板按需开启)", len(disabledSummary))
		}
	}

	// 清理废弃旧任务
	var oldTasks []models.Task
	if err := database.DB.Where("source_id = ? AND type = ?", masterTaskID, constant.TaskTypeNormal).Find(&oldTasks).Error; err == nil {
		for _, ot := range oldTasks {
			if !foundTaskNames[ot.Name] {
				log("  - [移除] 旧版本已废弃任务: %s", ot.Name)
				relation.DataRelation.CleanRelations(ot.ID, constant.RelationTypeTaskTag)
				relation.DataRelation.CleanRelations(ot.ID, constant.RelationTypeTaskEnv)
				database.DB.Unscoped().Where("id = ?", ot.ID).Delete(&models.Task{})
			}
		}
	}

	return createdTasks, updatedTasks, nil
}

// --------------------------------------------------------------------------
// 过程函数 6: 持久化 models.App 实体记录
// --------------------------------------------------------------------------
func (a *AppApplier) saveAppRecord(manifest *AppManifest, rawYAML []byte, manifestPath string, activeScenarioID string) error {
	var existingApp models.App
	res := database.DB.Where("id = ?", manifest.ID).Limit(1).Find(&existingApp)
	appRecord := models.App{
		ID:              manifest.ID,
		Name:            manifest.Name,
		Version:         manifest.Version,
		Author:          manifest.Author,
		Category:        manifest.Category,
		Description:     manifest.Description,
		Icon:            manifest.Icon,
		Homepage:        manifest.Homepage,
		ManifestPath:    manifestPath,
		ManifestRaw:     models.BigText(string(rawYAML)),
		CurrentScenario: activeScenarioID,
		Status:          constant.AppStatusInstalled,
		UpdatedAt:       models.Now(),
	}

	if res.RowsAffected > 0 {
		appRecord.CreatedAt = existingApp.CreatedAt
		return database.DB.Model(&existingApp).Updates(&appRecord).Error
	}

	appRecord.CreatedAt = models.Now()
	return database.DB.Create(&appRecord).Error
}

// normalizeCron 确保 Cron 表达式具有 6 个字段 (若为 5 位标准格式则自动补齐秒位 0)
func normalizeCron(cron string) string {
	fields := strings.Fields(cron)
	if len(fields) == 5 {
		return "0 " + cron
	}
	return cron
}

// captureOutput 捕获指定函数执行期间向 os.Stdout 和 os.Stderr 输出的全部内容，避免刷屏
func captureOutput(fn func()) string {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		fn()
		return ""
	}
	os.Stdout = w
	os.Stderr = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	<-done
	_ = r.Close()

	return buf.String()
}

// smartLogWriter 智能行流式日志过滤器：过滤控制台旋转动画/无意义转义符，提炼高价值业务回显
type smartLogWriter struct {
	rawBuf      io.Writer
	onHighlight func(line string)
	lineBuf     bytes.Buffer
}

func (w *smartLogWriter) Write(p []byte) (n int, err error) {
	if w.rawBuf != nil {
		_, _ = w.rawBuf.Write(p)
	}
	for _, b := range p {
		if b == '\n' || b == '\r' {
			if w.lineBuf.Len() > 0 {
				line := strings.TrimSpace(utils.ToUTF8(w.lineBuf.Bytes()))
				w.lineBuf.Reset()
				if line != "" && w.isHighlightLine(line) {
					w.onHighlight(line)
				}
			}
		} else {
			w.lineBuf.WriteByte(b)
		}
	}
	return len(p), nil
}

func (w *smartLogWriter) isHighlightLine(line string) bool {
	// 识别业务主动回显行 (如 ">> ...")
	if strings.HasPrefix(line, ">>") {
		return true
	}
	// 关键失败/告警
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "error") || strings.HasPrefix(lower, "fatal") {
		return true
	}
	return false
}

// saveMasterAppTask 将应用实体数据持久化保存到 tasks 表 (Type = "app", Config 存 Unified——TaskConfig JSON)
func (a *AppApplier) saveMasterAppTask(manifest *AppManifest, rawYAML []byte, appDir string, manifestPath string, targetScenarioID string, envValues map[string]string, opts ApplyOptions) (*models.Task, string, error) {
	activeScenarioID := targetScenarioID
	if len(manifest.Scenarios) > 0 {
		for i := range manifest.Scenarios {
			sc := &manifest.Scenarios[i]
			if activeScenarioID != "" && sc.ID == activeScenarioID {
				break
			} else if activeScenarioID == "" && sc.Default {
				activeScenarioID = sc.ID
				break
			}
		}
		if activeScenarioID == "" {
			activeScenarioID = manifest.Scenarios[0].ID
		}
	}

	appCfg := models.AppTaskConfig{
		ID:              manifest.ID,
		Name:            manifest.Name,
		Version:         manifest.Version,
		Author:          manifest.Author,
		Category:        manifest.Category,
		LastCommit:      manifest.LastCommit,
		Description:     manifest.Description,
		Icon:            manifest.Icon,
		Homepage:        manifest.Homepage,
		ManifestPath:    manifestPath,
		ManifestRaw:     string(rawYAML),
		CurrentScenario: activeScenarioID,
		Status:          constant.AppStatusInstalled,
		EnvValues:       envValues,
		BuildOpts: &models.AppBuildOpts{
			ForceSetup:   opts.ForceSetup,
			SkipSetup:    opts.SkipSetup,
			SkipSync:     opts.SkipSync,
			OverwriteEnv: opts.OverwriteEnv,
		},
	}

	appSourceID := "app:" + manifest.ID
	var existingTask models.Task
	tx := database.DB.Where("source_id = ? AND type = ?", appSourceID, constant.TaskTypeApp).Limit(1).Find(&existingTask)

	// 构造统一的分区块配置结构 UnifiedTaskConfig
	var unified models.UnifiedTaskConfig
	if tx.RowsAffected > 0 && string(existingTask.UnifiedConfig) != "" {
		unified = models.ParseUnifiedTaskConfig(string(existingTask.UnifiedConfig))
	}

	unified.App = &appCfg

	// 尝试解析传入的策略
	if opts.UnifiedConfig != "" {
		passedUnified := models.ParseUnifiedTaskConfig(opts.UnifiedConfig)
		if passedUnified.Common != nil {
			if unified.Common == nil {
				unified.Common = &models.CommonConfig{}
			}
			if passedUnified.Common.Concurrency > 0 {
				unified.Common.Concurrency = passedUnified.Common.Concurrency
			}
			unified.Common.AllEnvs = passedUnified.Common.AllEnvs
		}
	}

	finalUnifiedStr := unified.ToJSON()

	if tx.RowsAffected > 0 {
		existingTask.Name = manifest.Name
		existingTask.WorkDir = appDir
		existingTask.SourceID = appSourceID
		existingTask.UnifiedConfig = models.BigText(finalUnifiedStr)
		if opts.Schedule != "" {
			existingTask.Schedule = opts.Schedule
		}
		if opts.RandomRange > 0 {
			existingTask.RandomRange = opts.RandomRange
		}
		if opts.Timeout > 0 {
			existingTask.Timeout = opts.Timeout
		}
		existingTask.RetryCount = opts.RetryCount
		existingTask.RetryInterval = opts.RetryInterval
		if opts.CleanConfig != "" {
			existingTask.CleanConfig = opts.CleanConfig
		}
		existingTask.UpdatedAt = models.Now()
		err := database.DB.Model(&existingTask).Select("Name", "WorkDir", "SourceID", "UnifiedConfig", "Schedule", "RandomRange", "Timeout", "RetryCount", "RetryInterval", "CleanConfig", "UpdatedAt").Updates(&existingTask).Error
		if err != nil {
			return nil, "", err
		}
		return &existingTask, activeScenarioID, nil
	}

	newTask := models.Task{
		ID:            utils.GenerateID(),
		Name:          manifest.Name,
		Type:          constant.TaskTypeApp,
		SourceID:      appSourceID,
		WorkDir:       appDir,
		UnifiedConfig: models.BigText(finalUnifiedStr),
		Schedule:      opts.Schedule,
		RandomRange:   opts.RandomRange,
		Timeout:       opts.Timeout,
		RetryCount:    opts.RetryCount,
		RetryInterval: opts.RetryInterval,
		CleanConfig:   opts.CleanConfig,
		Enabled:       utils.BoolPtr(true),
		CreatedAt:     models.Now(),
		UpdatedAt:     models.Now(),
	}
	err := database.DB.Create(&newTask).Error
	if err != nil {
		return nil, "", err
	}
	return &newTask, activeScenarioID, nil
}

