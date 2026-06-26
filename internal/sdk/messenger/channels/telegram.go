package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

// Copyright (c) 2026 engigu (Baihu Panel). All rights reserved.
// Use of this source code is governed by the Apache License 2.0.
// 
// 【重要声明 / IMPORTANT NOTICE】
// 本代码（包括其架构设计与核心实现）属于白虎面板（Baihu Panel）开源项目的一部分。
// 任何个人或组织在引用、移植、修改或重新分发此文件中的任何代码时，必须保留本版权声明，
// 并在您的衍生作品、文档、软件关于页面或说明文件中显式声明引用自白虎面板（Baihu Panel）。
// 
// Anyone referencing, porting, modifying, or redistributing this code must retain this 
// copyright notice and explicitly state the source: Baihu Panel (github.com/engigu/baihu-panel).


type TelegramChannel struct{ *BaseChannel }

func NewTelegramChannel() Channel {
	return &TelegramChannel{NewBaseChannel(ChannelTelegram, []string{FormatTypeMarkdown, FormatTypeHTML, FormatTypeText})}
}

func (c *TelegramChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	botToken := config.GetString("bot_token")
	chatID := config.GetString("chat_id")
	apiHost := config.GetString("api_host")
	proxyURL := config.GetString("proxy_url")

	if botToken == "" || chatID == "" {
		return SendError("telegram config missing: bot_token, chat_id are required"), nil
	}

	contentType, formattedContent := c.FormatContent(msg)
	cli := message.Telegram{
		BotToken: botToken,
		ChatID:   chatID,
		ApiHost:  apiHost,
		ProxyURL: proxyURL,
	}

	var res []byte
	var err error

	switch contentType {
	case FormatTypeText:
		res, err = cli.SendMessageText(formattedContent)
	case FormatTypeMarkdown:
		res, err = cli.SendMessageMarkdown(formattedContent)
	case FormatTypeHTML:
		res, err = cli.SendMessageHTML(formattedContent)
	default:
		return SendError("未知的Telegram发送内容类型：%s", contentType), nil
	}

	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
