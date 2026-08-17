package clibase

import (
	"flag"
	"fmt"
	"os"
	"strings"

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

// VisualFormat 根据字符的视觉显示列宽（中文字符/宽字符计为2列，ASCII计为1列），
// 将字符串进行精确等宽填充或安全截断追加 ".."，确保混合字符输出下控制台表格严丝合缝强制对齐。
func VisualFormat(s string, targetVisualWidth int) string {
	w := 0
	var sb strings.Builder
	runes := []rune(s)

	// 先计算总视觉宽度
	totalW := 0
	for _, r := range runes {
		if r > 127 {
			totalW += 2
		} else {
			totalW += 1
		}
	}

	if totalW <= targetVisualWidth {
		return s + strings.Repeat(" ", targetVisualWidth-totalW)
	}

	// 如果总宽度超出，进行精准截断并追加 ".."
	maxContentW := targetVisualWidth - 2
	for _, r := range runes {
		rw := 1
		if r > 127 {
			rw = 2
		}
		if w+rw > maxContentW {
			break
		}
		sb.WriteRune(r)
		w += rw
	}

	res := sb.String() + ".."
	// 补齐末尾可能相差的1个空格列宽
	if w+2 < targetVisualWidth {
		res += strings.Repeat(" ", targetVisualWidth-(w+2))
	}
	return res
}
