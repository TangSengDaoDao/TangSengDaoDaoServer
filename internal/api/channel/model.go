package channel

import (
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/api/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/api/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/config"
)

type channelResp struct {
	Channel struct {
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
	} `json:"channel"`
	ParentChannel *struct {
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
	} `json:"parent_channel,omitempty"`
	Username    string            `json:"username,omitempty"` // 频道唯一标识（目前只有机器人有用到）
	Name        string            `json:"name"`               // 频道名称
	Logo        string            `json:"logo"`               // 频道logo
	Remark      string            `json:"remark"`             // 频道备注
	Status      int               `json:"status"`             //  频道状态 0.正常 1.正常  2.黑名单
	Online      int               `json:"online"`             // 是否在线
	LastOffline int64             `json:"last_offline"`       // 最后一次离线
	DeviceFlag  config.DeviceFlag `json:"device_flag"`        // 设备标记
	Receipt     int               `json:"receipt"`            // 消息是否回执
	Robot       int               `json:"robot"`              // 是否是机器人
	Category    string            `json:"category"`           // 频道类别
	// 设置
	Stick    int `json:"stick"`     // 是否置顶
	Mute     int `json:"mute"`      // 是否免打扰
	ShowNick int `json:"show_nick"` // 是否显示昵称
	// 个人特有
	Follow      int `json:"follow"`       // 是否已关注 0.未关注（陌生人） 1.已关注（好友）
	BeDeleted   int `json:"be_deleted"`   // 是否被对方删除
	BeBlacklist int `json:"be_blacklist"` // 是否被对方拉入黑名单
	// 群特有
	Notice    string `json:"notice"`    // 群公告
	Save      int    `json:"save"`      // 群是否保存
	Forbidden int    `json:"forbidden"` // 群是否全员禁言
	Invite    int    `json:"invite"`    // 是否开启邀请

	Flame       int `json:"flame"`        // 阅后即焚
	FlameSecond int `json:"flame_second"` // 阅后即焚秒数

	// 此内容在扩展内容内
	// Screenshot          int `json:"screenshot"`             // 是否开启截屏通知
	// ForbiddenAddFriend  int `json:"forbidden_add_friend"`   // 是否禁止群内添加好友
	// JoinGroupRemind     int `json:"join_group_remind"`      // 是否开启进群提醒
	// RevokeRemind        int `json:"revoke_remind"`          // 是否开启撤回通知
	// chatPwdOn           int `json:"chat_pwd_on"`            // 是否开启聊天密码
	// AllowViewHistoryMsg int `json:"allow_view_history_msg"` // 是否允许新成员查看群历史记录

	Extra map[string]interface{} `json:"extra"` // 扩展内容
}

func newChannelRespWithUserDetailResp(user *user.UserDetailResp) *channelResp {

	resp := &channelResp{}
	resp.Channel.ChannelID = user.UID
	resp.Channel.ChannelType = uint8(common.ChannelTypePerson)
	resp.Name = user.Name
	resp.Username = user.Username
	resp.Logo = fmt.Sprintf("users/%s/avatar", user.UID)
	resp.Mute = user.Mute
	resp.Stick = user.Top
	resp.Receipt = user.Receipt
	resp.Robot = user.Robot
	resp.Online = user.Online
	resp.LastOffline = int64(user.LastOffline)
	resp.DeviceFlag = user.DeviceFlag
	resp.Category = user.Category
	resp.Follow = user.Follow
	resp.Remark = user.Remark
	resp.Status = user.Status
	resp.BeBlacklist = user.BeBlacklist
	resp.BeDeleted = user.BeDeleted
	resp.Flame = user.Flame
	resp.FlameSecond = user.FlameSecond
	extraMap := make(map[string]interface{})
	extraMap["sex"] = user.Sex
	extraMap["chat_pwd_on"] = user.ChatPwdOn
	extraMap["short_no"] = user.ShortNo
	extraMap["source_desc"] = user.SourceDesc
	extraMap["vercode"] = user.Vercode
	extraMap["screenshot"] = user.Screenshot
	extraMap["revoke_remind"] = user.RevokeRemind
	resp.Extra = extraMap

	return resp
}

func newChannelRespWithGroupResp(groupResp *group.GroupResp) *channelResp {
	resp := &channelResp{}
	resp.Channel.ChannelID = groupResp.GroupNo
	resp.Channel.ChannelType = uint8(common.ChannelTypeGroup)
	resp.Name = groupResp.Name
	resp.Remark = groupResp.Remark
	resp.Logo = fmt.Sprintf("groups/%s/avatar", groupResp.GroupNo)
	resp.Notice = groupResp.Notice
	resp.Mute = groupResp.Mute
	resp.Stick = groupResp.Top
	resp.Receipt = groupResp.Receipt
	resp.ShowNick = groupResp.ShowNick
	resp.Forbidden = groupResp.Forbidden
	resp.Invite = groupResp.Invite
	resp.Status = groupResp.Status
	resp.Save = groupResp.Save
	resp.Remark = groupResp.Remark
	resp.Flame = groupResp.Flame
	resp.FlameSecond = groupResp.FlameSecond
	extraMap := make(map[string]interface{})
	extraMap["forbidden_add_friend"] = groupResp.ForbiddenAddFriend
	extraMap["screenshot"] = groupResp.Screenshot
	extraMap["revoke_remind"] = groupResp.RevokeRemind
	extraMap["join_group_remind"] = groupResp.JoinGroupRemind
	extraMap["chat_pwd_on"] = groupResp.ChatPwdOn
	extraMap["allow_view_history_msg"] = groupResp.AllowViewHistoryMsg
	extraMap["group_type"] = groupResp.GroupType

	if groupResp.MemberCount != 0 {
		extraMap["member_count"] = groupResp.MemberCount
	}
	if groupResp.OnlineCount != 0 {
		extraMap["online_count"] = groupResp.OnlineCount
	}
	if groupResp.Quit != 0 {
		extraMap["quit"] = groupResp.Quit
	}
	if groupResp.Role != 0 {
		extraMap["role"] = groupResp.Role
	}
	if groupResp.ForbiddenExpirTime != 0 {
		extraMap["forbidden_expir_time"] = groupResp.ForbiddenExpirTime
	}

	resp.Extra = extraMap

	return resp
}
