package mcp

import (
	"fmt"
	"os"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/logger"
	internalmcp "github.com/engigu/baihu-panel/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func printMainHelp() {
	fmt.Fprintf(os.Stderr, "\n白虎面板 Model Context Protocol (MCP) 服务工具\n\n")
	fmt.Fprintf(os.Stderr, "用法:\n")
	fmt.Fprintf(os.Stderr, "  baihu mcp\n\n")
	fmt.Fprintf(os.Stderr, "说明:\n")
	fmt.Fprintf(os.Stderr, "  以标准输入输出 (Stdio JSON-RPC) 方式启动 MCP 服务，供 Claude Desktop、Cursor、Cline 等 AI 客户端直接调用。\n\n")
}

func Run(args []string) {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printMainHelp()
		return
	}

	// 关键防护：Stdio 协议强依赖标准输出 (stdout) 传输 JSON-RPC 数据包，
	// 任何系统底层打印到 stdout 的字符都会破坏协议流。因此将面板内部所有日志重定向到 stderr。
	core := logger.NewCustomCore(zapcore.WarnLevel, zapcore.AddSync(os.Stderr))
	logger.SetOutput(zap.New(core))

	if err := clibase.InitContext(true); err != nil {
		fmt.Fprintf(os.Stderr, "初始化白虎环境失败: %v\n", err)
		os.Exit(1)
	}

	deps := internalmcp.InitDefaultDeps()
	mcpServer := internalmcp.NewBaihuMCPServer(deps)

	if err := server.ServeStdio(mcpServer); err != nil {
		fmt.Fprintf(os.Stderr, "MCP Stdio 服务异常退出: %v\n", err)
		os.Exit(1)
	}
}
