package app

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/engigu/baihu-panel/internal/services/app"
)

func Run(args []string) {
	printHelp := func() {
		fmt.Fprintf(os.Stderr, "\n白虎面板应用管理工具 (App Engine)\n\n")
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  baihu app <子命令> [参数]\n\n")
		fmt.Fprintf(os.Stderr, "可用子命令:\n")
		fmt.Fprintf(os.Stderr, "  apply   解析并应用本地或远程 app.yaml 规范文件\n")
		fmt.Fprintf(os.Stderr, "  list    查看当前已安装的应用清单\n")
		fmt.Fprintf(os.Stderr, "  info    查看指定已安装应用的详细信息与任务列表\n")
		fmt.Fprintf(os.Stderr, "  switch  一键切换指定应用的运行场景预设\n")
		fmt.Fprintf(os.Stderr, "  remove  卸载指定应用并清理关联任务\n\n")
		fmt.Fprintf(os.Stderr, "示例:\n")
		fmt.Fprintf(os.Stderr, "  baihu app apply ./apps/BiliBiliToolPro/app.yaml\n")
		fmt.Fprintf(os.Stderr, "  baihu app apply https://raw.githubusercontent.com/xxx/app.yaml --scenario minimal\n")
		fmt.Fprintf(os.Stderr, "  baihu app list\n")
		fmt.Fprintf(os.Stderr, "  baihu app switch bilibili-tool-pro --scenario hardcore\n")
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

	fs.StringVar(&scenario, "scenario", "", "指定激活的应用场景模板 ID")
	fs.BoolVar(&skipSetup, "skip-setup", false, "跳过前置环境检测与依赖安装步骤")
	fs.BoolVar(&skipSync, "skip-sync", false, "跳过代码源同步步骤")
	fs.BoolVar(&forceSetup, "force-setup", false, "强制重新执行依赖安装与编译（跳过 check 探活）")
	fs.BoolVar(&overwriteEnv, "overwrite-env", false, "是否覆盖已有同名环境变量")
	fs.StringVar(&overwriteTaskStr, "overwrite-task", "", "是否覆盖并同步更新受控任务 (true/false)")

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

	opts := app.ApplyOptions{
		ScenarioID:    scenario,
		SkipSetup:     skipSetup,
		SkipSync:      skipSync,
		ForceSetup:    forceSetup,
		OverwriteEnv:  overwriteEnv,
		OverwriteTask: overwriteTaskPtr,
		LogWriter:     os.Stdout,
	}

	res, err := app.DefaultAppService.ApplyApp(targetPathOrURL, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[失败] 部署应用失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n[成功] 应用 '%s' 已就绪 (激活场景: %s)\n", res.AppName, res.ActiveScenario)
}

func runList(args []string) {
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "应用标识 (ID)\t应用名称\t版本\t分类\t生效场景\t安装时间")
	fmt.Fprintln(w, "-------------\t--------\t----\t----\t--------\t--------")
	for _, a := range apps {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			a.ID, a.Name, a.Version, a.Category, a.CurrentScenario, a.CreatedAt.Time().Format("2006-01-02 15:04"))
	}
	w.Flush()
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
	fmt.Printf("应用简介:       %s\n", appEntity.Description)
	fmt.Printf("来源文件:       %s\n", appEntity.ManifestPath)
	fmt.Printf("安装时间:       %s\n", appEntity.CreatedAt.Time().Format("2006-01-02 15:04:05"))
	fmt.Printf("更新时间:       %s\n", appEntity.UpdatedAt.Time().Format("2006-01-02 15:04:05"))
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
	fs.BoolVar(&cleanData, "clean-data", false, "同时删除应用所占用的数据及代码目录")

	normArgs := normalizeArgs(args)
	if err := fs.Parse(normArgs); err != nil {
		return
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) == 0 {
		fmt.Fprintln(os.Stderr, "错误: 必须提供要卸载的应用 ID")
		fmt.Fprintln(os.Stderr, "用法: baihu app remove <app-id> [--clean-data]")
		os.Exit(1)
	}

	appID := parsedArgs[0]
	if err := app.DefaultAppService.RemoveApp(appID, cleanData, os.Stdout); err != nil {
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
			// 如果是非布尔选项 (如 --scenario val 或 --overwrite-task val)，连同下一个值一起存入 flags
			if (arg == "--scenario" || arg == "-scenario" || arg == "--overwrite-task" || arg == "-overwrite-task") && !strings.Contains(arg, "=") && i+1 < len(args) && len(args[i+1]) > 0 && args[i+1][0] != '-' {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}
	return append(flags, positional...)
}
