package app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/app"
)

func Run(args []string) {
	printHelp := func() {
		fmt.Fprintf(os.Stderr, "\n白虎面板应用管理工具 (App Engine)\n\n")
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  baihu app <子命令> [参数]\n\n")
		fmt.Fprintf(os.Stderr, "可用子命令:\n")
		fmt.Fprintf(os.Stderr, "  apply   解析并应用本地或远程 app.yaml 规范文件\n")
		fmt.Fprintf(os.Stderr, "  list    查看当前已安装的应用清单 (--updates 仅显示可更新项)\n")
		fmt.Fprintf(os.Stderr, "  info    查看指定已安装应用的详细信息与任务列表\n")
		fmt.Fprintf(os.Stderr, "  switch  一键切换指定应用的运行场景预设\n")
		fmt.Fprintf(os.Stderr, "  update  一键更新指定应用至更新源最新版本\n")
		fmt.Fprintf(os.Stderr, "  check-updates 检测全部已装应用的可用更新 (--notify 同时落通知并推送渠道)\n")
		fmt.Fprintf(os.Stderr, "  auto-update 开/关指定应用的自动更新（调度调和内经安全阀自动更新）\n")
		fmt.Fprintf(os.Stderr, "  remove  卸载指定应用并清理关联任务\n\n")
		fmt.Fprintf(os.Stderr, "示例:\n")
		fmt.Fprintf(os.Stderr, "  baihu app apply ./apps/BiliBiliToolPro/app.yaml\n")
		fmt.Fprintf(os.Stderr, "  baihu app apply https://raw.githubusercontent.com/xxx/app.yaml --scenario minimal\n")
		fmt.Fprintf(os.Stderr, "  baihu app list\n")
		fmt.Fprintf(os.Stderr, "  baihu app list --updates\n")
		fmt.Fprintf(os.Stderr, "  baihu app switch bilibili-tool-pro --scenario hardcore\n")
		fmt.Fprintf(os.Stderr, "  baihu app update bilibili-tool-pro\n")
		fmt.Fprintf(os.Stderr, "  baihu app update bilibili-tool-pro --env API_KEY=xxx\n")
		fmt.Fprintf(os.Stderr, "  baihu app check-updates --notify\n")
		fmt.Fprintf(os.Stderr, "  baihu app auto-update bilibili-tool-pro on\n")
		fmt.Fprintf(os.Stderr, "  baihu app remove bilibili-tool-pro --clean-data\n\n")
	}

	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printHelp()
		return
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "apply":
		runApply(subArgs)
	case "list":
		runList(subArgs)
	case "info":
		runInfo(subArgs)
	case "switch":
		runSwitch(subArgs)
	case "update":
		runUpdate(subArgs)
	case "check-updates":
		runCheckUpdates(subArgs)
	case "auto-update":
		runAutoUpdate(subArgs)
	case "remove", "delete":
		runRemove(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", subCmd)
		printHelp()
		os.Exit(1)
	}
}

