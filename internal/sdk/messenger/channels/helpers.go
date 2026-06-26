package channels

import (
	"encoding/json"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
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


// replaceBodyPlaceholder 替换自定义 webhook body 中的 TEXT 占位符
func replaceBodyPlaceholder(body string, content string) string {
	data, _ := json.Marshal(content)
	dataStr := strings.Trim(string(data), "\"")
	return strings.Replace(body, "TEXT", dataStr, -1)
}

// createAliyunSMSClient 创建阿里云短信客户端 (V2.0)
func createAliyunSMSClient(accessKeyId, accessKeySecret, regionId string) (*dysmsapi.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
		RegionId:        tea.String(regionId),
	}
	// 设置端点，通常为 dysmsapi.aliyuncs.com
	config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
	return dysmsapi.NewClient(config)
}

// sendAliyunSMS 发送短信 (V2.0)
func sendAliyunSMS(client *dysmsapi.Client, phoneNumber, signName, templateCode, content string, extra map[string]any) (string, error) {
	templateParam := map[string]interface{}{
		"content": content,
	}
	for k, v := range extra {
		templateParam[k] = v
	}
	templateParamJSON, _ := json.Marshal(templateParam)

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phoneNumber),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(string(templateParamJSON)),
	}

	response, err := client.SendSms(request)
	if err != nil {
		return "", fmt.Errorf("发送短信失败: %s", err.Error())
	}

	if response.Body == nil || tea.StringValue(response.Body.Code) != "OK" {
		msg := "Unknown Error"
		code := "Unknown Code"
		if response.Body != nil {
			msg = tea.StringValue(response.Body.Message)
			code = tea.StringValue(response.Body.Code)
		}
		return "", fmt.Errorf("发送失败: %s - %s", code, msg)
	}

	return fmt.Sprintf("RequestId: %s, BizId: %s", tea.StringValue(response.Body.RequestId), tea.StringValue(response.Body.BizId)), nil
}
