package message

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Manager 消息管理
type Manager struct {
	ctx *config.Context
	log.Log
	userService  user.IService
	groupService group.IService
	managerDB    *managerDB
}

// NewManager NewManager
func NewManager(ctx *config.Context) *Manager {
	return &Manager{
		ctx:          ctx,
		Log:          log.NewTLog("MessageManager"),
		userService:  user.NewService(ctx),
		groupService: group.NewService(ctx),
		managerDB:    newManagerDB(ctx),
	}
}

// Route 路由配置
func (m *Manager) Route(r *wkhttp.WKHttp) {
	auth := r.Group("/v1/manager", m.ctx.AuthMiddleware(r))
	{
		auth.POST("/message/send", m.sendMsg)                         // 发送消息
		auth.POST("message/sendfriends", m.sendMsgToFriends)          // 给某个用户代发消息
		auth.GET("/message", m.list)                                  // 代发消息记录
		auth.POST("/message/sendall", m.sendMsgToAllUsers)            // 给所有用户发送一条消息
		auth.GET("/message/record", m.record)                         // 消息记录
		auth.GET("/message/recordpersonal", m.recordpersonal)         // 单聊聊天记录
		auth.POST("/message/prohibit_words", m.addProhibitWords)      // 添加违禁词
		auth.GET("/message/prohibit_words", m.prohibitWords)          // 查询违禁词
		auth.DELETE("/message/prohibit_words", m.deleteProhibitWords) // 删除违禁词
		auth.DELETE("/message", m.delete)                             // 删除消息
	}
}
func (m *Manager) sendMsgToFriends(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	type ReqVO struct {
		UID     string   `json:"uid"`
		ToUIDs  []string `json:"to_uids"`
		Content string   `json:"content"`
	}
	var req ReqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if req.UID == "" {
		c.ResponseError(errors.New("发送者不能为空"))
		return
	}
	if req.Content == "" {
		c.ResponseError(errors.New("发送内容不能为空"))
		return
	}
	if len(req.ToUIDs) == 0 {
		c.ResponseError(errors.New("发送消息的订阅者不能为空"))
		return
	}
	go m.sendMessageToFriends(req.ToUIDs, req.UID, req.Content)
	c.ResponseOK()
}

