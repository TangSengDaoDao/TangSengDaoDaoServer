package user

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	chservice "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/channel/service"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/source"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Friend 好友
type Friend struct {
	ctx *config.Context
	log.Log
	db            *friendDB
	settingDB     *SettingDB
	userDB        *DB
	onlineService IOnlineService
	userService   IService
}

// NewFriend 创建
func NewFriend(ctx *config.Context) *Friend {
	f := &Friend{
		ctx:           ctx,
		Log:           log.NewTLog("Friend"),
		userDB:        NewDB(ctx),
		db:            newFriendDB(ctx),
		onlineService: NewOnlineService(ctx),
		settingDB:     NewSettingDB(ctx.DB()),
		userService:   NewService(ctx),
	}
	f.ctx.AddEventListener(event.FriendSure, f.handleFriendSure)
	f.ctx.AddEventListener(event.FriendDelete, f.handleDeleteFriend)
	return f
}

// Route 配置路由规则
func (f *Friend) Route(r *wkhttp.WKHttp) {
	friend := r.Group("/v1/friend", f.ctx.AuthMiddleware(r))
	{
		friend.POST("/apply", f.friendApply)           // 好友申请
		friend.GET("/apply", f.apply)                  // 好友申请列表
		friend.DELETE("/apply/:to_uid", f.deleteApply) // 删除好友申请
		friend.PUT("/refuse", f.refuseApply)           // 拒绝申请
		friend.POST("/sure", f.friendSure)             // 好友确认
		friend.GET("/sync", f.friendSync)              // 同步好友
		friend.GET("/search", f.friendSearch)          // 查询好友
		friend.PUT("/remark", f.remark)                //好友备注
	}
	friends := r.Group("/v1/friends", f.ctx.AuthMiddleware(r))
	{
		friends.DELETE("/:uid", f.delete) //删除好友
	}
}

// 通过或拒绝申请
func (f *Friend) refuseApply(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	toUid := c.Param("to_uid")
	if toUid == "" {
		c.ResponseError(errors.New("好友ID不能为空"))
		return
	}

	apply, err := f.db.queryApplyWithUidAndToUid(loginUID, toUid)
	if err != nil {
		f.Error("查询申请记录错误", zap.Error(err))
		c.ResponseError(errors.New("查询申请记录错误"))
		return
	}
	if apply == nil || apply.UID != loginUID {
		c.ResponseError(errors.New("申请记录不存在"))
		return
	}
	apply.Status = 2
	err = f.db.updateApply(apply)
	if err != nil {
		f.Error("修改申请记录错误", zap.Error(err))
		c.ResponseError(errors.New("修改申请记录错误"))
		return
	}
	c.ResponseOK()
}

// 删除好友申请记录
func (f *Friend) deleteApply(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	toUid := c.Param("to_uid")
	if toUid == "" {
		c.ResponseError(errors.New("id不能为空"))
		return
	}
	err := f.db.deleteApplyWithUidAndToUid(loginUID, toUid)
	if err != nil {
		f.Error("删除申请记录错误", zap.Error(err))
		c.ResponseError(errors.New("删除申请记录错误"))
		return
	}
	c.ResponseOK()
}

// 好友申请列表
func (f *Friend) apply(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	page := c.Query("page_index")
	size := c.Query("page_size")
	pageIndex, _ := strconv.Atoi(page)
	pageSize, _ := strconv.Atoi(size)
	applys, err := f.db.queryApplysWithPage(loginUID, uint64(pageSize), uint64(pageIndex))
	if err != nil {
		f.Error("查询好友申请列表错误", zap.Error(err))
		c.ResponseError(errors.New("查询好友申请列表错误"))
		return
	}
	list := make([]*friendApplyResp, 0)
	if len(applys) > 0 {
		for _, apply := range applys {
			list = append(list, &friendApplyResp{
				Id:        apply.Id,
				UID:       apply.UID,
				ToUID:     apply.ToUID,
				Remark:    apply.Remark,
				Status:    apply.Status,
				Token:     apply.Token,
				CreatedAt: apply.CreatedAt.String(),
			})
		}
	}
	c.Response(list)
}

