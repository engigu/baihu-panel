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


type WxPusher struct {
	AppToken      string   `json:"appToken"`
	Content       string   `json:"content"`
	ContentType   int      `json:"contentType"`
	TopicIds      []int    `json:"topicIds,omitempty"`
	Uids          []string `json:"uids,omitempty"`
	Url           string   `json:"url,omitempty"`
	VerifyPayType int      `json:"verifyPayType,omitempty"`
}

type wxPusherResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Uid       string `json:"uid"`
		TopicId   int    `json:"topicId"`
		MessageId int    `json:"messageId"`
		Code      int    `json:"code"`
		Status    string `json:"status"`
	} `json:"data"`
}

func (w *WxPusher) Send() (string, error) {
	apiUrl := "https://wxpusher.zjiecode.com/api/send/message"

	body, err := json.Marshal(w)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(apiUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res wxPusherResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return string(respBody), err
	}

	if res.Code == 1000 {
		return string(respBody), nil
	}

	return string(respBody), fmt.Errorf("WxPusher error: %s (code: %d)", res.Msg, res.Code)
}
