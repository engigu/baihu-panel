package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/services"
)

// SendInternalRequest 向常驻后台主服务安全发送内部通信请求
// relPath 传入相对内部接口路径 (如: "/internal/tasks/execute/xxx")，方法内部会自动补充完整的协议、端口及 "/api/v1" 前缀，
// 并自动获取 security.secret 密钥种入 X-Internal-Token 头部。
func SendInternalRequest(method, relPath string, payload interface{}) ([]byte, int, error) {
	appCfg := services.GetConfig()
	if appCfg == nil {
		return nil, 0, fmt.Errorf("加载系统配置失败")
	}

	relPath = strings.TrimPrefix(relPath, "/")
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/%s", appCfg.Server.Port, relPath)

	var bodyReader io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("序列化请求负载失败: %v", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	settings := services.NewSettingsService()
	secret := settings.Get("security", "secret")

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("创建 HTTP 请求失败: %v", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Internal-Token", secret)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("网络连接失败，请确保白虎面板常驻后台服务正在运行中: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	return bodyBytes, resp.StatusCode, err
}
