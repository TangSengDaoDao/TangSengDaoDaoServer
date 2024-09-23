package message

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// 置顶或取消置顶消息
func (m *Message) pinnedMessage(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	loginName := c.GetLoginName()
	type reqVO struct {
		MessageID   string `json:"message_id"`   // 消息唯一ID
		MessageSeq  uint32 `json:"message_seq"`  // 消息序列号
		ChannelID   string `json:"channel_id"`   // 频道唯一ID
		ChannelType uint8  `json:"channel_type"` // 频道类型
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if req.ChannelID == "" {
		c.ResponseError(errors.New("频道ID不能为空"))
		return
	}
	if req.MessageID == "" {
		c.ResponseError(errors.New("消息ID不能为空"))
		return
	}
	if req.MessageSeq <= 0 {
		c.ResponseError(errors.New("消息seq不合法"))
		return
	}

	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(loginUID, req.ChannelID)
	} else if req.ChannelType == common.ChannelTypeGroup.Uint8() {
		groupInfo, err := m.groupService.GetGroupDetail(req.ChannelID, loginUID)
		if err != nil {
			m.Error("查询群组信息错误", zap.Error(err))
			c.ResponseError(errors.New("查询群组信息错误"))
			return
		}
		if groupInfo == nil || groupInfo.Status != 1 {
			c.ResponseError(errors.New("群不存在或已删除"))
			return
		}
		isCreatorOrManager, err := m.groupService.IsCreatorOrManager(req.ChannelID, loginUID)
		if err != nil {
			m.Error("查询用户在群内权限错误", zap.Error(err))
			c.ResponseError(errors.New("查询用户在群内权限错误"))
			return
		}
		if !isCreatorOrManager && groupInfo.AllowMemberPinnedMessage == 0 {
			c.ResponseError(errors.New("普通成员不允许置顶消息"))
			return
		}
	}
	message, err := m.db.queryMessageWithMessageID(fakeChannelID, req.MessageID)
	if err != nil {
		m.Error("查询消息错误", zap.Error(err))
		c.ResponseError(errors.New("查询消息错误"))
		return
	}
	if message == nil {
		c.ResponseError(errors.New("该不存在或已删除"))
		return
	}

	messageExtra, err := m.messageExtraDB.queryWithMessageID(req.MessageID)
	if err != nil {
		m.Error("查询消息扩展信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询消息扩展信息错误"))
		return
	}
	if messageExtra != nil && messageExtra.IsDeleted == 1 {
		c.ResponseError(errors.New("该消息不存在或已删除"))
		return
	}
	appConfig, err := m.commonService.GetAppConfig()
	if err != nil {
		m.Error("查询配置错误", zap.Error(err))
		c.ResponseError(errors.New("查询配置错误"))
		return
	}
	var maxCount = 10
	if appConfig != nil {
		maxCount = appConfig.ChannelPinnedMessageMaxCount
	}
	currentCount, err := m.pinnedDB.queryCountWithChannel(fakeChannelID, req.ChannelType)
	if err != nil {
		m.Error("查询当前置顶消息数量错误", zap.Error(err))
		c.ResponseError(errors.New("查询当前置顶消息数量错误"))
		return
	}
	pinnedMessage, err := m.pinnedDB.queryWithMessageId(fakeChannelID, req.ChannelType, req.MessageID)
	if err != nil {
		m.Error("查询置顶消息错误", zap.Error(err))
		c.ResponseError(errors.New("查询置顶消息错误"))
		return
	}
	if currentCount >= int64(maxCount) && (pinnedMessage == nil || pinnedMessage.IsDeleted == 1) {
		c.ResponseError(errors.New("置顶数量已达到上限"))
		return
	}

	tx, err := m.db.session.Begin()
	if err != nil {
		m.Error("开启事务错误", zap.Error(err))
		c.ResponseError(errors.New("开启事务错误"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	isPinned := 0
	isSendSystemMsg := false
	if pinnedMessage == nil {
		err = m.pinnedDB.insert(&pinnedMessageModel{
			MessageId:   req.MessageID,
			ChannelID:   fakeChannelID,
			ChannelType: req.ChannelType,
			IsDeleted:   0,
			MessageSeq:  req.MessageSeq,
			Version:     time.Now().UnixMilli(),
		})
		if err != nil {
			tx.Rollback()
			m.Error("新增置顶消息错误", zap.Error(err))
			c.ResponseError(errors.New("新增置顶消息错误"))
			return
		}
		isSendSystemMsg = true
		isPinned = 1
	} else {
		if pinnedMessage.IsDeleted == 1 {
			pinnedMessage.IsDeleted = 0
			isPinned = 1
		} else {
			pinnedMessage.IsDeleted = 1
			isPinned = 0
		}
		pinnedMessage.Version = time.Now().UnixMilli()
		err = m.pinnedDB.update(pinnedMessage)
		if err != nil {
			tx.Rollback()
			m.Error("取消置顶消息错误", zap.Error(err))
			c.ResponseError(errors.New("取消置顶消息错误"))
			return
		}
	}
	version := m.genMessageExtraSeq(fakeChannelID)
	err = m.messageExtraDB.insertOrUpdatePinnedTx(&messageExtraModel{
		MessageID:   req.MessageID,
		MessageSeq:  req.MessageSeq,
		ChannelID:   fakeChannelID,
		ChannelType: req.ChannelType,
		IsPinned:    isPinned,
		Version:     version,
	}, tx)
	if err != nil {
		tx.Rollback()
		c.ResponseErrorf("更新消息置顶状态失败！", err)
		return
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		c.ResponseErrorf("事务提交失败！", err)
		return
	}
	err = m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		FromUID:     c.GetLoginUID(),
		CMD:         common.CMDSyncPinnedMessage,
	})

	if err != nil {
		m.Error("发送cmd失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if isSendSystemMsg {
		var payloadMap map[string]interface{}
		err := util.ReadJsonByByte(message.Payload, &payloadMap)
		if err != nil {
			m.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(message.Payload)))
			c.ResponseOK()
			return
		}
		var contentType int = 0
		var content string = ""
		if payloadMap["type"] != nil {
			contentTypeI, _ := payloadMap["type"].(json.Number).Int64()
			contentType = int(contentTypeI)
		}
		if contentType == common.Text.Int() {
			content = payloadMap["content"].(string)
			content = fmt.Sprintf("`%s`", content)
		} else {
			content = common.GetDisplayText(contentType)
		}
		mesageContent := fmt.Sprintf("{0} 置顶了%s", content)
		err = m.ctx.SendMessage(&config.MsgSendReq{
			Header: config.MsgHeader{
				NoPersist: 0,
				RedDot:    1,
				SyncOnce:  0, // 只同步一次
			},
			ChannelID:   req.ChannelID,
			ChannelType: req.ChannelType,
			FromUID:     loginUID,
			Payload: []byte(util.ToJson(map[string]interface{}{
				"from_uid":  loginUID,
				"from_name": loginName,
				"content":   mesageContent,
				"extra": []config.UserBaseVo{
					{
						UID:  loginUID,
						Name: loginName,
					},
				},
				"type": common.Tip,
			})),
		})
		if err != nil {
			m.Warn("发送解散群消息错误", zap.Error(err))
		}
	}
	c.ResponseOK()
}