// 删除好友
func (f *Friend) delete(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	uid := c.Param("uid")
	if uid == "" {
		c.ResponseError(errors.New("用户uid不能为空"))
		return
	}
	tx, err := f.ctx.DB().Begin()
	util.CheckErr(err)

	version := f.ctx.GenSeq(common.FriendSeqKey)
	// err = f.db.updateRelationshipTx(loginUID, uid, 1, 1, "", version, tx) // 不能删除sourceVercode 如果删除了 已有会话发起加好友会提示验证码不为空
	err = f.db.updateRelationship2Tx(loginUID, uid, 1, 1, version, tx)
	if err != nil {
		util.CheckErr(tx.Rollback())
		f.Error("删除好友错误", zap.Error(err))
		c.ResponseError(errors.New("删除好友错误"))
		return
	}
	err = f.db.updateAloneTx(uid, loginUID, 1, tx)
	if err != nil {
		util.CheckErr(tx.Rollback())
		f.Error("修改好友单项关系错误", zap.Error(err))
		c.ResponseError(errors.New("修改好友单项关系错误"))
		return
	}
	// 发布删除好友事件
	eventID, err := f.ctx.EventBegin(&wkevent.Data{
		Event: event.FriendDelete,
		Type:  wkevent.Message,
		Data: map[string]interface{}{
			"uid":    loginUID,
			"to_uid": uid,
		},
	}, tx)
	if err != nil {
		f.Error("发送删除好友事件失败", zap.Error(err))
		tx.Rollback()
		c.ResponseError(errors.New("发送删除好友事件失败"))
		return
	}
	userSetting, err := f.settingDB.querySettingByUIDAndToUID(loginUID, uid)
	if err != nil {
		tx.Rollback()
		f.Error("查询用户好友设置错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户好友设置错误"))
		return
	}
	if userSetting != nil {
		userSetting.ChatPwdOn = 0
		userSetting.Top = 0
		userSetting.Mute = 0
		userSetting.Receipt = 1
		userSetting.Screenshot = 1
		userSetting.RevokeRemind = 0
		userSetting.Remark = ""
		userSetting.Flame = 0
		userSetting.FlameSecond = 0
		err := f.settingDB.updateUserSettingModelWithToUIDTx(userSetting, loginUID, uid, tx)
		if err != nil {
			tx.Rollback()
			f.Error("重置好友设置错误", zap.Error(err))
			c.ResponseError(errors.New("重置好友设置错误"))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		f.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	f.ctx.EventCommit(eventID)

	err = f.ctx.SendChannelUpdate(config.ChannelReq{
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
	}, config.ChannelReq{
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
	})
	if err != nil {
		f.Warn("发送频道更新命令失败！", zap.Error(err))
	}

	err = f.ctx.SendFriendDelete(&config.MsgFriendDeleteReq{
		FromUID: loginUID,
		ToUID:   uid,
	})
	if err != nil {
		f.Error("发送删除好友的cmd失败！", zap.Error(err))
	}

	c.ResponseOK()
}

// 好友申请
func (f *Friend) friendApply(c *wkhttp.Context) {
	fromUID := c.GetLoginUID()
	fromName := c.GetLoginName()

	var req applyReq
	if err := c.BindJSON(&req); err != nil {
		f.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	if fromUID == req.ToUID {
		c.ResponseError(errors.New("不能添加自己为好友！"))
		return
	}
	// 是否是好友
	isFriendLoginUser, err := f.db.IsFriend(fromUID, req.ToUID)
	if err != nil {
		f.Error("查询是否是好友失败！", zap.Error(err), zap.String("uid", fromUID), zap.String("toUid", req.ToUID))
		c.ResponseError(errors.New("查询是否是好友失败！"))
		return
	}
	isFriendToUser, err := f.db.IsFriend(req.ToUID, fromUID)
	if err != nil {
		f.Error("查询是否是好友失败！", zap.Error(err), zap.String("uid", fromUID), zap.String("toUid", req.ToUID))
		c.ResponseError(errors.New("查询是否是好友失败！"))
		return
	}
	if isFriendLoginUser && isFriendToUser {
		c.ResponseError(errors.New("已经是好友，不能再申请！"))
		return
	}

	toUser, err := f.userDB.QueryByUID(req.ToUID)
	if err != nil {
		f.Error("查询接收者用户信息失败！", zap.Error(err), zap.String("uid", fromUID))
		c.ResponseError(errors.New("查询用户信息失败！"))
		return
	}
	if toUser == nil || toUser.IsDestroy == 1 {
		f.Error("接收好友请求的用户不存在！", zap.String("to_uid", req.ToUID))
		c.ResponseError(errors.New("接收好友请求的用户不存在！"))
		return
	}
	if req.Vercode == "" {
		friend, err := f.db.queryWithUID(fromUID, req.ToUID)
		if err != nil {
			f.Error("查询好友信息错误", zap.String("to_uid", req.ToUID))
			c.ResponseError(errors.New("查询好友信息错误"))
			return
		}
		if friend == nil {
			f.Error("好友信息不存在", zap.String("to_uid", req.ToUID))
			c.ResponseError(errors.New("好友信息不存在"))
			return
		}
		if friend.SourceVercode == "" {
			f.Error("验证码不能为空", zap.String("to_uid", req.ToUID))
			c.ResponseError(errors.New("验证码不能为空"))
			return
		}
		req.Vercode = friend.SourceVercode
	}

	//验证code是否有效
	err = source.CheckRequestAddFriendCode(req.Vercode, fromUID)
	if err != nil {
		c.ResponseError(err)
		return
	}
	// 设置token
	token := util.GenerUUID()

	err = f.ctx.Cache().SetAndExpire(f.ctx.GetConfig().Cache.FriendApplyTokenCachePrefix+token+toUser.UID, util.ToJson(map[string]interface{}{
		"from_uid": fromUID,
		"vercode":  req.Vercode,
		"remark":   req.Remark,
	}), f.ctx.GetConfig().Cache.FriendApplyExpire)
	if err != nil {
		f.Error("设置申请token失败！", zap.Error(err))
		c.ResponseError(errors.New("设置申请token失败！"))
		return
	}
	// 查询好友申请记录
	apply, err := f.db.queryApplyWithUidAndToUid(req.ToUID, fromUID)
	if err != nil {
		f.Error("查询好友申请记录错误", zap.String("to_uid", req.ToUID))
		c.ResponseError(errors.New("查询好友申请记录错误"))
		return
	}
	// 查询用户红点
	userRedDot, err := f.userDB.queryUserRedDot(req.ToUID, UserRedDotCategoryFriendApply)
	if err != nil {
		f.Error("查询用户通讯录红点信息错误", zap.String("to_uid", req.ToUID))
		c.ResponseError(errors.New("查询用户通讯录红点信息错误"))
		return
	}
	tx, _ := f.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	if apply == nil {
		err = f.db.insertApplyTx(&FriendApplyModel{
			Status: 0,
			UID:    req.ToUID,
			ToUID:  fromUID,
			Remark: req.Remark,
			Token:  token,
		}, tx)
		if err != nil {
			tx.Rollback()
			f.Error("新增好友申请记录错误", zap.String("to_uid", req.ToUID))
			c.ResponseError(errors.New("新增好友申请记录错误"))
			return
		}
	} else {
		apply.Status = 0
		err = f.db.updateApplyTx(apply, tx)
		if err != nil {
			tx.Rollback()
			f.Error("修改好友申请记录错误", zap.String("to_uid", req.ToUID))
			c.ResponseError(errors.New("修改好友申请记录错误"))
			return
		}
	}
	// 新增红点
	if userRedDot == nil {
		err = f.userDB.insertUserRedDotTx(&userRedDotModel{
			UID:      req.ToUID,
			Count:    1,
			IsDot:    0,
			Category: UserRedDotCategoryFriendApply,
		}, tx)
		if err != nil {
			tx.Rollback()
			f.Error("新增用户通讯录红点信息错误", zap.String("to_uid", req.ToUID))
			c.ResponseError(errors.New("新增用户通讯录红点信息错误"))
			return
		}
	} else {
		userRedDot.Count++
		err = f.userDB.updateUserRedDotTx(userRedDot, tx)
		if err != nil {
			tx.Rollback()
			f.Error("修改用户通讯录红点信息错误", zap.String("to_uid", req.ToUID))
			c.ResponseError(errors.New("修改用户通讯录红点信息错误"))
			return
		}
	}
	if err = tx.Commit(); err != nil {
		tx.Rollback()
		f.Error("提交事物错误", zap.Error(err))
		c.ResponseError(errors.New("提交事物错误"))
		return
	}
	// 发送消息
	err = f.ctx.SendCMD(config.MsgCMDReq{
		CMD:         common.CMDFriendRequest,
		ChannelID:   toUser.UID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		Param: map[string]interface{}{
			"apply_uid":  fromUID,
			"apply_name": fromName,
			"to_uid":     toUser.UID,
			"remark":     req.Remark,
			"token":      token,
		},
	})
	if err != nil {
		f.Error("发送好友申请失败！", zap.Error(err))
		c.ResponseError(errors.New("发送好友申请失败！"))
		return
	}
	c.ResponseOK()
}

// 确认好友
func (f *Friend) friendSure(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	name := c.GetLoginName()
	var req sureReq
	if err := c.BindJSON(&req); err != nil {
		f.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	key := f.ctx.GetConfig().Cache.FriendApplyTokenCachePrefix + req.Token + loginUID
	tokenVaule, err := f.ctx.Cache().Get(key) // 获取申请人的uid
	if err != nil {
		f.Error("获取好友申请token的信息失败！", zap.Error(err), zap.String("key", key))
		c.ResponseError(errors.New("获取好友申请token的信息失败！"))
		return
	}
	valueMap, err := util.JsonToMap(tokenVaule)
	if err != nil {
		f.Error("获取token信息错误", zap.Error(err), zap.String("key", key))
		c.ResponseError(errors.New("获取token信息错误"))
		return
	}

	loginUser, err := f.userDB.QueryByUID(loginUID)
	if err != nil {
		f.Error("查询用户信息失败！", zap.Error(err), zap.String("uid", loginUID))
		c.ResponseError(errors.New("查询用户信息失败！"))
		return
	}
	if loginUser == nil || loginUser.IsDestroy == 1 {
		f.Error("当前用户不存在或已注销！", zap.String("uid", loginUID))
		c.ResponseError(errors.New("当前用户不存在或已注销！"))
		return
	}

	applyUID := valueMap["from_uid"].(string)
	vercode := valueMap["vercode"].(string)
	remark := ""
	if valueMap["remark"] != nil {
		remark = valueMap["remark"].(string)
	}

	applyUser, err := f.userDB.QueryByUID(applyUID)
	if err != nil {
		f.Error("查询申请人用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询申请人用户信息失败！"))
		return
	}
	if applyUser == nil || applyUser.IsDestroy == 1 {
		f.Error("申请人不存在或已注销！", zap.String("uid", applyUID))
		c.ResponseError(errors.New("申请人不存在"))
		return
	}
	if remark == "" {
		if applyUser != nil {
			remark = fmt.Sprintf("我是%s", applyUser.Name)
		}
	}
	if strings.TrimSpace(applyUID) == "" || strings.TrimSpace(vercode) == "" {
		c.ResponseError(errors.New("好友申请无效或已过期！"))
		return
	}
	channelServiceObj := register.GetService(ChannelServiceName)
	var channelService chservice.IService
	if channelServiceObj != nil {
		channelService = channelServiceObj.(chservice.IService)
	}
	if channelService != nil {
		if applyUser.MsgExpireSecond > 0 {
			err = channelService.CreateOrUpdateMsgAutoDelete(common.GetFakeChannelIDWith(applyUID, loginUID), common.ChannelTypePerson.Uint8(), applyUser.MsgExpireSecond)
			if err != nil {
				f.Warn("设置消息自动删除失败", zap.Error(err))
			}
		}
	}
	// 是否是好友
	applyFriendModel, err := f.db.queryWithUID(loginUID, applyUID)
	if err != nil {
		f.Error("查询是否是好友失败！", zap.Error(err), zap.String("uid", loginUID), zap.String("toUid", applyUID))
		c.ResponseError(errors.New("查询是否是好友失败！"))
		return
	}
	// 添加好友到数据库
	tx, _ := f.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	version := f.ctx.GenSeq(common.FriendSeqKey)
	if applyFriendModel == nil {
		// 验证code
		err = source.CheckSource(vercode)
		if err != nil {
			c.ResponseError(err)
			return
		}

		util.CheckErr(err)
		err = f.db.InsertTx(&FriendModel{
			UID:           loginUID,
			ToUID:         applyUID,
			Version:       version,
			Initiator:     0,
			IsAlone:       0,
			Vercode:       fmt.Sprintf("%s@%d", util.GenerUUID(), common.Friend),
			SourceVercode: vercode,
		}, tx)
		if err != nil {
			util.CheckErr(tx.Rollback())
			c.ResponseError(errors.New("添加好友失败！"))
			return
		}
	} else {
		err = f.db.updateRelationshipTx(loginUID, applyUID, 0, 0, vercode, version, tx)
		if err != nil {
			util.CheckErr(tx.Rollback())
			c.ResponseError(errors.New("修改好友关系失败"))
			return
		}
	}
	// 是否是好友
	loginFriendModel, err := f.db.queryWithUID(applyUID, loginUID)
	//loginIsFriend, err := f.db.IsFriend(applyUID, loginUID)
	if err != nil {
		util.CheckErr(tx.Rollback())
		f.Error("查询被添加者是否是好友失败！", zap.Error(err), zap.String("uid", loginUID), zap.String("toUid", applyUID))
		c.ResponseError(errors.New("查询被添加者是否是好友失败！"))
		return
	}
	if loginFriendModel == nil {
		err = f.db.InsertTx(&FriendModel{
			UID:           applyUID,
			ToUID:         loginUID,
			Version:       version,
			Initiator:     1,
			IsAlone:       0,
			Vercode:       fmt.Sprintf("%s@%d", util.GenerUUID(), common.Friend),
			SourceVercode: vercode,
		}, tx)
		if err != nil {
			util.CheckErr(tx.Rollback())
			c.ResponseError(errors.New("添加好友失败！"))
			return
		}
	} else {
		err = f.db.updateRelationshipTx(applyUID, loginUID, 0, 0, vercode, version, tx)
		if err != nil {
			util.CheckErr(tx.Rollback())
			c.ResponseError(errors.New("修改好友关系失败"))
			return
		}
	}
	// 发布好友确认事件
	eventID, err := f.ctx.EventBegin(&wkevent.Data{
		Event: event.FriendSure,
		Type:  wkevent.None,
		Data: map[string]interface{}{
			"uid":    loginUID,
			"to_uid": applyUID,
		},
	}, tx)
	if err != nil {
		f.Error("发送好友确认事件失败", zap.Error(err))
		tx.Rollback()
		c.ResponseError(errors.New("发送好友确认事件失败"))
		return
	}
	// 查询好友申请记录
	apply, err := f.db.queryApplyWithUidAndToUid(loginUID, applyUID)
	if err != nil {
		f.Error("查询好友申请记录错误", zap.Error(err))
		tx.Rollback()
		c.ResponseError(errors.New("查询好友申请记录错误"))
		return
	}
	if apply != nil {
		apply.Status = 1
		err = f.db.updateApplyTx(apply, tx)
		if err != nil {
			f.Error("修改好友申请记录错误", zap.Error(err))
			tx.Rollback()
			c.ResponseError(errors.New("修改好友申请记录错误"))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		f.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	f.ctx.EventCommit(eventID)

	// 发送确认消息给对方
	err = f.ctx.SendCMD(config.MsgCMDReq{
		CMD:         common.CMDFriendAccept,
		Subscribers: []string{applyUID, loginUID},
		Param: map[string]interface{}{
			"to_uid":    applyUID,
			"from_uid":  loginUID,
			"from_name": name,
		},
	})
	if err != nil {
		f.Error("发送消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送消息失败！"))
		return
	}
	payload := []byte(util.ToJson(map[string]interface{}{
		"content": "我们已经是好友了，可以愉快的聊天了！",
		"type":    common.Tip,
	}))

	err = f.ctx.SendMessage(&config.MsgSendReq{
		FromUID:     loginUID,
		ChannelID:   applyUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		Payload:     payload,
		Header: config.MsgHeader{
			RedDot: 1,
		},
	})
	if err != nil {
		f.Error("发送通过好友请求消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送通过好友请求消息失败！"))
		return
	}

	payload = []byte(util.ToJson(map[string]interface{}{
		"content": remark,
		"type":    common.Text,
	}))

	err = f.ctx.SendMessage(&config.MsgSendReq{
		FromUID:     applyUID,
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		Payload:     payload,
		Header: config.MsgHeader{
			RedDot: 1,
		},
	})
	if err != nil {
		f.Error("发送接受好友请求消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送接受好友请求消息失败！"))
		return
	}

	err = f.ctx.Cache().Delete(key)
	if err != nil {
		f.Error("删除缓存数据错误", zap.Error(err))
		c.ResponseError(errors.New("删除缓存数据错误"))
		return
	}
	c.ResponseOK()
}

// 同步好友
func (f *Friend) friendSync(c *wkhttp.Context) {
	uid := c.MustGet("uid").(string)
	limit, _ := strconv.ParseUint(c.Query("limit"), 10, 64)
	if limit <= 0 {
		limit = 1000
	}
	version, _ := strconv.ParseInt(c.Query("version"), 10, 64)
	apiVersion, _ := strconv.ParseInt(c.Query("api_version"), 10, 64)
	var friends []*FriendModel
	var err error
	// 同步好友
	if apiVersion == 0 {
		c.ResponseError(errors.New("旧API已被废弃"))
		return
	} else {
		friends, err = f.db.SyncFriends(version, uid, limit)
		if err != nil {
			f.Error("同步好友信息错误！", zap.Error(err))
			c.ResponseError(errors.New("同步好友信息错误！"))
			return
		}
	}

	friendUIDs := make([]string, 0, len(friends))
	if len(friends) > 0 {
		for _, f := range friends {
			friendUIDs = append(friendUIDs, f.ToUID)
		}
	}
	userDetails, err := f.userService.GetUserDetails(friendUIDs, c.GetLoginUID())
	if err != nil {
		f.Error("获取用户详情失败！", zap.Error(err))
		c.ResponseError(errors.New("获取用户详情失败！"))
		return
	}
	userDetailMap := map[string]*UserDetailResp{}
	if len(userDetails) > 0 {
		for _, userDetail := range userDetails {
			userDetailMap[userDetail.UID] = userDetail
		}
	}
	resps := make([]*friendResp, 0)
	if len(friends) > 0 {
		for _, f := range friends {
			resp := &friendResp{}
			resp.IsDeleted = f.IsDeleted
			resp.Version = f.Version
			resp.Vercode = f.Vercode
			userDetail := userDetailMap[f.ToUID]
			if userDetail != nil {
				resp.UserDetailResp = *userDetail
			}
			resps = append(resps, resp)
		}
	}
	c.JSON(http.StatusOK, resps)
}

func (f *Friend) friendSearch(c *wkhttp.Context) {
	uid := c.MustGet("uid").(string)
	keyword := c.Query("keyword")
	friends, err := f.db.QueryFriendsWithKeyword(uid, keyword)
	if err != nil {
		f.Error("查询好友数据失败！", zap.Error(err))
		c.ResponseError(errors.New("查询好友数据失败！"))
		return
	}
	resps := make([]*friendResp, 0)
	if len(friends) > 0 {
		for _, f := range friends {
			resp := &friendResp{}
			blacklist := 1
			if f.Blacklist == 1 {
				blacklist = 2
			}
			resp.From(f, blacklist, 0)
			resps = append(resps, resp)
		}
	}
	c.JSON(http.StatusOK, resps)
}

// 设置好友备注
func (f *Friend) remark(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	var req remarkReq
	if err := c.BindJSON(&req); err != nil {
		f.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if req.UID == "" {
		c.ResponseError(errors.New("用户uid不能为空"))
		return
	}
	settingM, err := f.settingDB.querySettingByUIDAndToUID(loginUID, req.UID)
	if err != nil {
		f.Error("查询设置信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询设置信息失败！"))
		return
	}
	if settingM == nil {
		settingM = &SettingModel{
			UID:    loginUID,
			ToUID:  req.UID,
			Remark: req.Remark,
		}
		err = f.settingDB.InsertUserSettingModel(settingM)
		if err != nil {
			f.Error("添加用户设置失败！", zap.Error(err))
			c.ResponseError(errors.New("添加用户设置失败！"))
			return
		}
	} else {
		settingM.Remark = req.Remark
		err = f.settingDB.UpdateUserSettingModel(settingM)
		if err != nil {
			f.Error("修改用户备注错误", zap.Error(err))
			c.ResponseError(errors.New("修改用户备注错误"))
			return
		}
	}

	err = f.ctx.SendChannelUpdateToUser(loginUID, config.ChannelReq{
		ChannelID:   req.UID,
		ChannelType: common.ChannelTypePerson.Uint8(),
	})
	if err != nil {
		f.Warn("修改备注-发送频道更新消息失败", zap.Error(err))
	}
	c.ResponseOK()
}

// ---------- vo ----------
// 好友申请请求
type applyReq struct {
	ToUID   string `json:"to_uid"`  // 向谁申请好友
	Remark  string `json:"remark"`  // 备注
	Vercode string `json:"vercode"` // 验证码
}

// 修改好友备注请求
type remarkReq struct {
	UID    string `json:"uid"`    //好友UID
	Remark string `json:"remark"` //备注名称
}

func (r applyReq) Check() error {
	if strings.TrimSpace(r.ToUID) == "" {
		return errors.New("好友的ID不能为空！")
	}
	// if strings.TrimSpace(r.Vercode) == "" {
	// 	return errors.New("验证码不能为空！")
	// }
	return nil
}

type sureReq struct {
	Token string `json:"token"` // 收到申请的token
}

func (r sureReq) Check() error {
	if strings.TrimSpace(r.Token) == "" {
		return errors.New("接收申请的token不能为空！")
	}
	return nil
}

type friendResp struct {
	UserDetailResp

	// ID        int64  `json:"id"`
	// ToUID     string `json:"to_uid"`
	// ToName    string `json:"to_name"`
	// ToRemark  string `json:"to_remark"`
	// Mute      int    `json:"mute"`
	// Top       int    `json:"top"`
	// Version   int64  `json:"version"`
	// CreatedAt string `json:"created_at"`
	// UpdatedAt string `json:"updated_at"`
	// IsDeleted int    `json:"is_deleted"`
	// ShortNo   string `json:"short_no"`
	// Code      string `json:"code"`
	// ChatPwdOn int    `json:"chat_pwd_on"`
	// Status    int    `json:"status"`
	// Receipt   int    `json:"receipt"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	IsDeleted int    `json:"is_deleted"`
	Version   int64  `json:"version"`
}

type friendApplyResp struct {
	Id        int64  `json:"id"`
	UID       string `json:"uid"`
	ToUID     string `json:"to_uid"`
	Remark    string `json:"remark"`
	Status    int    `json:"status"` // 状态 0.未处理 1.通过 2.拒绝
	Token     string `json:"token"`
	CreatedAt string `json:"created_at"`
}

func (f *friendResp) From(m *DetailModel, blacklist int, beBlacklist int) {
	f.UID = m.ToUID
	f.Name = m.ToName
	f.Mute = m.Mute
	f.Top = m.Top
	f.ShortNo = m.ShortNo
	f.Code = m.Vercode
	f.Vercode = m.Vercode
	f.Remark = m.Remark
	f.ChatPwdOn = m.ChatPwdOn
	f.Status = blacklist
	f.Receipt = m.Receipt
	f.Follow = 1
	f.Version = m.Version
	f.IsDeleted = m.IsDeleted
	f.Category = m.ToCategory
	f.Robot = m.Robot
	f.CreatedAt = m.CreatedAt.String()
	f.UpdatedAt = m.UpdatedAt.String()
	f.BeDeleted = m.IsAlone
	f.BeBlacklist = beBlacklist
}
