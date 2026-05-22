package channels

import (
	"fmt"
	"github.com/engigu/baihu-panel/internal/sdk/message"
	"strconv"
	"strings"
)

type WxPusherChannel struct{ *BaseChannel }

func NewWxPusherChannel() Channel {
	return &WxPusherChannel{NewBaseChannel(ChannelWxPusher, []string{FormatTypeText})}
}

func (c *WxPusherChannel) Send(config ChannelConfig, msg *Message) (*Result, error) {
	appToken := config.GetString("app_token")
	if appToken == "" {
		return SendError("wxpusher config missing: app_token is required"), nil
	}

	uidsStr := config.GetString("uids")
	topicIdsStr := config.GetString("topic_ids")
	verifyPayTypeStr := config.GetString("verify_pay_type")

	if uidsStr == "" && topicIdsStr == "" {
		return SendError("wxpusher config missing: uids or topic_ids is required"), nil
	}

	var uids []string
	if uidsStr != "" {
		uids = strings.Split(uidsStr, ",")
		for i := range uids {
			uids[i] = strings.TrimSpace(uids[i])
		}
	}

	var topicIds []int
	if topicIdsStr != "" {
		ids := strings.Split(topicIdsStr, ",")
		for _, idStr := range ids {
			idStr = strings.TrimSpace(idStr)
			if id, err := strconv.Atoi(idStr); err == nil {
				topicIds = append(topicIds, id)
			}
		}
	}

	verifyPayType := 0
	if verifyPayTypeStr != "" {
		if v, err := strconv.Atoi(verifyPayTypeStr); err == nil {
			verifyPayType = v
		}
	}

	_, formattedContent := c.FormatContent(msg)

	// 如果有标题，将标题和内容合并
	content := formattedContent
	if msg.Title != "" {
		content = fmt.Sprintf("%s\n\n%s", msg.Title, formattedContent)
	}

	cli := message.WxPusher{
		AppToken:      appToken,
		Content:       content,
		ContentType:   1, // 仅支持文字
		Uids:          uids,
		TopicIds:      topicIds,
		VerifyPayType: verifyPayType,
	}

	res, err := cli.Send()
	if err != nil {
		return ErrorResult(res, err), nil
	}
	return SuccessResult(res), nil
}
