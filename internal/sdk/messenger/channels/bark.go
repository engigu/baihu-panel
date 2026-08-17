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


type BarkChannel struct{ *BaseChannel }

func NewBarkChannel() Channel {
	return &BarkChannel{NewBaseChannel(ChannelBark, []string{FormatTypeText})}
}

func (c *BarkChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	pushKey := config.GetString("push_key")
	if pushKey == "" {
		return SendError("bark config missing: push_key is required"), nil
	}

	cli := message.Bark{
		PushKey:  pushKey,
		Archive:  config.GetString("archive"),
		Group:    config.GetString("group"),
		Sound:    config.GetString("sound"),
		Icon:     config.GetString("icon"),
		Level:    config.GetString("level"),
		URL:      config.GetString("url"),
		Key:      config.GetString("key"),
		IV:       config.GetString("iv"),
		Server:   config.GetString("server"),
		Badge:    config.GetString("badge"),
		Copy:     config.GetString("copy"),
		AutoCopy: config.GetString("auto_copy"),
		ProxyURL: config.GetString("proxy_url"),
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