func (m *Message) clearPinnedMessage(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	var req struct {
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if req.ChannelID == "" {
		c.ResponseError(errors.New("频道ID不能为空"))
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(loginUID, req.ChannelID)
	} else {
		// 查询权限
		isCreatorOrManager, err := m.groupService.IsCreatorOrManager(req.ChannelID, loginUID)
		if err != nil {
			m.Error("查询用户在群内权限错误", zap.Error(err))
			c.ResponseError(errors.New("查询用户在群内权限错误"))
			return
		}
		if !isCreatorOrManager {
			c.ResponseError(errors.New("用户无权清空置顶消息"))
			return
		}
	}
	pinnedMsgs, err := m.pinnedDB.queryWithUnDeletedMessage(fakeChannelID, req.ChannelType)
	if err != nil {
		m.Error("查询置顶消息错误", zap.Error(err))
		c.ResponseError(errors.New("查询置顶消息错误"))
		return
	}
	messageIds := make([]string, 0)
	if len(pinnedMsgs) <= 0 {
		c.ResponseOK()
		return
	}

	for _, msg := range pinnedMsgs {
		messageIds = append(messageIds, msg.MessageId)
	}
	messageUserExtras, err := m.messageUserExtraDB.queryWithMessageIDsAndUID(messageIds, loginUID)
	if err != nil {
		m.Error("查询用户消息扩展字段失败！", zap.Error(err))
		c.ResponseError(errors.New("查询用户消息扩展字段失败！"))
		return
	}
	channelOffsetM, err := m.channelOffsetDB.queryWithUIDAndChannel(loginUID, fakeChannelID, req.ChannelType)
	if err != nil {
		m.Error("查询频道偏移量失败！", zap.Error(err))
		c.ResponseError(errors.New("查询频道偏移量失败！"))
		return
	}
	updateModel := make([]*pinnedMessageModel, 0)
	for _, msg := range pinnedMsgs {
		isAdd := true
		if len(messageUserExtras) > 0 {
			for _, extra := range messageUserExtras {
				if extra.MessageID == msg.MessageId && extra.MessageIsDeleted == 1 {
					isAdd = false
					break
				}
			}
		}
		if channelOffsetM != nil && msg.MessageSeq <= channelOffsetM.MessageSeq {
			isAdd = false
		}
		if isAdd {
			msg.IsDeleted = 1
			msg.Version = time.Now().UnixMilli()
			updateModel = append(updateModel, msg)
		}
	}
	if len(updateModel) == 0 {
		c.ResponseOK()
		return
	}
	tx, err := m.db.session.Begin()
	if err != nil {
		m.Error("开启事务错误", zap.Error(err))
		c.ResponseError(errors.New("开启事务错误"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	for _, msg := range updateModel {
		err = m.pinnedDB.updateTx(msg, tx)
		if err != nil {
			tx.Rollback()
			m.Error("删除置顶消息错误", zap.Error(err))
			c.ResponseError(errors.New("删除置顶消息错误"))
			return
		}

		version := m.genMessageExtraSeq(fakeChannelID)
		err = m.messageExtraDB.insertOrUpdatePinnedTx(&messageExtraModel{
			MessageID:   msg.MessageId,
			MessageSeq:  msg.MessageSeq,
			ChannelID:   fakeChannelID,
			ChannelType: req.ChannelType,
			IsPinned:    0,
			Version:     version,
		}, tx)
		if err != nil {
			tx.Rollback()
			m.Error("修改消息扩展置顶状态错误", zap.Error(err))
			c.ResponseErrorf("修改消息扩展置顶状态错误", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		c.ResponseErrorf("事务提交失败！", err)
		return
	}
	err = m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		FromUID:     c.GetLoginUID(),
		CMD:         common.CMDSyncPinnedMessage,
	})

	if err != nil {
		m.Error("发送cmd失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

func (m *Message) syncPinnedMessage(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	var req struct {
		Version     int64  `json:"version"`
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if req.ChannelID == "" {
		c.ResponseError(errors.New("频道ID不能为空"))
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(loginUID, req.ChannelID)
	}
	pinnedMsgs, err := m.pinnedDB.queryWithChannelIDAndVersion(fakeChannelID, req.ChannelType, req.Version)
	if err != nil {
		m.Error("查询置顶消息错误", zap.Error(err))
		c.ResponseError(errors.New("查询置顶消息错误"))
		return
	}
	messageSeqs := make([]uint32, 0)
	messageIds := make([]string, 0)
	list := make([]*MsgSyncResp, 0)
	pinnedMessageList := make([]*pinnedMessageResp, 0)
	if len(pinnedMsgs) <= 0 {
		c.Response(map[string]interface{}{
			"pinned_messages": pinnedMessageList,
			"messages":        list,
		})
		return
	}

	for _, msg := range pinnedMsgs {
		messageSeqs = append(messageSeqs, msg.MessageSeq)
		messageIds = append(messageIds, msg.MessageId)
	}

	resp, err := m.ctx.IMGetWithChannelAndSeqs(req.ChannelID, req.ChannelType, loginUID, messageSeqs)
	if err != nil {
		m.Error("查询频道内的消息失败！", zap.Error(err), zap.String("req", util.ToJson(req)))
		c.ResponseError(errors.New("查询频道内的消息失败！"))
		return
	}

	if resp == nil || len(resp.Messages) == 0 {
		c.Response(map[string]interface{}{
			"pinned_messages": pinnedMessageList,
			"messages":        list,
		})
		return
	}
	// 消息全局扩张
	messageExtras, err := m.messageExtraDB.queryWithMessageIDsAndUID(messageIds, loginUID)
	if err != nil {
		m.Error("查询消息扩展字段失败！", zap.Error(err))
		c.ResponseError(errors.New("查询用户消息扩展错误"))
		return
	}
	messageExtraMap := map[string]*messageExtraDetailModel{}
	if len(messageExtras) > 0 {
		for _, messageExtra := range messageExtras {
			messageExtraMap[messageExtra.MessageID] = messageExtra
		}
	}
	// 消息用户扩张
	messageUserExtras, err := m.messageUserExtraDB.queryWithMessageIDsAndUID(messageIds, loginUID)
	if err != nil {
		m.Error("查询用户消息扩展字段失败！", zap.Error(err))
		c.ResponseError(errors.New("查询用户消息扩展字段失败！"))
		return
	}
	messageUserExtraMap := map[string]*messageUserExtraModel{}
	if len(messageUserExtras) > 0 {
		for _, messageUserExtraM := range messageUserExtras {
			messageUserExtraMap[messageUserExtraM.MessageID] = messageUserExtraM
		}
	}
	// 查询消息回应
	messageReaction, err := m.messageReactionDB.queryWithMessageIDs(messageIds)
	if err != nil {
		m.Error("查询消息回应数据错误", zap.Error(err))
		c.ResponseError(errors.New("查询消息回应数据错误"))
		return
	}
	messageReactionMap := map[string][]*reactionModel{}
	if len(messageReaction) > 0 {
		for _, reaction := range messageReaction {
			msgReactionList := messageReactionMap[reaction.MessageID]
			if msgReactionList == nil {
				msgReactionList = make([]*reactionModel, 0)
			}
			msgReactionList = append(msgReactionList, reaction)
			messageReactionMap[reaction.MessageID] = msgReactionList
		}
	}
	channelOffsetM, err := m.channelOffsetDB.queryWithUIDAndChannel(loginUID, fakeChannelID, req.ChannelType)
	if err != nil {
		m.Error("查询频道偏移量失败！", zap.Error(err))
		c.ResponseError(errors.New("查询频道偏移量失败！"))
		return
	}
	// 频道偏移
	channelIds := make([]string, 0)
	channelIds = append(channelIds, fakeChannelID)
	channelSettings, err := m.channelService.GetChannelSettings(channelIds)
	if err != nil {
		m.Error("查询频道设置错误", zap.Error(err), zap.String("req", util.ToJson(req)))
		c.ResponseError(errors.New("查询频道设置错误"))
		return
	}
	var channelOffsetMessageSeq uint32 = 0
	if len(channelSettings) > 0 && channelSettings[0].OffsetMessageSeq > 0 {
		channelOffsetMessageSeq = channelSettings[0].OffsetMessageSeq
	}
	for _, message := range resp.Messages {
		if channelOffsetM != nil && message.MessageSeq <= channelOffsetM.MessageSeq {
			continue
		}
		msgResp := &MsgSyncResp{}
		messageIDStr := strconv.FormatInt(message.MessageID, 10)
		messageExtra := messageExtraMap[messageIDStr]
		messageUserExtra := messageUserExtraMap[messageIDStr]
		msgResp.from(message, loginUID, messageExtra, messageUserExtra, messageReactionMap[messageIDStr], channelOffsetMessageSeq)
		list = append(list, msgResp)
	}

	for _, msg := range pinnedMsgs {
		messageUserExtra := messageUserExtraMap[msg.MessageId]
		if messageUserExtra != nil && messageUserExtra.MessageIsDeleted == 1 {
			msg.IsDeleted = 1
		}
		if channelOffsetM != nil && msg.MessageSeq <= channelOffsetM.MessageSeq {
			msg.IsDeleted = 1
		}
		toChannelID := common.GetToChannelIDWithFakeChannelID(msg.ChannelID, loginUID)
		pinnedMessageList = append(pinnedMessageList, &pinnedMessageResp{
			MessageID:   msg.MessageId,
			MessageSeq:  msg.MessageSeq,
			ChannelID:   toChannelID,
			ChannelType: msg.ChannelType,
			IsDeleted:   msg.IsDeleted,
			Version:     msg.Version,
			CreatedAt:   msg.CreatedAt.String(),
			UpdatedAt:   msg.UpdatedAt.String(),
		})
	}
	c.Response(map[string]interface{}{
		"pinned_messages": pinnedMessageList,
		"messages":        list,
	})
}

func (m *Message) deletePinnedMessage(channelID string, channelType uint8, messageIds []string, loginUID string, tx *dbr.Tx) error {
	fakeChannelID := channelID
	if channelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(channelID, loginUID)
	}
	pinnedMessages, err := m.pinnedDB.queryWithMessageIds(fakeChannelID, channelType, messageIds)
	if err != nil {
		m.Error("查询置顶消息错误", zap.Error(err))
		return errors.New("查询置顶消息错误")
	}
	if len(pinnedMessages) == 0 {
		return nil
	}
	for _, pinnedMessage := range pinnedMessages {
		pinnedMessage.IsDeleted = 1
		pinnedMessage.Version = time.Now().UnixMilli()
		err = m.pinnedDB.updateTx(pinnedMessage, tx)
		if err != nil {
			tx.Rollback()
			m.Error("取消置顶消息错误", zap.Error(err))
			return errors.New("取消置顶消息错误")
		}
	}

	err = m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   channelID,
		ChannelType: channelType,
		FromUID:     loginUID,
		CMD:         common.CMDSyncPinnedMessage,
	})

	if err != nil {
		m.Warn("发送cmd失败！", zap.Error(err))
	}
	return nil
}

type pinnedMessageResp struct {
	MessageID   string `json:"message_id"`
	MessageSeq  uint32 `json:"message_seq"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	IsDeleted   int8   `json:"is_deleted"`
	Version     int64  `json:"version"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
