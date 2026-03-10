package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

type BarkChannel struct{ *BaseChannel }

func NewBarkChannel() Channel {
	return &BarkChannel{NewBaseChannel(ChannelBark, []string{FormatTypeText})}
}

func (c *BarkChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	pushKey := config.GetString("push_key")
	if pushKey == "" {
		return SendError("bark config missing: push_key is required"), nil
	}

	cli := message.Bark{
		PushKey: pushKey,
		Archive: config.GetString("archive"),
		Group:   config.GetString("group"),
		Sound:   config.GetString("sound"),
		Icon:    config.GetString("icon"),
		Level:   config.GetString("level"),
		URL:     config.GetString("url"),
		Key:     config.GetString("key"),
		IV:      config.GetString("iv"),
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
