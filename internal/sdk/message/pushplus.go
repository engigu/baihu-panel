package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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


type PushPlus struct {
	Token       string `json:"token"`
	Topic       string `json:"topic,omitempty"`
	Template    string `json:"template,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Webhook     string `json:"webhook,omitempty"`
	CallbackUrl string `json:"callbackUrl,omitempty"`
	To          string `json:"to,omitempty"`
}

type pushPlusData struct {
	PushPlus
	Title   string `json:"title"`
	Content string `json:"content"`
}

type pushPlusResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

func (p *PushPlus) Request(title, content string) (string, error) {
	url := "https://www.pushplus.plus/send"

	data := pushPlusData{
		PushPlus: *p,
		Title:    title,
		Content:  content,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		// Try old URL if first one fails or as fallback
		urlOld := "http://pushplus.hxtrip.com/send"
		resp, err = http.Post(urlOld, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return "", err
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res pushPlusResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return string(respBody), err
	}

	if res.Code == 200 {
		return string(respBody), nil
	}

	return string(respBody), fmt.Errorf("PushPlus error: %s (code: %d)", res.Msg, res.Code)
}
