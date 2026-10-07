package message

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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

type ServerChan struct {
	SendKey string `json:"sendkey"`
	Channel string `json:"channel,omitempty"`
	OpenID  string `json:"openid,omitempty"`
	APIURL  string `json:"api_url,omitempty"`
}

type serverChanResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Errno   int    `json:"errno"`
	ErrMsg  string `json:"errmsg"`
	Data    struct {
		PushID  string `json:"pushid"`
		ReadKey string `json:"readkey"`
		Error   string `json:"error"`
		Errno   int    `json:"errno"`
	} `json:"data"`
}

func (s *ServerChan) Request(title, desp string) (string, error) {
	sendKey := strings.TrimSpace(s.SendKey)
	if sendKey == "" {
		return "", fmt.Errorf("ServerChan sendkey is required")
	}

	apiURL := strings.TrimSpace(s.APIURL)
	if apiURL == "" {
		if strings.HasPrefix(strings.ToUpper(sendKey), "SCU") {
			apiURL = fmt.Sprintf("https://sc.ftqq.com/%s.send", sendKey)
		} else {
			apiURL = fmt.Sprintf("https://sctapi.ftqq.com/%s.send", sendKey)
		}
	} else {
		if strings.Contains(apiURL, "{sendkey}") {
			apiURL = strings.ReplaceAll(apiURL, "{sendkey}", sendKey)
		} else if !strings.HasSuffix(apiURL, ".send") {
			apiURL = fmt.Sprintf("%s/%s.send", strings.TrimSuffix(apiURL, "/"), sendKey)
		}
	}

	form := url.Values{}
	form.Set("title", title)
	form.Set("desp", desp)
	if s.Channel != "" {
		form.Set("channel", s.Channel)
	}
	if s.OpenID != "" {
		form.Set("openid", s.OpenID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res serverChanResponse
	if err := json.Unmarshal(body, &res); err != nil {
		if resp.StatusCode == http.StatusOK {
			return string(body), nil
		}
		return string(body), fmt.Errorf("ServerChan HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Turbo 版判断 code == 0，老版 SCKEY 判断 errno == 0
	if res.Code == 0 && res.Errno == 0 {
		return string(body), nil
	}

	errMsg := res.Message
	if errMsg == "" {
		errMsg = res.ErrMsg
	}
	if errMsg == "" {
		errMsg = res.Data.Error
	}
	if errMsg == "" {
		errMsg = string(body)
	}

	return string(body), fmt.Errorf("ServerChan error: %s (code: %d)", errMsg, res.Code)
}
