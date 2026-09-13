package clibase

import (
	"flag"
	"fmt"
	"os"

	"github.com/engigu/baihu-panel/internal/bootstrap"
	"github.com/engigu/baihu-panel/internal/services"
)

// InitContext 统一封装命令行所需的初始化上下文逻辑
func InitContext(requireSettings bool) error {
	bootstrap.InitBasicForCmd()
	if requireSettings {
		settingsService := services.NewSettingsService()
		if err := settingsService.InitSettings(); err != nil {
			return fmt.Errorf("初始化系统设置失败: %w", err)
		}
	}
	return nil
}

// PrintDBConfigHint 打印标准化的连接或检索失败时的排查指引
func PrintDBConfigHint(commandExample string) {
	fmt.Println(">> 提示: 程序当前可能连接到了默认的空 SQLite 数据库。")
	fmt.Println(">> 若您的生产环境使用的是 MySQL 或指定路径配置，请在执行命令时携带配置文件路径环境变量，例如:")
	fmt.Printf(">> BH_CONFIG_PATH=/app/data/config.ini baihu %s\n", commandExample)
}

// PrintSubCommandUsage 打印一致风格的子程序帮助信息
func PrintSubCommandUsage(title, usageStr, exampleStr string, fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "\n%s\n\n", title)
	fmt.Fprintf(os.Stderr, "用法:\n")
	fmt.Fprintf(os.Stderr, "  %s\n\n", usageStr)
	if fs != nil {
		fmt.Fprintf(os.Stderr, "参数说明:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}
	if exampleStr != "" {
		fmt.Fprintf(os.Stderr, "示例:\n")
		fmt.Fprintf(os.Stderr, "%s\n\n", exampleStr)
	}
}

// VisualFormat 根据字符的视觉显示列宽（基于 go-runewidth，支持中文字符、Emoji、宽字符），
// 将字符串进行精确等宽填充或安全截断追加 ".."，确保终端表格强制对齐。
func VisualFormat(s string, targetVisualWidth int) string {
	return FormatCell(s, targetVisualWidth, AlignLeft)
}

