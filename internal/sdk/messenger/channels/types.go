package channels

// Message 统一消息内容
type Message struct {
	Title     string         `json:"title"`
	Text      string         `json:"text"`
	HTML      string         `json:"html"`
	Markdown  string         `json:"markdown"`
	URL       string         `json:"url"`
	ImageURL  string         `json:"image_url"`
	Summary   string         `json:"summary"`
	AtMobiles []string       `json:"at_mobiles"`
	AtUserIds []string       `json:"at_user_ids"`
	AtAll     bool           `json:"at_all"`
	Extra     map[string]any `json:"extra"`
}

func (m *Message) HasText() bool     { return m.Text != "" }
func (m *Message) HasHTML() bool     { return m.HTML != "" }
func (m *Message) HasMarkdown() bool { return m.Markdown != "" }

func (m *Message) GetAtMobiles() []string {
	if m.AtMobiles == nil {
		return []string{}
	}
	return m.AtMobiles
}

func (m *Message) GetAtUserIds() []string {
	if m.AtUserIds == nil {
		return []string{}
	}
	return m.AtUserIds
}

// ChannelConfig 渠道认证配置（Key-Value 形式，各渠道自行定义字段）
type ChannelConfig map[string]string

// GetString 安全获取配置值
func (c ChannelConfig) GetString(key string) string {
	if v, ok := c[key]; ok {
		return v
	}
	return ""
}

// Result 发送结果
type Result struct {
	Success  bool   `json:"success"`
	Response string `json:"response"` // 原始响应
	Error    string `json:"error"`    // 错误信息
}

// 消息格式类型常量
const (
	FormatTypeText     = "text"
	FormatTypeHTML     = "html"
	FormatTypeMarkdown = "markdown"
)

// 渠道类型常量
const (
	ChannelEmail           = "Email"
	ChannelDtalk           = "Dtalk"
	ChannelQyWeiXin        = "QyWeiXin"
	ChannelFeishu          = "Feishu"
	ChannelCustom          = "Custom"
	ChannelWeChatOFAccount = "WeChatOFAccount"
	ChannelAliyunSMS       = "AliyunSMS"
	ChannelTelegram        = "Telegram"
	ChannelBark            = "Bark"
	ChannelPushMe          = "PushMe"
	ChannelNtfy            = "Ntfy"
	ChannelGotify          = "Gotify"
	ChannelPushPlus        = "PushPlus"
)
