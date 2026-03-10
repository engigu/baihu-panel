package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

type QyWeiXinChannel struct{ *BaseChannel }

func NewQyWeiXinChannel() Channel {
	return &QyWeiXinChannel{NewBaseChannel(ChannelQyWeiXin, []string{FormatTypeMarkdown, FormatTypeText})}
}

func (c *QyWeiXinChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	accessToken := config.GetString("access_token")

	if accessToken == "" {
		return SendError("qyweixin config missing: access_token is required"), nil
	}

	contentType, formattedContent := c.FormatContent(msg)
	atList := []string{}
	atList = append(atList, msg.GetAtUserIds()...)
	atList = append(atList, msg.GetAtMobiles()...)
	if msg.AtAll {
		atList = append(atList, "@all")
	}

	cli := message.QyWeiXin{AccessToken: accessToken}
	var res []byte
	var err error

	if contentType == FormatTypeText {
		res, err = cli.SendMessageText(formattedContent, atList...)
	} else if contentType == FormatTypeMarkdown {
		res, err = cli.SendMessageMarkdown(msg.Title, formattedContent, atList...)
	} else {
		return SendError("未知的企业微信发送内容类型：%s", contentType), nil
	}

	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
