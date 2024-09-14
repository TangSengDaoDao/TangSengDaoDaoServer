package user

import (
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/source"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// 处理通过好友
func (f *Friend) handleFriendSure(data []byte, commit config.EventCommit) {
	var req map[string]interface{}
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		f.Error("好友关系处理通过好友申请参数有误")
		commit(err)
		return
	}
	uid := req["uid"].(string)
	toUID := req["to_uid"].(string)
	if uid == "" || toUID == "" {
		commit(errors.New("好友ID不能为空"))
		return
	}
	loginUidList := make([]string, 0, 1)
	loginUidList = append(loginUidList, toUID)
	err = f.ctx.IMWhitelistAdd(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   uid,
			ChannelType: common.ChannelTypePerson.Uint8(),
		},
		UIDs: loginUidList,
	})
	if err != nil {
		commit(errors.New("添加IM白名单错误"))
		return
	}
	applyUIDList := make([]string, 0, 1)
	applyUIDList = append(applyUIDList, uid)
	err = f.ctx.IMWhitelistAdd(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   toUID,
			ChannelType: common.ChannelTypePerson.Uint8(),
		},
		UIDs: applyUIDList,
	})
	if err != nil {
		commit(errors.New("添加IM白名单错误"))
		return
	}
	commit(nil)
}

// 处理删除好友
func (f *Friend) handleDeleteFriend(data []byte, commit config.EventCommit) {
	var req map[string]interface{}
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		f.Error("处理删除好友参数错误")
		commit(err)
		return
	}
	uid := req["uid"].(string)
	toUID := req["to_uid"].(string)
	if uid == "" || toUID == "" {
		commit(errors.New("好友ID不能为空"))
		return
	}

	err = f.ctx.IMWhitelistRemove(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   toUID,
			ChannelType: common.ChannelTypePerson.Uint8(),
		},
		UIDs: []string{uid},
	})
	if err != nil {
		commit(errors.New("移除IM白名单错误"))
		return
	}
	err = f.ctx.IMWhitelistRemove(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   uid,
			ChannelType: common.ChannelTypePerson.Uint8(),
		},
		UIDs: []string{toUID},
	})
	if err != nil {
		commit(errors.New("移除IM白名单错误"))
		return
	}
	commit(nil)
}

