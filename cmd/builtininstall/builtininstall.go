package builtininstall

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/utils"
)

func printHelp() {
	clibase.PrintSubCommandUsage("白虎面板内建依赖安装工具", "baihu builtininstall", "", nil)
}

// Run 执行内建包安装逻辑
func Run(args []string) {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printHelp()
		return
	}

	fs := flag.NewFlagSet("builtininstall", flag.ExitOnError)
	fs.Usage = printHelp

	if err := fs.Parse(args); err != nil {
		return
	}

	fmt.Println(">> [Builtin] 开始为 mise 环境安装内建包...")

	// 1. 确定内建包路径
	// 优先使用 /www/builtin (Docker 环境)，否则尝试相对于二进制文件的当前目录
	builtinPath := "/www/builtin"
	if _, err := os.Stat(builtinPath); os.IsNotExist(err) {
		// 回退到当前目录下的 builtin
		pwd, _ := os.Getwd()
		builtinPath = filepath.Join(pwd, "builtin")
	}

	if _, err := os.Stat(builtinPath); os.IsNotExist(err) {
		fmt.Printf(">> [Builtin] 错误: 找不到内建包目录: %s\n", builtinPath)
		return
	}

	// 2. 安装 Node.js 包
	installForLanguage("node", filepath.Join(builtinPath, "nodejs"))

	// 3. 安装 Python 包
	installForLanguage("python", filepath.Join(builtinPath, "python"))

	fmt.Println(">> [Builtin] 内建包安装流程完成")
}

func installForLanguage(lang, pkgPath string) {
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		fmt.Printf(">> [Builtin] 警告: %s 的内建包目录不存在: %s\n", lang, pkgPath)
		return
	}

	versions, err := utils.ListMiseInstalledVersions(lang)
	if err != nil {
		fmt.Printf(">> [Builtin] 错误: 获取 %s 的 mise 版本列表失败: %v\n", lang, err)
		return
	}

	if len(versions) == 0 {
		fmt.Printf(">> [Builtin] 未发现已安装的 %s 版本，跳过\n", lang)
		return
	}

	for _, v := range versions {
		fmt.Printf(">> [Builtin] 正在为 %s@%s 安装内建包...\n", lang, v)

		// 尝试直接获取该版本的物理安装路径，直接调用 bin/npm 或 bin/pip
		var binCmd string
		var subCmdArgs []string

		cmdWhere := exec.Command("mise", "where", lang+"@"+v)
		whereOut, whereErr := cmdWhere.CombinedOutput()
		installDir := strings.TrimSpace(string(whereOut))

		if whereErr == nil && installDir != "" {
			if lang == "node" {
				candidate := filepath.Join(installDir, "bin", "npm")
				if runtime.GOOS == "windows" {
					candidate = filepath.Join(installDir, "npm.cmd")
				}
				if _, err := os.Stat(candidate); err == nil {
					binCmd = candidate
				}
			} else if lang == "python" {
				candidate := filepath.Join(installDir, "bin", "pip")
				if runtime.GOOS == "windows" {
					candidate = filepath.Join(installDir, "Scripts", "pip.exe")
				}
				if _, err := os.Stat(candidate); err == nil {
					binCmd = candidate
				}
			}
		}

		var cmd *exec.Cmd
		if binCmd != "" {
			// 直接使用版本内部的包管理器，完全绕过全局 shims 和环境变量缺失问题
			if lang == "node" {
				subCmdArgs = []string{"i", "-g", pkgPath}
			} else {
				subCmdArgs = []string{"install", "--force-reinstall", pkgPath}
			}
			cmd = exec.Command(binCmd, subCmdArgs...)
		} else {
			// 回退到 mise exec 模式
			if lang == "node" {
				subCmdArgs = []string{"npm", "i", "-g", pkgPath}
			} else {
				subCmdArgs = []string{"pip", "install", "--force-reinstall", pkgPath}
			}
			fullArgs := utils.BuildMiseCommandArgsSimple(subCmdArgs, lang, v)
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", append([]string{"/c"}, fullArgs...)...)
			} else {
				cmd = exec.Command(fullArgs[0], fullArgs[1:]...)
			}
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf(">> [Builtin] 错误: 为 %s@%s 安装失败: %v\n输出: %s\n", lang, v, err, string(out))
		} else {
			fmt.Printf(">> [Builtin] 为 %s@%s 安装成功\n", lang, v)
		}
	}
}