func runApply(args []string) {
	fs := flag.NewFlagSet("app apply", flag.ExitOnError)
	var scenario string
	var skipSetup bool
	var skipSync bool
	var forceSetup bool
	var overwriteTaskStr string
	var overwriteEnv bool
	var noLock bool
	var sourcePath string
	var refreshManifest bool

	fs.StringVar(&scenario, "scenario", "", "指定激活的应用场景模板 ID")
	fs.BoolVar(&skipSetup, "skip-setup", false, "跳过前置环境检测与依赖安装步骤")
	fs.BoolVar(&skipSync, "skip-sync", false, "跳过代码源同步步骤")
	fs.BoolVar(&forceSetup, "force-setup", false, "强制重新执行依赖安装与编译（跳过 check 探活）")
	fs.BoolVar(&overwriteEnv, "overwrite-env", false, "是否覆盖已有同名环境变量")
	fs.StringVar(&overwriteTaskStr, "overwrite-task", "", "是否覆盖并同步更新受控任务 (true/false)")
	// 内部选项：定时调和路径由调度器持有 per-app 锁，子进程不得重复加锁（否则父子互等自锁）
	fs.BoolVar(&noLock, "no-lock", false, "跳过 per-app 互斥锁（仅供调度器调和路径内部使用）")
	// 内部选项：调和落库时保留已保存更新源（URL/本地路径），避免临时清单快照路径覆盖 manifest_path
	fs.StringVar(&sourcePath, "source-path", "", "落库保留的更新源路径（仅供调度器调和路径内部使用）")
	// 内部选项：自动更新路径（P3）——调和前先经安全阀与每日节流刷新至更新源最新清单
	fs.BoolVar(&refreshManifest, "refresh-manifest", false, "调和前自动刷新至更新源最新清单（仅供调度器调和路径内部使用）")

	normArgs := normalizeArgs(args)
	if err := fs.Parse(normArgs); err != nil {
		return
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 必须提供 app.yaml 的文件路径或远程 URL")
		fmt.Fprintln(os.Stderr, "用法: baihu app apply <file-path-or-url> [--scenario <id>] [--force-setup]")
		os.Exit(1)
	}

	targetPathOrURL := parsedArgs[0]

	var overwriteTaskPtr *bool
	if overwriteTaskStr != "" {
		val := strings.ToLower(overwriteTaskStr) == "true" || overwriteTaskStr == "1"
		overwriteTaskPtr = &val
	}

	// 自动更新路径（P3，design/app-update-proposal.md §10）：调和命令携带 --refresh-manifest 时，
	// 先尝试经安全阀与每日节流刷新至更新源最新版本；刷新未执行（节流/已是最新/源不可达/安全阀拦截降级）
	// 时回退为按传入的本地清单快照正常调和——挂起中（FreezeSync）冻结代码同步；刷新中断（Apply 阶段失败）则如实失败留痕
	autoFreezeSync := false
	if refreshManifest {
		refreshed, freezeSync := tryAutoRefreshBeforeApply(targetPathOrURL)
		if refreshed {
			return
		}
		autoFreezeSync = freezeSync
	}

	opts := app.ApplyOptions{
		ScenarioID:        scenario,
		SkipSetup:         skipSetup,
		SkipSync:          skipSync || autoFreezeSync,
		ForceSetup:        forceSetup,
		OverwriteEnv:      overwriteEnv,
		OverwriteTask:     overwriteTaskPtr,
		SkipAppLock:       noLock,
		SavedManifestPath: sourcePath,
		LogWriter:         os.Stdout,
	}

	// 挂起中冻结代码同步：代码停旧版本、旧契约继续生效，补值或上游回退后次日自动刷新成功即自愈
	if autoFreezeSync && !skipSync {
		fmt.Fprintln(os.Stdout, "[提示] 更新挂起中（新增必填环境变量缺值待补值）：本次调和跳过代码同步，应用停留旧版本")
	}

	res, err := app.DefaultAppService.ApplyApp(targetPathOrURL, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[失败] 部署应用失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n[成功] 应用 '%s' 已就绪 (激活场景: %s)\n", res.AppName, res.ActiveScenario)
}

// tryAutoRefreshBeforeApply 自动更新前置刷新（仅供调和路径内部调用）：按本地快照清单 ID 定位已装应用，
// 复用服务端 UpdateApp 编排（配置保留 + 409 安全阀 + 每日节流）。返回 refreshed=true 表示刷新已生效（无需再调和）；
// 返回 refreshed=false 表示未刷新（应回退为按本地快照正常调和），freezeSync=true 表示闸门/安全阀挂起中，
// 回退调和必须跳过代码同步（代码停旧版本，避免「代码新、清单旧」的半更新状态）
func tryAutoRefreshBeforeApply(localYmlPath string) (refreshed bool, freezeSync bool) {
	manifest, err := app.ParseManifestFromFile(localYmlPath)
	if err != nil || strings.TrimSpace(manifest.ID) == "" {
		fmt.Fprintf(os.Stderr, "[警告] 自动刷新前置解析失败，回退为按当前清单调和: %v\n", err)
		return false, false
	}

	res, err := app.DefaultAppService.UpdateApp(manifest.ID, app.AppUpdateOptions{
		Auto:     true,
		SkipLock: true, // 调和路径父进程已持有 per-app 锁，子进程跳过加锁避免互等自锁
	}, os.Stdout)
	if err == nil && res != nil {
		fmt.Printf("\n[成功] 应用 '%s' 已自动更新至 v%s（用户配置与定制项已保留）\n", res.AppName, res.Version)
		return true, false
	}

	var skip *app.AppAutoRefreshSkipError
	if errors.As(err, &skip) {
		fmt.Printf("\n[跳过] 自动刷新未执行: %s\n", skip.Reason)
		return false, skip.FreezeSync
	}

	// 非 Skip 类错误 = 更新真实中断（如依赖安装失败），如实以失败退出留痕，不回退调和掩盖问题
	fmt.Fprintf(os.Stderr, "\n[失败] 自动更新中断: %v\n", err)
	os.Exit(1)
	return true, false
}

// runList 查看已安装应用清单；--updates 时检测更新源并仅显示可更新项
func runList(args []string) {
	fs := flag.NewFlagSet("app list", flag.ExitOnError)
	var showUpdates bool
	fs.BoolVar(&showUpdates, "updates", false, "检测更新源并仅显示可更新的应用")

	if err := fs.Parse(normalizeArgs(args)); err != nil {
		return
	}

	apps, err := app.DefaultAppService.ListApps()
	if err != nil {
		fmt.Fprintf(os.Stderr, "查询应用列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(apps) == 0 {
		fmt.Println("当前暂无已安装的应用。")
		fmt.Println("提示: 可使用 'baihu app apply <path-to-app.yaml>' 安装应用。")
		return
	}

	updates := make(map[string]app.AppUpdateInfo)
	if showUpdates {
		infos, err := app.DefaultAppService.CheckUpdates()
		if err != nil {
			fmt.Fprintf(os.Stderr, "检测应用更新失败: %v\n", err)
			os.Exit(1)
		}
		for _, info := range infos {
			if info.Updatable {
				updates[info.TaskID] = info
			}
		}
		if len(updates) == 0 {
			fmt.Println("所有已安装应用均为最新版本。")
			return
		}
	}

	if showUpdates {
		table := clibase.NewTable("应用标识 (ID)", "应用名称", "当前版本", "可更新至", "更新原因", "来源")
		for _, a := range apps {
			info, ok := updates[a.ID]
			if !ok {
				continue
			}
			reason := "版本变更"
			if info.Reason == "manifest_changed" {
				reason = "定义变更"
			}
			table.AddRow(a.ID, a.Name, info.InstalledVersion, info.LatestVersion, reason, info.UpdateSource)
		}
		table.Render()
	} else {
		table := clibase.NewTable("应用标识 (ID)", "应用名称", "版本", "分类", "生效场景", "安装时间")
		for _, a := range apps {
			table.AddRow(a.ID, a.Name, a.Version, a.Category, a.CurrentScenario, a.CreatedAt.Time().Format("2006-01-02 15:04"))
		}
		table.Render()
	}
}

func runInfo(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 必须提供应用 ID")
		fmt.Fprintln(os.Stderr, "用法: baihu app info <app-id>")
		os.Exit(1)
	}

	appID := args[0]
	appEntity, err := app.DefaultAppService.GetApp(appID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取应用信息失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("应用 ID:        %s\n", appEntity.ID)
	fmt.Printf("应用名称:       %s\n", appEntity.Name)
	fmt.Printf("当前版本:       %s\n", appEntity.Version)
	fmt.Printf("应用作者:       %s\n", appEntity.Author)
	fmt.Printf("所属分类:       %s\n", appEntity.Category)
	fmt.Printf("应用状态:       %s\n", appEntity.Status)
	fmt.Printf("当前场景:       %s\n", appEntity.CurrentScenario)
	if appEntity.Homepage != "" {
		fmt.Printf("项目主页:       %s\n", appEntity.Homepage)
	}
	if appEntity.Previous != nil {
		fmt.Printf("可回滚版本:     %s (更新前快照)\n", appEntity.Previous.Version)
	}
	fmt.Printf("应用简介:       %s\n", appEntity.Description)
	fmt.Printf("来源文件:       %s\n", appEntity.ManifestPath)
	fmt.Printf("安装时间:       %s\n", appEntity.CreatedAt.Time().Format("2006-01-02 15:04:05"))
	fmt.Printf("更新时间:       %s\n", appEntity.UpdatedAt.Time().Format("2006-01-02 15:04:05"))
}

// repeatFlags 可重复出现的字符串 flag（如 --env KEY=VALUE --env OTHER=x）
type repeatFlags []string

func (r *repeatFlags) String() string { return strings.Join(*r, ",") }
func (r *repeatFlags) Set(v string) error {
	*r = append(*r, v)
	return nil
}

// runUpdate 一键更新指定应用到更新源最新版本（复用服务端配置保留编排，与 Web/MCP 行为一致）
func runUpdate(args []string) {
	fs := flag.NewFlagSet("app update", flag.ExitOnError)
	var skipSetup bool
	var forceSetup bool
	var envFlags repeatFlags
	fs.BoolVar(&skipSetup, "skip-setup", false, "跳过环境与依赖安装（默认继承应用已保存构建选项）")
	fs.BoolVar(&forceSetup, "force-setup", false, "强制重新执行依赖安装与编译")
	fs.Var(&envFlags, "env", "409 安全阀补值，格式 KEY=VALUE，可重复传入")

	if err := fs.Parse(normalizeArgs(args)); err != nil {
		return
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 必须提供应用 ID")
		fmt.Fprintln(os.Stderr, "用法: baihu app update <app-id> [--env KEY=VALUE] [--force-setup]")
		os.Exit(1)
	}

	appID := parsedArgs[0]
	envValues := make(map[string]string)
	for _, pair := range envFlags {
		k, v, found := strings.Cut(pair, "=")
		k = strings.TrimSpace(k)
		if !found || k == "" {
			fmt.Fprintf(os.Stderr, "错误: --env 参数格式应为 KEY=VALUE（收到 %q）\n", pair)
			os.Exit(1)
		}
		envValues[k] = v
	}

	res, err := app.DefaultAppService.UpdateApp(appID, app.AppUpdateOptions{
		EnvValues:  envValues,
		SkipSetup:  skipSetup,
		ForceSetup: forceSetup,
	}, os.Stdout)
	if err != nil {
		var conflict *app.AppUpdateConflictError
		if errors.As(err, &conflict) {
			fmt.Fprintf(os.Stderr, "\n[拦截] 更新被安全阀暂停，新版本存在缺失值的必填环境变量:\n")
			for _, m := range conflict.MissingEnvs {
				fmt.Fprintf(os.Stderr, "  - %s (%s)\n", m.Key, m.Label)
			}
			fmt.Fprintf(os.Stderr, "请补值后重放: baihu app update %s --env KEY=VALUE ...\n", appID)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\n[失败] 更新应用失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n[成功] 应用 '%s' 已更新至 v%s（用户配置与定制项已保留）\n", res.AppName, res.Version)
}

// runCheckUpdates 检测全部已装应用的可用更新；--notify 时复用系统级每日检测落通知中心并推送渠道
// （面板内置 SysCron 每日调度与外部 cron 用户共用同一编排）
func runCheckUpdates(args []string) {
	fs := flag.NewFlagSet("app check-updates", flag.ExitOnError)
	var notify bool
	fs.BoolVar(&notify, "notify", false, "检测到新可更新版本时落通知中心未读提示并推送通知渠道")

	if err := fs.Parse(normalizeArgs(args)); err != nil {
		return
	}

	infos, err := app.DefaultAppService.CheckUpdates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "检测应用更新失败: %v\n", err)
		os.Exit(1)
	}

	updatable := 0
	table := clibase.NewTable("应用标识 (ID)", "应用名称", "当前版本", "最新版本", "状态", "来源")
	for _, info := range infos {
		status := "已是最新"
		switch {
		case info.Updatable && info.Reason == "manifest_changed":
			status = "定义有更新"
			updatable++
		case info.Updatable:
			status = "可更新"
			updatable++
		case info.Reason == "source_mismatch":
			status = "更新源归属异常"
		case info.Reason == "fetch_failed":
			status = "检测失败"
		case info.Reason == "no_source":
			status = "无更新源"
		}
		table.AddRow(info.TaskID, info.AppName, info.InstalledVersion, info.LatestVersion, status, info.UpdateSource)
	}
	table.Render()

	if updatable == 0 {
		fmt.Println("\n所有已安装应用均为最新版本。")
		return
	}

	if notify {
		notified := services.NotifyAppUpdateResults(infos, nil, nil, nil)
		if notified > 0 {
			fmt.Printf("\n检测到 %d 个应用有新版本，已落通知中心并推送绑定渠道。\n", notified)
		} else {
			fmt.Printf("\n检测到 %d 个应用可更新，但均已在近期提醒过，本次跳过重复推送。\n", updatable)
		}
	} else {
		fmt.Printf("\n检测到 %d 个应用可更新（--notify 可同时通知）。\n", updatable)
	}
}

// runAutoUpdate 开/关指定应用的自动更新（与 Web 更新抽屉、MCP set_app_auto_update 三端一致）
func runAutoUpdate(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "错误: 必须提供应用 ID 以及目标状态 (on/off)\n")
		fmt.Fprintf(os.Stderr, "用法: baihu app auto-update <app-id> [on|off]\n")
		os.Exit(1)
	}
	appID := args[0]
	enabled := true
	if len(args) >= 2 {
		switch strings.ToLower(strings.TrimSpace(args[1])) {
		case "on", "true", "1":
			enabled = true
		case "off", "false", "0":
			enabled = false
		default:
			fmt.Fprintf(os.Stderr, "错误: 未知状态 %q，请使用 on 或 off\n", args[1])
			os.Exit(1)
		}
	}

	dto, err := app.DefaultAppService.SetAppAutoUpdate(appID, enabled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "设置自动更新失败: %v\n", err)
		os.Exit(1)
	}
	state := "已开启"
	if !enabled {
		state = "已关闭"
	}
	fmt.Printf("[成功] 应用 '%s' 自动更新%s（当前版本 v%s）\n", dto.Name, state, dto.Version)
}

