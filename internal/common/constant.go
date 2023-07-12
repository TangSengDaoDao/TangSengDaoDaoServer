package common

import (
	"bytes"
	"encoding/json"
	"errors"
)

// ErrData 数据格式有误
var ErrData = errors.New("数据格式有误！")

// ChannelType 频道类型
type ChannelType uint8

const (
	// ChannelTypeNone 没有指定频道
	ChannelTypeNone ChannelType = iota
	// ChannelTypePerson 个人频道
	ChannelTypePerson
	// ChannelTypeGroup           群频道
	ChannelTypeGroup
	// ChannelTypeCustomerService 客服频道
	ChannelTypeCustomerService
	// ChannelTypeCommunity 社区
	ChannelTypeCommunity
	// ChannelTypeCommunityTopic 话题
	ChannelTypeCommunityTopic
	//  ChannelTypeInfo 资讯类频道
	ChannelTypeInfo
)

// GroupMemberStatus 群成员状态
type GroupMemberStatus int

const (
	// GroupMemberStatusNormal 正常
	GroupMemberStatusNormal GroupMemberStatus = 1
	// GroupMemberStatusBlacklist 黑名单
	GroupMemberStatusBlacklist GroupMemberStatus = 2
)

// GroupMemberRole 群成员角色
type GroupMemberRole int

const (
	// GroupMemberRoleCreater 群主
	GroupMemberRoleCreater GroupMemberRole = 1
	// GroupMemberRoleManager 管理员
	GroupMemberRoleManager GroupMemberRole = 2
	// GroupMemberRoleNormal 成员
	GroupMemberRoleNormal GroupMemberRole = 0
)

// GroupAllowViewHistoryMsgStatus 新成员是否能查看历史消息
type GroupAllowViewHistoryMsgStatus int

const (
	// GroupAllowViewHistoryMsgDisabled 不能查看历史消息
	GroupAllowViewHistoryMsgDisabled GroupAllowViewHistoryMsgStatus = 0
	// GroupAllowViewHistoryMsgEnabled 能查看历史消息
	GroupAllowViewHistoryMsgEnabled GroupAllowViewHistoryMsgStatus = 1
)

// Uint8 转换为uint8
func (c ChannelType) Uint8() uint8 {
	return uint8(c)
}

const (
	// GroupMemberSeqKey 群成员序列号key
	GroupMemberSeqKey = "groupMember"
	// GroupSettingSeqKey 群设置序列号key
	GroupSettingSeqKey = "groupSetting"
	// GroupSeqKey 群序列号key
	GroupSeqKey = "group"
	// UserSettingSeqKey 用户设置序列号key
	UserSettingSeqKey = "userSetting"
	// UserSeqKey 用户序列号
	UserSeqKey = "user"
	// FriendSeqKey 好友
	FriendSeqKey = "friend"
	// MessageExtraSeqKey 消息扩展序号
	MessageExtraSeqKey = "messageExtra"
	// MessageReactionSeqKey 消息回应序号
	MessageReactionSeqKey = "messageReaction"
	// RobotSeqKey 机器人序号
	RobotSeqKey = "robot"
	// RobotEventSeqKey 机器人事件序号
	RobotEventSeqKey = "robotEventSeq:"
	// SensitiveWordsKey 敏感词序号
	SensitiveWordsKey = "sensitiveWords"
	// ReminderKey 提醒项序号
	RemindersKey = "reminders"
	// SyncConversationExtraKey 同步最近会话扩展
	SyncConversationExtraKey = "syncConversationExtra"
	// ProhibitWord 违禁词
	ProhibitWordKey = "ProhibitWord"
)

// 群属性key
const (
	// GroupAttrKeyName 群名称
	GroupAttrKeyName = "name"
	// GroupAttrKeyNotice 群公告
	GroupAttrKeyNotice = "notice"
	// GroupAttrKeyForbidden 群禁言
	GroupAttrKeyForbidden = "forbidden"
	// GroupAttrKeyInvite 邀请确认
	GroupAttrKeyInvite = "invite"
	// GroupAttrKeyForbiddenAddFriend 群内禁止加好友
	GroupAttrKeyForbiddenAddFriend = "forbidden_add_friend"
	// GroupAttrKeyStatus 群状态
	GroupAttrKeyStatus = "status"
	// GroupAllowViewHistoryMsg 是否允许新成员查看历史消息
	GroupAllowViewHistoryMsg = "allow_view_history_msg"
)

