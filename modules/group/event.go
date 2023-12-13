package group

import (
	"errors"
	"fmt"
	"strings"

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

// 处理创建组织或部门事件
func (g *Group) handleOrgOrDeptCreateEvent(data []byte, commit config.EventCommit) {
	var req config.MsgOrgOrDeptCreateReq
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		g.Error("解析JSON失败！", zap.Error(err))
		commit(nil)
		return
	}
	groupModel, err := g.db.QueryWithGroupNo(req.GroupNo)
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
		// 创建群
		version := g.ctx.GenSeq(common.GroupSeqKey)
		err = g.db.InsertTx(&Model{
			GroupNo:             req.GroupNo,
			Name:                req.Name,
			Creator:             req.Operator,
			Status:              GroupStatusNormal,
			Version:             version,
			Invite:              1,
			AllowViewHistoryMsg: 1,
			Category:            req.GroupCategory,
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
			GroupNo: req.GroupNo,
			UID:     req.Operator,
			Role:    MemberRoleCreator,
			Status:  int(common.GroupMemberStatusNormal),
			Version: memberVersion,
			Vercode: fmt.Sprintf("%s@%d", util.GenerUUID(), common.GroupMember),
		}, tx)
		if err != nil {
			g.Error("设置群创建者失败")
			tx.Rollback()
			commit(err)
			return
		}
		realMemberUids := make([]string, 0)
		if len(req.Members) > 0 {
			for _, member := range req.Members {
				realMemberUids = append(realMemberUids, member.EmployeeUid)
				memberVersion := g.ctx.GenSeq(common.GroupMemberSeqKey)
				err = g.db.InsertMemberTx(&MemberModel{
					GroupNo: req.GroupNo,
					UID:     member.EmployeeUid,
					Role:    MemberRoleCommon,
					Status:  int(common.GroupMemberStatusNormal),
					Version: memberVersion,
					Vercode: fmt.Sprintf("%s@%d", util.GenerUUID(), common.GroupMember),
				}, tx)
				if err != nil {
					g.Error("添加群成员错误")
					tx.Rollback()
					commit(err)
					return
				}
			}
		}

		realMemberUids = append(realMemberUids, req.Operator)
		// 创建IM频道
		err = g.ctx.IMCreateOrUpdateChannel(&config.ChannelCreateReq{
			ChannelID:   req.GroupNo,
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
	// 发送一条系统消息
	content := fmt.Sprintf("欢迎%s加入%s，新成员入群可查看所有历史消息", req.OperatorName, req.Name)
	err = g.ctx.SendMessage(&config.MsgSendReq{
		Header: config.MsgHeader{
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
			"type":      common.GroupMemberAdd,
		})),
	})
	if err != nil {
		g.Error("发送系统消息错误")
		commit(err)
		return
	}
	commit(nil)
}

