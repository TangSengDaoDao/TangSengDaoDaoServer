package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

// 提醒已完成
func (m *Message) reminderDone(c *wkhttp.Context) {
	var ids []int64
	if err := c.BindJSON(&ids); err != nil {
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(ids) == 0 {
		c.ResponseError(errors.New("数据不能为空！"))
		return
	}
	loginUID := c.GetLoginUID()
	tx, _ := m.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	err := m.remindersDB.insertDonesTx(ids, loginUID, tx)
	if err != nil {
		tx.Rollback()
		m.Error("添加done失败！", zap.Error(err))
		c.ResponseError(errors.New("添加done失败！"))
		return
	}
	for _, id := range ids {
		version := m.ctx.GenSeq(common.RemindersKey)
		err = m.remindersDB.updateVersionTx(version, id, tx)
		if err != nil {
			tx.Rollback()
			m.Error("更新提醒项版本失败！", zap.Error(err))
			c.ResponseError(errors.New("更新提醒项版本失败！"))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		m.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	err = m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		CMD:         common.CMDSyncReminders,
	})
	if err != nil {
		m.Error("发送同步提醒项cmd失败！", zap.Error(err))
		c.ResponseError(errors.New("发送同步提醒项cmd失败！"))
		return
	}
	c.ResponseOK()
}

// 提醒内容同步
func (m *Message) reminderSync(c *wkhttp.Context) {
	var req struct {
		Version    int64    `json:"version"`
		Limit      uint64   `json:"limit"`
		ChannelIDs []string `json:"channel_ids"`
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	loginUID := c.GetLoginUID()
	reminders, err := m.remindersDB.sync(loginUID, req.Version, req.Limit, req.ChannelIDs)
	if err != nil {
		m.Error("同步提醒项失败！", zap.Error(err))
		c.ResponseError(errors.New("同步提醒项失败！"))
		return
	}
	reminderResps := make([]*reminderResp, 0, len(reminders))
	for _, reminder := range reminders {
		reminderResps = append(reminderResps, newReminderResp(reminder))
	}
	c.JSON(http.StatusOK, reminderResps)
}

func (m *Message) listenerMessages(messages []*config.MessageResp) {

	reminders := m.getReminders(messages) // 提醒
	if len(reminders) > 0 {
		m.handleReminders(reminders)
	}

}

func (m *Message) getReminders(messages []*config.MessageResp) []*remindersModel {
	var reminders []*remindersModel
	if reminders == nil {
		reminders = make([]*remindersModel, 0, len(messages))
	}
	for _, message := range messages {
		payloadMap, err := message.GetPayloadMap()
		if err != nil {
			m.Warn("解码消息payload失败！,跳过", zap.Error(err))
			continue
		}
		if payloadMap == nil {
			continue
		}
		if m.hasMention(payloadMap) {
			all, uids := m.getMention(payloadMap)
			if all {
				version := m.ctx.GenSeq(common.RemindersKey)
				reminders = append(reminders, &remindersModel{
					ChannelID:    message.ChannelID,
					ChannelType:  message.ChannelType,
					ClientMsgNo:  message.ClientMsgNo,
					Publisher:    message.FromUID,
					MessageID:    fmt.Sprintf("%d", message.MessageID),
					MessageSeq:   message.MessageSeq,
					ReminderType: ReminderTypeMentionMe,
					IsLocate:     1,
					Version:      version,
					Text:         "[有人@我]",
				})
			} else if len(uids) > 0 {
				for _, uid := range uids {
					version := m.ctx.GenSeq(common.RemindersKey)
					reminders = append(reminders, &remindersModel{
						ChannelID:    message.ChannelID,
						ChannelType:  message.ChannelType,
						Publisher:    message.FromUID,
						MessageID:    fmt.Sprintf("%d", message.MessageID),
						MessageSeq:   message.MessageSeq,
						ReminderType: ReminderTypeMentionMe,
						UID:          uid,
						IsLocate:     1,
						Version:      version,
						Text:         "[有人@我]",
					})
				}
			}
		}
		// 申请入群
		contentType := m.contentType(payloadMap)
		if contentType == common.GroupMemberInvite.Int() {
			if payloadMap["visibles"] != nil {
				visibleObjs := payloadMap["visibles"].([]interface{})
				for _, visibleObj := range visibleObjs {
					version := m.ctx.GenSeq(common.RemindersKey)
					reminders = append(reminders, &remindersModel{
						ChannelID:    message.ChannelID,
						ChannelType:  message.ChannelType,
						MessageID:    fmt.Sprintf("%d", message.MessageID),
						MessageSeq:   message.MessageSeq,
						ReminderType: ReminderTypeApplyJoinGroup,
						UID:          visibleObj.(string),
						IsLocate:     1,
						Version:      version,
						Text:         "[进群申请]",
					})
				}
			}
		}
	}
	return reminders
}

func (m *Message) handleReminders(reminders []*remindersModel) {
	if len(reminders) > 0 {
		err := m.remindersDB.inserts(reminders)
		if err != nil {
			m.Error("插入提醒项失败！", zap.Error(err))
		}
		channels := make([]*config.ChannelReq, 0)
		uids := make([]string, 0)
		for _, reminder := range reminders {
			if reminder.UID == "" {
				channels = append(channels, &config.ChannelReq{
					ChannelID:   reminder.ChannelID,
					ChannelType: reminder.ChannelType,
				})
			} else {
				uids = append(uids, reminder.UID)
			}
		}
		if len(channels) > 0 {
			for _, channel := range channels {
				err = m.ctx.SendCMD(config.MsgCMDReq{
					NoPersist:   true,
					ChannelID:   channel.ChannelID,
					ChannelType: channel.ChannelType,
					CMD:         common.CMDSyncReminders,
				})
				if err != nil {
					m.Error("发送cmd[CMDSyncReminders]失败！", zap.Error(err))
				}
			}
		}
		if len(uids) > 0 {
			err = m.ctx.SendCMD(config.MsgCMDReq{
				NoPersist:   true,
				Subscribers: uids,
				CMD:         common.CMDSyncReminders,
			})
			if err != nil {
				m.Error("发送cmd[CMDSyncReminders]失败！", zap.Error(err))
			}
		}
	}
}

func (m *Message) hasMention(payloadMap map[string]interface{}) bool {
	return payloadMap["mention"] != nil
}

func (m *Message) getMention(payloadMap map[string]interface{}) (all bool, uids []string) {
	mentionMap := payloadMap["mention"].(map[string]interface{})
	if mentionMap["all"] != nil {
		allI, _ := mentionMap["all"].(json.Number).Int64()
		if allI == 1 {
			all = true
		}
	}
	if mentionMap["uids"] != nil {
		uidObjs := mentionMap["uids"].([]interface{})
		uids = make([]string, 0, len(uidObjs))
		for _, uidObj := range uidObjs {
			uids = append(uids, uidObj.(string))
		}
	}
	return
}

func (m *Message) contentType(payloadMap map[string]interface{}) int {
	if payloadMap["type"] != nil {
		contentTypeI, _ := payloadMap["type"].(json.Number).Int64()
		return int(contentTypeI)
	}
	return 0
}

type reminderResp struct {
	ID           int64                  `json:"id"`
	ChannelID    string                 `json:"channel_id"`
	ChannelType  uint8                  `json:"channel_type"`
	Publisher    string                 `json:"publisher"`
	MessageSeq   uint32                 `json:"message_seq"`
	MessageID    string                 `json:"message_id"`
	ReminderType ReminderType           `json:"reminder_type"`
	UID          string                 `json:"uid"`
	Text         string                 `json:"text"`
	Data         map[string]interface{} `json:"data,omitempty"`
	IsLocate     int                    `json:"is_locate"`
	Version      int64                  `json:"version"`
	Done         int                    `json:"done"`
}

func newReminderResp(m *remindersDetailModel) *reminderResp {

	var dataMap map[string]interface{}
	if m.Data != "" {
		dataMap, _ = util.JsonToMap(m.Data)
	}

	return &reminderResp{
		ID:           m.Id,
		ChannelID:    m.ChannelID,
		ChannelType:  m.ChannelType,
		MessageSeq:   m.MessageSeq,
		MessageID:    m.MessageID,
		ReminderType: ReminderType(m.ReminderType),
		Publisher:    m.Publisher,
		UID:          m.UID,
		Text:         m.Text,
		Data:         dataMap,
		IsLocate:     m.IsLocate,
		Version:      m.Version,
		Done:         m.Done,
	}
}
