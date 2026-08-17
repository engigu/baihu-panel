package constant

const (
	// CookieActiveInterconnectNodeID 穿越状态下标识目标子节点 ID 的 Cookie 键名
	CookieActiveInterconnectNodeID = "active_interconnect_node_id"

	// SectionInterconnect 互联设置分组
	SectionInterconnect = "interconnect"

	// 互联设置相关 Key
	KeyInterconnectToken       = "interconnect_token"
	KeyInterconnectParentURL   = "interconnect_parent_url"
	KeyInterconnectParentToken = "interconnect_parent_token"
	KeyInterconnectRole        = "interconnect_role"

	// 互联角色
	InterconnectRoleMaster = "master"
	InterconnectRoleChild  = "child"

	// 互联系统事件
	EventInterconnectChildStatus = "interconnect_child_status"
)
