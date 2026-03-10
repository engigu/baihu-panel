package controllers

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unicode/utf8"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"

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
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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

	if runtime.GOOS == "windows" {
		tc.handlePipeMode(conn, userID)
	} else {
		tc.handlePtyMode(conn, userID)
	}
}

// handlePtyMode 使用 PTY 处理终端（Unix/macOS）
func (tc *TerminalController) handlePtyMode(conn *websocket.Conn, userID string) {
	// 发送 PTY 模式标识
	conn.WriteMessage(websocket.TextMessage, []byte("__PTY_MODE__"))

	cmd := utils.NewShellCmd()

	if absDir, err := filepath.Abs(constant.ScriptsWorkDir); err == nil {
		cmd.Dir = absDir
	}

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// 注入 baihu 命令环境变量
	if absBinDir, err := filepath.Abs(filepath.Join(constant.DataDir, "bin")); err == nil {
		pathStr := absBinDir + string(os.PathListSeparator) + os.Getenv("PATH")
		cmd.Env = append(cmd.Env, "PATH="+pathStr)
	}

	// 注入环境变量（支持同名合并）
	envVars := tc.envService.GetFormattedEnvVarsByUserID(userID)
	cmd.Env = append(cmd.Env, envVars...)

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
		conn.WriteMessage(websocket.TextMessage, data)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				text := toUTF8(buf[:n])
				writeMessage([]byte(text))
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if _, err := ptmx.Write(message); err != nil {
			break
		}
	}

	cmd.Process.Kill()
	cmd.Wait()
	wg.Wait()
}

// handlePipeMode 使用 pipe 处理终端（Windows）
func (tc *TerminalController) handlePipeMode(conn *websocket.Conn, userID string) {
	// 发送 pipe 模式标识
	conn.WriteMessage(websocket.TextMessage, []byte("__PIPE_MODE__"))

	cmd := utils.NewShellCmd()

	if absDir, err := filepath.Abs(constant.ScriptsWorkDir); err == nil {
		cmd.Dir = absDir
	}

	// 注入环境变量
	cmd.Env = os.Environ()

	// 注入 baihu 命令环境变量
	if absBinDir, err := filepath.Abs(filepath.Join(constant.DataDir, "bin")); err == nil {
		pathStr := absBinDir + string(os.PathListSeparator) + os.Getenv("PATH")
		cmd.Env = append(cmd.Env, "PATH="+pathStr)
	}

	// 注入环境变量（支持同名合并）
	envVars := tc.envService.GetFormattedEnvVarsByUserID(userID)
	cmd.Env = append(cmd.Env, envVars...)

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
		conn.WriteMessage(websocket.TextMessage, data)
	}

	readOutput := func(reader io.Reader) {
		defer wg.Done()
		defer func() { recover() }()
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				text := toUTF8(buf[:n])
				writeMessage([]byte(text))
			}
		}
	}

	wg.Add(2)
	go readOutput(stdout)
	go readOutput(stderr)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if _, err := stdin.Write(message); err != nil {
			break
		}
	}

	stdin.Close()
	cmd.Process.Kill()
	cmd.Wait()
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
