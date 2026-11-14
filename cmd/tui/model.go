package tui

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/engigu/baihu-panel/internal/constant"
)

type sessionState int

const (
	stateMainMenu sessionState = iota
	stateSubMenu
	stateArgsMenu
	stateInput
	stateConfirm
)

type activePane int

const (
	paneMenu activePane = iota
	paneLogs
)

type Model struct {
	state      sessionState
	activePane activePane
	commands   []constant.CommandSpec
	cursor     int

	selectedCmd *constant.CommandSpec
	selectedSub string
	subKeys     []string

	// 输入框组件
	textInput   textinput.Model
	inputTitle  string
	inputPrompt string
	inputHelp   string
	inputPrefix []string
	isCustom    bool

	// 确认框
	confirmTitle  string
	confirmPrompt string
	confirmAction []string

	// 日志专区视口与记录
	viewport viewport.Model
	logs     []LogEntry
	running  bool

	// 状态反馈
	lastMessage string
	isError     bool

	width      int
	height     int
	initialCmd tea.Cmd
}

func NewModel() *Model {
	ti := textinput.New()
	ti.Placeholder = "请输入参数..."
	ti.CharLimit = 256
	ti.Width = 22

	vp := viewport.New(60, 20)

	m := &Model{
		state:      stateMainMenu,
		activePane: paneMenu,
		commands:   GetDynamicCommands(),
		cursor:     0,
		textInput:  ti,
		viewport:   vp,
	}

	m.AppendLog(LogSystem, fmt.Sprintf("白虎终端控制台已启动 (%s - %s/%s)", constant.Version, runtime.GOOS, runtime.GOARCH))
	m.AppendLog(LogSystem, fmt.Sprintf("动态加载 %d 个系统命令，按 Tab 切换面板焦点", len(m.commands)))

	return m
}

func (m *Model) Init() tea.Cmd {
	if m.initialCmd != nil {
		return tea.Batch(textinput.Blink, m.initialCmd)
	}
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerH := 4
		footerH := 3
		bodyH := m.height - headerH - footerH
		if bodyH < 8 {
			bodyH = 8
		}

		if m.width >= 80 {
			leftW := int(float64(m.width) * 0.25)
			if leftW < 25 {
				leftW = 25
			}
			if leftW > 30 {
				leftW = 30
			}
			rightW := m.width - leftW - 3
			m.viewport.Width = rightW - 4
			m.viewport.Height = bodyH - 4
			m.textInput.Width = leftW - 6
		} else {
			m.viewport.Width = m.width - 6
			m.viewport.Height = (bodyH / 2) - 4
			m.textInput.Width = m.width - 8
		}
		m.refreshViewport()
		return m, nil

	case cmdFinishedMsg:
		m.running = false
		if msg.output != "" {
			lines := strings.Split(strings.TrimRight(msg.output, "\r\n"), "\n")
			for _, line := range lines {
				line = strings.TrimRight(line, "\r")
				m.AppendLog(LogOutput, line)
			}
		}
		if msg.err != nil {
			m.isError = true
			m.lastMessage = fmt.Sprintf("指令 [%s] 异常: %v", msg.cmd, msg.err)
			m.AppendLog(LogError, fmt.Sprintf("指令 [%s] 执行失败: %v (耗时: %v)", msg.cmd, msg.err, msg.duration.Round(time.Millisecond)))
		} else {
			m.isError = false
			m.lastMessage = fmt.Sprintf("指令 [%s] 成功", msg.cmd)
			m.AppendLog(LogSuccess, fmt.Sprintf("指令 [%s] 执行完成 (耗时: %v)", msg.cmd, msg.duration.Round(time.Millisecond)))
		}
		return m, nil

	case tea.KeyMsg:
		// 全局快捷键响应
		switch msg.String() {
		case "tab":
			if m.activePane == paneMenu {
				m.activePane = paneLogs
				m.AppendLog(LogSystem, "焦点已切换至日志区，可按 ↑/↓/PgUp/PgDn 滚动查看")
			} else {
				m.activePane = paneMenu
			}
			return m, nil

		case "ctrl+l":
			m.ClearLogs()
			m.lastMessage = "日志已清空"
			return m, nil
		}

		// 焦点处于日志查看区域时的按键处理
		if m.activePane == paneLogs {
			switch msg.String() {
			case "esc":
				m.activePane = paneMenu
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		// 焦点处于控制菜单区域时的按键处理
		switch m.state {
		case stateMainMenu:
			return m.updateMainMenu(msg)
		case stateSubMenu:
			return m.updateSubMenu(msg)
		case stateArgsMenu:
			return m.updateArgsMenu(msg)
		case stateInput:
			return m.updateInput(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		}
	}

	return m, nil
}

func (m *Model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = len(m.commands) - 1
		}

	case "down", "j":
		if m.cursor < len(m.commands)-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}

	case "c", "C":
		m.state = stateInput
		m.isCustom = true
		m.inputTitle = "自由命令行执行"
		m.inputPrompt = "请输入 baihu 完整参数 (例如: task list -size 5):"
		m.inputHelp = "输入完整子命令参数后回车执行，按 Esc 取消"
		m.inputPrefix = nil
		m.textInput.Reset()
		m.textInput.Focus()
		return m, textinput.Blink

	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.commands) {
			spec := m.commands[m.cursor]
			return m.selectCommand(&spec)
		}

	default:
		// 支持数字快捷键跳转
		if idx, err := strconv.Atoi(msg.String()); err == nil && idx >= 1 && idx <= len(m.commands) {
			m.cursor = idx - 1
			spec := m.commands[m.cursor]
			return m.selectCommand(&spec)
		}
	}

	return m, nil
}

