package main

import (
	"fmt"
	"os"

	"github.com/engigu/baihu-panel/cmd"
	"github.com/engigu/baihu-panel/internal/bootstrap"
	"github.com/engigu/baihu-panel/internal/constant"
)

// @title Baihu Panel API
// @version 1.0
// @description Baihu Panel OpenAPI Server documentation.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8052
// @BasePath /open2api/v1
// @query.collection.format multi
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the API token.

func printHelp() {
	fmt.Println("\n白虎面板 (Baihu Panel) - 现代化的服务器管理面板")
	fmt.Println("用法:")
	fmt.Println("  baihu <命令> [参数]")
	fmt.Println("可用命令:")
	for _, info := range constant.Commands {
		fmt.Printf("  %-15s %s\n", info.Name, info.Description)
	}
	fmt.Println("\n使用 'baihu <命令> --help' 查看具体命令的参数说明。")
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	commandName := os.Args[1]

	if commandName == "server" {
		bootstrap.New().Run()
		return
	}

	if handler, ok := cmd.Handlers[commandName]; ok {
		if commandName != "-v" && commandName != "-V" && commandName != "version" {
			bootstrap.InitBasicForCmd() // 专为命令行工具定制启动基础环境，屏蔽后台启动刷屏日志
		}
		handler(os.Args[2:])
		return
	}

	fmt.Printf("Unknown command: %s\n", commandName)
	printHelp()
	os.Exit(1)
}
