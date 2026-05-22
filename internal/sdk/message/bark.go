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
)

type barkResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Bark struct {
	PushKey  string
	Archive  string
	Group    string
	Sound    string
	Icon     string
	Level    string
	URL      string
	Key      string
	IV       string
	Server   string
	Badge    string
	Copy     string
	AutoCopy string
	ProxyURL string // 可选的代理地址
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
	if b.Key != "" && b.IV != "" {
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

		ciphertext, err := b.encryptPayload(string(jsonData))
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %v", err)
		}

		postData = map[string]interface{}{
			"ciphertext": ciphertext,
			"iv":         b.IV,
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

func (b *Bark) encryptPayload(payload string) (string, error) {
	key := []byte(b.Key)
	iv := []byte(b.IV)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	paddedPayload := b.pkcs7Pad([]byte(payload), aes.BlockSize)
	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(paddedPayload))
	mode.CryptBlocks(ciphertext, paddedPayload)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (b *Bark) pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
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
