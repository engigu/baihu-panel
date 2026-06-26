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


type NtfyChannel struct{ *BaseChannel }

func NewNtfyChannel() Channel {
	return &NtfyChannel{NewBaseChannel(ChannelNtfy, []string{FormatTypeText})}
}

func (c *NtfyChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	topic := config.GetString("topic")
	if topic == "" {
		return SendError("ntfy config missing: topic is required"), nil
	}

	cli := message.Ntfy{
		Url:      config.GetString("url"),
		Topic:    topic,
		Priority: config.GetString("priority"),
		Icon:     config.GetString("icon"),
		Token:    config.GetString("token"),
		Username: config.GetString("username"),
		Password: config.GetString("password"),
		Actions:  config.GetString("actions"),
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
