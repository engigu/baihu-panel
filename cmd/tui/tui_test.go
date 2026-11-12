package tui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/engigu/baihu-panel/cmd"
	"github.com/engigu/baihu-panel/cmd/tui"
	"github.com/engigu/baihu-panel/internal/constant"
)

func TestTUI_Registered(t *testing.T) {
	cmd.InitHandlers()

	reg, ok := cmd.Handlers["tui"]
	if !ok {
		t.Fatalf("预期 'tui' 命令已在 cmd.Handlers 中注册，实际未找到")
	}
	if reg.Handler == nil {
		t.Fatalf("预期 'tui' Handler 非空")
	}

	foundInConst := false
	for _, c := range constant.Commands {
		if c.Name == "tui" {
			foundInConst = true
			break
		}
	}
	if !foundInConst {
		t.Fatalf("预期 'tui' 存在于 constant.Commands 列表中，实际未找到")
	}
}

func TestTUI_DynamicCommandsExcludeSelf(t *testing.T) {
	cmds := tui.GetDynamicCommands()
	if len(cmds) == 0 {
		t.Fatalf("动态命令列表不应为空")
	}

	for _, c := range cmds {
		if strings.EqualFold(c.Name, "tui") {
			t.Fatalf("动态命令列表严禁包含 'tui' 本身，发现嵌套项: %s", c.Name)
		}
	}

	if spec := tui.FindCommandSpec("task"); spec == nil || spec.Name != "task" {
		t.Fatalf("预期能查找到 'task' 命令规范，实际返回 nil")
	}
	if spec := tui.FindCommandSpec("TASK"); spec == nil || spec.Name != "task" {
		t.Fatalf("预期大小写不敏感查找 'TASK'，实际返回 nil")
	}
	if spec := tui.FindCommandSpec("tui"); spec != nil {
		t.Fatalf("预期查找 'tui' 返回 nil（禁止嵌套），实际返回非空")
	}
	if spec := tui.FindCommandSpec("exit"); spec == nil || spec.Name != "exit" {
		t.Fatalf("预期动态命令包含 'exit' 退出项，实际未找到")
	}
}

func TestTUI_ExtractUniqueFlags(t *testing.T) {
	rawFlags := []string{"--name", "--type", "--page", "--size", "-name", "-type", "-page", "-size"}
	unique := tui.ExtractUniqueFlags(rawFlags)

	if len(unique) != 4 {
		t.Fatalf("预期去重后得到 4 个选项，实际为 %d: %v", len(unique), unique)
	}

	for _, f := range unique {
		if !strings.HasPrefix(f, "--") {
			t.Errorf("预期优先保留 '--' 前缀形式，实际得到 %s", f)
		}
	}
}

func TestTUI_SortedSubcommandKeys(t *testing.T) {
	subcmds := map[string]string{
		"remove":  "卸载",
		"apply":   "应用",
		"list":    "列表",
		"switch":  "切换",
		"unknown": "其他",
	}

	keys := tui.GetSortedSubcommandKeys(subcmds)
	if len(keys) != len(subcmds) {
		t.Fatalf("预期子命令数量一致")
	}

	if keys[0] != "list" {
		t.Errorf("预期常用操作 'list' 排序在最前，实际首项为 %s", keys[0])
	}
}

func TestTUI_BubbleTeaModel(t *testing.T) {
	m := tui.NewModel()
	if cmd := m.Init(); cmd == nil {
		t.Fatalf("Init() 应当返回文本框光标闪烁 cmd")
	}

	// 验证初始渲染内容包含标题与日志面板
	view := m.View()
	if !strings.Contains(view, "Baihu Console TUI") {
		t.Fatalf("View() 输出应包含系统标题，实际输出: %s", view)
	}
	if !strings.Contains(view, "Bubble Tea") {
		t.Fatalf("View() 输出应包含 Bubble Tea 引擎标识")
	}
	if !strings.Contains(view, "运行与执行日志") {
		t.Fatalf("View() 输出应包含专门的日志区域标识")
	}

	// 验证下移光标
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newM.(*tui.Model)

	// 验证按 'c' 切换到自由命令行模式
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newM.(*tui.Model)
	inputView := m.View()
	if !strings.Contains(inputView, "自由命令行执行") {
		t.Fatalf("按 c 应当进入自由命令行模式，实际输出: %s", inputView)
	}

	// 验证自定义模式输入包含 tui 时被拦截
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("tui task list")})
	m = newM.(*tui.Model)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(*tui.Model)
	if !strings.Contains(m.View(), "不允许在 TUI 中嵌套调用 tui") {
		t.Fatalf("预期拦截嵌套 tui 调用，实际未拦截: %s", m.View())
	}

	// 验证 Esc 取消返回
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newM.(*tui.Model)

	// 验证按 'q' 触发退出
	_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quitCmd == nil {
		t.Fatalf("按 q 应当返回退出 cmd")
	}
}

