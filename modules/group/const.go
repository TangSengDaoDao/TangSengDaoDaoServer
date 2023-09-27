package group

// 群状态
const (
	// GroupStatusDisabled 已禁用
	GroupStatusDisabled = 0
	// GroupStatusNormal 正常
	GroupStatusNormal = 1
)

// 群成员角色
const (
	// MemberRoleCommon 普通成员
	MemberRoleCommon = 0
	// MemberRoleCreator 创建者
	MemberRoleCreator = 1
	// MemberRoleManager 管理者
	MemberRoleManager = 2
)

const (
	// InviteStatusWait 等待确认
	InviteStatusWait = 0
	// InviteStatusOK 已确认
	InviteStatusOK = 1
)

// 群类型
type GroupType int

const (
	GroupTypeCommon GroupType = iota // 普通群
	GroupTypeSuper                   // 超大群
)

const (
	ChannelServiceName = "channel"
)