// 批量处理组织或部门成员改变部门事件
func (g *Group) handleOrgOrDeptEmployeeUpdate(data []byte, commit config.EventCommit) {
	var req config.MsgOrgOrDeptEmployeeUpdateReq
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		g.Error("解析JSON失败！", zap.Error(err))
		commit(nil)
		return
	}
	if len(req.Members) == 0 {
		g.Error("数据不能为空", zap.Error(errors.New("数据不能为空")))
		commit(nil)
		return
	}
	groupNos := make([]string, 0)
	for _, m := range req.Members {
		groupNos = append(groupNos, m.GroupNo)
	}
	groups, err := g.db.QueryGroupsWithGroupNos(groupNos)
	if err != nil {
		g.Error("批量查询群信息错误")
		commit(err)
		return
	}
	// 真实存在的群聊
	realList := make([]*config.OrgOrDeptEmployeeVO, 0)
	for _, m := range req.Members {
		isAdd := false
		for _, g := range groups {
			if m.GroupNo == g.GroupNo {
				isAdd = true
				break
			}
		}
		if isAdd {
			realList = append(realList, &config.OrgOrDeptEmployeeVO{
				Operator:     m.Operator,
				OperatorName: m.OperatorName,
				EmployeeUid:  m.EmployeeUid,
				EmployeeName: m.EmployeeName,
				GroupNo:      m.GroupNo,
				Action:       m.Action,
			})
		}
	}
	type tempVO struct {
		Operator     string
		OperatorName string
		EmployeeUid  string
		EmployeeName string
		Action       string
	}
	// 通过群编号分组
	list := make(map[string][]*tempVO, 0)
	for _, m := range realList {
		tempDatas := list[m.GroupNo]
		if len(tempDatas) == 0 {
			tempDatas = make([]*tempVO, 0)
		}
		tempDatas = append(tempDatas, &tempVO{
			Operator:     m.Operator,
			OperatorName: m.OperatorName,
			EmployeeUid:  m.EmployeeUid,
			EmployeeName: m.EmployeeName,
			Action:       m.Action,
		})
		list[m.GroupNo] = tempDatas
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

	// 添加或修改群成员
	for groupNo, members := range list {
		for _, member := range members {
			version := g.ctx.GenSeq(common.GroupMemberSeqKey)
			existDelete, err := g.db.ExistMemberDelete(member.EmployeeUid, groupNo)
			if err != nil {
				g.Error("查询是否存在删除成员失败！", zap.Error(err))
				tx.Rollback()
				commit(err)
				return
			}
			if member.Action == "add" {
				newMember := &MemberModel{
					GroupNo:   groupNo,
					InviteUID: member.Operator,
					UID:       member.EmployeeUid,
					Vercode:   fmt.Sprintf("%s@%d", util.GenerUUID(), common.GroupMember),
					Version:   version,
					Status:    int(common.GroupMemberStatusNormal),
					Robot:     0,
				}
				if existDelete {
					err = g.db.recoverMemberTx(newMember, tx)
				} else {
					err = g.db.InsertMemberTx(newMember, tx)
				}
				if err != nil {
					g.Error("添加群成员失败！", zap.Error(err))
					tx.Rollback()
					commit(err)
					return
				}
			} else {
				// 删除
				err = g.db.DeleteMemberTx(groupNo, member.EmployeeUid, version, tx)
				if err != nil {
					g.Error("删除群成员失败！", zap.Error(err))
					tx.Rollback()
					commit(err)
					return
				}
			}
		}
	}

	// 发布事件
	type tempMsgVO struct {
		GroupNo string
		Members []*tempVO
	}
	addMembers := make([]*tempMsgVO, 0)
	deleteMembers := make([]*tempMsgVO, 0)
	for groupNo, members := range list {
		tempList := make([]*tempVO, 0)
		for _, member := range members {
			tempList = append(tempList, &tempVO{
				Operator:     member.Operator,
				OperatorName: member.OperatorName,
				EmployeeUid:  member.EmployeeUid,
				EmployeeName: member.EmployeeName,
			})
			if member.Action == "add" {
				addMembers = append(addMembers, &tempMsgVO{
					GroupNo: groupNo,
					Members: tempList,
				})
			} else {
				deleteMembers = append(deleteMembers, &tempMsgVO{
					GroupNo: groupNo,
					Members: tempList,
				})
			}
		}
	}
	// 添加IM订阅者和发布入群消息
	for _, m := range addMembers {
		groupName := ""
		for _, group := range groups {
			if m.GroupNo == group.GroupNo {
				groupName = group.Name
				break
			}
		}
		//userBaseVos := make([]*config.UserBaseVo, 0)
		members := make([]string, 0)
		params := make([]string, 0, len(m.Members))
		for index := range m.Members {
			params = append(params, fmt.Sprintf("{%d}", index))
			members = append(members, m.Members[index].EmployeeUid)
			// userBaseVos = append(userBaseVos, &config.UserBaseVo{
			// 	UID:  m.Members[index].EmployeeUid,
			// 	Name: m.Members[index].EmployeeName,
			// })
		}
		err = g.ctx.IMAddSubscriber(&config.SubscriberAddReq{
			ChannelID:   m.GroupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			Subscribers: members,
		})
		if err != nil {
			g.Error("调用IM的订阅接口失败！", zap.Error(err))
			tx.RollbackUnlessCommitted()
			commit(err)
			return
		}

		// eventID, err := g.ctx.EventBegin(&wkevent.Data{
		// 	Event: event.OrgOrDeptEmployeeAddMsg,
		// 	Type:  wkevent.None,
		// 	Data: &config.MsgOrgOrDeptEmployeeAddReq{
		// 		GroupNo: m.GroupNo,
		// 		Name:    groupName,
		// 		Members: userBaseVos,
		// 	},
		// }, tx)
		// if err != nil {
		// 	tx.RollbackUnlessCommitted()
		// 	g.Error("开启事件失败！", zap.Error(err))
		// 	commit(err)
		// 	return
		// }
		// g.ctx.EventCommit(eventID)
		content := fmt.Sprintf("欢迎%s 加入 %s，新成员入群可查看所有历史消息", strings.Join(params, ","), groupName)
		err = g.ctx.SendMessage(&config.MsgSendReq{
			Header: config.MsgHeader{
				NoPersist: 0,
				RedDot:    1,
				SyncOnce:  0, // 只同步一次
			},
			ChannelID:   m.GroupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			Payload: []byte(util.ToJson(map[string]interface{}{
				// "from_uid":  operator,
				// "from_name": operatorName,
				"content": content,
				"extra":   members,
				"type":    common.GroupMemberAdd,
			})),
		})
		if err != nil {
			g.Error("发送新增组织或部门群成员消息错误", zap.Error(err))
			commit(nil)
			tx.RollbackUnlessCommitted()
			return
		}
	}

	if err = tx.Commit(); err != nil {
		g.Error("事物提交失败")
		tx.Rollback()
		commit(err)
		return
	}
	if len(deleteMembers) > 0 {
		for _, m := range deleteMembers {
			members := make([]string, 0)
			for index := range m.Members {
				members = append(members, m.Members[index].EmployeeUid)
			}
			err = g.ctx.IMRemoveSubscriber(&config.SubscriberRemoveReq{
				ChannelID:   m.GroupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
				Subscribers: members,
			})
			if err != nil {
				g.Error("调用IM的订阅接口失败！", zap.Error(err))
				commit(err)
				return
			}
			// 发送群成员更新命令
			err = g.ctx.SendCMD(config.MsgCMDReq{
				ChannelID:   m.GroupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
				CMD:         common.CMDGroupMemberUpdate,
				Param: map[string]interface{}{
					"group_no": m.GroupNo,
				},
			})
			if err != nil {
				g.Error("发送更新群成员cmd消息错误", zap.Error(err))
				commit(err)
				return
			}
		}
	}
	commit(nil)
}

