package webui

import (
	"fmt"
	"os"
	"strings"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/services"
)

func printMainHelp() {
	fmt.Fprintf(os.Stderr, "\n白虎面板 WebUI 命令行管理工具\n\n")
	fmt.Fprintf(os.Stderr, "用法:\n")
	fmt.Fprintf(os.Stderr, "  baihu webui <子命令> [参数]\n\n")
	fmt.Fprintf(os.Stderr, "可用子命令:\n")
	fmt.Fprintf(os.Stderr, "  list       列出当前安装的所有前端资源包\n")
	fmt.Fprintf(os.Stderr, "  set        设置激活指定的 WebUI\n")
	fmt.Fprintf(os.Stderr, "  reset      一键回退到系统默认的内置 WebUI\n")
	fmt.Fprintf(os.Stderr, "  delete     删除指定的 WebUI 资源包\n\n")
}

func Run(args []string) {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printMainHelp()
		return
	}

	subCommand := args[0]
	subArgs := args[1:]

	switch subCommand {
	case "list":
		runList(subArgs)
	case "set":
		runSet(subArgs)
	case "reset":
		runReset(subArgs)
	case "delete":
		runDelete(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", subCommand)
		printMainHelp()
	}
}

func initServices() *services.WebUIService {
	clibase.InitContext(false)
	settingsService := services.NewSettingsService()
	return services.NewWebUIService(settingsService)
}

func runList(args []string) {
	svc := initServices()
	list, err := svc.GetWebUIs()
	if err != nil {
		fmt.Printf(">> 获取WebUI列表失败: %v\n", err)
		return
	}

	settingsService := services.NewSettingsService()
	activeWebUI := settingsService.Get("site", "active_webui")
	if activeWebUI == "" {
		activeWebUI = "default"
	}

	fmt.Println(strings.Repeat("=", 100))
	fmt.Printf("%s | %s | %s | %s | %s\n",
		clibase.VisualFormat("名称", 20),
		clibase.VisualFormat("版本", 12),
		clibase.VisualFormat("作者", 15),
		clibase.VisualFormat("状态", 10),
		clibase.VisualFormat("描述", 30),
	)
	fmt.Println(strings.Repeat("-", 100))
	
	for _, w := range list {
		status := "-"
		if w.Name == activeWebUI {
			status = "使用中"
		}
		
		fmt.Printf("%s | %s | %s | %s | %s\n",
			clibase.VisualFormat(w.Name, 20),
			clibase.VisualFormat(w.Version, 12),
			clibase.VisualFormat(w.Author, 15),
			clibase.VisualFormat(status, 10),
			clibase.VisualFormat(w.Description, 30),
		)
	}
	fmt.Println(strings.Repeat("=", 100))
}

func runSet(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "错误: 缺少目标 WebUI 名称。\n用法: baihu webui set <name>\n")
		return
	}
	name := args[0]
	svc := initServices()
	err := svc.SetActiveWebUI(name)
	if err != nil {
		fmt.Printf(">> 设置激活WebUI失败: %v\n", err)
		return
	}
	fmt.Printf(">> 成功激活 WebUI: %s\n", name)
}

func runReset(args []string) {
	svc := initServices()
	err := svc.SetActiveWebUI("default")
	if err != nil {
		fmt.Printf(">> 回退默认WebUI失败: %v\n", err)
		return
	}
	fmt.Println(">> 成功回退到内置默认 WebUI")
}

func runDelete(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "错误: 缺少目标 WebUI 名称。\n用法: baihu webui delete <name>\n")
		return
	}
	name := args[0]
	svc := initServices()
	err := svc.DeleteWebUI(name)
	if err != nil {
		fmt.Printf(">> 删除WebUI失败: %v\n", err)
		return
	}
	fmt.Printf(">> 成功删除 WebUI: %s\n", name)
}
