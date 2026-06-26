package channels

import (
	"fmt"
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


type EmailChannel struct{ *BaseChannel }

func NewEmailChannel() Channel {
	return &EmailChannel{NewBaseChannel(ChannelEmail, []string{FormatTypeHTML, FormatTypeText})}
}

func (c *EmailChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	server := config.GetString("server")
	portStr := config.GetString("port")
	account := config.GetString("account")
	passwd := config.GetString("passwd")
	fromName := config.GetString("from_name")
	toAccount := config.GetString("to_account")

	if server == "" || account == "" || passwd == "" {
		return SendError("email config missing: server, account, passwd are required"), nil
	}
	if toAccount == "" {
		return SendError("email config missing: to_account is required"), nil
	}

	port, _ := strconv.Atoi(portStr)
	contentType, formattedContent := c.FormatContent(msg)

	var emailer message.EmailMessage
	emailer.Init(server, port, account, passwd, fromName)

	var errMsg string
	switch contentType {
	case FormatTypeText:
		errMsg = emailer.SendTextMessage(toAccount, msg.Title, formattedContent)
	case FormatTypeHTML:
		errMsg = emailer.SendHtmlMessage(toAccount, msg.Title, formattedContent)
	default:
		errMsg = fmt.Sprintf("未知的邮件发送内容类型：%s", contentType)
	}

	if errMsg != "" {
		return ErrorResultStr("", errMsg), nil
	}
	return SuccessResult(""), nil
}
