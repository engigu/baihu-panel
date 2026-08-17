package clibase

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/engigu/baihu-panel/internal/bootstrap"
)

// CallInternalAPI 封装底层进程间 HTTP 通信，统一处理网络连接错误及业务级异常提取
func CallInternalAPI(method, endpoint string, payload any) ([]byte, error) {
	bodyBytes, statusCode, err := bootstrap.SendInternalRequest(method, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("无法连接到主程序后台服务: %w", err)
	}

	if statusCode != 200 {
		return bodyBytes, fmt.Errorf("后台服务响应异常 (状态码: %d): %s", statusCode, strings.TrimSpace(string(bodyBytes)))
	}

	// 尝试通用结构体嗅探，提取业务级逻辑拒绝原因
	var res struct {
		Data struct {
			Success *bool  `json:"success"`
			Error   string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &res); err == nil {
		if res.Data.Success != nil && !*res.Data.Success {
			errReason := res.Data.Error
			if errReason == "" {
				errReason = strings.TrimSpace(string(bodyBytes))
			}
			return bodyBytes, fmt.Errorf("%s", errReason)
		}
	}

	return bodyBytes, nil
}
