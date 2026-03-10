package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

type WeChatOFAccountChannel struct{ *BaseChannel }

func NewWeChatOFAccountChannel() Channel {
	return &WeChatOFAccountChannel{NewBaseChannel(ChannelWeChatOFAccount, []string{FormatTypeText})}
}

func (c *WeChatOFAccountChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	appID := config.GetString("appID")
	appSecret := config.GetString("appsecret")
	tempID := config.GetString("tempid")
	toAccount := config.GetString("to_account")

	if appID == "" || appSecret == "" {
		return SendError("wechat config missing: appID, appsecret are required"), nil
	}
	if toAccount == "" {
		return SendError("wechat config missing: to_account is required"), nil
	}

	_, formattedContent := c.FormatContent(msg)
	cli := message.WeChatOFAccount{
		AppID:      appID,
		AppSecret:  appSecret,
		TemplateID: tempID,
		ToUser:     toAccount,
		URL:        msg.URL,
	}

	res, err := cli.Send(msg.Title, formattedContent)
	if err != nil {
		return ErrorResult(res, err), nil
	}
	return SuccessResult(res), nil
}
