package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

type NtfyChannel struct{ *BaseChannel }

func NewNtfyChannel() Channel {
	return &NtfyChannel{NewBaseChannel(ChannelNtfy, []string{FormatTypeText})}
}

func (c *NtfyChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	topic := config.GetString("topic")
	if topic == "" {
		return SendError("ntfy config missing: topic is required"), nil
	}

	cli := message.Ntfy{
		Url:      config.GetString("url"),
		Topic:    topic,
		Priority: config.GetString("priority"),
		Icon:     config.GetString("icon"),
		Token:    config.GetString("token"),
		Username: config.GetString("username"),
		Password: config.GetString("password"),
		Actions:  config.GetString("actions"),
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
