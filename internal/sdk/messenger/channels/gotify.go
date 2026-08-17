package channels

import (
	"github.com/engigu/baihu-panel/internal/sdk/message"
	"strconv"
)

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


type GotifyChannel struct{ *BaseChannel }

func NewGotifyChannel() Channel {
	return &GotifyChannel{NewBaseChannel(ChannelGotify, []string{FormatTypeText})}
}

func (c *GotifyChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	url := config.GetString("url")
	token := config.GetString("token")

	if url == "" || token == "" {
		return SendError("gotify config missing: url and token are required"), nil
	}

	priority, _ := strconv.Atoi(config.GetString("priority"))
	cli := message.Gotify{
		Url:      url,
		Token:    token,
		Priority: priority,
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