// 命令消息
const (
	// CMDChannelUpdate 频道信息更新
	CMDChannelUpdate = "channelUpdate"
	// CMDGroupMemberUpdate 群成员更新
	CMDGroupMemberUpdate = "memberUpdate"
	// CMDConversationUnreadClear 未读数清空
	CMDConversationUnreadClear = "unreadClear"
	// CMDGroupAvatarUpdate 群头像更新
	CMDGroupAvatarUpdate = "groupAvatarUpdate"
	// CMDCommunityAvatarUpdate 社区头像更新
	CMDCommunityAvatarUpdate = "communityAvatarUpdate"
	// CMDCommunityCoverUpdate 社区封面更新
	CMDCommunityCoverUpdate = "communityCoverUpdate"
	// CMDConversationDelete 删除最近会话
	CMDConversationDelete = "conversationDelete"
	// CMDFriendRequest 好友申请
	CMDFriendRequest = "friendRequest"
	// friendAccept 接受好友申请
	CMDFriendAccept = "friendAccept"
	// friendDeleted 好友被删除
	CMDFriendDeleted = "friendDeleted"
	// userAvatarUpdate 个人头像更新
	CMDUserAvatarUpdate = "userAvatarUpdate"
	// 输入中
	CMDTyping = "typing"
	// 在线状态
	CMDOnlineStatus = "onlineStatus"
	// 动态点赞或评论消息
	CMDMomentMsg = "momentMsg"
	// 同步消息扩展数据
	CMDSyncMessageExtra = "syncMessageExtra"
	// 同步消息回应数据
	CMDSyncMessageReaction = "syncMessageReaction"
	// 退出pc登录
	CMDPCQuit = "pcQuit"
	// 最近会话被删除
	CMDConversationDeleted = "conversationDeleted"
	// 同步提醒项
	CMDSyncReminders = "syncReminders"
	// 同步最近会话扩展
	CMDSyncConversationExtra = "syncConversationExtra"
)

// UserDeviceTokenPrefix 用户设备token缓存前缀
const UserDeviceTokenPrefix = "userDeviceToken:"

// UserDeviceBadgePrefix 用户设备红点
const UserDeviceBadgePrefix = "userDeviceBadge"

// QRCodeCachePrefix 二维码缓存前缀
const QRCodeCachePrefix = "qrcode:"

// AuthCodeCachePrefix 授权code
const AuthCodeCachePrefix = "authcode:"

// AuthCodeType 认证代码类型
type AuthCodeType string

// AuthCodeTypeJoinGroup 进群授权code
const AuthCodeTypeJoinGroup AuthCodeType = "joinGroup"

// AuthCodeTypeGroupMemberInvite 群成员邀请
const AuthCodeTypeGroupMemberInvite AuthCodeType = "groupMemberInvite"

// AuthCodeTypeScanLogin 扫描登录
const AuthCodeTypeScanLogin AuthCodeType = "scanLogin"

// DeviceType 设备类型
type DeviceType string

const (
	// DeviceTypeIOS iOS设备
	DeviceTypeIOS DeviceType = "IOS"

	// DeviceTypeMI 小米设备
	DeviceTypeMI DeviceType = "MI"

	// DeviceTypeHMS 华为设备
	DeviceTypeHMS DeviceType = "HMS"

	// DeviceTypeOPPO oppo设备
	DeviceTypeOPPO DeviceType = "OPPO"

	// DeviceTypeVIVO vivo设备
	DeviceTypeVIVO DeviceType = "VIVO"
)

// QRCodeType 二维码类型
type QRCodeType string

const (
	// QRCodeTypeGroup 群聊
	QRCodeTypeGroup QRCodeType = "group"
	// QRCodeTypeScanLogin 扫描登录
	QRCodeTypeScanLogin QRCodeType = "scanLogin"
)

// ScanLoginStatus 扫码状态
type ScanLoginStatus string

const (
	// ScanLoginStatusExpired 二维码过期
	ScanLoginStatusExpired ScanLoginStatus = "expired"
	// ScanLoginStatusWaitScan 等待扫码
	ScanLoginStatusWaitScan ScanLoginStatus = "waitScan"
	// ScanLoginStatusScanned 已扫描
	ScanLoginStatusScanned ScanLoginStatus = "scanned"
	// ScanLoginStatusAuthed 已授权
	ScanLoginStatusAuthed ScanLoginStatus = "authed"
)

// VercodeType 加好友验证码类型
type VercodeType int

const (
	// User 搜索
	User VercodeType = 1
	// GroupMember 群成员
	GroupMember VercodeType = 2
	// QRCode 二维码
	QRCode VercodeType = 3
	// Friend 好友
	Friend VercodeType = 4
	// 手机联系人
	MailList VercodeType = 5
)

// QRCodeModel QRCodeModel
type QRCodeModel struct {
	Type QRCodeType             `json:"type"` // 二维码类型
	Data map[string]interface{} `json:"data"`
}

// UserStatus 用户状态
type UserStatus int

const (
	// UserAvailable 可用
	UserAvailable UserStatus = 1
	// UserDisable 禁用
	UserDisable UserStatus = 0
)

// NewQRCodeModel NewQRCodeModel
func NewQRCodeModel(typ QRCodeType, data map[string]interface{}) *QRCodeModel {
	return &QRCodeModel{
		Type: typ,
		Data: data,
	}
}

// MarshalJSON MarshalJSON
func (q QRCodeType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(string(q))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON UnmarshalJSON
func (q *QRCodeType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	*q = QRCodeType(j)
	return nil
}

type RTCCallType int

const (
	RTCCallTypeAudio RTCCallType = 0 // 语音通话
	RTCCallTypeVideo RTCCallType = 1 // 视频通话
)

type RTCResultType int

const (
	RTCResultTypeCancel RTCResultType = 0 // 取消通话
	RTCResultTypeHangup RTCResultType = 1 // 挂断通话
	RTCResultTypeMissed RTCResultType = 2 // 未接听
	RTCResultTypeRefuse RTCResultType = 3 // 拒绝接听
)
