package channels

import (
	"fmt"
	"github.com/engigu/baihu-panel/internal/sdk/message"
	"strconv"
)

type EmailChannel struct{ *BaseChannel }

func NewEmailChannel() Channel {
	return &EmailChannel{NewBaseChannel(ChannelEmail, []string{FormatTypeHTML, FormatTypeText})}
}

func (c *EmailChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	server := config.GetString("server")
	portStr := config.GetString("port")
	account := config.GetString("account")
	passwd := config.GetString("passwd")
	fromName := config.GetString("from_name")
	toAccount := config.GetString("to_account")

	if server == "" || account == "" || passwd == "" {
		return SendError("email config missing: server, account, passwd are required"), nil
	}
	if toAccount == "" {
		return SendError("email config missing: to_account is required"), nil
	}

	port, _ := strconv.Atoi(portStr)
	contentType, formattedContent := c.FormatContent(msg)

	var emailer message.EmailMessage
	emailer.Init(server, port, account, passwd, fromName)

	var errMsg string
	if contentType == FormatTypeText {
		errMsg = emailer.SendTextMessage(toAccount, msg.Title, formattedContent)
	} else if contentType == FormatTypeHTML {
		errMsg = emailer.SendHtmlMessage(toAccount, msg.Title, formattedContent)
	} else {
		errMsg = fmt.Sprintf("未知的邮件发送内容类型：%s", contentType)
	}

	if errMsg != "" {
		return ErrorResultStr("", errMsg), nil
	}
	return SuccessResult(""), nil
}
