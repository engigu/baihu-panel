package channels

import (
	"fmt"
	"github.com/engigu/baihu-panel/internal/sdk/message"
	"strconv"
	"strings"
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


type WxPusherChannel struct{ *BaseChannel }

func NewWxPusherChannel() Channel {
	return &WxPusherChannel{NewBaseChannel(ChannelWxPusher, []string{FormatTypeText})}
}

func (c *WxPusherChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	appToken := config.GetString("app_token")
	if appToken == "" {
		return SendError("wxpusher config missing: app_token is required"), nil
	}

	uidsStr := config.GetString("uids")
	topicIdsStr := config.GetString("topic_ids")
	verifyPayTypeStr := config.GetString("verify_pay_type")

	if uidsStr == "" && topicIdsStr == "" {
		return SendError("wxpusher config missing: uids or topic_ids is required"), nil
	}

	var uids []string
	if uidsStr != "" {
		uids = strings.Split(uidsStr, ",")
		for i := range uids {
			uids[i] = strings.TrimSpace(uids[i])
		}
	}

	var topicIds []int
	if topicIdsStr != "" {
		ids := strings.Split(topicIdsStr, ",")
		for _, idStr := range ids {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.Atoi(idStr); err == nil {
				topicIds = append(topicIds, id)
			}
		}
	}

	verifyPayType := 0
	if verifyPayTypeStr != "" {
		if v, err := strconv.Atoi(verifyPayTypeStr); err == nil {
			verifyPayType = v
		}
	}

	_, formattedContent := c.FormatContent(msg)

	// 如果有标题，将标题和内容合并
	content := formattedContent
	if msg.Title != "" {
		content = fmt.Sprintf("%s\n\n%s", msg.Title, formattedContent)
	}

	cli := message.WxPusher{
		AppToken:      appToken,
		Content:       content,
		ContentType:   1, // 仅支持文字
		Uids:          uids,
		TopicIds:      topicIds,
		VerifyPayType: verifyPayType,
	}

	res, err := cli.Send()
	if err != nil {
		return ErrorResult(res, err), nil
	}
	return SuccessResult(res), nil
}
