package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Run 启动基于 Bubble Tea 驱动的终端交互式控制台
func Run(args []string) {
	// 1. 严格拦截嵌套调用 tui
	if len(args) > 0 {
		target := strings.ToLower(args[0])
		if target == "tui" {
			fmt.Fprintln(os.Stderr, ">> 提示: 不能在 TUI 中嵌套调用 tui 指令。")
			return
		}
		if target == "-h" || target == "--help" || target == "help" {
			printHelp()
			return
		}

		// 2. 携带有效子模块参数时（如 `baihu tui task`），直接进入对应模块
		if spec := FindCommandSpec(target); spec != nil {
			m := NewModel()
			_, initCmd := m.selectCommand(spec)
			m.initialCmd = initCmd
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				fmt.Fprintf(os.Stderr, ">> TUI 运行异常: %v\n", err)
			}
			return
		}

		fmt.Fprintf(os.Stderr, ">> 未知子命令: %s\n\n", target)
		printHelp()
		return
	}

	// 启动全功能主控制台
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, ">> TUI 运行异常: %v\n", err)
	}
}

func printHelp() {
	fmt.Println("\n白虎面板交互式终端管理控制台 (Bubble Tea TUI)")
	fmt.Println("用法:")
	fmt.Println("  baihu tui [子模块名称]")
	fmt.Println("示例:")
	fmt.Println("  baihu tui          # 启动全功能动态交互式主控制台")
	fmt.Println("  baihu tui task     # 直接动态进入任务管理菜单")
	fmt.Println("  baihu tui app      # 直接动态进入应用中心菜单")
	fmt.Println("  baihu tui webui    # 直接动态进入前端资源菜单")
	fmt.Println()
}
