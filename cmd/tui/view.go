package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/engigu/baihu-panel/internal/constant"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D7D7"))

	headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3B4252")).
			Padding(0, 1)

	paneBorderMenuFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00D7D7")).
				Padding(0, 1)

	paneBorderMenuUnfocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#4C566A")).
				Padding(0, 1)

	paneBorderLogsFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFAF00")).
				Padding(0, 1)

	paneBorderLogsUnfocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#4C566A")).
				Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E9F0"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7B88A1"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88C0D0"))

	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EBCB8B")).
			Bold(true)

	statusSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A3BE8C")).
				Bold(true)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#BF616A")).
				Bold(true)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D08770")).
				Bold(true)

	// 日志配色
	logTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#616E88"))

	logSystemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#81A1C1")).
			Bold(true)

	logExecStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B48EAD")).
			Bold(true)

	logOutputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D8DEE9"))

	logSuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A3BE8C")).
			Bold(true)

	logErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BF616A")).
			Bold(true)

	logInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EBCB8B"))

	logDividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4C566A"))

	logCmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")).
			Bold(true)

	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00D7D7")).
			Padding(1, 1)
)

func (m *Model) View() string {
	width := m.width
	if width <= 0 {
		width = 110
	}
	height := m.height
	if height <= 0 {
		height = 30
	}

	header := m.renderHeader(width)
	footer := m.renderFooter(width)

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)

	bodyHeight := height - headerHeight - footerHeight
	if bodyHeight < 10 {
		bodyHeight = 10
	}

	var body string
	if width >= 80 {
		// 宽屏模式：左右分栏
		// 左侧控制菜单紧凑窄栏：固定占用 25~30 列宽，留出 75%+ 宽阔视野给右侧日志区
		leftWidth := int(float64(width) * 0.25)
		if leftWidth < 25 {
			leftWidth = 25
		}
		if leftWidth > 30 {
			leftWidth = 30
		}
		rightWidth := width - leftWidth - 3
		if rightWidth < 35 {
			rightWidth = 35
		}

		leftPane := m.renderLeftPane(leftWidth, bodyHeight)
		rightPane := m.renderRightPane(rightWidth, bodyHeight)
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	} else {
		// 极窄屏模式：上下分栏
		halfHeight := bodyHeight / 2
		if halfHeight < 6 {
			halfHeight = 6
		}
		leftPane := m.renderLeftPane(width-2, halfHeight)
		rightPane := m.renderRightPane(width-2, bodyHeight-halfHeight)
		body = lipgloss.JoinVertical(lipgloss.Left, leftPane, rightPane)
	}

	return fmt.Sprintf("%s\n%s\n%s", header, body, footer)
}

func (m *Model) renderHeader(totalWidth int) string {
	title := titleStyle.Render("白虎面板终端交互控制台 (Baihu Console TUI)")
	statusText := "就绪"
	if m.running {
		statusText = statusRunningStyle.Render("执行中...")
	} else if m.isError {
		statusText = statusErrorStyle.Render("异常")
	} else {
		statusText = statusSuccessStyle.Render("正常")
	}

	sub := descStyle.Render(fmt.Sprintf("版本: %s | 系统: %s/%s | 驱动: Bubble Tea | 状态: %s",
		constant.Version, runtime.GOOS, runtime.GOARCH, statusText))

	content := fmt.Sprintf("%s\n%s", title, sub)
	return headerBoxStyle.Width(totalWidth - 2).Render(content)
}

func (m *Model) renderLeftPane(w, h int) string {
	style := paneBorderMenuUnfocused
	if m.activePane == paneMenu {
		style = paneBorderMenuFocused
	}

	innerW := w - 4
	if innerW < 10 {
		innerW = 10
	}

	var content string
	switch m.state {
	case stateMainMenu:
		content = m.renderMainMenuContent(innerW)
	case stateSubMenu:
		content = m.renderSubMenuContent(innerW)
	case stateArgsMenu:
		content = m.renderArgsMenuContent(innerW)
	case stateInput:
		content = m.renderInputContent(innerW)
	case stateConfirm:
		content = m.renderConfirmContent(innerW)
	}

	return style.Width(w).Height(h).Render(content)
}

func (m *Model) renderMainMenuContent(innerW int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("【控制中心】") + "\n\n")

	for i, cmd := range m.commands {
		cursor := "  "
		badge := badgeStyle.Render(fmt.Sprintf("[%2d]", i+1))
		nameText := cmd.Name

		if m.cursor == i && m.activePane == paneMenu {
			cursor = selectedStyle.Render("> ")
			nameText = selectedStyle.Render(nameText)
		} else {
			nameText = normalStyle.Render(nameText)
		}
		s.WriteString(fmt.Sprintf("%s%s %s\n", cursor, badge, nameText))
	}

	// 底部紧凑展示当前光标选中的命令功能描述
	if m.cursor >= 0 && m.cursor < len(m.commands) {
		cur := m.commands[m.cursor]
		s.WriteString("\n" + descStyle.Render("── "+cur.Name+" ──") + "\n")
		descWrapped := lipgloss.NewStyle().Width(innerW).Render(cur.Description)
		s.WriteString(helpStyle.Render(descWrapped) + "\n")
	}

	s.WriteString("\n" + helpStyle.Render("↑/↓:移动 Enter:选 C:自由"))
	return s.String()
}

