package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

type FeishuChannel struct{ *BaseChannel }

func NewFeishuChannel() Channel {
	return &FeishuChannel{NewBaseChannel(ChannelFeishu, []string{FormatTypeMarkdown, FormatTypeText})}
}

func (c *FeishuChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	accessToken := config.GetString("access_token")
	secret := config.GetString("secret")

	if accessToken == "" {
		return SendError("feishu config missing: access_token is required"), nil
	}

	contentType, formattedContent := c.FormatContent(msg)
	atMobiles := msg.GetAtMobiles()
	atUserIds := msg.GetAtUserIds()
	atList := append(atMobiles, atUserIds...)
	if msg.AtAll {
		atList = append(atList, "all")
	}

	cli := message.Feishu{AccessToken: accessToken, Secret: secret}
	var res []byte
	var err error

	if contentType == FormatTypeText {
		res, err = cli.SendMessageText(formattedContent, atList...)
	} else if contentType == FormatTypeMarkdown {
		res, err = cli.SendMessageMarkdown(msg.Title, formattedContent, atList...)
	} else {
		return SendError("未知的飞书发送内容类型：%s", contentType), nil
	}

	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
