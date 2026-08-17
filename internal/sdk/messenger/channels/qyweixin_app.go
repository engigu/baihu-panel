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

type QyWeiXinAppChannel struct{ *BaseChannel }

func NewQyWeiXinAppChannel() Channel {
	return &QyWeiXinAppChannel{NewBaseChannel(ChannelQyWeiXinApp, []string{FormatTypeMarkdown, FormatTypeText})}
}

func (c *QyWeiXinAppChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	corpID := config.GetString("corpid")
	agentID := config.GetString("agentid")
	secret := config.GetString("secret")

	if corpID == "" || agentID == "" || secret == "" {
		return SendError("qyweixin app config missing: corpid, agentid, secret are required"), nil
	}

	toUser := config.GetString("to_user")
	toParty := config.GetString("to_party")
	toTag := config.GetString("to_tag")
	apiHost := config.GetString("api_host")
	proxyURL := config.GetString("proxy_url")

	// 如果没有指定接收者，默认发送给所有成员
	if toUser == "" && toParty == "" && toTag == "" {
		toUser = "@all"
	}

	contentType, formattedContent := c.FormatContent(msg)

	cli := message.QyWeiXinApp{
		CorpID:   corpID,
		AgentID:  agentID,
		Secret:   secret,
		ApiHost:  apiHost,
		ProxyURL: proxyURL,
	}

	var res []byte
	var err error

	switch contentType {
	case FormatTypeText:
		res, err = cli.SendTextMessage(toUser, toParty, toTag, formattedContent)
	case FormatTypeMarkdown:
		res, err = cli.SendMarkdownMessage(toUser, toParty, toTag, formattedContent)
	default:
		return SendError("未知的企业微信应用发送内容类型：%s", contentType), nil
	}

	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