func (m *Model) renderSubMenuContent(innerW int) string {
	var s strings.Builder
	cmdName := m.selectedCmd.Name
	s.WriteString(titleStyle.Render("【"+cmdName+" 子命令】") + "\n\n")

	for i, k := range m.subKeys {
		cursor := "  "
		badge := badgeStyle.Render(fmt.Sprintf("[%d]", i+1))
		keyText := k

		if m.cursor == i && m.activePane == paneMenu {
			cursor = selectedStyle.Render("> ")
			keyText = selectedStyle.Render(keyText)
		} else {
			keyText = normalStyle.Render(keyText)
		}
		s.WriteString(fmt.Sprintf("%s%s %s\n", cursor, badge, keyText))
	}

	// 底部紧凑展示当前选中的子命令说明
	if m.cursor >= 0 && m.cursor < len(m.subKeys) {
		k := m.subKeys[m.cursor]
		desc := m.selectedCmd.SubCommands[k]
		s.WriteString("\n" + descStyle.Render("── "+k+" ──") + "\n")
		descWrapped := lipgloss.NewStyle().Width(innerW).Render(desc)
		s.WriteString(helpStyle.Render(descWrapped) + "\n")
	}

	s.WriteString("\n" + helpStyle.Render("↑/↓:移动 Enter:选 Esc:返"))
	return s.String()
}

func (m *Model) renderArgsMenuContent(innerW int) string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("【"+m.selectedCmd.Name+" 选项】") + "\n\n")

	for i, arg := range m.selectedCmd.Args {
		cursor := "  "
		badge := badgeStyle.Render(fmt.Sprintf("[%d]", i+1))
		argText := normalStyle.Render(arg)

		if m.cursor == i && m.activePane == paneMenu {
			cursor = selectedStyle.Render("> ")
			argText = selectedStyle.Render(arg)
		}
		s.WriteString(fmt.Sprintf("%s%s %s\n", cursor, badge, argText))
	}

	s.WriteString("\n" + helpStyle.Render("↑/↓:移动 Enter:选 Esc:返"))
	return s.String()
}

func (m *Model) renderInputContent(innerW int) string {
	previewCmd := strings.Join(m.inputPrefix, " ")
	if m.textInput.Value() != "" {
		if previewCmd != "" {
			previewCmd += " " + m.textInput.Value()
		} else {
			previewCmd = m.textInput.Value()
		}
	}

	promptWrapped := lipgloss.NewStyle().Width(innerW).Render(m.inputPrompt)
	helpWrapped := lipgloss.NewStyle().Width(innerW).Render(m.inputHelp)

	return fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n\n%s",
		titleStyle.Render("【"+m.inputTitle+"】"),
		normalStyle.Render(promptWrapped),
		m.textInput.View(),
		descStyle.Render(">> 预览:"),
		selectedStyle.Render("baihu "+previewCmd),
		helpStyle.Render(helpWrapped),
		descStyle.Render("Enter:确认  Esc:取消"),
	)
}

func (m *Model) renderConfirmContent(innerW int) string {
	promptWrapped := lipgloss.NewStyle().Width(innerW).Render(m.confirmPrompt)
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		titleStyle.Render("【"+m.confirmTitle+"】"),
		normalStyle.Render(promptWrapped),
		badgeStyle.Render("[Y]确认  [N/Esc]取消"),
	)
}

func (m *Model) renderRightPane(w, h int) string {
	style := paneBorderLogsUnfocused
	percent := int(m.viewport.ScrollPercent() * 100)
	titlePrefix := fmt.Sprintf("【运行与执行日志】 [%d%%] (按 Tab 切换焦点)", percent)

	if m.activePane == paneLogs {
		style = paneBorderLogsFocused
		titlePrefix = badgeStyle.Render(fmt.Sprintf("【运行与执行日志】★ 已聚焦 [%d%%] (PgUp/PgDn滚动 • Esc返回 • Ctrl+L清空)", percent))
	} else {
		titlePrefix = titleStyle.Render(titlePrefix)
	}

	// 动态精确对齐 viewport 实际可用区域，防止外层边框截断最新产生的日志行
	vpW := w - 4
	if vpW < 10 {
		vpW = 10
	}
	vpH := h - 4
	if vpH < 4 {
		vpH = 4
	}
	if m.viewport.Width != vpW || m.viewport.Height != vpH {
		m.viewport.Width = vpW
		m.viewport.Height = vpH
		m.refreshViewport()
	}

	vpView := m.viewport.View()
	if len(m.logs) == 0 {
		vpView = descStyle.Render("\n  暂无日志输出，从左侧选择指令执行，或按 C 键输入自由指令。")
	}

	content := fmt.Sprintf("%s\n%s", titlePrefix, vpView)
	return style.Width(w).Height(h).Render(content)
}

func (m *Model) renderFooter(totalWidth int) string {
	var s strings.Builder
	shortcuts := "快捷键: Tab: 切换面板焦点  •  ↑/↓: 导航  •  Enter: 确认  •  C: 自由指令  •  Ctrl+L: 清空日志  •  Q: 退出"
	s.WriteString(helpStyle.Render(shortcuts))

	if m.lastMessage != "" {
		s.WriteString("\n")
		if m.isError {
			s.WriteString(statusErrorStyle.Render(">> " + m.lastMessage))
		} else {
			s.WriteString(statusSuccessStyle.Render(">> " + m.lastMessage))
		}
	}

	return s.String()
}