// SelectCommandForTest 用于单元测试动态路由验证
func (m *Model) SelectCommandForTest(spec *constant.CommandSpec) (tea.Model, tea.Cmd) {
	return m.selectCommand(spec)
}

func (m *Model) selectCommand(spec *constant.CommandSpec) (tea.Model, tea.Cmd) {
	m.selectedCmd = spec
	m.cursor = 0

	// 退出选项
	if spec.Name == "exit" {
		return m, tea.Quit
	}

	// 1. 包含子命令
	if len(spec.SubCommands) > 0 {
		m.state = stateSubMenu
		m.subKeys = GetSortedSubcommandKeys(spec.SubCommands)
		return m, nil
	}

	// 2. 包含参数候选
	if len(spec.Args) > 0 {
		m.state = stateArgsMenu
		return m, nil
	}

	// 3. 需确认执行的命令（动态读取 spec.Confirm）
	if spec.Confirm {
		m.state = stateConfirm
		m.confirmTitle = fmt.Sprintf("%s (%s)", spec.Description, spec.Name)
		if spec.ConfirmPrompt != "" {
			m.confirmPrompt = spec.ConfirmPrompt
		} else {
			m.confirmPrompt = fmt.Sprintf("确认执行 %s 命令吗?", spec.Name)
		}
		m.confirmAction = []string{spec.Name}
		return m, nil
	}

	// 4. 需要输入参数或携带命令行选项的命令（动态读取 spec.ArgPrompt / spec.Flags）
	if spec.ArgPrompt != "" || len(spec.Flags) > 0 {
		unique := ExtractUniqueFlags(spec.Flags)
		m.state = stateInput
		m.isCustom = false
		m.inputTitle = fmt.Sprintf("%s (%s)", spec.Description, spec.Name)
		if spec.ArgPrompt != "" {
			m.inputPrompt = spec.ArgPrompt
		} else {
			m.inputPrompt = "请输入选项参数 (直接回车执行默认):"
		}

		if spec.ArgHelp != "" {
			m.inputHelp = spec.ArgHelp
		} else if len(unique) > 0 {
			m.inputHelp = fmt.Sprintf("可用选项: %s", strings.Join(unique, " "))
		} else {
			m.inputHelp = "输入参数后回车执行，按 Esc 取消返回"
		}

		m.inputPrefix = []string{spec.Name}
		m.textInput.Reset()
		if spec.ArgDefault != "" {
			m.textInput.SetValue(spec.ArgDefault)
		}
		m.textInput.Focus()
		return m, textinput.Blink
	}

	// 5. 无子命令、无参数的常规命令直接执行
	return m, m.runProcess(spec.Name)
}

