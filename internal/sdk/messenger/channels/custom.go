package channels

import (
	"encoding/json"

	"github.com/engigu/baihu-panel/internal/sdk/message"
)

type CustomChannel struct{ *BaseChannel }

func NewCustomChannel() Channel {
	return &CustomChannel{NewBaseChannel(ChannelCustom, []string{FormatTypeText})}
}

func (c *CustomChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	webhook := config.GetString("webhook")
	body := config.GetString("body")
	headersStr := config.GetString("headers")

	if webhook == "" {
		return SendError("custom config missing: webhook is required"), nil
	}

	var headers map[string]string
	if headersStr != "" {
		if err := json.Unmarshal([]byte(headersStr), &headers); err != nil {
			return SendError("custom config error: headers must be a valid JSON object"), nil
		}
	}

	_, formattedContent := c.FormatContent(msg)
	cli := message.CustomWebhook{}

	// 替换 body 模板中的 TEXT 占位符
	bodyStr := body
	if bodyStr != "" {
		bodyStr = replaceBodyPlaceholder(bodyStr, formattedContent)
	} else {
		bodyStr = formattedContent
	}

	res, err := cli.Request(webhook, bodyStr, headers)
	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
