package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/engigu/baihu-panel/internal/services"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerNotifyTools 注册消息通知与告警测试 (Notification Channels) 相关 MCP 工具
func registerNotifyTools(s *server.MCPServer, deps *Deps) {
	registerListNotifyChannelsTool(s, deps)
	registerTestNotificationTool(s, deps)
	registerSendNotificationTool(s, deps)
}

// 1. list_notify_channels
func registerListNotifyChannelsTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("list_notify_channels",
		mcp.WithDescription("列出白虎面板已配置的全部消息通知渠道（支持自动脱敏敏感机密）及系统支持的渠道类型"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListNotifyChannels(ctx, deps, req)
	})
}

func handleListNotifyChannels(_ context.Context, deps *Deps, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.NotifyService == nil {
		return mcp.NewToolResultError("NotifyService 未初始化"), nil
	}

	rawChannels := deps.NotifyService.GetChannels()

	type MaskedChannel struct {
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		Type      string            `json:"type"`
		Enabled   bool              `json:"enabled"`
		Config    map[string]string `json:"config"`
		CreatedAt string            `json:"created_at"`
	}

	sensitiveKeys := []string{"token", "secret", "password", "key", "webhook", "api_key", "bot_token"}

	masked := make([]MaskedChannel, 0, len(rawChannels))
	for _, ch := range rawChannels {
		safeConfig := make(map[string]string)
		for k, v := range ch.Config {
			isSensitive := false
			lowerK := strings.ToLower(k)
			for _, sk := range sensitiveKeys {
				if strings.Contains(lowerK, sk) {
					isSensitive = true
					break
				}
			}
			if isSensitive && len(v) > 6 {
				safeConfig[k] = v[:3] + "******" + v[len(v)-3:]
			} else if isSensitive {
				safeConfig[k] = "******"
			} else {
				safeConfig[k] = v
			}
		}

		masked = append(masked, MaskedChannel{
			ID:        ch.ID,
			Name:      ch.Name,
			Type:      ch.Type,
			Enabled:   ch.Enabled,
			Config:    safeConfig,
			CreatedAt: ch.CreatedAt.Time().Format("2006-01-02 15:04:05"),
		})
	}

	resData := map[string]interface{}{
		"total":                   len(masked),
		"channels":                masked,
		"supported_channel_types": services.SupportedChannelTypes,
	}
	data, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 2. test_notification
func registerTestNotificationTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("test_notification",
		mcp.WithDescription("向白虎面板中指定的通知渠道发送一条连通性测试消息，检验渠道配置与网络连通性是否正常"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("通知渠道 ID（可通过 list_notify_channels 获取）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleTestNotification(ctx, deps, req)
	})
}

func handleTestNotification(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.NotifyService == nil {
		return mcp.NewToolResultError("NotifyService 未初始化"), nil
	}

	channelID, err := req.RequireString("channel_id")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 channel_id"), nil
	}

	msg := &services.NotifyMessage{
		Title:   "🔔 白虎面板 MCP 连通性测试",
		Content: "这是一条来自白虎面板 MCP (Model Context Protocol) 服务的连通性测试通知，表明此渠道配置正确且可正常接收推送！",
		Format:  "text",
	}

	result := deps.NotifyService.SendByChannelID(channelID, msg)
	resData := map[string]interface{}{
		"channel_id": channelID,
		"success":    result.Success,
		"error":      result.Error,
	}
	data, _ := json.MarshalIndent(resData, "", "  ")
	if !result.Success {
		return mcp.NewToolResultError(fmt.Sprintf("渠道测试失败: %s\n%s", result.Error, string(data))), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// 3. send_notification
func registerSendNotificationTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("send_notification",
		mcp.WithDescription("借由白虎面板向指定或全部启用的通知渠道推送自定义告警、任务排查报告或状态消息"),
		mcp.WithString("title", mcp.Required(), mcp.Description("消息标题")),
		mcp.WithString("content", mcp.Required(), mcp.Description("消息详细内容（支持 Markdown）")),
		mcp.WithString("channel_id", mcp.Description("指定的通知渠道 ID；若留空则自动向所有已启用的渠道广播")),
		mcp.WithString("format", mcp.Description("内容格式：markdown (默认), text, html")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSendNotification(ctx, deps, req)
	})
}

func handleSendNotification(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.NotifyService == nil {
		return mcp.NewToolResultError("NotifyService 未初始化"), nil
	}

	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 title"), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 content"), nil
	}

	channelID := strings.TrimSpace(req.GetString("channel_id", ""))
	format := strings.TrimSpace(req.GetString("format", "markdown"))
	if format == "" {
		format = "markdown"
	}

	msg := &services.NotifyMessage{
		Title:   title,
		Content: content,
		Format:  format,
	}

	type ChannelSendResult struct {
		ChannelID   string `json:"channel_id"`
		ChannelName string `json:"channel_name"`
		Type        string `json:"type"`
		Success     bool   `json:"success"`
		Error       string `json:"error,omitempty"`
	}

	var sendResults []ChannelSendResult

	if channelID != "" {
		res := deps.NotifyService.SendByChannelID(channelID, msg)
		sendResults = append(sendResults, ChannelSendResult{
			ChannelID: channelID,
			Success:   res.Success,
			Error:     res.Error,
		})
	} else {
		channels := deps.NotifyService.GetChannels()
		for _, ch := range channels {
			if !ch.Enabled {
				continue
			}
			res := deps.NotifyService.SendToChannel(ch, msg)
			sendResults = append(sendResults, ChannelSendResult{
				ChannelID:   ch.ID,
				ChannelName: ch.Name,
				Type:        ch.Type,
				Success:     res.Success,
				Error:       res.Error,
			})
		}
	}

	allSuccess := true
	for _, r := range sendResults {
		if !r.Success {
			allSuccess = false
			break
		}
	}

	resData := map[string]interface{}{
		"success": allSuccess,
		"total":   len(sendResults),
		"results": sendResults,
	}
	data, _ := json.MarshalIndent(resData, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}