// 处理发送新增部门或组织群成员消息
// func (g *Group) handleOrgOrDeptEmployeeAddMsg(data []byte, commit config.EventCommit) {
// 	var req config.MsgOrgOrDeptEmployeeAddReq
// 	err := util.ReadJsonByByte(data, &req)
// 	if err != nil {
// 		g.Error("解析JSON失败！", zap.Error(err))
// 		commit(nil)
// 		return
// 	}
// 	if req.GroupNo == "" {
// 		g.Error("群编号不能为空", zap.Error(errors.New("群编号不能为空")))
// 		commit(nil)
// 		return
// 	}
// 	if len(req.Members) == 0 {
// 		g.Error("新增成员列表不能为空", zap.Error(errors.New("新增成员列表不能为空")))
// 		commit(nil)
// 		return
// 	}
// 	members := make([]*config.UserBaseVo, 0)
// 	params := make([]string, 0, len(req.Members))
// 	for index := range req.Members {
// 		params = append(params, fmt.Sprintf("{%d}", index))
// 		members = append(members, &config.UserBaseVo{
// 			UID:  req.Members[index].UID,
// 			Name: req.Members[index].Name,
// 		})
// 	}
// 	content := fmt.Sprintf("欢迎%s 加入 %s，新成员入群可查看所有历史消息", strings.Join(params, ","), req.Name)
// 	err = g.ctx.SendMessage(&config.MsgSendReq{
// 		Header: config.MsgHeader{
// 			NoPersist: 0,
// 			RedDot:    1,
// 			SyncOnce:  0, // 只同步一次
// 		},
// 		ChannelID:   req.GroupNo,
// 		ChannelType: common.ChannelTypeGroup.Uint8(),
// 		Payload: []byte(util.ToJson(map[string]interface{}{
// 			// "from_uid":  operator,
// 			// "from_name": operatorName,
// 			"content": content,
// 			"extra":   members,
// 			"type":    common.GroupMemberAdd,
// 		})),
// 	})
// 	if err != nil {
// 		g.Error("发送新增组织或部门群成员消息错误", zap.Error(err))
// 		commit(nil)
// 		return
// 	}
// 	commit(nil)
// }