// 处理用户注册
func (f *Friend) handleUserRegister(data []byte, commit config.EventCommit) {
	var req map[string]interface{}
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		f.Error("好友处理用户注册加入群聊参数有误")
		commit(err)
		return
	}
	if req == nil || req["invite_vercode"] == nil {
		commit(nil)
		return
	}

	inviteVercode := req["invite_vercode"].(string)
	if inviteVercode == "" {
		commit(nil)
		return
	}
	uid := req["uid"].(string)
	if uid == "" {
		f.Error("好友处理用户注册uid不能为空")
		commit(errors.New("好友处理用户注册uid不能为空"))
		return
	}
	inviteUid := req["invite_uid"].(string)
	if inviteUid == "" {
		f.Error("好友处理用户注册邀请者uid不能为空")
		commit(errors.New("好友处理用户注册邀请者uid不能为空"))
		return
	}
	// 是否是好友
	applyFriendModel, err := f.db.queryWithUID(uid, inviteUid)
	if err != nil {
		f.Error("查询是否是好友失败！", zap.Error(err), zap.String("uid", uid), zap.String("toUid", inviteUid))
		commit(errors.New("查询是否是好友失败！"))
		return
	}
	// 添加好友到数据库
	tx, err := f.ctx.DB().Begin()
	if err != nil {
		f.Error("开启事务失败！", zap.Error(err))
		commit(errors.New("开启事务失败！"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	version := f.ctx.GenSeq(common.FriendSeqKey)
	if applyFriendModel == nil {
		// 验证code
		err = source.CheckSource(inviteVercode)
		if err != nil {
			commit(err)
			return
		}

		util.CheckErr(err)
		err = f.db.InsertTx(&FriendModel{
			UID:           uid,
			ToUID:         inviteUid,
			Version:       version,
			Initiator:     0,
			IsAlone:       0,
			Vercode:       fmt.Sprintf("%s@%d", util.GenerUUID(), common.Friend),
			SourceVercode: inviteVercode,
		}, tx)
		if err != nil {
			tx.Rollback()
			commit(errors.New("添加好友失败！"))
			return
		}
	} else {
		err = f.db.updateRelationshipTx(uid, inviteUid, 0, 0, inviteVercode, version, tx)
		if err != nil {
			tx.Rollback()
			commit(errors.New("修改好友关系失败"))
			return
		}
	}
	// 是否是好友
	loginFriendModel, err := f.db.queryWithUID(inviteUid, uid)
	//loginIsFriend, err := f.db.IsFriend(applyUID, loginUID)
	if err != nil {
		tx.Rollback()
		f.Error("查询被添加者是否是好友失败！", zap.Error(err), zap.String("uid", uid), zap.String("toUid", inviteUid))
		commit(errors.New("查询被添加者是否是好友失败！"))
		return
	}
	if loginFriendModel == nil {
		err = f.db.InsertTx(&FriendModel{
			UID:           inviteUid,
			ToUID:         uid,
			Version:       version,
			Initiator:     1,
			IsAlone:       0,
			Vercode:       fmt.Sprintf("%s@%d", util.GenerUUID(), common.Friend),
			SourceVercode: inviteVercode,
		}, tx)
		if err != nil {
			tx.Rollback()
			commit(errors.New("添加好友失败！"))
			return
		}
	} else {
		err = f.db.updateRelationshipTx(inviteUid, uid, 0, 0, inviteVercode, version, tx)
		if err != nil {
			tx.Rollback()
			commit(errors.New("修改好友关系失败"))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		f.Error("提交事务失败！", zap.Error(err))
		commit(errors.New("提交事务失败！"))
		return
	}
	// 添加白名单
	loginUidList := make([]string, 0, 1)
	loginUidList = append(loginUidList, inviteUid)
	err = f.ctx.IMWhitelistAdd(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   uid,
			ChannelType: common.ChannelTypePerson.Uint8(),
		},
		UIDs: loginUidList,
	})
	if err != nil {
		commit(errors.New("添加IM白名单错误"))
		return
	}
	applyUIDList := make([]string, 0, 1)
	applyUIDList = append(applyUIDList, uid)
	err = f.ctx.IMWhitelistAdd(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   inviteUid,
			ChannelType: common.ChannelTypePerson.Uint8(),
		},
		UIDs: applyUIDList,
	})
	if err != nil {
		commit(errors.New("添加IM白名单错误"))
		return
	}
	userInfo, err := f.userDB.QueryByUID(uid)
	if err != nil {
		commit(errors.New("查询用户资料错误"))
		return
	}
	if userInfo == nil {
		commit(errors.New("用户不存在"))
		return
	}
	// 发送确认消息给对方
	err = f.ctx.SendCMD(config.MsgCMDReq{
		CMD:         common.CMDFriendAccept,
		Subscribers: []string{uid, inviteUid},
		Param: map[string]interface{}{
			"to_uid":    inviteUid,
			"from_uid":  uid,
			"from_name": userInfo.Name,
		},
	})
	if err != nil {
		f.Error("发送消息失败！", zap.Error(err))
		commit(errors.New("发送消息失败！"))
		return
	}
	content := "我们已经是好友了，可以愉快的聊天了！"
	if f.ctx.GetConfig().Friend.AddedTipsText != "" {
		content = f.ctx.GetConfig().Friend.AddedTipsText
	}
	// 发送消息
	payload := []byte(util.ToJson(map[string]interface{}{
		"content": content,
		"type":    common.Tip,
	}))

	err = f.ctx.SendMessage(&config.MsgSendReq{
		FromUID:     uid,
		ChannelID:   inviteUid,
		ChannelType: common.ChannelTypePerson.Uint8(),
		Payload:     payload,
		Header: config.MsgHeader{
			RedDot: 1,
		},
	})
	if err != nil {
		f.Error("发送通过好友请求消息失败！", zap.Error(err))
		commit(errors.New("发送通过好友请求消息失败！"))
		return
	}

	commit(nil)
}
