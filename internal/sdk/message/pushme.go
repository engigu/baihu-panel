package message

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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


type PushMe struct {
	PushKey string
	URL     string
	Date    string
	Type    string
}

func (p *PushMe) Request(title, content string) (string, error) {
	apiURL := p.URL
	if apiURL == "" {
		apiURL = "https://push.i-i.me/"
	}

	data := url.Values{}
	data.Set("push_key", p.PushKey)
	data.Set("title", title)
	data.Set("content", content)
	if p.Date != "" {
		data.Set("date", p.Date)
	}
	if p.Type != "" {
		data.Set("type", p.Type)
	}

	resp, err := http.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 200 && string(body) == "success" {
		return string(body), nil
	}

	return string(body), fmt.Errorf("PushMe response error: %s", string(body))
}
