package mcp

import (
	"context"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// DefaultMaxConcurrentToolExecutions 限制 MCP 服务同时并发执行工具的协程数上限，防范并发协程风暴
	DefaultMaxConcurrentToolExecutions = 4
	// DefaultToolExecutionTimeout 单个工具执行获取协程令牌的最大等待超时时间
	DefaultToolExecutionTimeout = 15 * time.Second
)

// ToolConcurrencyLimiter 创建并返回限制工具执行并发协程数的中间件（有界信号量机制）
func ToolConcurrencyLimiter(limit int) server.ToolHandlerMiddleware {
	if limit <= 0 {
		limit = DefaultMaxConcurrentToolExecutions
	}
	sem := make(chan struct{}, limit)

	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				return next(ctx, req)
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(DefaultToolExecutionTimeout):
				return mcp.NewToolResultError("系统繁忙：当前并发执行的 MCP 工具协程数已达上限，排队超时"), nil
			}
		}
	}
}

// NewBaihuMCPServer 创建并初始化白虎面板的 MCP 服务实例
func NewBaihuMCPServer(deps *Deps) *server.MCPServer {
	if deps == nil {
		deps = InitDefaultDeps()
	}

	s := server.NewMCPServer(
		"baihu-panel",
		constant.Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithInstructions("白虎面板 (Baihu Panel) 是极致轻量、高性能的自动化定时任务调度平台。你可以通过本 MCP 服务查询/触发/编排任务、管理环境变量、查看实时执行日志以及读写本地脚本。"),
	)

	// 挂载并发控制中间件，严格限制同时执行工具的协程数上限（默认为 4）
	s.Use(ToolConcurrencyLimiter(DefaultMaxConcurrentToolExecutions))

	// 注册各类领域工具
	registerTaskTools(s, deps)
	registerLogTools(s, deps)
	registerEnvTools(s, deps)
	registerFileTools(s, deps)
	registerSystemTools(s, deps)
	registerAppTools(s, deps)
	registerRepoTools(s, deps)
	registerNotifyTools(s, deps)

	// 注册上下文资源
	registerResources(s, deps)

	// 注册提示词模板
	registerPrompts(s, deps)

	return s
}
