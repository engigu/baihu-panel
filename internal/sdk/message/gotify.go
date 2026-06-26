package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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


type gotifyResponse struct {
	Id        int    `json:"id"`
	Message   string `json:"message"`
	ErrorCode int    `json:"errorCode"`
}

type Gotify struct {
	Url      string
	Token    string
	Priority int
}

func (g *Gotify) Request(title, content string) ([]byte, error) {
	// Construct the URL with token
	u, err := url.Parse(fmt.Sprintf("%s/message", g.Url))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("token", g.Token)
	u.RawQuery = q.Encode()

	data := map[string]interface{}{
		"title":    title,
		"message":  content,
		"priority": g.Priority,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(u.String(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r gotifyResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		return body, err
	}

	if r.Id == 0 {
		return body, fmt.Errorf("gotify response error: %s", string(body))
	}

	return body, nil
}