func TestTUI_HelpAndNoNest(t *testing.T) {
	// 验证拦截嵌套调用
	tui.Run([]string{"tui"})

	// 验证帮助指令
	tui.Run([]string{"--help"})
	tui.Run([]string{"-h"})
}

func TestTUI_DynamicDispatchAllCommands(t *testing.T) {
	cmds := tui.GetDynamicCommands()
	for _, spec := range cmds {
		m := tui.NewModel()
		newM, _ := m.SelectCommandForTest(&spec)
		testedModel := newM.(*tui.Model)
		view := testedModel.View()
		if view == "" {
			t.Errorf("命令 %s 动态分发后视图不应为空", spec.Name)
		}
	}
}

func TestTUI_DynamicDepinstall(t *testing.T) {
	spec := tui.FindCommandSpec("depinstall")
	if spec == nil {
		t.Fatalf("预期找到 depinstall 命令规范")
	}

	m := tui.NewModel()
	newM, _ := m.SelectCommandForTest(spec)
	view := newM.(*tui.Model).View()

	if !strings.Contains(view, "报错的任务日志ID") {
		t.Errorf("depinstall 视图应包含动态配置的 ArgPrompt 提示")
	}
	if !strings.Contains(view, "缺失依赖") {
		t.Errorf("depinstall 视图应包含动态配置的 ArgHelp 关键词")
	}
}

func TestTUI_DynamicServerConfirm(t *testing.T) {
	spec := tui.FindCommandSpec("server")
	if spec == nil {
		t.Fatalf("预期找到 server 命令规范")
	}

	m := tui.NewModel()
	newM, _ := m.SelectCommandForTest(spec)
	view := newM.(*tui.Model).View()

	if !strings.Contains(view, "确认") {
		t.Errorf("server 具有 Confirm=true，视图应包含确认提示")
	}
	if !strings.Contains(view, "server") {
		t.Errorf("server 视图应包含 server 命令名称")
	}
	if !strings.Contains(view, "Ctrl+C") {
		t.Errorf("server 视图应包含动态配置的 ConfirmPrompt 中的 Ctrl+C 关键描述")
	}
}

func TestTUI_ViewportScroll(t *testing.T) {
	m := tui.NewModel()
	// 模拟窗口大小设置
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = newM.(*tui.Model)

	// 添加大量日志
	for i := 1; i <= 40; i++ {
		m.AppendLog(tui.LogOutput, fmt.Sprintf("Line %d: This is a test log output", i))
	}

	// 此时触发 version 命令结束
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newM.(*tui.Model)

	m.AppendLog(tui.LogExec, ">> baihu version")
	m.AppendLog(tui.LogOutput, "baihu-panel dev")
	m.AppendLog(tui.LogSuccess, "指令 [version] 执行完成")

	vpView := m.View()

	if !strings.Contains(vpView, "baihu version") {
		t.Errorf("View 应包含最新的日志 'baihu version'，但没有找到！")
	}
}

