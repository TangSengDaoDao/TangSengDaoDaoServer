package message

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	commonapi "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/file"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
	"go.uber.org/zap"
)

// Message 消息相关API
type Message struct {
	ctx *config.Context
	log.Log
	db                  *DB
	messageReactionDB   *messageReactionDB
	userDB              *user.DB
	messageExtraDB      *messageExtraDB
	memberChangeDB      *memberChangeDB
	memberReadedDB      *memberReadedDB
	channelOffsetDB     *channelOffsetDB
	deviceOffsetDB      *deviceOffsetDB
	conversationExtradb *conversationExtraDB
	messageUserExtraDB  *messageUserExtraDB
	remindersDB         *remindersDB
	userService         user.IService
	groupService        group.IService
	commonService       commonapi.IService
	fileService         file.IService
}

// New New
func New(ctx *config.Context) *Message {

	m := &Message{

		ctx:                 ctx,
		Log:                 log.NewTLog("Message"),
		db:                  NewDB(ctx),
		userDB:              user.NewDB(ctx),
		messageExtraDB:      newMessageExtraDB(ctx),
		groupService:        group.NewService(ctx),
		memberChangeDB:      newMemberChangeDB(ctx),
		memberReadedDB:      newMemberReadedDB(ctx),
		conversationExtradb: newConversationExtraDB(ctx),
		messageReactionDB:   newMessageReactionDB(ctx),
		messageUserExtraDB:  newMessageUserExtraDB(ctx),
		channelOffsetDB:     newChannelOffsetDB(ctx),
		deviceOffsetDB:      newDeviceOffsetDB(ctx.DB()),
		remindersDB:         newRemindersDB(ctx),
		userService:         user.NewService(ctx),
		commonService:       commonapi.NewService(ctx),
		fileService:         file.NewService(ctx),
	}
	m.ctx.AddEventListener(event.GroupMemberAdd, m.handleGroupMemberAddEvent)
	return m
}

// Route 路由配置
func (m *Message) Route(r *wkhttp.WKHttp) {
	message := r.Group("/v1/message", m.ctx.AuthMiddleware(r))
	{

		message.POST("/sync", m.sync)                             // 同步消息 (写模式才用到 TODO：此方法未来将弃用)
		message.POST("/syncack/:last_message_seq", m.syncack)     // 同步消息回执 （写模式才用到 TODO：此方法未来将弃用）
		message.DELETE("", m.delete)                              // 删除消息
		message.POST("/revoke", m.revoke)                         // 撤回消息
		message.POST("/offset", m.offset)                         // 清除某频道消息
		message.PUT("/voicereaded", m.voiceReaded)                // 语音消息设置为已读
		message.POST("/search", m.search)                         // 消息搜索
		message.POST("/typing", m.typing)                         // 发送typing消息
		message.POST("/channel/sync", m.syncChannelMessage)       // 同步频道消息
		message.POST("/extra/sync", m.syncMessageExtra)           // 同步消息扩展
		message.POST("/readed", m.messageReaded)                  // 消息已读
		message.GET("/sync/sensitivewords", m.syncSensitiveWords) // 同步敏感词
		message.POST("/edit", m.messageEdit)                      // 消息编辑
		message.POST("/reminder/sync", m.reminderSync)            // 同步提醒
		message.POST("/reminder/done", m.reminderDone)            // 提醒已处理完成
		message.GET("/prohibit_words/sync", m.synccProhibitWords) // 同步违禁词
	}
	messages := r.Group("/v1/messages", m.ctx.AuthMiddleware(r))
	{
		// messages.PUT("/:message_id/voicereaded", m.voiceReaded)
		messages.GET("/:message_id/receipt", m.messageReceiptList) // 消息回执列表
	}
	// 回应
	reactions := r.Group("/v1/reactions", m.ctx.AuthMiddleware(r))
	{
		reactions.POST("", m.addOrCancelReaction) // 添加或取消回应
	}
	reaction := r.Group("/v1/reaction", m.ctx.AuthMiddleware(r))
	{
		reaction.POST("/sync", m.syncReaction)
	}
	msg := r.Group("/v1/message")
	{
		msg.POST("/send", m.sendMsg) // 代发消息
	}
	m.ctx.AddMessagesListener(m.listenerMessages) // 监听消息
}

