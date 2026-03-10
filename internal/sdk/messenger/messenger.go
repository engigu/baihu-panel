// Package messenger 提供统一的消息发送SDK
//
// 此包可独立于 Message-Push-Nest 的业务层（数据库、HTTP路由等）使用
// 适合在其他服务中直接引入来发送消息
//
// 快速使用：
//
//	result, err := messenger.Send("Telegram", messenger.ChannelConfig{
//	    "bot_token": "your-bot-token",
//	    "chat_id":   "your-chat-id",
//	}, &messenger.Message{
//	    Title: "Hello",
//	    Text:  "World",
//	})
package messenger

import (
	"fmt"
	"github.com/engigu/baihu-panel/internal/sdk/messenger/channels"
	"sync"
)

// 重导出 channels 包的类型，方便外部使用
type (
	Channel       = channels.Channel
	Message       = channels.Message
	ChannelConfig = channels.ChannelConfig
	Result        = channels.Result
	BaseChannel   = channels.BaseChannel
)

// 重导出常量
const (
	FormatTypeText     = channels.FormatTypeText
	FormatTypeHTML     = channels.FormatTypeHTML
	FormatTypeMarkdown = channels.FormatTypeMarkdown

	ChannelEmail           = channels.ChannelEmail
	ChannelDtalk           = channels.ChannelDtalk
	ChannelQyWeiXin        = channels.ChannelQyWeiXin
	ChannelFeishu          = channels.ChannelFeishu
	ChannelCustom          = channels.ChannelCustom
	ChannelWeChatOFAccount = channels.ChannelWeChatOFAccount
	ChannelAliyunSMS       = channels.ChannelAliyunSMS
	ChannelTelegram        = channels.ChannelTelegram
	ChannelBark            = channels.ChannelBark
	ChannelPushMe          = channels.ChannelPushMe
	ChannelNtfy            = channels.ChannelNtfy
	ChannelGotify          = channels.ChannelGotify
	ChannelPushPlus        = channels.ChannelPushPlus
)

// 重导出辅助函数
var (
	SuccessResult  = channels.SuccessResult
	ErrorResult    = channels.ErrorResult
	ErrorResultStr = channels.ErrorResultStr
	SendError      = channels.SendError
	NewBaseChannel = channels.NewBaseChannel
)

// channelFactory 渠道工厂注册表
var (
	channelFactories = map[string]func() Channel{}
	factoryMu        sync.RWMutex
)

func init() {
	// 注册所有内置渠道
	RegisterChannel(ChannelEmail, func() Channel { return channels.NewEmailChannel() })
	RegisterChannel(ChannelDtalk, func() Channel { return channels.NewDtalkChannel() })
	RegisterChannel(ChannelQyWeiXin, func() Channel { return channels.NewQyWeiXinChannel() })
	RegisterChannel(ChannelFeishu, func() Channel { return channels.NewFeishuChannel() })
	RegisterChannel(ChannelTelegram, func() Channel { return channels.NewTelegramChannel() })
	RegisterChannel(ChannelBark, func() Channel { return channels.NewBarkChannel() })
	RegisterChannel(ChannelNtfy, func() Channel { return channels.NewNtfyChannel() })
	RegisterChannel(ChannelGotify, func() Channel { return channels.NewGotifyChannel() })
	RegisterChannel(ChannelPushMe, func() Channel { return channels.NewPushMeChannel() })
	RegisterChannel(ChannelCustom, func() Channel { return channels.NewCustomChannel() })
	RegisterChannel(ChannelWeChatOFAccount, func() Channel { return channels.NewWeChatOFAccountChannel() })
	RegisterChannel(ChannelAliyunSMS, func() Channel { return channels.NewAliyunSMSChannel() })
	RegisterChannel(ChannelPushPlus, func() Channel { return channels.NewPushPlusChannel() })
}

// RegisterChannel 注册自定义渠道（可用于扩展）
func RegisterChannel(channelType string, factory func() Channel) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	channelFactories[channelType] = factory
}

// GetChannel 获取渠道实例
func GetChannel(channelType string) (Channel, error) {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	factory, ok := channelFactories[channelType]
	if !ok {
		return nil, fmt.Errorf("未知的渠道类型: %s", channelType)
	}
	return factory(), nil
}

// ListChannels 列出所有已注册的渠道类型
func ListChannels() []string {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	types := make([]string, 0, len(channelFactories))
	for t := range channelFactories {
		types = append(types, t)
	}
	return types
}

// Send 发送消息的便捷函数
//
// 参数：
//   - channelType: 渠道类型（如 "Telegram", "Dtalk" 等）
//   - config: 渠道必要的认证配置
//   - msg: 消息内容
//
// 使用示例：
//
//	result, err := messenger.Send("Ntfy", messenger.ChannelConfig{
//	    "topic": "my-topic",
//	}, &messenger.Message{
//	    Title: "Alert",
//	    Text:  "Something happened!",
//	})
func Send(channelType string, config ChannelConfig, msg *Message) (*Result, error) {
	ch, err := GetChannel(channelType)
	if err != nil {
		return nil, err
	}
	return ch.Send(config, msg)
}

// Client 消息发送客户端（支持预设默认配置）
type Client struct {
	defaultConfigs map[string]ChannelConfig
	mu             sync.RWMutex
}

// NewClient 创建消息发送客户端
func NewClient() *Client {
	return &Client{
		defaultConfigs: make(map[string]ChannelConfig),
	}
}

// SetDefaultConfig 为指定渠道设置默认配置
//
// 使用示例：
//
//	client := messenger.NewClient()
//	client.SetDefaultConfig("Telegram", messenger.ChannelConfig{
//	    "bot_token": "default-token",
//	    "chat_id":   "default-chat",
//	})
//	// 后续发送时不需要再传 config 的相关字段
//	result, err := client.Send("Telegram", nil, &messenger.Message{...})
func (c *Client) SetDefaultConfig(channelType string, config ChannelConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultConfigs[channelType] = config
}

// Send 使用客户端发送消息（会合并默认配置）
func (c *Client) Send(channelType string, config ChannelConfig, msg *Message) (*Result, error) {
	mergedConfig := c.mergeConfig(channelType, config)
	return Send(channelType, mergedConfig, msg)
}

func (c *Client) mergeConfig(channelType string, config ChannelConfig) ChannelConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	defaultConfig, hasDefault := c.defaultConfigs[channelType]
	if !hasDefault && config == nil {
		return ChannelConfig{}
	}
	if !hasDefault {
		return config
	}
	if config == nil {
		return defaultConfig
	}

	// 合并：config 覆盖 defaultConfig
	merged := make(ChannelConfig, len(defaultConfig)+len(config))
	for k, v := range defaultConfig {
		merged[k] = v
	}
	for k, v := range config {
		merged[k] = v
	}
	return merged
}
