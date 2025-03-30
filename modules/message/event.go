package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

func (m *Message) syncMessageReadedCount() {
	go m.startTimer()
}
func (m *Message) startTimer() {
	intervalSecond := m.ctx.GetConfig().Message.SyncReadedCountIntervalSecond
	if intervalSecond == 0 {
		intervalSecond = 3
	}
	ticker := time.NewTicker(time.Duration(intervalSecond) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.handleReadedMessageCount()
	}
}

// 处理消息已读数量
func (m *Message) handleReadedMessageCount() {
	keysStr, err := m.ctx.GetRedisConn().GetKeys(fmt.Sprintf("%s*", CacheReadedCountPrefix))
	if err != nil {
		m.Error("获取已读消息keys错误", zap.Error(err))
		return
	}
	messageIds := make([]string, 0)
	messages := make([]*messageReadedCountModel, 0)
	if len(keysStr) > 0 {
		for _, key := range keysStr {
			var messageExtra messageReadedCountModel
			msgStr, err := m.ctx.GetRedisConn().GetString(key)
			if err != nil {
				m.Error("通过key获取消息错误", zap.Error(err), zap.String("key", key))
				return
			}
			err = json.Unmarshal([]byte(msgStr), &messageExtra)
			if err != nil {
				m.Error("转换消息对象错误", zap.Error(err), zap.String("msgStr", msgStr))
				return
			}
			messages = append(messages, &messageExtra)
			messageIds = append(messageIds, messageExtra.MessageIDStr)
			m.mutex.Lock()
			err = m.ctx.GetRedisConn().Del(key)
			if err != nil {
				m.mutex.Unlock()
				m.Error("删除缓存错误", zap.Error(err), zap.String("key", key))
				return
			}
			m.mutex.Unlock()
		}
	}

	if len(messages) == 0 {
		return
	}
	// 分组
	messageChannelMap := make(map[string][]*messageReadedCountModel, 0)
	for _, message := range messages {
		fakeChannelID := message.ChannelID
		if message.ReqChannelType == common.ChannelTypePerson.Uint8() {
			fakeChannelID = common.GetFakeChannelIDWith(message.ReqChannelID, message.LoginUID)
		}
		list := messageChannelMap[fakeChannelID]
		if list == nil {
			list = make([]*messageReadedCountModel, 0)
		}
		list = append(list, &messageReadedCountModel{
			MessageID:      message.MessageID,
			MessageIDStr:   message.MessageIDStr,
			MessageSeq:     message.MessageSeq,
			FromUID:        message.FromUID,
			ChannelID:      message.ChannelID,
			ChannelType:    message.ChannelType,
			Revoke:         message.Revoke,
			Revoker:        message.Revoker,
			LoginUID:       message.LoginUID,
			ReqChannelID:   message.ReqChannelID,
			ReqChannelType: message.ReqChannelType,
		})
		messageChannelMap[fakeChannelID] = list
	}
	m.mutex.Lock()
	// 查询已经存在的消息扩展数据
	messageExtras, err := m.messageExtraDB.queryWithMessageIDs(messageIds)
	if err != nil {
		m.mutex.Unlock()
		m.Error("查询消息扩展数据错误", zap.Error(err))
		return
	}
	tx, err := m.ctx.DB().Begin()
	if err != nil {
		m.mutex.Unlock()
		m.Error("开启事务失败！", zap.Error(err))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			m.mutex.Unlock()
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	type sendCMDVO struct {
		ChannelID   string
		ChannelType uint8
		LoginUID    string
		FromUIDs    []string
	}
	sendCmds := make([]*sendCMDVO, 0)
	for fakeChannelID, msgs := range messageChannelMap {
		messageIDStrs := make([]string, 0)
		reqChannelType := common.ChannelTypePerson.Uint8()
		reqChannelID := ""
		reqLoginUID := ""
		if len(msgs) > 0 {
			reqChannelType = msgs[0].ReqChannelType
			reqChannelID = msgs[0].ReqChannelID
			reqLoginUID = msgs[0].LoginUID
			for _, tempMsg := range msgs {
				messageIDStrs = append(messageIDStrs, tempMsg.MessageIDStr)
			}
		}
		messageReadedCountMap, err := m.memberReadedDB.queryCountWithMessageIDs(fakeChannelID, reqChannelType, messageIDStrs)
		if err != nil {
			m.mutex.Unlock()
			tx.Rollback()
			m.Error("获取消息已读数量map失败！", zap.Error(err))
			return
		}
		fromUIDs := make([]string, 0, len(messages)) // 消息发送者
		for _, message := range msgs {
			fromUIDs = append(fromUIDs, message.FromUID)
			version := m.genMessageExtraSeq(fakeChannelID)
			count := messageReadedCountMap[message.MessageID]
			if message.ChannelType == common.ChannelTypePerson.Uint8() {
				count = 1
			}

			var tempMsgExtra *messageExtraModel
			for _, messageExtra := range messageExtras {
				if messageExtra.MessageID == message.MessageIDStr {
					tempMsgExtra = messageExtra
					tempMsgExtra.ReadedCount = count
					tempMsgExtra.Version = version
					break
				}
			}
			if tempMsgExtra == nil {
				err = m.messageExtraDB.insertTx(&messageExtraModel{
					MessageID:   message.MessageIDStr,
					MessageSeq:  message.MessageSeq,
					FromUID:     message.FromUID,
					ChannelID:   fakeChannelID,
					ChannelType: reqChannelType,
					ReadedCount: count,
					Version:     version,
				}, tx)
				if err != nil {
					m.mutex.Unlock()
					tx.Rollback()
					m.Error("添加消息扩展数据失败！", zap.Error(err), zap.Int64("messageID", message.MessageID), zap.String("channelID", fakeChannelID))
					return
				}
			} else {
				err = m.messageExtraDB.updateTx(tempMsgExtra, tx)
				if err != nil {
					m.mutex.Unlock()
					tx.Rollback()
					m.Error("更新消息扩展数据失败！", zap.Error(err), zap.Int64("messageID", message.MessageID), zap.String("channelID", fakeChannelID))
					return
				}
			}

		}
		if reqChannelType == common.ChannelTypePerson.Uint8() {
			sendCmds = append(sendCmds, &sendCMDVO{
				ChannelID:   reqChannelID,
				ChannelType: reqChannelType,
				LoginUID:    reqLoginUID,
			})
		} else {
			sendCmds = append(sendCmds, &sendCMDVO{
				ChannelID:   fakeChannelID,
				ChannelType: reqChannelType,
				FromUIDs:    fromUIDs,
			})
		}
	}
	m.mutex.Unlock()
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		m.Error("提交事物错误", zap.Error(err))
		return
	}

	if len(sendCmds) > 0 {
		for _, cmd := range sendCmds {
			if cmd.ChannelType == common.ChannelTypePerson.Uint8() {
				err = m.ctx.SendCMD(config.MsgCMDReq{
					NoPersist:   true,
					ChannelID:   cmd.ChannelID,
					ChannelType: cmd.ChannelType,
					FromUID:     cmd.LoginUID,
					CMD:         common.CMDSyncMessageExtra,
				})
			} else {
				err = m.ctx.SendCMD(config.MsgCMDReq{
					NoPersist:   true,
					Subscribers: cmd.FromUIDs, // 消息只发送给发送者
					CMD:         common.CMDSyncMessageExtra,
					Param: map[string]interface{}{
						"channel_id":   cmd.ChannelID,
						"channel_type": cmd.ChannelType,
					},
				})
			}
			if err != nil {
				m.Error("发送cmd消息错误", zap.Error(err))
				return
			}
		}
	}

}