func (m *Message) sendMsg(c *wkhttp.Context) {
	if !m.ctx.GetConfig().Message.SendMessageOn {
		c.ResponseError(errors.New("不支持代发消息"))
		return
	}
	var req struct {
		Token              string                 `json:"token"`                // 发送者
		ReceiveChannelID   string                 `json:"receive_channel_id"`   // 接受者id
		ReceiveChannelType uint8                  `json:"receive_channel_type"` // 接受类型
		Payload            map[string]interface{} `json:"payload"`              // 消息体
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("数据格式有误！", err)
		return
	}
	if req.Token == "" {
		c.ResponseError(errors.New("发送者token不能为空"))
		return
	}
	if req.ReceiveChannelID == "" {
		c.ResponseError(errors.New("接受channelID不能为空"))
		return
	}
	if req.Payload == nil {
		c.ResponseError(errors.New("消息不能为空"))
		return
	}
	uidAndName, err := m.ctx.Cache().Get(m.ctx.GetConfig().Cache.TokenCachePrefix + req.Token)
	if err != nil {
		m.Error("解析token错误", zap.Error(err))
		c.ResponseError(errors.New("解析token错误"))
		return
	}
	if strings.TrimSpace(uidAndName) == "" {
		c.ResponseError(errors.New("请先登录"))
		return
	}
	uidAndNames := strings.Split(uidAndName, "@")
	if len(uidAndNames) < 2 {
		c.ResponseError(errors.New("token错误"))
		return
	}
	uid := uidAndNames[0]
	if uid == "" {
		c.ResponseError(errors.New("发送者不能为空"))
		return
	}

	if req.ReceiveChannelType == common.ChannelTypePerson.Uint8() {
		sendUserIsFriend, err := m.userService.IsFriend(uid, req.ReceiveChannelID)
		if err != nil {
			m.Error("查询发送者与接受者好友关系错误", zap.Error(err))
			c.ResponseError(errors.New("查询好友关系错误"))
			return
		}
		if !sendUserIsFriend {
			c.ResponseError(errors.New("发送者与接受者不是好友"))
			return
		}
		recvUserIsFriend, err := m.userService.IsFriend(req.ReceiveChannelID, uid)
		if err != nil {
			m.Error("查询接受者与发送者好友关系错误", zap.Error(err))
			c.ResponseError(errors.New("查询接受者与发送者好友关系错误"))
			return
		}
		if !recvUserIsFriend {
			c.ResponseError(errors.New("接受者与发送者不是好友"))
			return
		}
	}
	if req.ReceiveChannelType == common.ChannelTypeGroup.Uint8() {
		isExist, err := m.groupService.ExistMember(req.ReceiveChannelID, uid)
		if err != nil {
			m.Error("查询发送者是否在群内错误", zap.Error(err))
			c.ResponseError(errors.New("查询发送者是否在群内错误"))
			return
		}
		if !isExist {
			c.ResponseError(errors.New("未在群内"))
			return
		}
	}
	err = m.sendMessage(req.ReceiveChannelID, req.ReceiveChannelType, uid, req.Payload)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

func (m *Message) sendMessage(channelID string, channelType uint8, fromUID string, payload map[string]interface{}) error {
	err := m.ctx.SendMessage(&config.MsgSendReq{
		Header: config.MsgHeader{
			RedDot: 1,
		},
		ChannelID:   channelID,
		ChannelType: channelType,
		FromUID:     fromUID,
		Payload:     []byte(util.ToJson(payload)),
	})
	if err != nil {
		m.Error("发送消息错误", zap.Error(err))
		return errors.New("发送消息错误")
	}
	return nil
}

// 消息编辑
func (m *Message) messageEdit(c *wkhttp.Context) {
	var req struct {
		MessageID   string `json:"message_id"`
		MessageSeq  uint32 `json:"message_seq"`
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
		ContentEdit string `json:"content_edit"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("数据格式有误！", err)
		return
	}
	if req.MessageID == "" {
		c.ResponseError(errors.New("消息ID不能为空！"))
		return
	}
	if req.MessageSeq == 0 {
		c.ResponseError(errors.New("消息序号不能为空！"))
		return
	}
	if req.ChannelID == "" {
		c.ResponseError(errors.New("频道ID不能为空！"))
		return
	}
	contentEdit := dbr.NewNullString(req.ContentEdit).String
	contentMD5 := util.MD5(contentEdit)

	exist, err := m.messageExtraDB.existContentEdit(req.MessageID, contentMD5)
	if err != nil {
		m.Error("查询是否存在相同正文失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否存在相同正文失败！"))
		return
	}
	if exist {
		m.Warn("存在相同编辑正文，不再处理！")
		c.ResponseOK()
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(c.GetLoginUID(), req.ChannelID)
	}

	version := m.genMessageExtraSeq(fakeChannelID)
	err = m.messageExtraDB.insertOrUpdateContentEdit(&messageExtraModel{
		MessageID:       req.MessageID,
		MessageSeq:      req.MessageSeq,
		ChannelID:       fakeChannelID,
		ChannelType:     req.ChannelType,
		ContentEdit:     dbr.NewNullString(req.ContentEdit),
		ContentEditHash: contentMD5,
		EditedAt:        int(time.Now().Unix()),
		Version:         version,
	})
	if err != nil {
		m.Error("添加或修改编辑内容失败！", zap.Error(err))
		c.ResponseError(errors.New("添加或修改编辑内容失败！"))
		return
	}

	err = m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		FromUID:     c.GetLoginUID(),
		CMD:         common.CMDSyncMessageExtra,
	})

	if err != nil {
		m.Error("发送cmd失败！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

// 消息已读
func (m *Message) messageReaded(c *wkhttp.Context) {
	var req struct {
		MessageIDs  []string `json:"message_ids"`
		ChannelID   string   `json:"channel_id"`
		ChannelType uint8    `json:"channel_type"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("数据格式有误！", err)
		return
	}
	if len(req.MessageIDs) == 0 {
		c.ResponseError(errors.New("message_ids不能为空！"))
		return
	}
	// var cloneNo string
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(req.ChannelID, c.GetLoginUID())
	}
	if len(req.MessageIDs) <= 0 {
		c.ResponseOK()
		return
	}
	messageIDStrs := util.RemoveRepeatedElement(req.MessageIDs)

	messages, err := m.db.queryMessagesWithMessageIDs(fakeChannelID, req.ChannelType, messageIDStrs)
	if err != nil {
		c.ResponseErrorf("查询消息失败！", err)
		return
	}
	if len(messages) <= 0 {
		m.Warn("没有读取到消息！", zap.Strings("messages", req.MessageIDs))
		c.ResponseError(errors.New("没有读取到消息！"))
		return
	}

	tx, _ := m.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()

	fromUIDs := make([]string, 0, len(messages)) // 消息发送者
	for _, message := range messages {
		fromUIDs = append(fromUIDs, message.FromUID)
		err := m.memberReadedDB.insertOrUpdateTx(&memberReadedModel{
			MessageID:   message.MessageID,
			ChannelID:   fakeChannelID,
			ChannelType: req.ChannelType,
			UID:         c.GetLoginUID(),
		}, tx)
		if err != nil {
			tx.Rollback()
			c.ResponseErrorf("添加已读数据失败！", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		c.ResponseErrorf("提交事务失败！", err)
		return
	}

	var messageReadedCountMap map[int64]int
	if req.ChannelType != common.ChannelTypePerson.Uint8() {
		messageReadedCountMap, err = m.memberReadedDB.queryCountWithMessageIDs(fakeChannelID, req.ChannelType, messageIDStrs)
		if err != nil {
			c.ResponseErrorf("获取消息已读数量map失败！", err)
			return
		}
	}

	tx2, _ := m.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx2.RollbackUnlessCommitted()
			panic(err)
		}
	}()

	for _, message := range messages {
		version := m.genMessageExtraSeq(fakeChannelID)
		count := messageReadedCountMap[message.MessageID]
		if req.ChannelType == common.ChannelTypePerson.Uint8() {
			count = 1
		}
		err = m.messageExtraDB.insertOrUpdateReadedCountTx(&messageExtraModel{
			MessageID:   strconv.FormatInt(message.MessageID, 10),
			MessageSeq:  message.MessageSeq,
			FromUID:     message.FromUID,
			ChannelID:   fakeChannelID,
			ChannelType: req.ChannelType,
			ReadedCount: count,
			Version:     version,
		}, tx2)
		if err != nil {
			tx2.Rollback()
			m.Error("添加或更新消息扩展数据失败！", zap.Error(err), zap.Int64("messageID", message.MessageID), zap.String("channelID", fakeChannelID))
			c.ResponseError(errors.New("添加或更新消息扩展数据失败！"))
			return
		}
	}

	if err := tx2.Commit(); err != nil {
		tx2.Rollback()
		c.ResponseErrorf("提交事务失败！", err)
		return
	}

	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		err = m.ctx.SendCMD(config.MsgCMDReq{
			NoPersist:   true,
			ChannelID:   req.ChannelID,
			ChannelType: req.ChannelType,
			FromUID:     c.GetLoginUID(),
			CMD:         common.CMDSyncMessageExtra,
		})
	} else {
		err = m.ctx.SendCMD(config.MsgCMDReq{
			NoPersist:   true,
			ChannelID:   req.ChannelID,
			ChannelType: req.ChannelType,
			Subscribers: fromUIDs, // 消息只发送给发送者
			CMD:         common.CMDSyncMessageExtra,
		})
	}

	if err != nil {
		c.ResponseErrorf("发送同步命令失败！", err)
		return
	}

	c.ResponseOK()

}

// 消息回执列表
func (m *Message) messageReceiptList(c *wkhttp.Context) {
	messageIDStr := c.Param("message_id")

	readed := c.Query("readed") // 查询已读未读的消息，0.未读 1.已读
	pIndex, pSize := c.GetPage()

	resps := make([]memberReceiptResp, 0)
	uids := make([]string, 0)
	if readed == "1" {
		memberReadedModels, err := m.memberReadedDB.queryWithMessageIDAndPage(messageIDStr, uint64(pIndex), uint64(pSize))
		if err != nil {
			c.ResponseErrorf("查询已读列表失败！", err)
			return
		}
		if len(memberReadedModels) > 0 {
			for _, memberReadedM := range memberReadedModels {
				uids = append(uids, memberReadedM.UID)
			}
		}
	}
	userResps, err := m.userService.GetUsers(uids)
	if err != nil {
		c.ResponseErrorf("查询用户数据失败！", err)
		return
	}
	userMap := map[string]*user.Resp{}
	if len(userResps) > 0 {
		for _, userResp := range userResps {
			userMap[userResp.UID] = userResp
		}
	}

	for _, uid := range uids {
		userResp := userMap[uid]
		var name string
		if userResp != nil {
			name = userResp.Name
		}
		resps = append(resps, memberReceiptResp{
			UID:  uid,
			Name: name,
		})
	}
	c.Response(resps)

}