func (m *Model) updateSubMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b", "B", "left":
		m.state = stateMainMenu
		m.cursor = 0
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = len(m.subKeys) - 1
		}

	case "down", "j":
		if m.cursor < len(m.subKeys)-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}

	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.subKeys) {
			subKey := m.subKeys[m.cursor]
			m.selectedSub = subKey
			return m.selectSubcommand(subKey)
		}

	default:
		if idx, err := strconv.Atoi(msg.String()); err == nil && idx >= 1 && idx <= len(m.subKeys) {
			m.cursor = idx - 1
			subKey := m.subKeys[m.cursor]
			m.selectedSub = subKey
			return m.selectSubcommand(subKey)
		}
	}

	return m, nil
}

func (m *Model) selectSubcommand(subKey string) (tea.Model, tea.Cmd) {
	cmdName := m.selectedCmd.Name
	subDesc := m.selectedCmd.SubCommands[subKey]

	prompt := ""
	if m.selectedCmd.SubPrompts != nil {
		prompt = m.selectedCmd.SubPrompts[subKey]
	}
	defaultVal := ""
	if m.selectedCmd.SubDefaults != nil {
		defaultVal = m.selectedCmd.SubDefaults[subKey]
	}

	var flags []string
	if m.selectedCmd.SubFlags != nil && len(m.selectedCmd.SubFlags[subKey]) > 0 {
		flags = ExtractUniqueFlags(m.selectedCmd.SubFlags[subKey])
	}

	if prompt != "" || len(flags) > 0 {
		m.state = stateInput
		m.isCustom = false
		m.inputTitle = fmt.Sprintf("[%s %s] %s", cmdName, subKey, subDesc)
		if prompt != "" {
			m.inputPrompt = prompt
		} else {
			m.inputPrompt = "请输入参数选项 (直接回车跳过):"
		}

		if len(flags) > 0 {
			m.inputHelp = fmt.Sprintf("支持选项参数: %s", strings.Join(flags, " "))
		} else {
			m.inputHelp = "输入参数后回车执行，按 Esc 取消返回"
		}

		m.inputPrefix = []string{cmdName, subKey}
		m.textInput.Reset()
		if defaultVal != "" {
			m.textInput.SetValue(defaultVal)
		}
		m.textInput.Focus()
		return m, textinput.Blink
	}

	// 无参数子命令直接执行
	return m, m.runProcess(cmdName, subKey)
}

func (m *Model) updateArgsMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b", "B", "left":
		m.state = stateMainMenu
		m.cursor = 0
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = len(m.selectedCmd.Args) - 1
		}

	case "down", "j":
		if m.cursor < len(m.selectedCmd.Args)-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}

	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.selectedCmd.Args) {
			arg := m.selectedCmd.Args[m.cursor]
			return m, m.runProcess(m.selectedCmd.Name, arg)
		}
	}

	return m, nil
}

func (m *Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.selectedCmd != nil && len(m.selectedCmd.SubCommands) > 0 {
			m.state = stateSubMenu
		} else {
			m.state = stateMainMenu
		}
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.textInput.Value())

		// 自定义命令行执行
		if m.isCustom {
			if val == "" {
				m.state = stateMainMenu
				return m, nil
			}
			args := strings.Fields(val)
			if len(args) > 0 && (strings.EqualFold(args[0], "exit") || strings.EqualFold(args[0], "quit")) {
				return m, tea.Quit
			}
			// 严格拦截嵌套 tui
			if len(args) > 0 && strings.EqualFold(args[0], "tui") {
				m.isError = true
				m.lastMessage = "不允许在 TUI 中嵌套调用 tui 指令"
				m.AppendLog(LogError, "尝试嵌套调用 tui 指令，已被系统安全机制拦截")
				return m, nil
			}
			m.state = stateMainMenu
			return m, m.runProcess(args...)
		}

		// 常规带前缀的命令
		var finalArgs []string
		finalArgs = append(finalArgs, m.inputPrefix...)
		if val != "" {
			finalArgs = append(finalArgs, strings.Fields(val)...)
		}

		if m.selectedCmd != nil && len(m.selectedCmd.SubCommands) > 0 {
			m.state = stateSubMenu
		} else {
			m.state = stateMainMenu
		}

		return m, m.runProcess(finalArgs...)
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.state = stateMainMenu
		return m, m.runProcess(m.confirmAction...)

	case "n", "N", "esc", "b", "B":
		m.state = stateMainMenu
		m.AppendLog(LogInfo, fmt.Sprintf("取消操作: %s", strings.Join(m.confirmAction, " ")))
		return m, nil
	}

	return m, nil
}
