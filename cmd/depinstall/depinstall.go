package depinstall

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/database"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/services/deps"
	"github.com/engigu/baihu-panel/internal/utils"
)

// Run 依赖自动补全命令入口
func Run(args []string) {
	if len(args) == 0 {
		fmt.Println("用法: baihu depinstall <log_id>")
		return
	}

	logID := args[0]

	// 初始化基础环境和数据库连接
	if err := clibase.InitContext(true); err != nil {
		fmt.Printf(">> 初始化环境失败: %v\n", err)
		return
	}

	var log models.TaskLog
	if err := database.DB.Where("id = ?", logID).First(&log).Error; err != nil {
		fmt.Printf(">> 未找到指定的任务日志 (ID: %s): %v\n", logID, err)
		return
	}

	var task models.Task
	if err := database.DB.Where("id = ?", log.TaskID).First(&task).Error; err != nil {
		fmt.Printf(">> 未找到对应的任务 (TaskID: %s): %v\n", log.TaskID, err)
		return
	}

	logOutput, err := utils.DecompressFromBase64(string(log.Output))
	if err != nil {
		fmt.Printf(">> 解压日志失败: %v\n", err)
		return
	}

	// 找出任务配置的语言
	taskLangs := task.GetLanguages()
	if len(taskLangs) == 0 {
		fmt.Println(">> 提示: 当前任务未配置具体语言环境，请手动指定语言类型（例如 python3, node 等）:")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Println(">> 已取消补全。")
			return
		}
		taskLangs = append(taskLangs, map[string]string{
			"name":    input,
			"version": "",
		})
	}

	var allDetected []string
	langToPkgMap := make(map[string][]string)

	for _, langMap := range taskLangs {
		langName := langMap["name"]
		if langName == "" {
			continue
		}
		detected, found := deps.DetectMissingDependencies(langName, logOutput)
		if found {
			langToPkgMap[langName] = detected
			allDetected = append(allDetected, detected...)
		}
	}

	reader := bufio.NewReader(os.Stdin)

	// 如果没有检测到任何缺失的包，允许用户手动输入
	if len(allDetected) == 0 {
		fmt.Println(">> 分析完毕: 未从最近一次的任务运行日志中检测到缺失依赖模式。")
		fmt.Println(">> 您可以手动输入想要安装的依赖包名称（多个包用空格分隔，若不安装请直接回车退出）:")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Println(">> 已退出依赖补全。")
			return
		}
		// 默认分配到任务的第一个语言环境
		defaultLang := taskLangs[0]["name"]
		langToPkgMap[defaultLang] = strings.Fields(input)
		allDetected = append(allDetected, langToPkgMap[defaultLang]...)
	} else {
		fmt.Println(">> 分析结果: 从运行日志中检测到以下缺失依赖包：")
		for langName, pkgs := range langToPkgMap {
			fmt.Printf("   [%s]: %s\n", langName, strings.Join(pkgs, ", "))
		}
		fmt.Println(">> 是否确认自动安装上述依赖包？(y/N):")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm != "y" && confirm != "yes" {
			fmt.Println(">> 用户已取消安装操作。")
			return
		}
	}

	fmt.Println("==================================================================")
	fmt.Println(">> 开始执行依赖安装，请稍候...")
	fmt.Println("==================================================================")

	var failedPkgs []string
	depService := services.NewDependencyService()

	for langName, pkgs := range langToPkgMap {
		var langVersion string
		for _, lm := range taskLangs {
			if lm["name"] == langName {
				langVersion = lm["version"]
				break
			}
		}

		m := deps.GetManager(langName)
		if m == nil {
			fmt.Printf(">> 错误: 不支持的语言类型: %s\n", langName)
			failedPkgs = append(failedPkgs, pkgs...)
			continue
		}

		for _, pkg := range pkgs {
			dep := &models.Dependency{
				Name:        pkg,
				Language:    langName,
				LangVersion: langVersion,
			}

			cmdStr, err := m.GetInstallCommand(dep)
			if err != nil {
				fmt.Printf(">> 无法生成 %s 包 [%s] 的安装命令: %v\n", langName, pkg, err)
				failedPkgs = append(failedPkgs, pkg)
				continue
			}

			// 去除命令末尾的 success/failed echo 重定向，因为我们需要捕获退出状态并在控制台展示原始流程
			if idx := strings.Index(cmdStr, " && echo"); idx != -1 {
				cmdStr = cmdStr[:idx]
			}

			fmt.Printf(">> 正在安装 [%s] -> 执行指令: %s\n", pkg, cmdStr)

			execCmd := utils.NewShellCommandCmd(cmdStr)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin

			runErr := execCmd.Run()
			if runErr != nil {
				fmt.Printf(">> 【失败】依赖包 [%s] 安装出错。\n\n", pkg)
				failedPkgs = append(failedPkgs, pkg)
			} else {
				fmt.Printf(">> 【成功】依赖包 [%s] 安装成功！\n\n", pkg)
				// 成功后记录到依赖表
				_ = depService.Create(dep)
			}
		}
	}

	fmt.Println("==================================================================")
	if len(failedPkgs) > 0 {
		fmt.Printf(">> 依赖补全已结束。其中以下依赖包安装失败，请用户自行判断/手动处理：\n")
		for _, fp := range failedPkgs {
			fmt.Printf("   - %s\n", fp)
		}
	} else {
		fmt.Println(">> 恭喜！所有依赖包安装成功！")
	}
	fmt.Println("==================================================================")
}
