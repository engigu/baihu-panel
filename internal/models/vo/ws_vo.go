package vo

// WSMessage 通用 WebSocket 消息结构
type WSMessage struct {
	Type      string      `json:"type"`      // 事件类型: task_status, notice, system_stats
	Timestamp int64       `json:"timestamp"` // 毫秒时间戳
	Payload   interface{} `json:"payload"`   // 负载数据
}
