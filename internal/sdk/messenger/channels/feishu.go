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


type FeishuChannel struct{ *BaseChannel }

func NewFeishuChannel() Channel {
	return &FeishuChannel{NewBaseChannel(ChannelFeishu, []string{FormatTypeMarkdown, FormatTypeText})}
}

func (c *FeishuChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	accessToken := config.GetString("access_token")
	secret := config.GetString("secret")

	if accessToken == "" {
		return SendError("feishu config missing: access_token is required"), nil
	}

	contentType, formattedContent := c.FormatContent(msg)
	atMobiles := msg.GetAtMobiles()
	atUserIds := msg.GetAtUserIds()
	atList := append(atMobiles, atUserIds...)
	if msg.AtAll {
		atList = append(atList, "all")
	}

	cli := message.Feishu{AccessToken: accessToken, Secret: secret}
	var res []byte
	var err error

	switch contentType {
	case FormatTypeText:
		res, err = cli.SendMessageText(formattedContent, atList...)
	case FormatTypeMarkdown:
		res, err = cli.SendMessageMarkdown(msg.Title, formattedContent, atList...)
	default:
		return SendError("未知的飞书发送内容类型：%s", contentType), nil
	}

	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