//	func (m *Message) getCacheMessageReactionSeq(uid, channelID string, channelType uint8) (int64, error) {
//		versionStr, err := m.ctx.GetRedisConn().Hget(fmt.Sprintf("messageReactionSeq:%s", uid), fmt.Sprintf("%s-%d", channelID, channelType))
//		if err != nil {
//			return 0, err
//		}
//		if versionStr == "" {
//			return 0, nil
//		}
//		version, _ := strconv.ParseInt(versionStr, 10, 64)
//		return version, nil
//	}
func (m *Message) getMessageExtraVersion(uid, source, channelID string, channelType uint8) (int64, error) {
	versionStr, err := m.ctx.GetRedisConn().Hget(fmt.Sprintf("messageExtraVersion:%s%s", uid, source), fmt.Sprintf("%s-%d", channelID, channelType))
	if err != nil {
		return 0, err
	}
	if versionStr == "" {
		return 0, nil
	}
	version, _ := strconv.ParseInt(versionStr, 10, 64)
	return version, nil

}

func (m *Message) setMessageExtraVersion(uid, channelID string, channelType uint8, source string, messageExtraVersion int64) error {
	err := m.ctx.GetRedisConn().Hset(fmt.Sprintf("messageExtraVersion:%s%s", uid, source), fmt.Sprintf("%s-%d", channelID, channelType), fmt.Sprintf("%d", messageExtraVersion))
	if err != nil {
		return err
	}
	return nil
}

// 同步扩展消息数据
func (m *Message) syncMessageExtra(c *wkhttp.Context) {
	var req struct {
		ChannelID    string `json:"channel_id"`
		ChannelType  uint8  `json:"channel_type"`
		ExtraVersion int64  `json:"extra_version"`
		Source       string `json:"source"` // 操作源
		Limit        int    `json:"limit"`  // 数据限制
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("数据格式有误！", err)
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(c.GetLoginUID(), req.ChannelID)
	}
	cacheExtraVersion, err := m.getMessageExtraVersion(c.GetLoginUID(), req.Source, fakeChannelID, req.ChannelType)
	if err != nil {
		c.ResponseErrorf("从缓存中获取消息扩展版本失败！", err)
		return
	}
	extraVersion := req.ExtraVersion
	if cacheExtraVersion >= extraVersion {
		extraVersion = cacheExtraVersion
	} else {
		err = m.setMessageExtraVersion(c.GetLoginUID(), fakeChannelID, req.ChannelType, req.Source, extraVersion)
		if err != nil {
			c.ResponseErrorf("缓存最大的消息扩展版本失败！", err)
			return
		}

	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}
	if strings.TrimSpace(req.ChannelID) == "" {
		c.ResponseError(errors.New("频道ID不能为空！"))
		return
	}
	extraModels, err := m.messageExtraDB.sync(extraVersion, fakeChannelID, req.ChannelType, uint64(limit), c.GetLoginUID())
	if err != nil {
		c.ResponseErrorf("同步消息扩展数据失败！", err)
		return
	}
	resps := make([]*messageExtraResp, 0, len(extraModels))
	if len(extraModels) > 0 {
		for _, extraModel := range extraModels {
			resps = append(resps, newMessageExtraResp(extraModel))
		}
	}
	c.Response(resps)
}

// 同步频道消息
func (m *Message) syncChannelMessage(c *wkhttp.Context) {
	var req config.SyncChannelMessageReq
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}

	// 如果当前用户不在群内，则直接返回空消息数组
	if req.ChannelType == common.ChannelTypeGroup.Uint8() {
		exist, err := m.groupService.ExistMember(req.ChannelID, c.GetLoginUID())
		if err != nil {
			m.Error("查询是否在群内存在失败！", zap.Error(err))
			c.ResponseError(errors.New("查询是否在群内存在失败！"))
			return
		}
		if !exist {
			c.JSON(http.StatusOK, &syncChannelMessageResp{
				StartMessageSeq: req.EndMessageSeq,
				EndMessageSeq:   req.EndMessageSeq,
				PullMode:        req.PullMode,
				Messages:        make([]*MsgSyncResp, 0),
			})
			return
		}
	}
	req.LoginUID = c.GetLoginUID()
	resp, err := m.ctx.IMSyncChannelMessage(req)
	if err != nil {
		m.Error("同步频道内的消息失败！", zap.Error(err), zap.String("req", util.ToJson(req)))
		c.ResponseError(errors.New("同步频道内的消息失败！"))
		return
	}
	fmt.Println("resp----messages-->", len(resp.Messages))

	c.Response(newSyncChannelMessageResp(resp, c.GetLoginUID(), req.DeviceUUID, req.ChannelID, req.ChannelType, m.messageExtraDB, m.messageUserExtraDB, m.messageReactionDB, m.channelOffsetDB, m.deviceOffsetDB))
}

// 输入中
func (m *Message) typing(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	loginName := c.MustGet("name").(string)
	var req struct {
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(err)
		return
	}
	channelID := req.ChannelID
	channelType := req.ChannelType
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		channelID = loginUID
	}
	// 发送输入中的命令
	err := m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		CMD:         common.CMDTyping,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		Param: map[string]interface{}{
			"from_uid":     loginUID,
			"from_name":    loginName,
			"channel_id":   channelID,
			"channel_type": channelType,
		},
	})
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

// 搜索消息
func (m *Message) search(c *wkhttp.Context) {
	var req struct {
		UID         string `json:"uid"` // 搜索的消息限定这某个用户内
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
		ContentType int    `json:"content_type"` // 正文类型
		Keyword     string `json:"keyword"`
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	uid := c.MustGet("uid").(string)
	req.UID = uid
	fmt.Println("req->", req)
	resp, err := network.Post(fmt.Sprintf("%s/message/search", m.ctx.GetConfig().WuKongIM.APIURL), []byte(util.ToJson(req)), nil)
	if err != nil {
		m.Error("调用搜索失败！", zap.Error(err))
		c.ResponseError(errors.New("调用搜索失败！"))
		return
	}
	err = m.handlerIMError(resp)
	if err != nil {
		m.Error("调用搜索错误！", zap.Error(err))
		c.ResponseError(errors.New("调用搜索错误！"))
		return
	}
	var results []map[string]interface{}
	err = util.ReadJsonByByte([]byte(resp.Body), &results)
	if err != nil {
		m.Error("解析搜索数据失败！", zap.Error(err))
		c.ResponseError(errors.New("解析搜索数据失败！"))
		return
	}
	c.JSON(http.StatusOK, results)
}

// 语音消息设置为已读
func (m *Message) voiceReaded(c *wkhttp.Context) {
	var req *voiceReadedReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("数据格式有误！", err)
		return
	}
	if err := req.check(); err != nil {
		c.ResponseError(err)
		return
	}
	loginUID := c.GetLoginUID()

	err := m.messageUserExtraDB.insertOrUpdateVoiceRead(&messageUserExtraModel{
		UID:         loginUID,
		MessageID:   req.MessageID,
		MessageSeq:  req.MessageSeq,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		VoiceReaded: 1,
	})
	if err != nil {
		c.ResponseErrorf("修改语音已读状态失败！", err)
		return
	}
	c.ResponseOK()
}

