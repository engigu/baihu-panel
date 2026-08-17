package restore

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/services"
)

func printHelp() {
	clibase.PrintSubCommandUsage("白虎面板系统数据恢复工具", "baihu restore <备份文件.zip>", "  baihu restore backup_20231027.zip", nil)
}

func Run(args []string) {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printHelp()
		return
	}

	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	fs.Usage = printHelp

	if err := fs.Parse(args); err != nil {
		return
	}

	parsedArgs := fs.Args()
	if len(parsedArgs) < 1 {
		fmt.Fprintf(os.Stderr, "错误: 必须提供备份文件路径\n")
		fs.Usage()
		return
	}

	backupFile := parsedArgs[0]
	absPath, err := filepath.Abs(backupFile)
	if err != nil {
		fmt.Printf("文件路径解析失败: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Printf("错误: 备份文件 '%s' 不存在\n", absPath)
		os.Exit(1)
	}

	// 必须初始化环境与数据库才能恢复数据
	clibase.InitContext(false)

	backupService := services.NewBackupService()
	fmt.Printf("正在从 '%s' 恢复系统数据，请勿强制中断...\n", absPath)
	err = backupService.Restore(absPath)
	if err != nil {
		fmt.Printf("恢复备份失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--------------------------------------------------")
	fmt.Println("系统备份恢复成功！")
	fmt.Println("注意：部分设定可能需要重启后台服务后才能完全生效。")
	fmt.Println("--------------------------------------------------")
}
