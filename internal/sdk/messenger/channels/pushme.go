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


type PushMeChannel struct{ *BaseChannel }

func NewPushMeChannel() Channel {
	return &PushMeChannel{NewBaseChannel(ChannelPushMe, []string{FormatTypeText})}
}

func (c *PushMeChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	pushKey := config.GetString("push_key")
	if pushKey == "" {
		return SendError("pushme config missing: push_key is required"), nil
	}

	cli := message.PushMe{
		PushKey: pushKey,
		URL:     config.GetString("url"),
		Date:    config.GetString("date"),
		Type:    config.GetString("type"),
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(res, err), nil
	}
	return SuccessResult(res), nil
}
