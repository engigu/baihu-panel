package channels

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
)

// replaceBodyPlaceholder 替换自定义 webhook body 中的 TEXT 占位符
func replaceBodyPlaceholder(body string, content string) string {
	data, _ := json.Marshal(content)
	dataStr := strings.Trim(string(data), "\"")
	return strings.Replace(body, "TEXT", dataStr, -1)
}

// createAliyunSMSClient 创建阿里云短信客户端
func createAliyunSMSClient(accessKeyId, accessKeySecret, regionId string) (*dysmsapi.Client, error) {
	return dysmsapi.NewClientWithAccessKey(regionId, accessKeyId, accessKeySecret)
}

// sendAliyunSMS 发送短信
func sendAliyunSMS(client *dysmsapi.Client, phoneNumber, signName, templateCode, content string, extra map[string]any) (string, error) {
	templateParam := map[string]interface{}{
		"content": content,
	}
	if extra != nil {
		for k, v := range extra {
			templateParam[k] = v
		}
	}
	templateParamJSON, _ := json.Marshal(templateParam)

	request := dysmsapi.CreateSendSmsRequest()
	request.Scheme = "https"
	request.PhoneNumbers = phoneNumber
	request.SignName = signName
	request.TemplateCode = templateCode
	request.TemplateParam = string(templateParamJSON)

	response, err := client.SendSms(request)
	if err != nil {
		return "", fmt.Errorf("发送短信失败: %s", err.Error())
	}

	if response.Code != "OK" {
		return "", fmt.Errorf("发送失败: %s - %s", response.Code, response.Message)
	}

	return fmt.Sprintf("RequestId: %s, BizId: %s", response.RequestId, response.BizId), nil
}