func TestTUI_TabFocusAndClearLogs(t *testing.T) {
	m := tui.NewModel()

	// 验证初始状态处于菜单焦点
	if !strings.Contains(m.View(), "按 Tab 切换焦点") {
		t.Errorf("初始状态右侧应提示按 Tab 切换焦点")
	}

	// 按 Tab 切换到日志区
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(*tui.Model)
	if !strings.Contains(m.View(), "已聚焦") {
		t.Errorf("按下 Tab 后日志面板标题应显示已聚焦")
	}

	// 再次按 Tab 切换回菜单
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(*tui.Model)
	if !strings.Contains(m.View(), "按 Tab 切换焦点") {
		t.Errorf("再次按下 Tab 后应切回菜单焦点")
	}

	// 验证按 Ctrl+L 清空日志
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = newM.(*tui.Model)
	if !strings.Contains(m.View(), "日志记录已清空") {
		t.Errorf("按 Ctrl+L 后日志视图应包含已清空提示")
	}
}

func TestTUI_SequentialExecution(t *testing.T) {
	m := tui.NewModel()
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	m = newM.(*tui.Model)

	// 1. 模拟执行 app 命令
	appSpec := tui.FindCommandSpec("app")
	newM, _ = m.SelectCommandForTest(appSpec)
	m = newM.(*tui.Model)

	// 模拟执行完成返回大量表格数据
	appOutput := `应用标识 (ID) | 应用名称 | 版本 | 分类 | 生效场景 | 安装时间
========================================================================================
dalpi68c5br5ph049mk0 | B站全自动化助手 (BilibiliToolPro) | 4.2.3 | 福利签到 | minimal | 2026-09-17 15:39
dal1028c5br1bs08m4d0 | 京东全功能自动化助手 (JDPro) | 1.0.0 | 福利签到 | standard | 2026-09-16 11:42`

	// 模拟触发异步任务完成
	m.AppendLog(tui.LogExec, ">> baihu app list")
	for _, line := range strings.Split(appOutput, "\n") {
		m.AppendLog(tui.LogOutput, line)
	}
	m.AppendLog(tui.LogSuccess, "指令 [app list] 执行完成 (耗时: 120ms)")

	// 2. 模拟随后直接执行 version 命令
	verSpec := tui.FindCommandSpec("version")
	newM, cmd := m.SelectCommandForTest(verSpec)
	m = newM.(*tui.Model)
	if cmd == nil {
		t.Fatalf("version 应当返回异步执行 cmd")
	}

	// 模拟 version 命令执行完成消息
	m.AppendLog(tui.LogOutput, "baihu-panel v2.0.0-rc1")
	m.AppendLog(tui.LogSuccess, "指令 [version] 执行完成 (耗时: 15ms)")

	view := m.View()
	if !strings.Contains(view, "baihu version") {
		t.Errorf("执行 version 后，日志视口必须包含 'baihu version'")
	}
	if !strings.Contains(view, "v2.0.0-rc1") {
		t.Errorf("执行 version 后，日志视口必须包含版本输出")
	}
	if !strings.Contains(view, "100%") {
		t.Errorf("执行新指令后日志视口滚动位置应当自动对齐到底部 100%%")
	}

	// 严禁每行出现 [OUT] 标识
	if strings.Contains(view, "[OUT]") {
		t.Errorf("日志输出区域严禁出现 '[OUT]' 冗余标识")
	}

	// 验证包含明显的分隔线与原样输出内容
	if !strings.Contains(view, "──") {
		t.Errorf("命令执行前必须包含明显的分隔线样式")
	}
	if !strings.Contains(view, "B站全自动化助手") {
		t.Errorf("执行内容必须原样输出")
	}
}

func TestTUI_ExitOption(t *testing.T) {
	m := tui.NewModel()
	exitSpec := tui.FindCommandSpec("exit")
	if exitSpec == nil {
		t.Fatalf("必须存在 exit 命令规范")
	}

	// 1. 验证菜单选择 exit 触发 tea.Quit
	_, cmd := m.SelectCommandForTest(exitSpec)
	if cmd == nil {
		t.Fatalf("选择 exit 命令应当返回退出 cmd")
	}

	// 2. 验证自由输入 exit 触发退出
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newM.(*tui.Model)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("exit")})
	m = newM.(*tui.Model)
	_, exitCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if exitCmd == nil {
		t.Fatalf("输入 exit 应当触发退出 cmd")
	}
}
