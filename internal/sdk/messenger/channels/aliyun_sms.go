package channels

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