func (m *Manager) sendMessageToFriends(toUids []string, fromUID string, content string) error {
	err := m.ctx.SendMessageBatch(&config.MsgSendBatch{
		Header: config.MsgHeader{
			RedDot: 1,
		},
		FromUID: fromUID,
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": content,
			"type":    1,
		})),
		Subscribers: toUids,
	})
	if err != nil {
		m.Error("发送消息错误", zap.Error(err))
		return errors.New("发送消息错误")
	}
	return nil
}
func (m *Manager) delete(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	type msgVO struct {
		MessageID  string `json:"message_id"`
		MessageSeq uint32 `json:"message_seq"`
	}
	type reqVO struct {
		List        []*msgVO `json:"list"`
		ChannelID   string   `json:"channel_id"`
		FromUID     string   `json:"from_uid"`
		ChannelType uint8    `json:"channel_type"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(req.List) == 0 {
		c.ResponseError(errors.New("删除的msgIds不能为空"))
		return
	}
	if req.ChannelType == uint8(common.ChannelTypePerson) && (req.FromUID == "" || req.ChannelID == req.FromUID) {
		c.ResponseError(errors.New("单聊fromuid不能为空且不能和channelId一致"))
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(req.ChannelID, req.FromUID)
	}
	tx, _ := m.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	for _, msg := range req.List {
		version := m.genMessageExtraSeq(fakeChannelID)
		err := m.managerDB.updateMsgExtraVersionAndDeletedTx(&messageExtraModel{
			ChannelID:   fakeChannelID,
			ChannelType: req.ChannelType,
			MessageID:   msg.MessageID,
			MessageSeq:  msg.MessageSeq,
			IsDeleted:   1,
			Version:     version,
		}, tx)
		if err != nil {
			tx.Rollback()
			m.Error(common.ErrData.Error(), zap.Error(err))
			c.ResponseError(errors.New("删除消息错误"))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		m.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		err = m.ctx.SendCMD(config.MsgCMDReq{
			NoPersist:   false,
			ChannelID:   req.ChannelID,
			ChannelType: req.ChannelType,
			CMD:         common.CMDSyncMessageExtra,
			FromUID:     req.FromUID,
			Param: map[string]interface{}{
				"channel_id":   req.ChannelID,
				"channel_type": req.ChannelType,
			},
		})
	} else {
		err = m.ctx.SendCMD(config.MsgCMDReq{
			NoPersist:   false,
			ChannelID:   req.ChannelID,
			ChannelType: req.ChannelType,
			CMD:         common.CMDSyncMessageExtra,
			Param: map[string]interface{}{
				"channel_id":   req.ChannelID,
				"channel_type": req.ChannelType,
			},
		})
	}

	if err != nil {
		m.Error("发送cmd失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}
func (m *Manager) deleteProhibitWords(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	is_deleted := c.Query("is_deleted")
	isDeleted, _ := strconv.Atoi(is_deleted)
	id := c.Query("id")
	if id == "" || (isDeleted != 0 && isDeleted != 1) {
		c.ResponseError(errors.New("参数错误"))
		return
	}
	tempID, _ := strconv.Atoi(id)
	words, err := m.managerDB.queryProhibitWordsWithID(tempID)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询违禁词错误"))
		return
	}
	if words == nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("操作的违禁词不存在"))
		return
	}
	words.IsDeleted = isDeleted
	words.Version = m.ctx.GenSeq(common.ProhibitWordKey)
	err = m.managerDB.updateProhibitWord(words)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("修改违禁词错误"))
		return
	}
	c.ResponseOK()
}

func (m *Manager) prohibitWords(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	pageIndex, pageSize := c.GetPage()
	searchKey := c.Query("search_key")
	var result []*prohibitWordsModel
	var count int64 = 0
	if searchKey == "" {
		result, err = m.managerDB.queryProhibitWords(uint64(pageIndex), uint64(pageSize))
		if err != nil {
			m.Error(common.ErrData.Error(), zap.Error(err))
			c.ResponseError(errors.New("查询违禁词列表错误"))
			return
		}
		count, err = m.managerDB.queryProhibitWordsCount()
		if err != nil {
			if err != nil {
				m.Error(common.ErrData.Error(), zap.Error(err))
				c.ResponseError(errors.New("查询违禁词总数错误"))
				return
			}
		}
	} else {
		result, err = m.managerDB.queryProhibitWordsWithContentAndPage(searchKey, uint64(pageIndex), uint64(pageSize))
		if err != nil {
			m.Error(common.ErrData.Error(), zap.Error(err))
			c.ResponseError(errors.New("搜索查询违禁词列表错误"))
			return
		}

		count, err = m.managerDB.queryProhibitWordsCountWithContent(searchKey)
		if err != nil {
			if err != nil {
				m.Error(common.ErrData.Error(), zap.Error(err))
				c.ResponseError(errors.New("查询搜索违禁词总数错误"))
				return
			}
		}
	}

	list := make([]*prohibitWordsVO, 0)

	if len(result) > 0 {
		for _, word := range result {
			list = append(list, &prohibitWordsVO{
				Content:   word.Content,
				CreatedAt: word.CreatedAt.String(),
				IsDeleted: word.IsDeleted,
				Version:   word.Version,
				Id:        word.Id,
			})
		}
	}
	c.Response(map[string]interface{}{
		"list":  list,
		"count": count,
	})
}
func (m *Manager) addProhibitWords(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	content := c.Query("content")
	if content == "" {
		c.ResponseError(errors.New("违禁词不能为空"))
		return
	}
	model, err := m.managerDB.queryProhibitWordsWithContent(content)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询违禁词错误"))
		return
	}
	version := m.ctx.GenSeq(common.ProhibitWordKey)
	if model != nil {
		model.IsDeleted = 0
		model.Version = version
		err = m.managerDB.updateProhibitWord(model)
		if err != nil {
			m.Error(common.ErrData.Error(), zap.Error(err))
			c.ResponseError(errors.New("修改违禁词错误"))
			return
		}
	} else {
		err = m.managerDB.insertProhibitWord(&prohibitWordsModel{
			IsDeleted: 0,
			Content:   content,
			Version:   version,
		})
		if err != nil {
			m.Error(common.ErrData.Error(), zap.Error(err))
			c.ResponseError(errors.New("新增违禁词错误"))
			return
		}
	}
	c.ResponseOK()
}
func (m *Manager) recordpersonal(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	uid := c.Query("uid")
	touid := c.Query("touid")
	pageIndex, pageSize := c.GetPage()
	if strings.TrimSpace(uid) == "" || strings.TrimSpace(touid) == "" {
		c.ResponseError(errors.New("uid不能为空"))
		return
	}
	channelID := common.GetFakeChannelIDWith(uid, touid)
	msgs, err := m.managerDB.queryWithChannelID(channelID, uint64(pageIndex), uint64(pageSize))
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询消息记录错误"))
		return
	}

	count, err := m.managerDB.queryRecordCount(channelID)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询消息总量错误"))
		return
	}
	list := make([]*recordVO, 0)
	if len(msgs) == 0 {
		c.Response(list)
		return
	}
	uids := make([]string, 0)
	msgIds := make([]int64, 0)
	for _, msg := range msgs {
		uids = append(uids, msg.FromUID)
		msgIds = append(msgIds, msg.MessageID)
	}
	msgExtrs, err := m.managerDB.queryMsgExtrWithMsgIds(msgIds)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询消息扩展错误"))
		return
	}
	userList, err := m.userService.GetUsers(uids)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询发送者信息错误"))
		return
	}
	for _, msg := range msgs {
		sendName := ""
		for _, user := range userList {
			if user.UID == msg.FromUID {
				sendName = user.Name
			}
		}
		isDeleted := 0
		revoke := 0
		editedAt := 0
		readedCount := 0
		var payloadMap map[string]interface{}
		for _, extr := range msgExtrs {
			msgID, _ := strconv.ParseInt(extr.MessageID, 10, 64)
			if msgID == msg.MessageID {
				isDeleted = extr.IsDeleted
				revoke = extr.Revoke
				editedAt = extr.EditedAt
				readedCount = extr.ReadedCount
				if extr.ContentEdit.String != "" {
					err := util.ReadJsonByByte([]byte(extr.ContentEdit.String), &payloadMap)
					if err != nil {
						log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(extr.ContentEdit.String)))
					}
				}
			}
		}
		if payloadMap == nil {
			err := util.ReadJsonByByte(msg.Payload, &payloadMap)
			if err != nil {
				log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(msg.Payload)))
			}
		}
		messageId := strconv.FormatInt(msg.MessageID, 10)
		list = append(list, &recordVO{
			MessageID:   messageId,
			Sender:      msg.FromUID,
			SenderName:  sendName,
			Payload:     payloadMap,
			Signal:      msg.Signal,
			IsDeleted:   isDeleted,
			CreatedAt:   msg.CreatedAt.String(),
			EditedAt:    editedAt,
			Revoke:      revoke,
			ReadedCount: readedCount,
		})
	}
	c.Response(&recordResp{
		Count: count,
		List:  list,
	})
}
func (m *Manager) record(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	var channelID = c.Query("channel_id")
	pageIndex, pageSize := c.GetPage()
	msgs, err := m.managerDB.queryWithChannelID(channelID, uint64(pageIndex), uint64(pageSize))
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询消息记录错误"))
		return
	}
	count, err := m.managerDB.queryRecordCount(channelID)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询消息总量错误"))
		return
	}

	list := make([]*recordVO, 0)
	if len(msgs) == 0 {
		c.Response(list)
		return
	}
	uids := make([]string, 0)
	msgIds := make([]int64, 0)
	for _, msg := range msgs {
		uids = append(uids, msg.FromUID)
		msgIds = append(msgIds, msg.MessageID)
	}
	msgExtrs, err := m.managerDB.queryMsgExtrWithMsgIds(msgIds)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询消息扩展错误"))
		return
	}
	userList, err := m.userService.GetUsers(uids)
	if err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(errors.New("查询发送者信息错误"))
		return
	}
	for _, msg := range msgs {
		sendName := ""
		for _, user := range userList {
			if user.UID == msg.FromUID {
				sendName = user.Name
			}
		}
		isDeleted := 0
		revoke := 0
		editedAt := 0
		readedCount := 0
		var payloadMap map[string]interface{}
		for _, extr := range msgExtrs {
			msgID, _ := strconv.ParseInt(extr.MessageID, 10, 64)
			if msgID == msg.MessageID {
				isDeleted = extr.IsDeleted
				revoke = extr.Revoke
				editedAt = extr.EditedAt
				readedCount = extr.ReadedCount
				if extr.ContentEdit.String != "" {
					err := util.ReadJsonByByte([]byte(extr.ContentEdit.String), &payloadMap)
					if err != nil {
						log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(extr.ContentEdit.String)))
					}
				}
			}
		}
		if payloadMap == nil {
			err := util.ReadJsonByByte(msg.Payload, &payloadMap)
			if err != nil {
				log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(msg.Payload)))
			}
		}

		messageId := strconv.FormatInt(msg.MessageID, 10)

		list = append(list, &recordVO{
			MessageID:   messageId,
			MessageSeq:  msg.MessageSeq,
			Sender:      msg.FromUID,
			SenderName:  sendName,
			Payload:     payloadMap,
			Signal:      0,
			IsDeleted:   isDeleted,
			CreatedAt:   msg.CreatedAt.String(),
			EditedAt:    editedAt,
			Revoke:      revoke,
			ReadedCount: readedCount,
		})
	}
	c.Response(&recordResp{
		Count: count,
		List:  list,
	})
}
func (m *Manager) sendMsgToAllUsers(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	type SendMsgReq struct {
		Content string `json:"content"`
	}
	var req SendMsgReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	userList, err := m.userService.GetAllUsers()
	if err != nil {
		c.ResponseError(err)
		return
	}
	uids := make([][]string, 0)
	tempUserList := make([]string, 0)
	for _, user := range userList {
		if len(tempUserList) == 1000 {
			uids = append(uids, tempUserList)
			tempUserList = make([]string, 0)
		}
		tempUserList = append(tempUserList, user.UID)
	}
	if len(tempUserList) > 0 {
		uids = append(uids, tempUserList)
	}
	go m.sendMessageBatch(uids, req.Content)
	c.ResponseOK()
}
func (m *Manager) sendMessageBatch(uids [][]string, content string) error {
	for _, list := range uids {
		err := m.ctx.SendMessageBatch(&config.MsgSendBatch{
			Header: config.MsgHeader{
				RedDot: 1,
			},
			FromUID: m.ctx.GetConfig().Account.SystemUID,
			Payload: []byte(util.ToJson(map[string]interface{}{
				"content": content,
				"type":    1,
			})),
			Subscribers: list,
		})
		if err != nil {
			m.Error("发送消息错误", zap.Error(err))
			return errors.New("发送消息错误")
		}
		time.Sleep(time.Second)
	}
	return nil
}

// 发送消息
func (m *Manager) sendMsg(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	var req managerSendMsgReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.check(); err != nil {
		c.ResponseError(err)
		return
	}
	var receiverName string = ""
	if req.ReceivedChannelType == int(common.ChannelTypePerson) {
		user, err := m.userService.GetUser(req.ReceivedChannelID)
		if err != nil {
			m.Error("查询接受的者信息错误", zap.Error(err), zap.String("uid", req.ReceivedChannelID))
			c.ResponseError(errors.New("查询接受的者信息错误"))
			return
		}
		if user == nil {
			c.ResponseError(errors.New("消息接受者用户不存在"))
			return
		}
		receiverName = user.Name
	}
	if req.ReceivedChannelType == int(common.ChannelTypeGroup) {
		group, err := m.groupService.GetGroupWithGroupNo(req.ReceivedChannelID)
		if err != nil {
			m.Error("查询接受群信息错误", zap.Error(err), zap.String("groupNo", req.ReceivedChannelID))
			c.ResponseError(errors.New("查询接受群信息错误"))
			return
		}
		if group == nil {
			c.ResponseError(errors.New("消息接受群不存在"))
			return
		}
		receiverName = group.Name
	}
	err = m.ctx.SendMessage(&config.MsgSendReq{
		Header: config.MsgHeader{
			RedDot: 1,
		},
		FromUID:     req.Sender,
		ChannelID:   req.ReceivedChannelID,
		ChannelType: uint8(req.ReceivedChannelType),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content":  req.Content,
			"type":     1,
			"from_uid": req.Sender,
		})),
	})
	if err != nil {
		m.Error("发送消息错误", zap.Error(err))
		c.ResponseError(errors.New("发送消息错误"))
		return
	}
	// 添加发送消息记录
	err = m.managerDB.insertMsgHistory(&managerMsgModel{
		Sender:              req.Sender,
		SenderName:          req.SenderName,
		ReceiverChannelType: req.ReceivedChannelType,
		Receiver:            req.ReceivedChannelID,
		ReceiverName:        receiverName,
		HandlerUID:          c.GetLoginUID(),
		HandlerName:         c.GetLoginName(),
		Content:             req.Content,
	})
	if err != nil {
		m.Error("添加发送消息记录错误", zap.Error(err))
		c.ResponseError(errors.New("添加发送消息记录错误"))
		return
	}
	c.ResponseOK()
}

// 代发消息列表
func (m *Manager) list(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	pageIndex, pageSize := c.GetPage()
	list, err := m.managerDB.queryMsgWithPage(uint64(pageSize), uint64(pageIndex))
	if err != nil {
		m.Error("查询代发消息记录错误", zap.Error(err))
		c.ResponseError(errors.New("查询代发消息记录错误"))
		return
	}
	count, err := m.managerDB.queryMsgCount()
	if err != nil {
		m.Error("查询代发消息总数错误", zap.Error(err))
		c.ResponseError(errors.New("查询代发消息总数错误"))
		return
	}
	result := make([]*managerSendMsgResp, 0)
	for _, model := range list {
		result = append(result, &managerSendMsgResp{
			Sender:              model.Sender,
			SenderName:          model.SenderName,
			Receiver:            model.Receiver,
			ReceiverName:        model.ReceiverName,
			ReceiverChannelType: model.ReceiverChannelType,
			HandlerUID:          model.HandlerUID,
			HandlerName:         model.HandlerName,
			Content:             model.Content,
			CreatedAt:           model.CreatedAt.String(),
		})
	}
	c.Response(map[string]interface{}{
		"count": count,
		"list":  result,
	})
}

func (m *managerSendMsgReq) check() error {
	if m.ReceivedChannelID == "" {
		return errors.New("接受者ID不能为空")
	}
	if m.Sender == "" {
		return errors.New("发送者ID不能为空")
	}
	if m.SenderName == "" {
		return errors.New("发送者名字不能为空")
	}
	if m.ReceivedChannelType != int(common.ChannelTypeGroup) && m.ReceivedChannelType != int(common.ChannelTypePerson) && m.ReceivedChannelType != int(common.ChannelTypeNone) {
		return errors.New("接受者类型错误")
	}
	return nil
}

func (m *Manager) genMessageExtraSeq(channelID string) int64 {
	return m.ctx.GenSeq(fmt.Sprintf("%s:%s", common.MessageExtraSeqKey, channelID))
}

type managerSendMsgReq struct {
	Sender              string `json:"sender"`                // 发送者uid
	SenderName          string `json:"sender_name"`           // 发送者名字
	ReceivedChannelID   string `json:"received_channel_id"`   // 接受者id
	ReceivedChannelType int    `json:"received_channel_type"` // 接受类型
	Content             string `json:"content"`               // 发送内容
}

type managerSendMsgResp struct {
	Receiver            string `json:"receiver"`              // 接受者uid
	ReceiverName        string `json:"receiver_name"`         // 接受者名字
	ReceiverChannelType int    `json:"receiver_channel_type"` // 接受者频道类型
	Sender              string `json:"sender"`                // 发送者uid
	SenderName          string `json:"sender_name"`           // 发送者名字
	HandlerUID          string `json:"handler_uid"`           // 操作者uid
	HandlerName         string `json:"handler_name"`          // 操作者名字
	Content             string `json:"content"`               // 发送内容
	CreatedAt           string `json:"created_at"`            // 发送时间
}
type recordResp struct {
	Count int64       `json:"count"`
	List  []*recordVO `json:"list"`
}
type recordVO struct {
	MessageID   string                 `json:"message_id"`   // 消息编号
	MessageSeq  uint32                 `json:"message_seq"`  // 消息序号
	Sender      string                 `json:"sender"`       // 发送者uid
	SenderName  string                 `json:"sender_name"`  // 发送者名字
	Signal      int                    `json:"signal"`       // 是否加密
	Payload     map[string]interface{} `json:"payload"`      // 发送内容
	IsDeleted   int                    `json:"is_deleted"`   // 是否删除
	ReadedCount int                    `json:"readed_count"` // 已读人数
	Revoke      int                    `json:"revoke"`       // 是否撤回
	CreatedAt   string                 `json:"created_at"`   // 发送时间
	EditedAt    int                    `json:"edited_at"`    // 编辑时间
}
type prohibitWordsVO struct {
	Id        int64  `json:"id"`
	Content   string `json:"content"`    // 违禁词
	IsDeleted int    `json:"is_deleted"` // 是否删除
	Version   int64  `json:"version"`    // 版本
	CreatedAt string `json:"created_at"` // 时间
}