// 处理扫码入群
func (m *Message) handleGroupMemberScanJoinEvent(data []byte, commit config.EventCommit) {
	var req *config.MsgGroupMemberScanJoin
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		m.Error("解析JSON失败！", zap.Error(err))
		commit(err)
		return
	}
	list := make([]*config.UserBaseVo, 0)
	list = append(list, &config.UserBaseVo{
		UID:  req.Scaner,
		Name: req.ScanerName,
	})
	err = m.updateMembersChannelOffset(req.GroupNo, list)
	if err != nil {
		commit(err)
		return
	}
	content := `“{0}”通过“{1}”的二维码加入群聊`
	err = m.ctx.SendMessage(&config.MsgSendReq{
		Header: config.MsgHeader{
			NoPersist: 0,
			RedDot:    1,
			SyncOnce:  0, // 只同步一次
		},
		ChannelID:   req.GroupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"extra": []config.UserBaseVo{
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
	if err != nil {
		commit(err)
		return
	}
	commit(nil)
}

// 处理群成员添加事件
func (m *Message) handleGroupMemberAddEvent(data []byte, commit config.EventCommit) {
	var req *config.MsgGroupMemberAddReq
	err := util.ReadJsonByByte(data, &req)
	if err != nil {
		m.Error("解析JSON失败！", zap.Error(err))
		commit(err)
		return
	}
	err = m.updateMembersChannelOffset(req.GroupNo, req.Members)
	if err != nil {
		commit(err)
		return
	}
	commit(nil)
	// groupInfo, err := m.groupService.GetGroupWithGroupNo(req.GroupNo)
	// if err != nil {
	// 	m.Error("查询群信息错误", zap.Error(err))
	// 	commit(err)
	// 	return
	// }
	// if groupInfo == nil {
	// 	m.Error("操作的群不存在")
	// 	commit(errors.New("操作的群不存在"))
	// 	return
	// }
	// // if groupInfo.AllowViewHistoryMsg == 1 {
	// // 	commit(nil)
	// // 	return
	// // }

	// maxSeq, err := m.db.queryMaxMessageSeq(req.GroupNo, common.ChannelTypeGroup.Uint8())
	// if err != nil {
	// 	m.Error("查询channel最大消息序号错误", zap.Error(err))
	// 	commit(errors.New("查询channel最大消息序号错误"))
	// 	return
	// }
	// list := make([]*channelOffsetModel, 0)
	// for _, member := range req.Members {
	// 	list = append(list, &channelOffsetModel{
	// 		UID:         member.UID,
	// 		ChannelID:   req.GroupNo,
	// 		ChannelType: common.ChannelTypeGroup.Uint8(),
	// 		MessageSeq:  maxSeq,
	// 	})
	// }
	// tx, err := m.ctx.DB().Begin()
	// if err != nil {
	// 	m.Error("开启事物错误", zap.Error(err))
	// 	commit(errors.New("开启事物错误"))
	// 	return
	// }
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		tx.RollbackUnlessCommitted()
	// 		panic(err)
	// 	}
	// }()

	// for _, model := range list {

	// 	err = m.channelOffsetDB.delete(model.UID, model.ChannelID, model.ChannelType, tx)
	// 	if err != nil {
	// 		m.Error("删除消息偏移量错误", zap.Error(err))
	// 		commit(err)
	// 		tx.Rollback()
	// 		return
	// 	}
	// 	if groupInfo.AllowViewHistoryMsg == int(common.GroupAllowViewHistoryMsgEnabled) {
	// 		model.MessageSeq = 0
	// 	}
	// 	err = m.channelOffsetDB.insertOrUpdateTx(model, tx)
	// 	if err != nil {
	// 		m.Error("添加或修改用户channel消息偏移错误", zap.Error(err))
	// 		commit(err)
	// 		tx.Rollback()
	// 		return
	// 	}
	// }
	// err = tx.Commit()
	// if err != nil {
	// 	m.Error("事物提交错误", zap.Error(err))
	// 	tx.RollbackUnlessCommitted()
	// 	commit(err)
	// 	return
	// }
	// commit(nil)
}

// 修改新成员channel偏移量
func (m *Message) updateMembersChannelOffset(groupNO string, members []*config.UserBaseVo) error {
	groupInfo, err := m.groupService.GetGroupWithGroupNo(groupNO)
	if err != nil {
		m.Error("查询群信息错误", zap.Error(err))
		return err
	}
	if groupInfo == nil {
		m.Error("操作的群不存在")
		return errors.New("操作的群不存在")
	}
	// if groupInfo.AllowViewHistoryMsg == 1 {
	// 	commit(nil)
	// 	return
	// }

	maxSeq, err := m.db.queryMaxMessageSeq(groupNO, common.ChannelTypeGroup.Uint8())
	if err != nil {
		m.Error("查询channel最大消息序号错误", zap.Error(err))
		return errors.New("查询channel最大消息序号错误")
	}
	list := make([]*channelOffsetModel, 0)
	for _, member := range members {
		list = append(list, &channelOffsetModel{
			UID:         member.UID,
			ChannelID:   groupNO,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			MessageSeq:  maxSeq,
		})
	}
	tx, err := m.ctx.DB().Begin()
	if err != nil {
		m.Error("开启事物错误", zap.Error(err))
		return errors.New("开启事物错误")
	}
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
			tx.Rollback()
			return err
		}
		if groupInfo.AllowViewHistoryMsg == int(common.GroupAllowViewHistoryMsgEnabled) {
			model.MessageSeq = 0
		}
		err = m.channelOffsetDB.insertOrUpdateTx(model, tx)
		if err != nil {
			m.Error("添加或修改用户channel消息偏移错误", zap.Error(err))
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		m.Error("事物提交错误", zap.Error(err))
		tx.RollbackUnlessCommitted()
		return err
	}
	return nil
}

type messageReadedCountModel struct {
	MessageID      int64
	MessageIDStr   string
	MessageSeq     uint32
	FromUID        string
	ChannelID      string
	ChannelType    uint8
	LoginUID       string
	ReqChannelID   string
	ReqChannelType uint8
	Revoke         int
	Revoker        string // 消息撤回者的uid
	ReadedCount    int    // 已读数量
	Version        int64  // 数据版本
}
