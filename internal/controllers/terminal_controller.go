package controllers

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
	"github.com/engigu/baihu-panel/internal/windows"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type TerminalController struct {
	envService *services.EnvService
}

func NewTerminalController(envService *services.EnvService) *TerminalController {
	return &TerminalController{
		envService: envService,
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: utils.CheckWSOrigin,
}

// toUTF8 将可能是 GBK 编码的字节转换为 UTF-8
func toUTF8(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	// 尝试从 GBK 转换
	reader := transform.NewReader(
		bufio.NewReader(
			&byteReader{data: data},
		),
		simplifiedchinese.GBK.NewDecoder(),
	)
	result, err := io.ReadAll(reader)
	if err != nil {
		return string(data)
	}
	return string(result)
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (tc *TerminalController) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 演示模式下禁用终端
	if constant.DemoMode {
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n\033[1;33m[演示模式] 终端功能已禁用\033[0m\r\n"))
		return
	}

	// Windows 使用 pipe 模式，Unix 使用 PTY 模式
	userID := c.GetString("userID")
	if userID == "" {
		userID = "1" // 兜底
	}
	if windows.IsWindows() {
		tc.handlePipeMode(conn, userID)
	} else {
		tc.handlePtyMode(conn, userID)
	}
}

// handlePtyMode 使用 PTY 处理终端（Unix/macOS）
func (tc *TerminalController) handlePtyMode(conn *websocket.Conn, userID string) {
	conn.SetReadLimit(constant.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(constant.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(constant.PongWait))
		return nil
	})

	// 发送 PTY 模式标识
	conn.WriteMessage(websocket.TextMessage, []byte("__PTY_MODE__"))

	cmd := utils.NewShellCmd()

	if absDir, err := filepath.Abs(constant.ScriptsWorkDir); err == nil {
		cmd.Dir = absDir
	}

	cmd.Env = tc.buildTerminalEnv(userID, "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error starting shell: "+err.Error()))
		return
	}
	defer ptmx.Close()

	pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	var wg sync.WaitGroup
	var connMu sync.Mutex

	writeMessage := func(data []byte) {
		connMu.Lock()
		defer connMu.Unlock()
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		conn.WriteMessage(websocket.TextMessage, data)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		var remainder []byte
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				chunk := append(remainder, buf[:n]...)
				lastSafe := len(chunk)
				for i := len(chunk); i > 0 && i > len(chunk)-4; i-- {
					if utf8.RuneStart(chunk[i-1]) {
						if !utf8.FullRune(chunk[i-1 : len(chunk)]) {
							lastSafe = i - 1
						}
						break
					}
				}
				safe := chunk[:lastSafe]
				
				if len(safe) == 0 && len(chunk) >= 4 {
					safe = chunk
					remainder = nil
				} else {
					remainder = make([]byte, len(chunk[lastSafe:]))
					copy(remainder, chunk[lastSafe:])
				}

				if len(safe) > 0 {
					text := toUTF8(safe)
					writeMessage([]byte(text))
				}
			}
			if err != nil {
				if len(remainder) > 0 {
					writeMessage([]byte(toUTF8(remainder)))
				}
				return
			}
		}
	}()

	// 启动 ping 协程
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(constant.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				connMu.Lock()
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					connMu.Unlock()
					return
				}
				connMu.Unlock()
			case <-pingDone:
				return
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// 处理调整窗口大小的消息
		if len(message) > 0 && message[0] == '{' {
			var resizeMsg struct {
				Type string `json:"type"`
				Rows uint16 `json:"rows"`
				Cols uint16 `json:"cols"`
			}
			if err := json.Unmarshal(message, &resizeMsg); err == nil && resizeMsg.Type == "resize" {
				pty.Setsize(ptmx, &pty.Winsize{Rows: resizeMsg.Rows, Cols: resizeMsg.Cols})
				continue
			}
		}

		if _, err := ptmx.Write(message); err != nil {
			break
		}
	}

	close(pingDone)
	cmd.Process.Kill()
	cmd.Wait()
	ptmx.Close() // Force close PTY to interrupt the blocking ptmx.Read() in the goroutine
	wg.Wait()
}

