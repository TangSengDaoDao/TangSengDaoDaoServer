package message

import (
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

// 处理群成员添加事件
func (m *Message) handleGroupMemberAddEvent(data []byte, commit config.EventCommit) {
	var req *config.MsgGroupMemberAddReq
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		m.Error("解析JSON失败！", zap.Error(err))
		commit(err)
		return
	}
	groupInfo, err := m.groupService.GetGroupWithGroupNo(req.GroupNo)
	if err != nil {
		m.Error("查询群信息错误", zap.Error(err))
		commit(err)
		return
	}
	if groupInfo == nil {
		m.Error("操作的群不存在")
		commit(errors.New("操作的群不存在"))
		return
	}
	// if groupInfo.AllowViewHistoryMsg == 1 {
	// 	commit(nil)
	// 	return
	// }

	maxSeq, err := m.db.queryMaxMessageSeq(req.GroupNo, common.ChannelTypeGroup.Uint8())
	if err != nil {
		m.Error("查询channel最大消息序号错误", zap.Error(err))
		commit(errors.New("查询channel最大消息序号错误"))
		return
	}
	list := make([]*channelOffsetModel, 0)
	for _, member := range req.Members {
		list = append(list, &channelOffsetModel{
			UID:         member.UID,
			ChannelID:   req.GroupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			MessageSeq:  maxSeq,
		})
	}
	tx, err := m.ctx.DB().Begin()
	util.CheckErr(err)
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()

	for _, model := range list {

		err = m.channelOffsetDB.delete(model.UID, model.ChannelID, model.ChannelType, tx)
		if err != nil {
			m.Error("删除消息偏移量错误", zap.Error(err))
			commit(err)
			tx.Rollback()
			return
		}
		if groupInfo.AllowViewHistoryMsg == int(common.GroupAllowViewHistoryMsgEnabled) {
			model.MessageSeq = 0
		}
		err = m.channelOffsetDB.insertOrUpdateTx(model, tx)
		if err != nil {
			m.Error("添加或修改用户channel消息偏移错误", zap.Error(err))
			commit(err)
			tx.Rollback()
			return
		}
	}
	err = tx.Commit()
	if err != nil {
		m.Error("事物提交错误", zap.Error(err))
		tx.RollbackUnlessCommitted()
		commit(err)
		return
	}
	commit(nil)
}