func runSwitch(args []string) {
	fs := flag.NewFlagSet("app switch", flag.ExitOnError)
	var scenario string
	fs.StringVar(&scenario, "scenario", "", "目标场景预设 ID (必填)")

	normArgs := normalizeArgs(args)
	if err := fs.Parse(normArgs); err != nil {
		return
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) == 0 || scenario == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须提供应用 ID 以及 --scenario 参数")
		fmt.Fprintln(os.Stderr, "用法: baihu app switch <app-id> --scenario <scenario-id>")
		os.Exit(1)
	}

	appID := parsedArgs[0]
	if err := app.DefaultAppService.SwitchScenario(appID, scenario, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "场景切换失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n[成功] 应用 '%s' 已切换至场景 '%s'\n", appID, scenario)
}

func runRemove(args []string) {
	fs := flag.NewFlagSet("app remove", flag.ExitOnError)
	var cleanData bool
	var cleanEnvs bool
	fs.BoolVar(&cleanData, "clean-data", true, "同时删除应用所占用的数据及代码目录")
	fs.BoolVar(&cleanEnvs, "clean-envs", true, "同时删除应用关联的环境变量与凭证")

	normArgs := normalizeArgs(args)
	if err := fs.Parse(normArgs); err != nil {
		return
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 必须提供要卸载的应用 ID")
		fmt.Fprintln(os.Stderr, "用法: baihu app remove <app-id> [--clean-data] [--clean-envs=false]")
		os.Exit(1)
	}

	appID := parsedArgs[0]
	if _, err := app.DefaultAppService.RemoveAppWithOptions(appID, app.AppRemoveOptions{
		CleanData: cleanData,
		CleanEnvs: cleanEnvs,
	}, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "卸载失败: %v\n", err)
		os.Exit(1)
	}
}

// normalizeArgs 将命令行中任意位置的 flags 提前，保证 flag.FlagSet 能完整解析
func normalizeArgs(args []string) []string {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 0 && arg[0] == '-' {
			flags = append(flags, arg)
			// 如果是非布尔选项 (如 --scenario val / --overwrite-task val / --env KEY=VALUE / --source-path val)，连同下一个值一起存入 flags
			isValuedFlag := arg == "--scenario" || arg == "-scenario" || arg == "--overwrite-task" || arg == "-overwrite-task" || arg == "--env" || arg == "-env" || arg == "--source-path" || arg == "-source-path"
			if isValuedFlag && !strings.Contains(arg, "=") && i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return append(flags, positional...)
}