// 同步回应数据
func (m *Message) syncReaction(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	var req struct {
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
		Seq         int64  `json:"seq"` // 同步序列号
		Limit       uint64 `json:"limit"`
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() {
		if !strings.Contains(req.ChannelID, "@") {
			fakeChannelID = common.GetFakeChannelIDWith(loginUID, req.ChannelID)
		}
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	// cacheReactionSeq, err := m.getCacheMessageReactionSeq(loginUID, req.ChannelID, req.ChannelType)
	// if err != nil {
	// 	m.Error("获取缓存messageSeq失败", zap.Error(err))
	// 	c.ResponseError(errors.New("获取缓存messageSeq失败"))
	// 	return
	// }
	// if req.Seq <= cacheReactionSeq {
	// 	req.Seq = cacheReactionSeq
	// }
	list, err := m.messageReactionDB.queryReactionWithChannelAndSeq(fakeChannelID, req.ChannelType, req.Seq, limit)
	if err != nil {
		m.Error("获取缓存seq错误", zap.Error(err))
		c.ResponseError(errors.New("获取缓存seq错误"))
		return
	}

	toChannelID := common.GetToChannelIDWithFakeChannelID(fakeChannelID, loginUID)

	reactions := make([]*reactionResp, 0)
	if len(list) > 0 {
		for _, model := range list {
			reactions = append(reactions, &reactionResp{
				UID:         model.UID,
				Name:        model.Name,
				ChannelID:   toChannelID,
				ChannelType: model.ChannelType,
				Seq:         model.Seq,
				MessageID:   model.MessageID,
				CreatedAt:   model.CreatedAt.String(),
				Emoji:       model.Emoji,
				IsDeleted:   model.IsDeleted,
			})
		}
	}
	c.JSON(http.StatusOK, reactions)
}

// 添加或取消回应
func (m *Message) addOrCancelReaction(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	loginName := c.GetLoginName()
	var req struct {
		MessageID   string `json:"message_id"`   // 消息唯一ID
		ChannelID   string `json:"channel_id"`   // 频道唯一ID
		ChannelType uint8  `json:"channel_type"` // 频道类型
		Emoji       string `json:"emoji"`        // 回应的emoji
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	model, err := m.messageReactionDB.queryReactionWithUIDAndMessageID(loginUID, req.MessageID)
	if err != nil {
		m.Error("查询登录用户是否回应消息错误", zap.Error(err))
		c.ResponseError(errors.New("查询登录用户是否回应消息错误"))
		return
	}
	fakeChannelID := req.ChannelID
	if req.ChannelType == common.ChannelTypePerson.Uint8() { // 如果是群则需要计算群成员是否变化 如果有变化则将群成员加入到克隆表
		fakeChannelID = common.GetFakeChannelIDWith(req.ChannelID, loginUID)
	}
	seq := m.genMessageReactionSeq(fakeChannelID) // 下次回复seq
	if model == nil {
		//新增回应
		err = m.messageReactionDB.insertReaction(&reactionModel{
			ChannelID:   fakeChannelID,
			ChannelType: req.ChannelType,
			UID:         loginUID,
			Name:        loginName,
			MessageID:   req.MessageID,
			Emoji:       req.Emoji,
			Seq:         seq,
			IsDeleted:   0,
		})
		if err != nil {
			m.Error("新增消息回应错误", zap.Error(err))
			c.ResponseError(errors.New("新增消息回应错误"))
			return
		}
	} else {
		model.Seq = seq
		if model.IsDeleted == 1 {
			model.IsDeleted = 0
			if model.Emoji != req.Emoji {
				model.Emoji = req.Emoji
			}
		} else {
			if model.Emoji == req.Emoji {
				model.IsDeleted = 1
			} else {
				model.Emoji = req.Emoji
			}
		}
		err = m.messageReactionDB.updateReactionStatus(model)
		if err != nil {
			m.Error("修改消息回应错误", zap.Error(err))
			c.ResponseError(errors.New("修改消息回应错误"))
			return
		}
	}

	//发送同步消息cmd
	err = m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   req.ChannelID,
		ChannelType: uint8(req.ChannelType),
		CMD:         common.CMDSyncMessageReaction,
		FromUID:     loginUID,
	})
	if err != nil {
		m.Error("发送同步命令失败！", zap.Error(err))
		c.ResponseErrorf("发送同步命令失败！", err)
		return
	}

	c.ResponseOK()
}
func (m *Message) handlerIMError(resp *rest.Response) error {
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			resultMap, err := util.JsonToMap(resp.Body)
			if err != nil {
				return err
			}
			if resultMap != nil && resultMap["msg"] != nil {
				return fmt.Errorf("IM Extend服务失败！ -> %s", resultMap["msg"])
			}
		}
		return fmt.Errorf("IM Extend服务返回状态[%d]失败！", resp.StatusCode)
	}
	return nil
}

// 同步消息回执
func (m *Message) syncack(c *wkhttp.Context) {
	uid := c.MustGet("uid").(string)
	lastMessageSeqStr := c.Param("last_message_seq")
	lastMessageSeq, err := strconv.ParseUint(lastMessageSeqStr, 10, 64)
	if err != nil {
		m.Error("last_message_seq格式有误！", zap.String("last_message_seq", lastMessageSeqStr))
		c.ResponseError(errors.New("last_message_seq格式有误！"))
		return
	}
	err = m.ctx.IMSyncMessageAck(&config.SyncackReq{
		UID:            uid,
		LastMessageSeq: uint32(lastMessageSeq),
	})
	if err != nil {
		m.Error("同步消息回执失败！", zap.Error(err), zap.String("uid", uid), zap.String("last_message_seq", lastMessageSeqStr))
		c.ResponseError(errors.New("同步消息回执失败！"))
		return
	}
	c.ResponseOK()
}

// 同步消息
func (m *Message) sync(c *wkhttp.Context) {
	uid := c.MustGet("uid").(string)
	var req syncReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	resps, err := m.ctx.IMSyncMessage(&config.MsgSyncReq{
		UID:        uid,
		MessageSeq: req.MaxMessageSeq,
		Limit:      req.Limit,
	})
	if err != nil {
		m.Error("同步消息失败！", zap.Error(err), zap.String("uid", uid))
		c.ResponseError(errors.New("同步消息失败！"))
		return
	}
	messageIDs := make([]string, 0, len(resps))
	for _, message := range resps {
		messageIDs = append(messageIDs, fmt.Sprintf("%d", message.MessageID))
	}

	// 全局扩充数据
	messageExtras, err := m.messageExtraDB.queryWithMessageIDs(messageIDs, c.GetLoginUID())
	if err != nil {
		log.Error("查询消息扩展字段失败！", zap.Error(err))
	}
	messageExtraMap := map[string]*messageExtraDetailModel{}
	if len(messageExtras) > 0 {
		for _, messageExtra := range messageExtras {
			messageExtraMap[messageExtra.MessageID] = messageExtra
		}
	}
	// 用户扩充数据
	messageUserExtras, err := m.messageUserExtraDB.queryWithMessageIDsAndUID(messageIDs, c.GetLoginUID())
	if err != nil {
		log.Error("查询用户消息扩展字段失败！", zap.Error(err))
	}
	messageUserExtraMap := map[string]*messageUserExtraModel{}
	if len(messageUserExtras) > 0 {
		for _, messageUserExtraM := range messageUserExtras {
			messageUserExtraMap[messageUserExtraM.MessageID] = messageUserExtraM
		}
	}

	channelOffsetM, err := m.channelOffsetDB.queryWithUIDAndChannel(c.GetLoginUID(), req.ChannelID, req.ChannelType)
	if err != nil {
		m.Error("查询偏移量失败！", zap.Error(err))
		c.ResponseError(errors.New("查询偏移量失败！"))
		return
	}

	respVos := make([]*MsgSyncResp, 0)
	for _, resp := range resps {
		if channelOffsetM != nil && resp.MessageSeq <= channelOffsetM.MessageSeq {
			continue
		}
		messageIDStr := strconv.FormatInt(resp.MessageID, 10)
		messageExtraM := messageExtraMap[messageIDStr]
		messageUserExtraM := messageUserExtraMap[messageIDStr]
		respVo := &MsgSyncResp{}
		respVo.from(resp, c.GetLoginUID(), messageExtraM, messageUserExtraM, nil)
		respVos = append(respVos, respVo)
	}

	c.JSON(http.StatusOK, respVos)
}

