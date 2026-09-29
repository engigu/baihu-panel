package message

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
	"github.com/engigu/baihu-panel/internal/utils"
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


type barkResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Bark struct {
	PushKey         string
	Archive         string
	Group           string
	Sound           string
	Icon            string
	Level           string
	URL             string
	Server          string
	Badge           string
	Copy            string
	AutoCopy        string
	ProxyURL        string // 可选的代理地址
	CipherEnable    string // 加密开关
	CipherAlgorithm string // 加密算法
	CipherMode      string // 加密模式
	CipherPadding   string // 加密填充
	CipherKey       string // 加密密钥
}

func (b *Bark) Request(title, content string) ([]byte, error) {
	data := map[string]interface{}{
		"device_key": b.PushKey,
		"title":      title,
		"body":       content,
	}
	if b.Archive != "" {
		data["isArchive"] = b.Archive
	}
	if b.Group != "" {
		data["group"] = b.Group
	}
	if b.Sound != "" {
		data["sound"] = b.Sound
	}
	if b.Icon != "" {
		data["icon"] = b.Icon
	}
	if b.Level != "" {
		data["level"] = b.Level
	}
	if b.URL != "" {
		data["url"] = b.URL
	}
	if b.Badge != "" {
		data["badge"] = b.Badge
	}
	if b.Copy != "" {
		data["copy"] = b.Copy
	}
	if b.AutoCopy != "" {
		data["autoCopy"] = b.AutoCopy
	}

	server := b.Server
	if server == "" {
		server = "https://api.day.app"
	}
	server = strings.TrimSuffix(server, "/")
	apiURL := server + "/push"

	// If PushKey is a full URL, we might be using an old-style custom URL
	if strings.HasPrefix(b.PushKey, "http") {
		apiURL = b.PushKey
	}

	var postData interface{}
	if b.CipherEnable == "true" {
		// Encrypted Request
		// 1. Prepare the full notification payload (without device_key, as specified for encryption)
		encryptData := make(map[string]interface{})
		for k, v := range data {
			if k != "device_key" {
				encryptData[k] = v
			}
		}

		jsonData, err := json.Marshal(encryptData)
		if err != nil {
			return nil, err
		}

		ciphertext, cipherIv, err := b.encryptPayload(string(jsonData))
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %v", err)
		}

		postData = map[string]interface{}{
			"ciphertext": ciphertext,
			"iv":         cipherIv,
			"device_key": b.PushKey,
		}
	} else {
		// Normal request
		postData = data
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return nil, err
	}

	// 使用带超时的客户端
	client := b.getHTTPClient()
	resp, err := client.Post(apiURL, "application/json;charset=utf-8", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r barkResponse
	err = json.Unmarshal(body, &r)
	if err != nil {
		// If not JSON, return the raw body as it might be a simple success message from some servers
		if resp.StatusCode == 200 {
			return body, nil
		}
		return body, err
	}

	if r.Code != 200 && resp.StatusCode != 200 {
		return body, fmt.Errorf("bark response error: %s", string(body))
	}
	return body, nil
}

// encryptPayload 加密
// 参考逻辑: https://github.com/hotlcc/MoviePilot-Plugins-Third/blob/main/plugins/mergemessagenotify/channel/custom/bark.py
func (b *Bark) encryptPayload(payload string) (*string, *string, error) {
	if b.CipherMode == "" {
		return nil, nil, fmt.Errorf("加密模式不能为空")
	}

	plaintextBytes := []byte(payload)
	cipherKeyBytes := []byte(b.CipherKey)

	var padded []byte
	switch b.CipherMode {
		case "CBC", "ECB":
			padded = b.pkcs7Pad(plaintextBytes, aes.BlockSize)
		case "GCM":
			padded = plaintextBytes
		default:
			return nil, nil, fmt.Errorf("加密模式无效: %s", b.CipherMode)
	}

	block, err := aes.NewCipher(cipherKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("AES加密失败: %w", err)
	}

	var cipherTextBytes []byte
	var ivStr string

	switch b.CipherMode {
		case "ECB":
			cipherTextBytes, err = aesECBEncrypt(block, padded)
			if err != nil {
				return nil, nil, fmt.Errorf("AES加密失败: %w", err)
			}
			ivStr = ""
		case "CBC":
			ivStr = utils.RandomString(16)
			ivBytes := []byte(ivStr)

			cbcMode := cipher.NewCBCEncrypter(block, ivBytes)
			cipherTextBytes = make([]byte, len(padded))
			cbcMode.CryptBlocks(cipherTextBytes, padded)
		case "GCM":
			ivStr = utils.RandomString(12)
			ivBytes := []byte(ivStr)

			gcm, err := cipher.NewGCM(block)
			if err != nil {
				return nil, nil, fmt.Errorf("AES加密失败: %w", err)
			}
			cipherTextBytes = gcm.Seal(nil, ivBytes, padded, nil)
		default:
			return nil, nil, fmt.Errorf("加密模式无效: %s", b.CipherMode)
	}

	b64 := base64.StdEncoding.EncodeToString(cipherTextBytes)
	return &b64, &ivStr, nil
}

func (b *Bark) pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func aesECBEncrypt(block cipher.Block, src []byte) ([]byte, error) {
	if len(src)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("ECB input not multiple of block size")
	}
	dst := make([]byte, len(src))
	for i := 0; i < len(src); i += block.BlockSize() {
		block.Encrypt(dst[i:i+block.BlockSize()], src[i:i+block.BlockSize()])
	}
	return dst, nil
}

// getHTTPClient 获取 HTTP 客户端（含超时和代理）
func (b *Bark) getHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	if b.ProxyURL != "" {
		proxyURL, err := url.Parse(b.ProxyURL)
		if err == nil {
			if strings.HasPrefix(strings.ToLower(b.ProxyURL), "socks5://") {
				dialer, err := b.createSOCKS5Dialer(proxyURL)
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
func (b *Bark) createSOCKS5Dialer(proxyURL *url.URL) (proxy.ContextDialer, error) {
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
		Timeout:   20 * time.Second,
		KeepAlive: 20 * time.Second,
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
