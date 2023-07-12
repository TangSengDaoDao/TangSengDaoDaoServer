package config

import (
	"fmt"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
)

// SendGroupCreate 发送群创建的消息
func (c *Context) SendGroupCreate(req *MsgGroupCreateReq) error {
	members := req.Members
	if members == nil {
		members = make([]*UserBaseVo, 0)
	}

	params := make([]string, 0, len(members))
	newMembers := make([]*UserBaseVo, 0, len(members))
	i := 0
	for _, member := range members {
		if member.UID == req.Creator {
			continue
		}
		newMembers = append(newMembers, member)
		params = append(params, fmt.Sprintf("{%d}", i))
		i++
	}
	content := fmt.Sprintf("%s邀请%s加入群聊", req.CreatorName, strings.Join(params, ","))

	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"creator":      req.Creator,
			"creator_name": req.CreatorName,
			"content":      content,
			"version":      req.Version,
			"extra":        newMembers,
			"type":         common.GroupCreate,
		})),
	})
}

// SendGroupUnableAddDestoryAccount 发送无法添加注销账号到群聊
func (c *Context) SendUnableAddDestoryAccountInGroup(req *MsgGroupCreateReq) error {
	members := req.Members
	if members == nil {
		members = make([]*UserBaseVo, 0)
	}

	params := make([]string, 0, len(members))
	newMembers := make([]*UserBaseVo, 0, len(members))
	i := 0
	for _, member := range members {
		if member.UID == req.Creator {
			continue
		}
		newMembers = append(newMembers, member)
		params = append(params, fmt.Sprintf("{%d}", i))
		i++
	}
	content := fmt.Sprintf("用户%s已注销，无法添加到群聊", strings.Join(params, ","))

	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"extra":   newMembers,
			"type":    common.Tip,
		})),
	})
}

// SendGroupUpdate 发送群更新消息
func (c *Context) SendGroupUpdate(req *MsgGroupUpdateReq) error {
	// if req.Data == nil {
	// 	c.Error("发送群更新消息失败！没有data数据")
	// 	return nil
	// }
	content := "{0}"
	switch req.Attr {
	case common.GroupAttrKeyName:
		content += fmt.Sprintf(`修改群名为"%s"`, req.Data[common.GroupAttrKeyName])
		break
	case common.GroupAttrKeyNotice:
		notice := req.Data[common.GroupAttrKeyNotice]
		if notice == "" {
			content += "清空了群公告"
		} else {
			content += fmt.Sprintf(`修改群公告为"%s"`, notice)
		}
		break
	case common.GroupAttrKeyForbidden:
		forbidden, _ := req.Data[common.GroupAttrKeyForbidden]
		if forbidden == "1" {
			content += fmt.Sprintf(`开启了群禁言`)
		} else {
			content += fmt.Sprintf(`关闭了群禁言`)
		}
		break
	case common.GroupAttrKeyInvite:
		invite, _ := req.Data[common.GroupAttrKeyInvite]
		if invite == "1" {
			content += fmt.Sprintf(`已启用“群聊邀请确认”，群成员需群主或管理员确认才能邀请朋友进群。`)
		} else {
			content += fmt.Sprintf(`已恢复默认进群方式。`)
		}
		break

	case common.GroupAttrKeyStatus:
		status, _ := req.Data[common.GroupAttrKeyStatus]
		if status == "1" {
			content += fmt.Sprintf(`解禁了该群`)
		} else {
			content += fmt.Sprintf(`封禁了该群`)
		}
		break
	}
	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"extra": []UserBaseVo{
				{
					UID:  req.Operator,
					Name: req.OperatorName,
				},
			},
			"data": req.Data,
			"type": common.GroupUpdate,
		})),
	})
}

// SendGroupMemberAdd 发送群成员添加消息
func (c *Context) SendGroupMemberAdd(req *MsgGroupMemberAddReq) error {
	members := req.Members
	if members == nil {
		members = make([]*UserBaseVo, 0)
	}

	params := make([]string, 0, len(members))
	for index := range members {
		params = append(params, fmt.Sprintf("{%d}", index))
	}
	content := fmt.Sprintf("%s邀请%s加入群聊", req.OperatorName, strings.Join(params, ","))

	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"from_uid":  req.Operator,
			"from_name": req.OperatorName,
			"content":   content,
			"extra":     members,
			"type":      common.GroupMemberAdd,
		})),
	})
}

// 群升级通知
func (c *Context) SendGroupUpgrade(groupNo string) error {
	content := fmt.Sprintf("群成员超过%d，将自动升级为超级群", c.cfg.GroupUpgradeWhenMemberCount)
	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"type":    common.GroupUpgrade,
		})),
	})
}

// SendGroupMemberBeRemove 发送群成员被移除的消息(发送给被踢的群成员)
func (c *Context) SendGroupMemberBeRemove(req *MsgGroupMemberRemoveReq) error {
	if len(req.Members) <= 0 {
		return nil
	}
	subscribers := make([]string, 0, len(req.Members))
	for _, member := range req.Members {
		subscribers = append(subscribers, member.UID)
	}
	setting := Setting{
		NoUpdateConversation: true,
	}
	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		Setting:     setting.ToUint8(),
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Subscribers: subscribers,
		Payload: []byte(util.ToJson(map[string]interface{}{
			// "from_uid":  req.Operator,
			// "from_name": req.OperatorName,
			"content":  "你被{0}移除群聊",
			"visibles": subscribers,
			"extra": []UserBaseVo{
				{
					UID:  req.Operator,
					Name: req.OperatorName,
				},
			},
			"type": common.GroupMemberBeRemove,
		})),
	})
}

