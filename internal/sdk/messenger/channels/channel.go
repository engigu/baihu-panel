package channels

import "fmt"

// Channel 渠道接口 - SDK 版本，零业务依赖
type Channel interface {
	// GetType 返回渠道类型标识
	GetType() string
	// GetSupportedFormats 返回支持的消息格式
	GetSupportedFormats() []string
	// Send 发送消息
	Send(config ChannelConfig, msg *Message) (*Result, error)
}

// BaseChannel 渠道基础实现
type BaseChannel struct {
	channelType      string
	supportedFormats []string
}

func NewBaseChannel(channelType string, supportedFormats []string) *BaseChannel {
	return &BaseChannel{channelType: channelType, supportedFormats: supportedFormats}
}

func (c *BaseChannel) GetType() string              { return c.channelType }
func (c *BaseChannel) GetSupportedFormats() []string { return c.supportedFormats }

// FormatContent 根据渠道支持的格式选择最佳内容
func (c *BaseChannel) FormatContent(msg *Message) (formatType string, content string) {
	for _, ft := range c.supportedFormats {
		switch ft {
		case FormatTypeMarkdown:
			if msg.HasMarkdown() {
				return FormatTypeMarkdown, msg.Markdown
			}
		case FormatTypeHTML:
			if msg.HasHTML() {
				return FormatTypeHTML, msg.HTML
			}
		case FormatTypeText:
			if msg.HasText() {
				return FormatTypeText, msg.Text
			}
		}
	}
	if msg.HasText() {
		return FormatTypeText, msg.Text
	}
	return FormatTypeText, ""
}

// SuccessResult 创建成功结果
func SuccessResult(response string) *Result {
	return &Result{Success: true, Response: response}
}

// ErrorResult 创建失败结果
func ErrorResult(response string, err error) *Result {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return &Result{Success: false, Response: response, Error: errMsg}
}

// ErrorResultStr 创建失败结果（字符串错误）
func ErrorResultStr(response string, errMsg string) *Result {
	return &Result{Success: false, Response: response, Error: errMsg}
}

// SendError 发送失败时的格式化错误
func SendError(format string, args ...any) *Result {
	return &Result{Success: false, Error: fmt.Sprintf(format, args...)}
}
