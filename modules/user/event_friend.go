package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/pkg/errors"
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
