package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

type PushPlusChannel struct{ *BaseChannel }

func NewPushPlusChannel() Channel {
	return &PushPlusChannel{NewBaseChannel(ChannelPushPlus, []string{FormatTypeText, FormatTypeHTML, FormatTypeMarkdown})}
}

func (c *PushPlusChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	token := config.GetString("token")
	if token == "" {
		return SendError("pushplus config missing: token is required"), nil
	}

	cli := message.PushPlus{
		Token:       token,
		Topic:       config.GetString("topic"),
		Template:    config.GetString("template"),
		Channel:     config.GetString("channel"),
		Webhook:     config.GetString("webhook"),
		CallbackUrl: config.GetString("callback_url"),
		To:          config.GetString("to"),
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
