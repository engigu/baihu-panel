package channels

import (
	"github.com/engigu/baihu-panel/internal/sdk/message"
	"strconv"
)

type GotifyChannel struct{ *BaseChannel }

func NewGotifyChannel() Channel {
	return &GotifyChannel{NewBaseChannel(ChannelGotify, []string{FormatTypeText})}
}

func (c *GotifyChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	url := config.GetString("url")
	token := config.GetString("token")

	if url == "" || token == "" {
		return SendError("gotify config missing: url and token are required"), nil
	}

	priority, _ := strconv.Atoi(config.GetString("priority"))
	cli := message.Gotify{
		Url:      url,
		Token:    token,
		Priority: priority,
	}

	res, err := cli.Request(msg.Title, msg.Text)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
