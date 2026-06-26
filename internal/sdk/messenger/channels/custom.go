package channels

import (
	"encoding/json"

	"github.com/engigu/baihu-panel/internal/sdk/message"
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


type CustomChannel struct{ *BaseChannel }

func NewCustomChannel() Channel {
	return &CustomChannel{NewBaseChannel(ChannelCustom, []string{FormatTypeText})}
}

func (c *CustomChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	webhook := config.GetString("webhook")
	body := config.GetString("body")
	headersStr := config.GetString("headers")

	if webhook == "" {
		return SendError("custom config missing: webhook is required"), nil
	}

	var headers map[string]string
	if headersStr != "" {
		if err := json.Unmarshal([]byte(headersStr), &headers); err != nil {
			return SendError("custom config error: headers must be a valid JSON object"), nil
		}
	}

	_, formattedContent := c.FormatContent(msg)
	cli := message.CustomWebhook{}

	// 替换 body 模板中的 TEXT 占位符
	bodyStr := body
	if bodyStr != "" {
		bodyStr = replaceBodyPlaceholder(bodyStr, formattedContent)
	} else {
		bodyStr = formattedContent
	}

	res, err := cli.Request(webhook, bodyStr, headers)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
