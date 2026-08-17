package cmd

import (
	"fmt"
	"os"

	"github.com/engigu/baihu-panel/internal/bootstrap"
	"github.com/engigu/baihu-panel/internal/constant"
)

// CommandSpec 兼容引用 constant.CommandSpec
type CommandSpec = constant.CommandSpec

// CommandInfo 兼容引用 constant.CommandInfo
type CommandInfo = constant.CommandInfo

// Commands 兼容引用 constant.Commands
var Commands = constant.Commands

// CommandHandler 定义命令执行函数
type CommandHandler func(args []string)

// CommandRegistration 声明式维护命令处理句柄及其执行策略
type CommandRegistration struct {
	Handler        CommandHandler
	RequireContext bool // 是否需要初始化数据库/日志基础环境
}

// Handlers 维护全系统可用命令的执行入口
var Handlers = map[string]CommandRegistration{}

// RegisterHandler 供各个子命令注册处理函数（默认初始化基础环境）
func RegisterHandler(name string, handler CommandHandler) {
	RegisterHandlerWithConfig(name, handler, true)
}

// RegisterHandlerWithConfig 支持显式配置是否需要初始化基础环境
func RegisterHandlerWithConfig(name string, handler CommandHandler, requireContext bool) {
	Handlers[name] = CommandRegistration{
		Handler:        handler,
		RequireContext: requireContext,
	}
}

// PrintHelp 打印根命令帮助信息
func PrintHelp() {
	fmt.Println("\nBaihu Panel - 极致轻量、高性能的自动化任务调度平台")
	fmt.Println("用法:")
	fmt.Println("  baihu <命令> [参数]")
	fmt.Println("可用命令:")
	for _, info := range constant.Commands {
		fmt.Printf("  %-15s %s\n", info.Name, info.Description)
	}
	fmt.Println("\n使用 'baihu <命令> --help' 查看具体命令的参数说明。")
	fmt.Println()
}

// Execute 统一路由与分发所有命令行指令
func Execute(args []string) {
	if len(args) < 2 {
		PrintHelp()
		os.Exit(1)
	}

	commandName := args[1]

	reg, ok := Handlers[commandName]
	if !ok {
		fmt.Printf("Unknown command: %s\n", commandName)
		PrintHelp()
		os.Exit(1)
	}

	if reg.RequireContext {
		bootstrap.InitBasicForCmd() // 专为命令行工具定制启动基础环境，屏蔽后台启动刷屏日志
	}

	reg.Handler(args[2:])
}
