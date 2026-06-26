package channels

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


type AliyunSMSChannel struct{ *BaseChannel }

func NewAliyunSMSChannel() Channel {
	return &AliyunSMSChannel{NewBaseChannel(ChannelAliyunSMS, []string{FormatTypeText})}
}

func (c *AliyunSMSChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	accessKeyId := config.GetString("access_key_id")
	accessKeySecret := config.GetString("access_key_secret")
	signName := config.GetString("sign_name")
	regionId := config.GetString("region_id")
	phoneNumber := config.GetString("phone_number")
	templateCode := config.GetString("template_code")

	if accessKeyId == "" || accessKeySecret == "" || signName == "" {
		return SendError("aliyun sms config missing: access_key_id, access_key_secret, sign_name are required"), nil
	}
	if phoneNumber == "" || templateCode == "" {
		return SendError("aliyun sms config missing: phone_number, template_code are required"), nil
	}

	_, formattedContent := c.FormatContent(msg)

	if regionId == "" {
		regionId = "cn-hangzhou"
	}

	client, err := createAliyunSMSClient(accessKeyId, accessKeySecret, regionId)
	if err != nil {
		return SendError("创建阿里云短信客户端失败: %s", err.Error()), nil
	}

	result, err := sendAliyunSMS(client, phoneNumber, signName, templateCode, formattedContent, msg.Extra)
	if err != nil {
		return ErrorResult("", err), nil
	}
	return SuccessResult(result), nil
}
