package group

import (
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/pool"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

// handleRegisterUserEvent 用户注册时加入系统群
func (g *Group) handleRegisterUserEvent(data []byte, commit config.EventCommit) {
	appconfig, _ := g.commonService.GetAppConfig()
	if appconfig != nil && appconfig.NewUserJoinSystemGroup == 0 {
		commit(nil)
		return
	}
	var req map[string]interface{}
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		g.Error("处理用户注册加入群聊参数有误")
		commit(err)
		return
	}
	uid := req["uid"].(string)
	if uid == "" {
		g.Error("处理用户注册加入群聊UID不能为空")
		commit(errors.New("处理用户注册加入群聊UID不能为空"))
		return
	}
	//查询群聊是否存在
	groupModel, err := g.db.QueryWithGroupNo(g.ctx.GetConfig().Account.SystemGroupID)
	if err != nil {
		g.Error("查询群详情失败")
		commit(err)
		return
	}
	tx, err := g.db.session.Begin()
	if err != nil {
		g.Error("开启事物失败")
		tx.Rollback()
		commit(err)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			commit(err.(error))
			panic(err)
		}
	}()
	if groupModel == nil {
		//创建群
		version := g.ctx.GenSeq(common.GroupSeqKey)
		err = g.db.InsertTx(&Model{
			GroupNo: g.ctx.GetConfig().Account.SystemGroupID,
			Name:    g.ctx.GetConfig().Account.SystemGroupName,
			Creator: g.ctx.GetConfig().Account.SystemUID,
			Status:  GroupStatusNormal,
			Version: version,
		}, tx)
		if err != nil {
			g.Error("创建群聊失败")
			tx.Rollback()
			commit(err)
			return
		}
		//添加创建者
		memberVersion := g.ctx.GenSeq(common.GroupMemberSeqKey)
		err = g.db.InsertMemberTx(&MemberModel{
			GroupNo: g.ctx.GetConfig().Account.SystemGroupID,
			UID:     g.ctx.GetConfig().Account.SystemUID,
			Role:    MemberRoleCreator,
			Status:  int(common.GroupMemberStatusNormal),
			Version: memberVersion,
		}, tx)
		if err != nil {
			g.Error("设置系统群创建者失败")
			tx.Rollback()
			commit(err)
			return
		}
		realMemberUids := make([]string, 0)
		realMemberUids = append(realMemberUids, g.ctx.GetConfig().Account.SystemUID)
		// 创建IM频道
		err = g.ctx.IMCreateOrUpdateChannel(&config.ChannelCreateReq{
			ChannelID:   g.ctx.GetConfig().Account.SystemGroupID,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			Subscribers: realMemberUids,
		})
		if err != nil {
			g.Error("创建im频道失败")
			tx.Rollback()
			commit(err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		g.Error("事物提交失败")
		tx.Rollback()
		commit(err)
		return
	}

	//将新注册的用户添加到系统群
	realMemberUids := make([]string, 0)
	realMemberUids = append(realMemberUids, uid)
	err = g.addMembers(realMemberUids, g.ctx.GetConfig().Account.SystemGroupID, g.ctx.GetConfig().Account.SystemUID, "系统账号")
	if err != nil {
		g.Error("添加注册账号到系统群失败！")
		commit(err)
		return
	}
	commit(nil)
}

// 处理群成员添加事件
func (g *Group) handleGroupMemberAddEvent(data []byte, commit config.EventCommit) {

	g.ctx.EventPool.Work <- &pool.Job{
		Data: data,
		JobFunc: func(id int64, data interface{}) {
			var dataBytes = data.([]byte)
			var req *config.MsgGroupMemberAddReq
			err := util.ReadJsonByByte(dataBytes, &req)
			if err != nil {
				g.Error("解析JSON失败！", zap.Error(err))
				return
			}
			err = g.ctx.SendGroupMemberAdd(req)
		},
	}
}