// 删除消息
func (m *Message) delete(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	var reqs []*deleteReq
	if err := c.BindJSON(&reqs); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(reqs) == 0 {
		c.ResponseError(errors.New("参数不能为空！"))
		return
	}
	for _, req := range reqs {
		if err := req.check(); err != nil {
			c.ResponseError(err)
			return
		}
	}

	tx, _ := m.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	for _, req := range reqs {
		err := m.messageUserExtraDB.insertOrUpdateDeletedTx(&messageUserExtraModel{
			UID:              loginUID,
			MessageID:        req.MessageID,
			MessageSeq:       req.MessageSeq,
			ChannelID:        req.ChannelID,
			ChannelType:      req.ChannelType,
			MessageIsDeleted: 1,
		}, tx)
		if err != nil {
			tx.Rollback()
			m.Error("删除消息失败！", zap.Error(err))
			c.ResponseError(errors.New("删除消息失败！"))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		m.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}

	err := m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		CMD:         CMDMessageDeleted,
		Param: map[string]interface{}{
			"messages": reqs,
		},
	})
	if err != nil {
		m.Error("发送命令失败", zap.Error(err))
		c.ResponseError(errors.New("发送命令失败"))
		return
	}

	c.ResponseOK()
}

func (m *Message) genMessageExtraSeq(channelID string) int64 {
	return m.ctx.GenSeq(fmt.Sprintf("%s:%s", common.MessageExtraSeqKey, channelID))
}
func (m *Message) genMessageReactionSeq(channelID string) int64 {
	return m.ctx.GenSeq(fmt.Sprintf("%s:%s", common.MessageReactionSeqKey, channelID))
}

// 消息偏移
func (m *Message) offset(c *wkhttp.Context) {
	var req struct {
		ChannelID   string `json:"channel_id"`
		ChannelType uint8  `json:"channel_type"`
		MessageSeq  uint32 `json:"message_seq"`
	}
	if err := c.BindJSON(&req); err != nil {
		m.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	channelOffsetM, err := m.channelOffsetDB.queryWithUIDAndChannel(c.GetLoginUID(), req.ChannelID, req.ChannelType)
	if err != nil {
		m.Error("查询频道偏移数据失败！", zap.Error(err))
		c.ResponseError(errors.New("查询频道偏移数据失败！"))
		return
	}
	if channelOffsetM != nil {
		if channelOffsetM.MessageSeq >= req.MessageSeq {
			c.ResponseOK()
			return
		}
	}

	err = m.channelOffsetDB.insertOrUpdate(&channelOffsetModel{
		UID:         c.GetLoginUID(),
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		MessageSeq:  req.MessageSeq,
	})
	if err != nil {
		m.Error("清除失败！", zap.Error(err))
		c.ResponseError(errors.New("清除失败！"))
		return
	}
	// 清除最近会话的未读数（这里不管有没有未读数都调用清除）
	err = m.ctx.IMClearConversationUnread(config.ClearConversationUnreadReq{
		UID:         c.GetLoginUID(),
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		MessageSeq:  req.MessageSeq,
		Unread:      0,
	})
	if err != nil {
		m.Error("清除最近会话未读数失败！", zap.Error(err), zap.String("uid", c.GetLoginUID()), zap.String("channelID", req.ChannelID), zap.Uint8("channelType", req.ChannelType))
	}

	// 发送清空红点的命令
	err = m.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   c.GetLoginUID(),
		ChannelType: common.ChannelTypePerson.Uint8(),
		CMD:         common.CMDConversationUnreadClear,
		Param: map[string]interface{}{
			"channel_id":   req.ChannelID,
			"channel_type": req.ChannelType,
			"unread":       0,
		},
	})
	if err != nil {
		m.Error("命令发送失败！", zap.String("cmd", common.CMDConversationUnreadClear), zap.String("uid", c.GetLoginUID()), zap.String("channelID", req.ChannelID), zap.Uint8("channelType", req.ChannelType))
	}

	c.ResponseOK()
}

// 是否有撤回的权限
func (m *Message) hasRevokePermission(messageM *messageModel, loginUID string) (bool, error) {
	if messageM.FromUID == "" { // 没有fromUID的消息一般是命令类的消息，不被允许撤回
		return false, nil
	}
	if messageM.FromUID == loginUID { // 自己发的消息允许被撤回
		return true, nil
	}
	if messageM.ChannelType == common.ChannelTypeGroup.Uint8() { // 管理者或创建者可以撤回其他成员的消息
		creatorOrManager, err := m.groupService.IsCreatorOrManager(messageM.ChannelID, loginUID)
		if err != nil {
			return false, err
		}
		return creatorOrManager, nil
	}

	return false, nil
}

