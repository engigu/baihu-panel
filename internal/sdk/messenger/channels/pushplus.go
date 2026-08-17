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


type PushPlusChannel struct{ *BaseChannel }

func NewPushPlusChannel() Channel {
	return &PushPlusChannel{NewBaseChannel(ChannelPushPlus, []string{FormatTypeText, FormatTypeHTML, FormatTypeMarkdown})}
}

func (c *PushPlusChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	token := config.GetString("token")
	if token == "" {
		return SendError("pushplus config missing: token is required"), nil
	}

	cli := message.PushPlus{
		Token:       token,
		Topic:       config.GetString("topic"),
		Template:    config.GetString("template"),
		Channel:     config.GetString("channel"),
		Webhook:     config.GetString("webhook"),
		CallbackUrl: config.GetString("callback_url"),
		To:          config.GetString("to"),
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
