package message

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
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

type qywxAppTokenResponse struct {
	Code        int    `json:"errcode"`
	Msg         string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type qywxAppSendResponse struct {
	Code int    `json:"errcode"`
	Msg  string `json:"errmsg"`
}

type QyWeiXinApp struct {
	CorpID   string
	AgentID  string
	Secret   string
	ApiHost  string
	ProxyURL string
}

type appTokenCache struct {
	Token     string
	ExpiresAt time.Time
}

var (
	appTokenCacheMap sync.Map // key: CorpID_Secret, value: appTokenCache
)

// GetToken 获取 access_token (带内存缓存)
func (q *QyWeiXinApp) GetToken() (string, error) {
	cacheKey := q.CorpID + "_" + q.Secret

	if val, ok := appTokenCacheMap.Load(cacheKey); ok {
		cache := val.(appTokenCache)
		// 提前5分钟失效以保证稳定性
		if time.Now().Add(5 * time.Minute).Before(cache.ExpiresAt) {
			return cache.Token, nil
		}
	}

	// 从微信 API 获取
	token, expiresIn, err := q.requestNewToken()
	if err != nil {
		return "", err
	}

	// 存入缓存
	appTokenCacheMap.Store(cacheKey, appTokenCache{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
	})

	return token, nil
}

// requestNewToken 向企业微信接口获取新 Token
func (q *QyWeiXinApp) requestNewToken() (string, int, error) {
	apiHost := "https://qyapi.weixin.qq.com"
	if q.ApiHost != "" {
		apiHost = q.ApiHost
	}
	apiURL := fmt.Sprintf("%s/cgi-bin/gettoken?corpid=%s&corpsecret=%s", apiHost, q.CorpID, q.Secret)

	client := q.getHTTPClient()
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	var r qywxAppTokenResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", 0, err
	}

	if r.Code != 0 {
		return "", 0, fmt.Errorf("qyweixin app gettoken err: %d - %s", r.Code, r.Msg)
	}

	if r.AccessToken == "" {
		return "", 0, errors.New("qyweixin app gettoken returned empty token")
	}

	return r.AccessToken, r.ExpiresIn, nil
}

// SendMessage 发送统一包装消息
func (q *QyWeiXinApp) SendMessage(toUser, toParty, toTag, msgType string, contentMap map[string]interface{}) ([]byte, error) {
	token, err := q.GetToken()
	if err != nil {
		return nil, err
	}

	apiHost := "https://qyapi.weixin.qq.com"
	if q.ApiHost != "" {
		apiHost = q.ApiHost
	}
	apiURL := fmt.Sprintf("%s/cgi-bin/message/send?access_token=%s", apiHost, token)

	agentID, _ := strconv.Atoi(q.AgentID)
	msg := map[string]interface{}{
		"touser":   toUser,
		"toparty":  toParty,
		"totag":    toTag,
		"msgtype":  msgType,
		"agentid":  agentID,
		"safe":     0,
		msgType:    contentMap,
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	client := q.getHTTPClient()
	resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r qywxAppSendResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return body, err
	}

	if r.Code != 0 {
		return body, fmt.Errorf("qyweixin app send err: %d - %s", r.Code, r.Msg)
	}

	return body, nil
}

// SendTextMessage 文本消息
func (q *QyWeiXinApp) SendTextMessage(toUser, toParty, toTag, text string) ([]byte, error) {
	content := map[string]interface{}{
		"content": text,
	}
	return q.SendMessage(toUser, toParty, toTag, "text", content)
}

// SendMarkdownMessage Markdown消息
func (q *QyWeiXinApp) SendMarkdownMessage(toUser, toParty, toTag, text string) ([]byte, error) {
	content := map[string]interface{}{
		"content": text,
	}
	return q.SendMessage(toUser, toParty, toTag, "markdown", content)
}

// getHTTPClient 获取带代理的 HTTP 客户端
func (q *QyWeiXinApp) getHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if q.ProxyURL != "" && q.ApiHost == "" {
		proxyURL, err := url.Parse(q.ProxyURL)
		if err == nil {
			if strings.HasPrefix(strings.ToLower(q.ProxyURL), "socks5://") {
				dialer, err := q.createSOCKS5Dialer(proxyURL)
				if err == nil {
					client.Transport = &http.Transport{
						DialContext: dialer.DialContext,
					}
				}
			} else {
				client.Transport = &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				}
			}
		}
	}

	return client
}

// createSOCKS5Dialer 创建 SOCKS5 拨号器
func (q *QyWeiXinApp) createSOCKS5Dialer(proxyURL *url.URL) (proxy.ContextDialer, error) {
	host := proxyURL.Host
	var auth *proxy.Auth
	if proxyURL.User != nil {
		password, _ := proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     proxyURL.User.Username(),
			Password: password,
		}
	}

	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	dialer, err := proxy.SOCKS5("tcp", host, auth, baseDialer)
	if err != nil {
		return nil, err
	}

	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New("failed to convert to ContextDialer")
	}

	return contextDialer, nil
}
