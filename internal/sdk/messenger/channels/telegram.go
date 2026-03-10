package channels

import "github.com/engigu/baihu-panel/internal/sdk/message"

type TelegramChannel struct{ *BaseChannel }

func NewTelegramChannel() Channel {
	return &TelegramChannel{NewBaseChannel(ChannelTelegram, []string{FormatTypeMarkdown, FormatTypeHTML, FormatTypeText})}
}

func (c *TelegramChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	botToken := config.GetString("bot_token")
	chatID := config.GetString("chat_id")
	apiHost := config.GetString("api_host")
	proxyURL := config.GetString("proxy_url")

	if botToken == "" || chatID == "" {
		return SendError("telegram config missing: bot_token, chat_id are required"), nil
	}

	contentType, formattedContent := c.FormatContent(msg)
	cli := message.Telegram{
		BotToken: botToken,
		ChatID:   chatID,
		ApiHost:  apiHost,
		ProxyURL: proxyURL,
	}

	var res []byte
	var err error

	switch contentType {
	case FormatTypeText:
		res, err = cli.SendMessageText(formattedContent)
	case FormatTypeMarkdown:
		res, err = cli.SendMessageMarkdown(formattedContent)
	case FormatTypeHTML:
		res, err = cli.SendMessageHTML(formattedContent)
	default:
		return SendError("未知的Telegram发送内容类型：%s", contentType), nil
	}

	if err != nil {
		return ErrorResult(string(res), err), nil
	}
	return SuccessResult(string(res)), nil
}