// handlePipeMode 使用 pipe 处理终端（Windows）
func (tc *TerminalController) handlePipeMode(conn *websocket.Conn, userID string) {
	conn.SetReadLimit(constant.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(constant.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(constant.PongWait))
		return nil
	})

	// 发送 pipe 模式标识
	conn.WriteMessage(websocket.TextMessage, []byte("__PIPE_MODE__"))

	cmd := utils.NewShellCmd()

	if absDir, err := filepath.Abs(constant.ScriptsWorkDir); err == nil {
		cmd.Dir = absDir
	}

	// 注入环境变量
	cmd.Env = tc.buildTerminalEnv(userID)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}

	if err := cmd.Start(); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
		return
	}

	var wg sync.WaitGroup
	var connMu sync.Mutex

	writeMessage := func(data []byte) {
		connMu.Lock()
		defer connMu.Unlock()
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		conn.WriteMessage(websocket.TextMessage, data)
	}

	readOutput := func(reader io.Reader) {
		defer wg.Done()
		defer func() { recover() }()
		buf := make([]byte, 4096)
		var remainder []byte
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				chunk := append(remainder, buf[:n]...)
				lastSafe := len(chunk)
				for i := len(chunk); i > 0 && i > len(chunk)-4; i-- {
					if utf8.RuneStart(chunk[i-1]) {
						if !utf8.FullRune(chunk[i-1 : len(chunk)]) {
							lastSafe = i - 1
						}
						break
					}
				}
				safe := chunk[:lastSafe]
				
				if len(safe) == 0 && len(chunk) >= 4 {
					safe = chunk
					remainder = nil
				} else {
					remainder = make([]byte, len(chunk[lastSafe:]))
					copy(remainder, chunk[lastSafe:])
				}

				if len(safe) > 0 {
					text := toUTF8(safe)
					writeMessage([]byte(text))
				}
			}
			if err != nil {
				if len(remainder) > 0 {
					writeMessage([]byte(toUTF8(remainder)))
				}
				return
			}
		}
	}

	wg.Add(2)
	go readOutput(stdout)
	go readOutput(stderr)

	// 启动 ping 协程
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(constant.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				connMu.Lock()
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					connMu.Unlock()
					return
				}
				connMu.Unlock()
			case <-pingDone:
				return
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// 过滤调整窗口大小的消息（Windows Pipe 模式不支持调整尺寸，需过滤掉避免写入 stdin）
		if len(message) > 0 && message[0] == '{' {
			var resizeMsg struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(message, &resizeMsg); err == nil && resizeMsg.Type == "resize" {
				continue
			}
		}

		// 检测是否收到 Ctrl+C (ASCII 3)
		if len(message) == 1 && message[0] == 3 {
			windows.InterruptProcessGroup(cmd.Process.Pid)
		}

		if _, err := stdin.Write(message); err != nil {
			break
		}
	}

	close(pingDone)
	cmd.Process.Kill()
	cmd.Wait()

	if stdinCloser, ok := stdin.(io.Closer); ok {
		stdinCloser.Close()
	}
	if stdoutCloser, ok := stdout.(io.Closer); ok {
		stdoutCloser.Close()
	}
	if stderrCloser, ok := stderr.(io.Closer); ok {
		stderrCloser.Close()
	}

	wg.Wait()
}

// ExecuteShellCommand 执行单个命令并返回结果
func (tc *TerminalController) ExecuteShellCommand(c *gin.Context) {
	// 演示模式下禁止执行命令
	if constant.DemoMode {
		utils.BadRequest(c, "演示模式下不能执行命令")
		return
	}

	var req struct {
		Command string `json:"command" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	cmd := utils.NewShellCommandCmd(req.Command)
	userID := c.GetString("userID")
	if userID == "" {
		userID = "1" // 与 WebSocket 终端保持一致，保留原有兜底行为
	}
	cmd.Env = tc.buildTerminalEnv(userID)
	output, err := cmd.CombinedOutput()

	if err != nil {
		utils.Success(c, gin.H{
			"output": string(output),
			"error":  err.Error(),
		})
		return
	}

	utils.Success(c, gin.H{
		"output": string(output),
	})
}

func (tc *TerminalController) buildTerminalEnv(userID string, extraEnvs ...string) []string {
	env := os.Environ()
	env = append(env, extraEnvs...)

	// 注入 baihu 命令运行时路径与配置，保证在终端里手动执行 baihu 子命令时
	// 仍然能连接到与主服务一致的数据库，而不是回退到默认 sqlite。
	if absBinDir, err := filepath.Abs(filepath.Join(constant.DataDir, "bin")); err == nil {
		pathStr := absBinDir + string(os.PathListSeparator) + os.Getenv("PATH")
		env = append(env, "PATH="+pathStr)
	}
	env = append(env, utils.BuildRuntimeProcessEnv()...)

	// 注入环境变量（支持同名合并）
	if tc.envService != nil {
		envVars := tc.envService.GetFormattedEnvVarsByUserID(userID)
		env = append(env, envVars...)
	}

	// 为 Docker 环境或二进制版本注入所有 mise 已安装 Node 的全局依赖路径到 NODE_PATH (Issue-90)
	if !utils.IsInDocker() || (!strings.Contains(os.Args[0], "go-build") && !strings.Contains(os.Args[0], "tmp")) {
		versions, _ := utils.ListMiseInstalledVersions("node")
		var nodePaths []string
		for _, v := range versions {
			if p := utils.GetMiseNodePath(v); p != "" {
				nodePaths = append(nodePaths, p)
			}
		}
		if len(nodePaths) > 0 {
			sep := windows.GetPathSeparator()
			env = append(env, "NODE_PATH="+strings.Join(nodePaths, sep))
		}
	}

	return env
}

// GetCommands 获取所有可用的 cmd 列表及说明
func (tc *TerminalController) GetCommands(c *gin.Context) {
	var cmds []map[string]string
	for _, cmdInfo := range constant.Commands {
		cmds = append(cmds, map[string]string{
			"name":        cmdInfo.Name,
			"description": cmdInfo.Description,
		})
	}
	utils.Success(c, cmds)
}
