package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/engigu/baihu-panel/internal/utils"
)

type LogLevel string

const (
	LogInfo    LogLevel = "INFO"
	LogSystem  LogLevel = "SYSTEM"
	LogExec    LogLevel = "EXEC"
	LogOutput  LogLevel = "OUT"
	LogSuccess LogLevel = "OK"
	LogError   LogLevel = "ERR"
)

type LogEntry struct {
	Time    time.Time
	Level   LogLevel
	Message string
}

type cmdFinishedMsg struct {
	cmd      string
	output   string
	err      error
	duration time.Duration
}

func isInteractiveCommand(cmdName string) bool {
	switch strings.ToLower(cmdName) {
	case "server", "mcp", "resetpwd":
		return true
	default:
		return false
	}
}

func (m *Model) AppendLog(level LogLevel, msg string) {
	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
	}
	m.logs = append(m.logs, entry)
	m.refreshViewport()
}

func (m *Model) ClearLogs() {
	m.logs = nil
	m.AppendLog(LogSystem, "日志记录已清空")
}

func (m *Model) refreshViewport() {
	var sb strings.Builder
	wrapWidth := m.viewport.Width
	if wrapWidth <= 0 {
		wrapWidth = 60
	}
	wrapStyle := lipgloss.NewStyle().Width(wrapWidth)
	sepWidth := min(wrapWidth, 76)
	divider := logDividerStyle.Render(strings.Repeat("─", sepWidth))

	for _, l := range m.logs {
		timeStr := logTimeStyle.Render(l.Time.Format("15:04:05"))

		switch l.Level {
		case LogOutput:
			// 执行内容原样输出：不添加时间戳与 [OUT] 标识，保留原始表格排版
			wrapped := wrapStyle.Render(l.Message)
			sb.WriteString(wrapped)
			sb.WriteString("\n")

		case LogExec:
			// 每个命令前增加明显的分隔线与专业执行记录头
			badge := logExecStyle.Render("▶ [EXEC]")
			cmdText := logCmdStyle.Render(l.Message)
			sb.WriteString("\n" + divider + "\n")
			sb.WriteString(fmt.Sprintf("%s %s %s\n", timeStr, badge, cmdText))
			sb.WriteString(divider + "\n")

		case LogSuccess:
			badge := logSuccessStyle.Render("✔ [OK]")
			sb.WriteString(fmt.Sprintf("%s %s %s\n\n", timeStr, badge, l.Message))

		case LogError:
			badge := logErrorStyle.Render("✖ [ERR]")
			sb.WriteString(fmt.Sprintf("%s %s %s\n\n", timeStr, badge, l.Message))

		case LogSystem:
			badge := logSystemStyle.Render("[SYSTEM]")
			sb.WriteString(fmt.Sprintf("%s %s %s\n", timeStr, badge, l.Message))

		default:
			badge := logInfoStyle.Render("[INFO]")
			sb.WriteString(fmt.Sprintf("%s %s %s\n", timeStr, badge, l.Message))
		}
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m *Model) runProcess(args ...string) tea.Cmd {
	if len(args) == 0 {
		return nil
	}

	m.running = true
	cmdLine := strings.Join(args, " ")
	m.AppendLog(LogExec, "baihu "+cmdLine)

	if isInteractiveCommand(args[0]) {
		return m.executeInteractive(args...)
	}
	return m.executeCommandAsync(args...)
}

func (m *Model) executeCommandAsync(args ...string) tea.Cmd {
	return func() tea.Msg {
		startTime := time.Now()
		exe := utils.GetBaihuExecutable()
		c := exec.Command(exe, args...)
		c.Env = append(os.Environ(), utils.BuildRuntimeProcessEnv()...)

		out, err := c.CombinedOutput()
		return cmdFinishedMsg{
			cmd:      strings.Join(args, " "),
			output:   string(out),
			err:      err,
			duration: time.Since(startTime),
		}
	}
}

func (m *Model) executeInteractive(args ...string) tea.Cmd {
	exe := utils.GetBaihuExecutable()
	c := exec.Command(exe, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), utils.BuildRuntimeProcessEnv()...)

	startTime := time.Now()
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return cmdFinishedMsg{
			cmd:      strings.Join(args, " "),
			output:   "(控制台交互已完成返回)",
			err:      err,
			duration: time.Since(startTime),
		}
	})
}
