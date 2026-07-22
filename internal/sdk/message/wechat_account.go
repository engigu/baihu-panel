package message

import (
	"github.com/engigu/baihu-panel/internal/logger"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"
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


type WeChatOFAccount struct {
	AppID      string
	AppSecret  string
	ToUser     string
	TemplateID string
	URL        string
}

// 使用内存缓存进行token的存储
var memory = cache.NewMemory()

func (cw *WeChatOFAccount) Send(title string, content string) (string, error) {
	wc := wechat.NewWechat()
	cfg := &offConfig.Config{
		AppID:     cw.AppID,
		AppSecret: cw.AppSecret,
		Cache:     memory,
	}
	officialAccount := wc.GetOfficialAccount(cfg)

	// 获取 Access Token
	_, err := officialAccount.GetAccessToken()
	if err != nil {
		logger.Errorf("获取access token失败:%s", err)
		return "", err
	}

	msgData := make(map[string]*message.TemplateDataItem)
	msgData["content"] = &message.TemplateDataItem{
		Value: content,
	}
	msgData["title"] = &message.TemplateDataItem{
		Value: title,
	}

	// 创建模板消息
	templateMessage := &message.TemplateMessage{
		ToUser:     cw.ToUser,
		TemplateID: cw.TemplateID,
		URL:        cw.URL,
		Data:       msgData,
	}

	// 发送模板消息
	_, err = officialAccount.GetTemplate().Send(templateMessage)
	if err != nil {
		logger.Errorf("发送模板消息失败: %s", err)
		return "", err
	}
	return "", nil
}
