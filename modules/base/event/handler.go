package event

import (
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/pool"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

// Handler 事件处理者
type Handler func(model *Model)

var handlerMap = map[string]Handler{}

func (e *Event) registerHandlers() {
	handlerMap = map[string]Handler{
		GroupCreate:                  e.handleGroupCreateEvent,             // 群创建
		GroupUnableAddDestroyAccount: e.handleGroupUnableAddDestroyAccount, // 无法添加注销账号到群聊
		// GroupMemberAdd:             e.handleGroupMemberAddEvent,      // 群成员添加
		GroupMemberRemove:          e.handleGroupMemberRemoveEvent,   // 群成员移除
		GroupUpdate:                e.handleGroupUpdateEvent,         // 群更新
		GroupAvatarUpdate:          e.handleGroupAvatarUpdateEvent,   // 群头像更新
		GroupMemberScanJoin:        e.handleGroupMemberScanJoin,      // 群扫码入群
		GroupMemberTransferGrouper: e.handleGroupTransferGrouper,     // 群转让
		GroupMemberInviteRequest:   e.handleGroupMemberInviteRequest, // 群成员邀请
		// EventRedpacketReceive:      e.handleRedpacketReceive,         // 处理红包领取消息
	}
}

func (e *Event) handleEvent(model *Model) {
	handler := handlerMap[model.Event]
	if handler == nil {
		listeners := e.ctx.GetEventListeners(model.Event)
		if listeners == nil {
			e.Error("不支持的事件!", zap.String("event", model.Event))
			return
		}
		for _, listener := range listeners {
			listener([]byte(model.Data), func(err error) {
				e.updateEventStatus(err, model.VersionLock, model.Id)
			})
		}
		return
	}
	handler(model)
}

// 处理群创建事件
func (e *Event) handleGroupCreateEvent(model *Model) {

	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req *config.MsgGroupCreateReq
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			err = e.ctx.SendGroupCreate(req)
			e.updateEventStatus(err, model.VersionLock, model.Id)
		},
	}
}

// 处理群聊无法添加注销账号事件
func (e *Event) handleGroupUnableAddDestroyAccount(model *Model) {

	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req *config.MsgGroupCreateReq
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			err = e.ctx.SendUnableAddDestoryAccountInGroup(req)
			e.updateEventStatus(err, model.VersionLock, model.Id)
		},
	}
}

// 处理群更新事件
func (e *Event) handleGroupUpdateEvent(model *Model) {

	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req *config.MsgGroupUpdateReq
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			err = e.ctx.SendGroupUpdate(req)
			e.updateEventStatus(err, model.VersionLock, model.Id)
			err = e.ctx.SendChannelUpdateToGroup(req.GroupNo)
			if err != nil {
				e.Error("发送频道更新cmd失败！", zap.Error(err))
			}
		},
	}
}

// 处理群成员添加事件
// func (e *Event) handleGroupMemberAddEvent(model *Model) {

// 	e.ctx.EventPool.Work <- &pool.Job{
// 		Data: model,
// 		JobFunc: func(id int64, data interface{}) {
// 			var model = data.(*Model)
// 			var req *config.MsgGroupMemberAddReq
// 			err := util.ReadJsonByByte([]byte(model.Data), &req)
// 			if err != nil {
// 				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
// 				return
// 			}
// 			err = e.ctx.SendGroupMemberAdd(req)
// 			e.updateEventStatus(err, model.VersionLock, model.Id)
// 		},
// 	}
// }

// 处理群成员移除事件
func (e *Event) handleGroupMemberRemoveEvent(model *Model) {

	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req *config.MsgGroupMemberRemoveReq
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			err = e.ctx.SendGroupMemberRemove(req)
			e.updateEventStatus(err, model.VersionLock, model.Id)
		},
	}
}

// 处理群成头像更新事件
func (e *Event) handleGroupAvatarUpdateEvent(model *Model) {

	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req *config.CMDGroupAvatarUpdateReq
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			// 组合群头像
			downloadURLs := make([]string, 0, len(req.Members))
			for _, member := range req.Members {
				downloadURLs = append(downloadURLs, fmt.Sprintf("%s/users/%s/avatar", e.ctx.GetConfig().External.APIBaseURL, member))
			}
			uploadPath := e.ctx.GetConfig().GetGroupAvatarFilePath(req.GroupNo)
			_, err = e.fileService.DownloadAndMakeCompose(uploadPath, downloadURLs)
			if err != nil {
				e.Error("组合群头像失败！", zap.String("groupNo", req.GroupNo), zap.Any("members", req.Members), zap.Error(err))
				return
			}
			// 发送群头像更新命令
			err = e.ctx.SendCMD(config.MsgCMDReq{
				ChannelID:   req.GroupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
				CMD:         common.CMDGroupAvatarUpdate,
				Param: map[string]interface{}{
					"group_no": req.GroupNo,
				},
			})
			if err != nil {
				e.Error("发送群头像更新命令失败！", zap.String("groupNo", req.GroupNo), zap.Any("members", req.Members), zap.Error(err))
				return
			}
			e.updateEventStatus(err, model.VersionLock, model.Id)
		},
	}
}

// 群成员扫码加入
func (e *Event) handleGroupMemberScanJoin(model *Model) {
	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req config.MsgGroupMemberScanJoin
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			err = e.ctx.SendGroupMemberScanJoin(req)
			e.updateEventStatus(err, model.VersionLock, model.Id)
		},
	}
}

// 群主转让
func (e *Event) handleGroupTransferGrouper(model *Model) {
	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req config.MsgGroupTransferGrouper
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			err = e.ctx.SendGroupTransferGrouper(req)
			e.updateEventStatus(err, model.VersionLock, model.Id)
			err = e.ctx.SendGroupMemberUpdate(req.GroupNo)
			if err != nil {
				e.Error("发送群成员更新cmd失败！", zap.Error(err))
			}
		},
	}
}

// 群成员邀请
func (e *Event) handleGroupMemberInviteRequest(model *Model) {
	e.ctx.EventPool.Work <- &pool.Job{
		Data: model,
		JobFunc: func(id int64, data interface{}) {
			var model = data.(*Model)
			var req config.MsgGroupMemberInviteReq
			err := util.ReadJsonByByte([]byte(model.Data), &req)
			if err != nil {
				e.Error("解析JSON失败！", zap.Error(err), zap.String("data", model.Data))
				return
			}
			err = e.ctx.SendGroupMemberInviteReq(req)
			e.updateEventStatus(err, model.VersionLock, model.Id)
		},
	}
}