// SendGroupMemberRemove 发送群成员移除消息
func (c *Context) SendGroupMemberRemove(req *MsgGroupMemberRemoveReq) error {
	members := req.Members
	if members == nil {
		members = make([]*UserBaseVo, 0)
	}

	params := make([]string, 0, len(members))
	for index := range members {
		params = append(params, fmt.Sprintf("{%d}", index))
	}
	content := fmt.Sprintf("%s将%s移除群聊", req.OperatorName, strings.Join(params, ","))

	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			// "from_uid":  req.Operator,
			// "from_name": req.OperatorName,
			"content": content,
			"extra":   members,
			"type":    common.GroupMemberRemove,
		})),
	})
}

// SendGroupMemberScanJoin 发送群成员扫码加入消息
func (c *Context) SendGroupMemberScanJoin(req MsgGroupMemberScanJoin) error {
	content := fmt.Sprintf(`“{0}”通过“{1}”的二维码加入群聊`)
	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"extra": []UserBaseVo{
				{
					UID:  req.Scaner,
					Name: req.ScanerName,
				},
				{
					UID:  req.Generator,
					Name: req.GeneratorName,
				},
			},
			"type": common.GroupMemberScanJoin,
		}))})
}

// SendGroupTransferGrouper 群主转让
func (c *Context) SendGroupTransferGrouper(req MsgGroupTransferGrouper) error {
	content := fmt.Sprintf(`“{0}”已成为新群主`)
	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"extra": []UserBaseVo{
				{
					UID:  req.NewGrouper,
					Name: req.NewGrouperName,
				},
			},
			"type": common.GroupTransferGrouper,
		}))})
}

// SendGroupMemberInviteReq 群主转让
func (c *Context) SendGroupMemberInviteReq(req MsgGroupMemberInviteReq) error {
	content := fmt.Sprintf(`“{0}“想邀请%d位朋友加入群聊`, req.Num)
	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Subscribers: req.Subscribers,
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"extra": []UserBaseVo{
				{
					UID:  req.Inviter,
					Name: req.InviterName,
				},
			},
			"invite_no": req.InviteNo,
			"type":      common.GroupMemberInvite,
			"visibles":  req.Subscribers,
		}))})
}

// 发送某个用户退出群聊的消息
func (c *Context) SendGroupExit(groupNo string, uid string, name string) error {
	// 发送群成员退出群聊消息
	err := c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			RedDot: 1,
		},
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": "“{0}“退出群聊",
			"type":    common.GroupMemberQuit,
			"extra": []UserBaseVo{
				{
					UID:  uid,
					Name: name,
				},
			},
		})),
	})

	return err
}

func (c *Context) SendGroupMemberUpdate(groupNo string) error {
	return c.SendCMD(MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupMemberUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
}

// MsgGroupCreateReq 创建群请求
type MsgGroupCreateReq struct {
	Creator     string        `json:"creator"`      // 创建者
	CreatorName string        `json:"creator_name"` // 创建者名称
	GroupNo     string        `json:"group_no"`
	Version     int64         `json:"version"` // 数据版本
	Members     []*UserBaseVo `json:"members"`
}

// MsgGroupMemberInviteReq 群成员邀请请求
type MsgGroupMemberInviteReq struct {
	GroupNo     string   `json:"group_no"`     // 群编号
	InviteNo    string   `json:"invite_no"`    // 邀请编号
	Inviter     string   `json:"inviter"`      // 邀请者
	InviterName string   `json:"inviter_name"` // 邀请者名称
	Num         int      `json:"num"`          // 邀请成员数量
	Subscribers []string `json:"subscribers"`  // 消息订阅者
}

// MsgGroupTransferGrouper 群主转让
type MsgGroupTransferGrouper struct {
	GroupNo        string `json:"group_no"`
	OldGrouper     string `json:"old_grouper"`      // 老群主
	OldGrouperName string `json:"old_grouper_name"` // 老群主名称
	NewGrouper     string `json:"new_grouper"`      // 新群主
	NewGrouperName string `json:"new_grouper_name"` // 新群主名称
}

// MsgGroupMemberRemoveReq 移除群成员
type MsgGroupMemberRemoveReq struct {
	Operator     string        `json:"operator"`      // 操作者uid
	OperatorName string        `json:"operator_name"` // 操作者名称
	GroupNo      string        `json:"group_no"`      // 群编号
	Members      []*UserBaseVo `json:"members"`       // 邀请成员
}

// MsgGroupUpdateReq 群更新请求
type MsgGroupUpdateReq struct {
	GroupNo      string            `json:"group_no"`      // 群编号
	Operator     string            `json:"operator"`      // 操作者uid
	OperatorName string            `json:"operator_name"` // 操作者名称
	Attr         string            `json:"attr"`          // 修改群的属性
	Data         map[string]string `json:"data"`          // 数据
}

// MsgGroupMemberScanJoin 用户扫码加入群成员
type MsgGroupMemberScanJoin struct {
	GroupNo       string `json:"group_no"`       // 群编号
	Generator     string `json:"generator"`      // 二维码生成者uid
	GeneratorName string `json:"generator_name"` // 二维码生成者名称
	Scaner        string `json:"scaner"`         // 扫码者uid
	ScanerName    string `json:"scaner_name"`    // 扫码者名称
}