func (m *Message) cancelMentionReminderIfNeed(message *messageModel) {
	setting := config.SettingFromUint8(message.Setting)
	//  如果撤回的是@消息，需要取消提醒
	if !setting.Signal {
		var payloadMap map[string]interface{}
		if err := util.ReadJsonByByte(message.Payload, &payloadMap); err != nil {
			m.Warn("解码消息内容失败！", zap.Error(err))
		}
		if payloadMap != nil {
			if m.hasMention(payloadMap) {
				all, uids := m.getMention(payloadMap)
				if all {
					version := m.ctx.GenSeq(common.RemindersKey)
					err := m.remindersDB.deleteWithChannel(message.ChannelID, message.ChannelType, message.MessageID, version)
					if err != nil {
						m.Error("删除提醒项失败！", zap.Error(err))
					} else {
						err = m.ctx.SendCMD(config.MsgCMDReq{
							NoPersist:   true,
							ChannelID:   message.ChannelID,
							ChannelType: message.ChannelType,
							CMD:         common.CMDSyncReminders,
						})
						if err != nil {
							m.Error("发送cmd[CMDSyncReminders]失败！", zap.Error(err))
						}
					}
				} else if len(uids) > 0 {
					tx, _ := m.ctx.DB().Begin()
					defer func() {
						if err := recover(); err != nil {
							tx.RollbackUnlessCommitted()
							panic(err)
						}
					}()
					for _, uid := range uids {
						version := m.ctx.GenSeq(common.RemindersKey)
						err := m.remindersDB.deleteWithChannelAndUIDTx(message.ChannelID, message.ChannelType, uid, message.MessageID, version, tx)
						if err != nil {
							m.Error("删除用户提醒项失败！", zap.Error(err))
							tx.Rollback()
							return
						}
					}
					if err := tx.Commit(); err != nil {
						m.Error("提交事务失败！", zap.Error(err))
						tx.RollbackUnlessCommitted()
						return
					}
					err := m.ctx.SendCMD(config.MsgCMDReq{
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
	}
}

// 撤回消息
func (m *Message) revoke(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	messageID := c.Query("message_id")
	clientMsgNo := c.Query("client_msg_no") // TODO：后续版本不再使用messageID撤回，使用client_msg_no撤回，因为存在重试消息，clientMsgNo一样 但是messageID不一样
	channelID := c.Query("channel_id")
	channelType := c.Query("channel_type")

	if strings.TrimSpace(clientMsgNo) == "" {
		c.ResponseError(errors.New("撤回主键参数错误！"))
		return
	}

	//删除消息
	channelTypeI, _ := strconv.ParseUint(channelType, 10, 64)

	fakeChannelID := channelID
	if uint8(channelTypeI) == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(channelID, c.GetLoginUID())
	}

	var messageIDs = []string{}
	var err error

	if clientMsgNo != "" {
		messages, err := m.db.queryMessagesWithChannelClientMsgNo(fakeChannelID, uint8(channelTypeI), clientMsgNo)
		if err != nil {
			m.Error("撤回失败！", zap.String("fakeChannelID", fakeChannelID), zap.String("clientMsgNo", clientMsgNo), zap.String("loginUID", c.GetLoginUID()))
			c.ResponseErrorf("查询消息失败！", err)
			return
		}
		if len(messages) == 0 {
			c.ResponseError(errors.New("撤回失败！"))
			return
		}
		var message *messageModel
		if len(messages) > 0 {
			message = messages[0]
			for _, message := range messages {
				messageIDs = append(messageIDs, fmt.Sprintf("%d", message.MessageID))
			}
		}
		if message != nil {
			allow, err := m.hasRevokePermission(message, c.GetLoginUID())
			if err != nil {
				m.Error("权限判断失败！", zap.Error(err))
				c.ResponseError(errors.New("权限判断失败！"))
				return
			}
			if !allow {
				c.ResponseError(errors.New("无权限撤回此消息！"))
				return
			}

			m.cancelMentionReminderIfNeed(message)

		}
	}

	if len(messageIDs) == 0 {
		messageIDs = append(messageIDs, messageID)
	}

	tx, _ := m.db.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	for _, msgID := range messageIDs {
		version := m.genMessageExtraSeq(fakeChannelID)
		err = m.messageExtraDB.insertOrUpdateRevokeTx(&messageExtraModel{
			MessageID:   msgID,
			ChannelID:   fakeChannelID,
			ChannelType: uint8(channelTypeI),
			Revoke:      1,
			Version:     version,
			Revoker:     loginUID,
		}, tx)
		if err != nil {
			tx.Rollback()
			c.ResponseErrorf("更新消息为撤回状态失败！", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		c.ResponseErrorf("事务提交失败！", err)
		return
	}

	// err = m.ctx.SendCMD(config.MsgCMDReq{
	// 	NoPersist:   true,
	// 	ChannelID:   channelID,
	// 	ChannelType: uint8(channelTypeI),
	// 	FromUID:     loginUID,
	// 	CMD:         common.CMDSyncMessageExtra,
	// })
	// if err != nil {
	// 	c.ResponseErrorf("发送同步命令失败！", err)
	// 	return
	// }

	for _, msgID := range messageIDs {
		messageIDI, _ := strconv.ParseInt(msgID, 10, 64)
		// 发给指定频道
		err = m.ctx.SendRevoke(&config.MsgRevokeReq{
			Operator:     loginUID,
			OperatorName: c.GetLoginName(),
			FromUID:      loginUID,
			ChannelID:    channelID,
			ChannelType:  uint8(channelTypeI),
			MessageID:    messageIDI,
		})
		if err != nil {
			m.Error("发送撤回消息失败！", zap.Error(err))
			c.ResponseError(errors.New("发送撤回消息失败！"))
			return
		}
	}

	c.ResponseOK()

}

// 同步违禁词
func (m *Message) synccProhibitWords(c *wkhttp.Context) {
	version := c.Query("version")
	maxVersion, _ := strconv.ParseInt(version, 10, 64)
	list, err := m.db.queryProhibitWordsWithVersion(maxVersion)
	if err != nil {
		m.Error("同步违禁词错误", zap.Error(err))
		c.ResponseError(errors.New("同步违禁词错误"))
		return
	}
	result := make([]*ProhibitWordResp, 0)
	if len(list) > 0 {
		for _, word := range list {
			result = append(result, &ProhibitWordResp{
				Id:        word.Id,
				Content:   word.Content,
				IsDeleted: word.IsDeleted,
				CreatedAt: word.CreatedAt.String(),
			})
		}
	}
	c.Response(result)
}

// 同步敏感词
func (m *Message) syncSensitiveWords(c *wkhttp.Context) {
	type resp struct {
		Tips    string   `json:"tips"`
		List    []string `json:"list"`
		Version int64    `json:"version"`
	}
	reqVersion, _ := strconv.ParseInt(c.Query("version"), 10, 64)
	resultList := make([]string, 0)
	tips := ""
	if reqVersion < sensitiveWordsVersion {
		resultList = sensitive_words
		tips = "涉及私下交易、转账等资金问题，谨慎对待，谨防上当受骗，点击标题栏头像可投诉！"
	}
	c.Response(&resp{
		Tips:    tips,
		List:    resultList,
		Version: sensitiveWordsVersion,
	})
}

// // 接受IM的消息
// func (m *Message) notify(c *wkhttp.Context) {
// 	data, err := c.GetRawData()
// 	if err != nil {
// 		m.Error("notify读取数据失败！", zap.Error(err))
// 		c.ResponseError(err)
// 		return
// 	}
// 	var msgResps []msgResp
// 	err = util.ReadJsonByByte(data, &msgResps)
// 	if err != nil {
// 		m.Error("读取消息数据失败！", zap.Error(err))
// 		c.ResponseError(err)
// 		return
// 	}
// 	tx, _ := m.db.session.Begin()
// 	defer func() {
// 		if err := recover(); err != nil {
// 			tx.Rollback()
// 			panic(err)
// 		}
// 	}()
// 	messageIDS := make([]string, 0, len(msgResps))
// 	for _, msgResp := range msgResps {
// 		messageIDS = append(messageIDS, strconv.FormatUint(msgResp.MessageID, 10))
// 		messageModel := msgResp.ToModel()
// 		err = m.db.InsertTx(messageModel, tx)
// 		if err != nil {
// 			tx.Rollback()
// 			m.Error("添加消息失败！", zap.Any("msg", msgResp), zap.Error(err))
// 			c.ResponseError(err)
// 			return
// 		}
// 	}
// 	if err := tx.Commit(); err != nil {
// 		tx.Rollback()
// 		m.Error("提交事务失败！", zap.Error(err))
// 		c.ResponseError(err)
// 		return
// 	}
// 	c.Response(messageIDS)
// }

// ---------- vo ----------

type syncChannelMessageResp struct {
	StartMessageSeq uint32          `json:"start_message_seq"` // 开始序列号
	EndMessageSeq   uint32          `json:"end_message_seq"`   // 结束序列号
	PullMode        config.PullMode `json:"pull_mode"`         // 拉取模式
	More            int             `json:"more"`              // 是否还有更多 1.是 0.否
	Messages        []*MsgSyncResp  `json:"messages"`          // 消息数据
}

func newSyncChannelMessageResp(resp *config.SyncChannelMessageResp, loginUID string, deviceUUID string, channelID string, channelType uint8, messageExtraDB *messageExtraDB, messageUserExtraDB *messageUserExtraDB, messageReactionDB *messageReactionDB, channelOffsetDB *channelOffsetDB, deviceOffsetDB *deviceOffsetDB) *syncChannelMessageResp {
	messages := make([]*MsgSyncResp, 0, len(resp.Messages))
	if len(resp.Messages) > 0 {
		messageIDs := make([]string, 0, len(resp.Messages))
		for _, message := range resp.Messages {
			messageIDs = append(messageIDs, fmt.Sprintf("%d", message.MessageID))
		}

		// 消息全局扩张
		messageExtras, err := messageExtraDB.queryWithMessageIDs(messageIDs, loginUID)
		if err != nil {
			log.Error("查询消息扩展字段失败！", zap.Error(err))
		}
		messageExtraMap := map[string]*messageExtraDetailModel{}
		if len(messageExtras) > 0 {
			for _, messageExtra := range messageExtras {
				messageExtraMap[messageExtra.MessageID] = messageExtra
			}
		}

		// 消息用户扩张
		messageUserExtras, err := messageUserExtraDB.queryWithMessageIDsAndUID(messageIDs, loginUID)
		if err != nil {
			log.Error("查询用户消息扩展字段失败！", zap.Error(err))
		}
		messageUserExtraMap := map[string]*messageUserExtraModel{}
		if len(messageUserExtras) > 0 {
			for _, messageUserExtraM := range messageUserExtras {
				messageUserExtraMap[messageUserExtraM.MessageID] = messageUserExtraM
			}
		}

		// 查询消息回应
		messageReaction, err := messageReactionDB.queryWithMessageIDs(messageIDs)
		if err != nil {
			log.Error("查询消息回应数据错误", zap.Error(err))
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

		// 频道偏移
		channelOffsetM, err := channelOffsetDB.queryWithUIDAndChannel(loginUID, channelID, channelType)
		if err != nil {
			log.Error("查询频道偏移量失败！", zap.Error(err))
		}

		// 设备偏移
		deviceLastMessageSeq, err := deviceOffsetDB.queryMessageSeq(loginUID, deviceUUID, channelID, channelType)
		if err != nil {
			log.Error("查询设备消息偏移量失败！", zap.Error(err))
		}
		for _, message := range resp.Messages {
			if channelOffsetM != nil && message.MessageSeq <= channelOffsetM.MessageSeq {
				continue
			}
			if message.MessageSeq <= uint32(deviceLastMessageSeq) {
				continue
			}
			messageIDStr := strconv.FormatInt(message.MessageID, 10)
			messageExtra := messageExtraMap[messageIDStr]
			messageUserExtra := messageUserExtraMap[messageIDStr]
			msgResp := &MsgSyncResp{}
			msgResp.from(message, loginUID, messageExtra, messageUserExtra, messageReactionMap[strconv.FormatInt(message.MessageID, 10)])
			messages = append(messages, msgResp)
		}
	}
	return &syncChannelMessageResp{
		StartMessageSeq: resp.StartMessageSeq,
		EndMessageSeq:   resp.EndMessageSeq,
		PullMode:        resp.PullMode,
		Messages:        messages,
	}
}

// 消息头
type messageHeader struct {
	NoPersist int `json:"no_persist"` // 是否不持久化
	RedDot    int `json:"red_dot"`    // 是否显示红点
	SyncOnce  int `json:"sync_once"`  // 此消息只被同步或被消费一次
}

type syncReq struct {
	MaxMessageSeq uint32 `json:"max_message_seq"` // 客户端最大消息序列号
	Limit         int    `json:"limit"`           // 消息数量限制
	ChannelID     string `json:"channel_id"`      // 频道ID
	ChannelType   uint8  `json:"channel_type"`    // 频道类型
	Reverse       int    `json:"reverse"`         // 是否倒序
	Offset        int64  `json:"offset"`          // 偏移量
}

// type msgResp struct {
// 	MessageID   uint64 `json:"message_id"`   // 服务端的消息ID(全局唯一)
// 	FromUID     string `json:"from_uid"`     // 发送者UID
// 	ChannelID   string `json:"channel_id"`   // 频道ID
// 	ChannelType uint8  `json:"channel_type"` // 频道类型
// 	Timestamp   int64  `json:"timestamp"`    // 服务器消息时间戳(10位，到秒)
// 	Payload     []byte `json:"payload"`      // 消息内容
// }

// func (m msgResp) ToModel() *messageModel {
// 	var payloadMap map[string]interface{}
// 	err := util.ReadJsonByByte(m.Payload, &payloadMap)
// 	if err != nil {
// 		log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(m.Payload)))
// 	}
// 	contentType := 0
// 	if payloadMap != nil {
// 		if payloadMap["type"] != nil {
// 			contentTypeInt64, _ := payloadMap["type"].(json.Number).Int64()
// 			contentType = int(contentTypeInt64)
// 		}
// 		// if payloadMap["content"] != nil {
// 		// 	keyword = payloadMap["content"].(string)
// 		// }
// 	}
// 	return &messageModel{
// 		MessageID:   int64(m.MessageID),
// 		FromUID:     m.FromUID,
// 		ChannelID:   m.ChannelID,
// 		ChannelType: m.ChannelType,
// 		Timestamp:   m.Timestamp,
// 		Payload:     m.Payload,
// 		Type:        contentType,
// 	}
// }

type replyMsgSyncResp struct {
	Root     *config.MessageResp   `json:"root"`
	Messages []*config.MessageResp `json:"messages"`
}

// MgSyncResp 消息同步请求
type MsgSyncResp struct {
	Header        messageHeader          `json:"header"`                    // 消息头部
	Setting       uint8                  `json:"setting"`                   // 设置
	MessageID     int64                  `json:"message_id"`                // 服务端的消息ID(全局唯一)
	MessageIDStr  string                 `json:"message_idstr"`             // 服务端的消息ID(全局唯一)字符串形式
	MessageSeq    uint32                 `json:"message_seq"`               // 消息序列号 （用户唯一，有序递增）
	ClientMsgNo   string                 `json:"client_msg_no"`             // 客户端消息唯一编号
	StreamNo      string                 `json:"stream_no,omitempty"`       // 流编号
	FromUID       string                 `json:"from_uid"`                  // 发送者UID
	ToUID         string                 `json:"to_uid,omitempty"`          // 接受者uid
	ChannelID     string                 `json:"channel_id"`                // 频道ID
	ChannelType   uint8                  `json:"channel_type"`              // 频道类型
	Expire        uint32                 `json:"expire,omitempty"`          // expire
	Timestamp     int32                  `json:"timestamp"`                 // 服务器消息时间戳(10位，到秒)
	Payload       map[string]interface{} `json:"payload"`                   // 消息内容
	SignalPayload string                 `json:"signal_payload"`            // signal 加密后的payload base64编码,TODO: 这里为了兼容没加密的版本，所以新用SignalPayload字段
	ReplyCount    int                    `json:"reply_count,omitempty"`     // 回复集合
	ReplyCountSeq string                 `json:"reply_count_seq,omitempty"` // 回复数量seq
	ReplySeq      string                 `json:"reply_seq,omitempty"`       // 回复seq
	Reactions     []*reactionSimpleResp  `json:"reactions,omitempty"`       // 回应数据
	IsDeleted     int                    `json:"is_deleted"`                // 是否已删除
	VoiceStatus   int                    `json:"voice_status,omitempty"`    // 语音状态 0.未读 1.已读
	Streams       []*streamItemResp      `json:"streams,omitempty"`         // 流数据
	// ---------- 旧字段 这些字段都放到MessageExtra对象里了 ----------
	Readed       int    `json:"readed"`                 // 是否已读（针对于自己）
	Revoke       int    `json:"revoke,omitempty"`       // 是否撤回
	Revoker      string `json:"revoker,omitempty"`      // 消息撤回者
	ReadedCount  int    `json:"readed_count,omitempty"` // 已读数量
	UnreadCount  int    `json:"unread_count,omitempty"` // 未读数量
	ExtraVersion int64  `json:"extra_version"`          // 扩展数据版本号

	// 消息扩展字段
	MessageExtra *messageExtraResp `json:"message_extra,omitempty"` // 消息扩展

}

func (m *MsgSyncResp) from(msgResp *config.MessageResp, loginUID string, messageExtraM *messageExtraDetailModel, messageUserExtraM *messageUserExtraModel, reactionModels []*reactionModel) {
	m.Header.NoPersist = msgResp.Header.NoPersist
	m.Header.RedDot = msgResp.Header.RedDot
	m.Header.SyncOnce = msgResp.Header.SyncOnce
	m.Setting = msgResp.Setting
	m.MessageID = msgResp.MessageID
	m.MessageIDStr = strconv.FormatInt(msgResp.MessageID, 10)
	m.MessageSeq = msgResp.MessageSeq
	m.ClientMsgNo = msgResp.ClientMsgNo
	m.StreamNo = msgResp.StreamNo
	m.FromUID = msgResp.FromUID
	m.ToUID = msgResp.ToUID
	m.ChannelID = msgResp.ChannelID
	m.ChannelType = msgResp.ChannelType
	m.Expire = msgResp.Expire
	m.Timestamp = msgResp.Timestamp
	if messageExtraM != nil {
		// TODO: 后续这些字段可以废除了 都放MessageExtra对象里了
		m.IsDeleted = messageExtraM.IsDeleted
		m.Revoke = messageExtraM.Revoke
		m.Revoker = messageExtraM.Revoker
		m.ReadedCount = messageExtraM.ReadedCount
		m.Readed = messageExtraM.Readed
		m.ExtraVersion = messageExtraM.Version

		m.MessageExtra = newMessageExtraResp(messageExtraM)
	}

	setting := config.SettingFromUint8(msgResp.Setting)
	var payloadMap map[string]interface{}
	if setting.Signal {
		m.SignalPayload = base64.StdEncoding.EncodeToString(msgResp.Payload)
		payloadMap = map[string]interface{}{
			"type": common.SignalError.Int(),
		}
	} else {
		err := util.ReadJsonByByte(msgResp.Payload, &payloadMap)
		if err != nil {
			log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(msgResp.Payload)))
		}
		if len(payloadMap) > 0 {
			visibles := payloadMap["visibles"]
			if visibles != nil {
				visiblesArray := visibles.([]interface{})
				if len(visiblesArray) > 0 {
					m.IsDeleted = 1
					for _, limitUID := range visiblesArray {
						if limitUID == loginUID {
							m.IsDeleted = 0
						}
					}
				}
			}
		} else {
			payloadMap = map[string]interface{}{
				"type": common.ContentError.Int(),
			}
		}
	}

	if messageUserExtraM != nil {
		if m.IsDeleted == 0 {
			m.IsDeleted = messageUserExtraM.MessageIsDeleted
		}
		m.VoiceStatus = messageUserExtraM.VoiceReaded
	}

	if msgResp.Expire > 0 {
		if time.Now().Unix()-int64(msgResp.Expire) >= int64(msgResp.Timestamp) {
			m.IsDeleted = 1
		}
	}

	m.Payload = payloadMap

	msgReactionList := make([]*reactionSimpleResp, 0, len(reactionModels))
	if len(reactionModels) > 0 {
		for _, reaction := range reactionModels {
			msgReactionList = append(msgReactionList, &reactionSimpleResp{
				UID:       reaction.UID,
				Name:      reaction.Name,
				Seq:       reaction.Seq,
				IsDeleted: reaction.IsDeleted,
				Emoji:     reaction.Emoji,
				CreatedAt: reaction.CreatedAt.String(),
			})
		}
	}
	m.Reactions = msgReactionList

	if len(msgResp.Streams) > 0 {
		streams := make([]*streamItemResp, 0, len(msgResp.Streams))
		for _, streamItem := range msgResp.Streams {
			streams = append(streams, newStreamItemResp(streamItem))
		}
		m.Streams = streams
	}

}

type streamItemResp struct {
	StreamSeq   uint32         `json:"stream_seq"`    // 流序号
	ClientMsgNo string         `json:"client_msg_no"` // 客户端消息唯一编号
	Blob        map[string]any `json:"blob"`          // 消息内容
}

func newStreamItemResp(streamItem *config.StreamItemResp) *streamItemResp {
	var blobMap map[string]any
	err := util.ReadJsonByByte(streamItem.Blob, &blobMap)
	if err != nil {
		log.Warn("blob不是json格式！", zap.Error(err), zap.String("blob", string(streamItem.Blob)))
	}
	return &streamItemResp{
		ClientMsgNo: streamItem.ClientMsgNo,
		StreamSeq:   streamItem.StreamSeq,
		Blob:        blobMap,
	}
}

// 回应返回
type reactionResp struct {
	MessageID   string `json:"message_id"`   // 消息编号
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
	Seq         int64  `json:"seq"`          // 回复序列号
	UID         string `json:"uid"`          // 回应用户ID
	Name        string `json:"name"`         // 回应用户名
	Emoji       string `json:"emoji"`        // 回应的emoji
	IsDeleted   int    `json:"is_deleted"`   // 是否删除
	CreatedAt   string `json:"created_at"`
}

// 回应返回
type reactionSimpleResp struct {
	Seq       int64  `json:"seq"`        // 回复序列号
	UID       string `json:"uid"`        // 回应用户ID
	Name      string `json:"name"`       // 回应用户名
	Emoji     string `json:"emoji"`      // 回应的emoji
	IsDeleted int    `json:"is_deleted"` // 是否删除
	CreatedAt string `json:"created_at"`
}

// type userResp struct {
// 	UID       string `json:"uid"`
// 	Name      string `json:"name"`
// 	IsDeleted int    `json:"is_deleted"`
// }

type syncTotalResp struct {
	MessageID   string `json:"message_id"`   // 消息唯一ID
	Seq         string `json:"seq"`          // 回复序列号
	ChannelID   string `json:"channel_id"`   // 频道唯一ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
	Count       int    `json:"count"`        // 回复数量
}

type messageExtraResp struct {
	MessageID       int64                  `json:"message_id"`
	MessageIDStr    string                 `json:"message_id_str"`
	Revoke          int                    `json:"revoke,omitempty"`
	Revoker         string                 `json:"revoker,omitempty"`
	VoiceStatus     int                    `json:"voice_status,omitempty"`
	Readed          int                    `json:"readed,omitempty"`            // 是否已读（针对于自己）
	ReadedCount     int                    `json:"readed_count,omitempty"`      // 已读数量
	ReadedAt        int64                  `json:"readed_at,omitempty"`         // 已读时间
	IsMutualDeleted int                    `json:"is_mutual_deleted,omitempty"` // 双向删除
	ContentEdit     map[string]interface{} `json:"content_edit,omitempty"`      // 编辑后的正文
	EditedAt        int                    `json:"edited_at,omitempty"`         // 编辑时间 例如 12:23
	ExtraVersion    int64                  `json:"extra_version"`               // 数据版本
}

func newMessageExtraResp(m *messageExtraDetailModel) *messageExtraResp {

	messageID, _ := strconv.ParseInt(m.MessageID, 10, 64)

	var contentEditMap map[string]interface{}
	if m.ContentEdit.String != "" {
		err := util.ReadJsonByByte([]byte(m.ContentEdit.String), &contentEditMap)
		if err != nil {
			log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(m.ContentEdit.String)))
		}
	}

	var readedAt int64 = 0
	if m.ReadedAt.Valid {
		readedAt = m.ReadedAt.Time.Unix()
	}

	return &messageExtraResp{
		MessageID:       messageID,
		MessageIDStr:    m.MessageID,
		Revoke:          m.Revoke,
		Revoker:         m.Revoker,
		Readed:          m.Readed,
		ReadedAt:        readedAt,
		ReadedCount:     m.ReadedCount,
		ContentEdit:     contentEditMap,
		EditedAt:        m.EditedAt,
		IsMutualDeleted: m.IsDeleted,
		ExtraVersion:    m.Version,
	}
}

type memberReceiptResp struct {
	UID  string `json:"uid"`  // 成员uid
	Name string `json:"name"` // 成员名称
}

type ProhibitWordResp struct {
	Id        int64  `json:"id"`
	Content   string `json:"content"`    // 违禁词
	IsDeleted int    `json:"is_deleted"` // 是否删除
	Version   int64  `json:"version"`    // 版本
	CreatedAt string `json:"created_at"` // 时间
}
