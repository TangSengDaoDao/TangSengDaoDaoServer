package group

import (
	"errors"
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"go.uber.org/zap"
)

type settingContext struct {
	loginUID     string
	loginName    string
	groupSetting *Setting
	newSetting   bool
	g            *Group
}

func (c *settingContext) updateGroupSetting() error {
	if c.newSetting {
		err := c.g.settingDB.InsertSetting(c.groupSetting)
		if err != nil {
			return err
		}
	} else {
		err := c.g.settingDB.UpdateSetting(c.groupSetting)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *settingContext) sendChannelUpdate() error {
	return c.g.ctx.SendChannelUpdate(config.ChannelReq{
		ChannelID:   c.loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
	}, config.ChannelReq{
		ChannelID:   c.groupSetting.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
	})
}

func (c *settingContext) updateSettingAndSendCMD() error {
	err := c.updateGroupSetting()
	if err != nil {
		return err
	}
	return c.sendChannelUpdate()
}

type groupUpdateContext struct {
	loginUID   string
	loginName  string
	groupModel *Model
	g          *Group
}

func (g *groupUpdateContext) isManager() (bool, error) {
	isManager, err := g.g.db.QueryIsGroupManagerOrCreator(g.groupModel.GroupNo, g.loginUID)
	if err != nil {
		g.g.Error("查询是否是群管理者失败！", zap.Error(err))
		return false, err
	}
	return isManager, nil
}

func (g *groupUpdateContext) checkPermissions() error {
	isManager, err := g.isManager()
	if err != nil {
		return err
	}
	if !isManager {
		return errors.New("没有权限！")
	}
	return nil
}

func (g *groupUpdateContext) updateGroup() error {
	return g.g.db.Update(g.groupModel)
}

func (g *groupUpdateContext) commmitGroupUpdateEvent(key, value string) error {
	tx, err := g.g.ctx.DB().Begin()
	util.CheckErr(err)
	groupNo := g.groupModel.GroupNo
	// 发布群信息更新事件
	eventID, err := g.g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupUpdate,
		Type:  wkevent.Message,
		Data: &config.MsgGroupUpdateReq{
			GroupNo:      groupNo,
			Operator:     g.loginUID,
			OperatorName: g.loginName,
			Attr:         key,
			Data: map[string]string{
				key: value,
			},
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		g.g.Error("开启群更新事件失败！", zap.Error(err))
		return errors.New("开启群更新事件失败！")
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		g.g.Error("提交事务失败！", zap.Error(err))
		return errors.New("提交事务失败！")
	}
	g.g.ctx.EventCommit(eventID)

	g.g.ctx.SendChannelUpdateToGroup(groupNo) // 发送频道更新cmd

	return nil
}

type groupUpdateActionFnc func(ctx *groupUpdateContext, value interface{}) error

type groupSettingActionFnc func(ctx *settingContext, value interface{}) error

// 设置action
var settingActionMap = map[string]groupSettingActionFnc{
	"mute": func(ctx *settingContext, value interface{}) error { // 免打扰
		ctx.groupSetting.Mute = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"top": func(ctx *settingContext, value interface{}) error { // 会话置顶
		ctx.groupSetting.Top = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"save": func(ctx *settingContext, value interface{}) error { // 保存群
		ctx.groupSetting.Save = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"show_nick": func(ctx *settingContext, value interface{}) error { // 是否显示昵称
		ctx.groupSetting.ShowNick = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"chat_pwd_on": func(ctx *settingContext, value interface{}) error { // 聊天密码
		ctx.groupSetting.ChatPwdOn = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"screenshot": func(ctx *settingContext, value interface{}) error { // 截屏
		ctx.groupSetting.Screenshot = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"join_group_remind": func(ctx *settingContext, value interface{}) error { // 进群提醒
		ctx.groupSetting.JoinGroupRemind = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"revoke_remind": func(ctx *settingContext, value interface{}) error { // 撤回提醒
		ctx.groupSetting.RevokeRemind = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"receipt": func(ctx *settingContext, value interface{}) error { // 消息已读回执
		ctx.groupSetting.Receipt = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"remark": func(ctx *settingContext, value interface{}) error { // 群备注
		ctx.groupSetting.Remark = value.(string)
		return ctx.updateSettingAndSendCMD()
	},
	"flame": func(ctx *settingContext, value interface{}) error { // 阅后即焚开启
		ctx.groupSetting.Flame = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
	"flame_second": func(ctx *settingContext, value interface{}) error { // 阅后即焚时间
		ctx.groupSetting.FlameSecond = int(value.(float64))
		return ctx.updateSettingAndSendCMD()
	},
}

var groupUpdateActionMap = map[string]groupUpdateActionFnc{
	common.GroupAttrKeyForbidden: func(ctx *groupUpdateContext, value interface{}) error { // 群内禁言
		if err := ctx.checkPermissions(); err != nil {
			return err
		}
		ctx.groupModel.Forbidden = int(value.(float64))

		err := ctx.updateGroup()
		if err != nil {
			return err
		}

		groupNo := ctx.groupModel.GroupNo

		whitelistUIDs := make([]string, 0)
		if ctx.groupModel.Forbidden == 1 {
			managerOrCreaterUIDs, err := ctx.g.db.QueryGroupManagerOrCreatorUIDS(groupNo)
			if err != nil {
				return err
			}
			whitelistUIDs = managerOrCreaterUIDs
		}
		err = ctx.g.ctx.IMWhitelistSet(config.ChannelWhitelistReq{
			ChannelReq: config.ChannelReq{
				ChannelID:   groupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
			},
			UIDs: whitelistUIDs,
		})
		if err != nil {
			ctx.g.Error("设置禁言失败！", zap.Error(err))
			return errors.New("设置禁言失败！")
		}

		ctx.commmitGroupUpdateEvent(common.GroupAttrKeyForbidden, fmt.Sprintf("%d", ctx.groupModel.Forbidden))

		return nil
	},
	common.GroupAttrKeyForbiddenAddFriend: func(ctx *groupUpdateContext, value interface{}) error { // 群内禁止加好友
		if err := ctx.checkPermissions(); err != nil {
			return err
		}
		ctx.groupModel.ForbiddenAddFriend = int(value.(float64))
		err := ctx.updateGroup()
		if err != nil {
			return err
		}
		groupNo := ctx.groupModel.GroupNo
		// 通知群内成员更新频道
		err = ctx.g.ctx.SendChannelUpdateToGroup(groupNo)

		return err
	},
	common.GroupAttrKeyInvite: func(ctx *groupUpdateContext, value interface{}) error { // 邀请开关
		if err := ctx.checkPermissions(); err != nil {
			return err
		}
		ctx.groupModel.Invite = int(value.(float64))

		err := ctx.updateGroup()
		if err != nil {
			return err
		}

		return ctx.commmitGroupUpdateEvent(common.GroupAttrKeyInvite, fmt.Sprintf("%d", ctx.groupModel.Invite))
	},
	common.GroupAllowViewHistoryMsg: func(ctx *groupUpdateContext, value interface{}) error {
		if err := ctx.checkPermissions(); err != nil {
			return err
		}
		ctx.groupModel.AllowViewHistoryMsg = int(value.(float64))

		err := ctx.updateGroup()
		if err != nil {
			return err
		}
		groupNo := ctx.groupModel.GroupNo
		// 通知群内成员更新频道
		return ctx.g.ctx.SendChannelUpdateToGroup(groupNo)
	},
}