// 处理组织成员退出
func (g *Group) handleOrgEmployeeExit(data []byte, commit config.EventCommit) {
	var req config.OrgEmployeeExitReq
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		g.Error("解析JSON失败！", zap.Error(err))
		commit(nil)
		return
	}
	if req.Operator == "" {
		g.Error("退出用户uid不能为空", zap.Error(errors.New("退出用户uid不能为空")))
		commit(nil)
		return
	}
	if len(req.GroupNos) == 0 {
		g.Error("退出群列表不能为空", zap.Error(errors.New("退出群列表不能为空")))
		commit(nil)
		return
	}
	groups, err := g.db.QueryGroupsWithGroupNos(req.GroupNos)
	if err != nil {
		g.Error("查询群列表错误", zap.Error(err))
		commit(nil)
		return
	}
	if len(groups) == 0 {
		g.Error("所在群里不存在", zap.Error(errors.New("所在群里不存在")))
		commit(nil)
		return
	}
	realGroups := make([]string, 0)
	for _, groupNo := range req.GroupNos {
		for _, group := range groups {
			if groupNo == group.GroupNo {
				realGroups = append(realGroups, groupNo)
				break
			}
		}
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
	for _, groupNo := range realGroups {
		version := g.ctx.GenSeq(common.GroupMemberSeqKey)
		err = g.db.DeleteMemberTx(groupNo, req.Operator, version, tx)
		if err != nil {
			g.Error("删除群成员失败！", zap.Error(err))
			tx.Rollback()
			commit(err)
			return
		}
	}
	err = tx.Commit()
	if err != nil {
		g.Error("提交事物错误", zap.Error(err))
		tx.Rollback()
		commit(err)
		return
	}
	for _, groupNo := range realGroups {
		members := make([]string, 0)
		members = append(members, req.Operator)
		err = g.ctx.IMRemoveSubscriber(&config.SubscriberRemoveReq{
			ChannelID:   groupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			Subscribers: members,
		})
		if err != nil {
			g.Error("调用IM的订阅接口失败！", zap.Error(err))
			commit(err)
			return
		}
		// 发送群成员更新命令
		err = g.ctx.SendCMD(config.MsgCMDReq{
			ChannelID:   groupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			CMD:         common.CMDGroupMemberUpdate,
			Param: map[string]interface{}{
				"group_no": groupNo,
			},
		})
		if err != nil {
			g.Error("发送更新群成员cmd消息错误", zap.Error(err))
			commit(err)
			return
		}
	}
	commit(nil)
}
