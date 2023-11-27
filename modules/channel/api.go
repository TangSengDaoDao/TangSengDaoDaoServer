package channel

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/model"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

type Channel struct {
	ctx *config.Context
	log.Log
	userService      user.IService
	groupService     group.IService
	channelSettingDB *channelSettingDB
}

func New(ctx *config.Context) *Channel {
	return &Channel{
		ctx:              ctx,
		Log:              log.NewTLog("Channel"),
		userService:      user.NewService(ctx),
		groupService:     group.NewService(ctx),
		channelSettingDB: newChannelSettingDB(ctx),
	}
}

// Route 路由配置
func (ch *Channel) Route(r *wkhttp.WKHttp) {
	auth := r.Group("/v1", ch.ctx.AuthMiddleware(r))
	{
		auth.GET("/channel/state", ch.state)
		auth.GET("/channels/:channel_id/:channel_type", ch.channelGet)                                  // 获取频道信息
		auth.POST("/channels/:channel_id/:channel_type/message/autodelete", ch.setAutoDeleteForMessage) // 设置消息定时删除时间
	}
}

func (ch *Channel) channelGet(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	channelID := c.Param("channel_id")
	channelTypeI64, _ := strconv.ParseInt(c.Param("channel_type"), 10, 64)
	channelType := uint8(channelTypeI64)

	modules := register.GetModules(ch.ctx)
	var err error
	var channelResp *model.ChannelResp
	for _, m := range modules {
		if m.BussDataSource.ChannelGet != nil {
			channelResp, err = m.BussDataSource.ChannelGet(channelID, channelType, loginUID)
			if err != nil {
				if errors.Is(err, register.ErrDatasourceNotProcess) {
					continue
				}
				ch.Error("查询频道失败！", zap.Error(err))
				c.ResponseError(err)
				return
			}
			break
		}
	}
	if channelResp == nil {
		ch.Error("频道不存在！", zap.String("channel_id", channelID), zap.Uint8("channelType", channelType))
		c.ResponseError(errors.New("频道不存在！"))
		return
	}
	fakeChannelID := channelID
	if channelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(loginUID, channelID)
	}

	channelSettingM, err := ch.channelSettingDB.queryWithChannel(fakeChannelID, channelType) // TODO: 这里虽然暂时看着没啥用，后面可以统一频道的设置信息
	if err != nil {
		ch.Error("查询频道设置失败！", zap.Error(err))
		c.ResponseError(errors.New("查询频道设置失败！"))
		return
	}
	if channelSettingM != nil {
		if channelSettingM.ParentChannelID != "" {
			channelResp.ParentChannel = &struct {
				ChannelID   string `json:"channel_id"`
				ChannelType uint8  `json:"channel_type"`
			}{
				ChannelID:   channelSettingM.ParentChannelID,
				ChannelType: channelSettingM.ParentChannelType,
			}
		}
		if channelSettingM.MsgAutoDelete > 0 {
			channelResp.Extra["msg_auto_delete"] = channelSettingM.MsgAutoDelete
		}
	}

	c.JSON(http.StatusOK, channelResp)

}

func (ch *Channel) state(c *wkhttp.Context) {
	channelID := c.Query("channel_id")
	channelTypeI64, _ := strconv.ParseInt(c.Query("channel_type"), 10, 64)

	channelType := uint8(channelTypeI64)

	var signalOn uint8 = 0
	var onlineCount int = 0
	if channelType != common.ChannelTypePerson.Uint8() {

		members, err := ch.groupService.GetMembers(channelID)
		if err != nil {
			c.ResponseError(errors.New("查询群成员错误"))
			ch.Error("查询群成员错误", zap.Error(err))
			return
		}
		uids := make([]string, 0)
		if len(members) > 0 {
			for _, member := range members {
				uids = append(uids, member.UID)
			}
		}
		onlineUsers, err := ch.userService.GetUserOnlineStatus(uids)
		if err != nil {
			c.ResponseError(errors.New("查询群成员在线数量错误"))
			ch.Error("查询群成员在线数量错误", zap.Error(err))
			return
		}
		if len(onlineUsers) > 0 {
			for _, user := range onlineUsers {
				if user.Online == 1 {
					onlineCount += 1
				}
			}
		}
	}

	c.Response(stateResp{
		SignalOn:    signalOn,
		OnlineCount: onlineCount,
	})

}

func (ch *Channel) setAutoDeleteForMessage(c *wkhttp.Context) {
	channelID := c.Param("channel_id")
	channelTypeI64, _ := strconv.ParseInt(c.Param("channel_type"), 10, 64)
	channelType := uint8(channelTypeI64)

	var req struct {
		MsgAutoDelete int64 `json:"msg_auto_delete"` // 单位秒
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.ResponseError(errors.New("参数错误"))
		ch.Error("参数错误", zap.Error(err))
		return
	}

	loginUID := c.GetLoginUID()
	fakeChannelID := channelID
	if channelType == common.ChannelTypePerson.Uint8() {
		fakeChannelID = common.GetFakeChannelIDWith(loginUID, channelID)
	} else {
		isCreatorOrManager, err := ch.groupService.IsCreatorOrManager(channelID, loginUID)
		if err != nil {
			c.ResponseError(errors.New("查询群的创建者或管理员错误"))
			ch.Error("查询群的创建者或管理员错误", zap.Error(err))
			return
		}
		if !isCreatorOrManager {
			c.ResponseError(errors.New("没有权限设置"))
			ch.Error("没有权限设置")
			return
		}
	}

	if err := ch.channelSettingDB.insertOrAddMsgAutoDelete(fakeChannelID, channelType, req.MsgAutoDelete); err != nil {
		c.ResponseError(errors.New("设置失败"))
		ch.Error("设置失败", zap.Error(err))
		return
	}
	if req.MsgAutoDelete > 0 {
		payload := []byte(util.ToJson(map[string]interface{}{
			"content": fmt.Sprintf("{0}设置消息在 %s 后自动删除", formatSecondToDisplayTime(req.MsgAutoDelete)),
			"type":    common.Tip,
			"data": map[string]interface{}{
				"msg_auto_delete": req.MsgAutoDelete,
				"data_type":       "autoDeleteForMessage",
			},
			"extra": []config.UserBaseVo{
				{
					UID:  loginUID,
					Name: c.GetLoginName(),
				},
			},
		}))
		err := ch.ctx.SendMessage(&config.MsgSendReq{
			FromUID:     loginUID,
			ChannelID:   channelID,
			ChannelType: channelType,
			Payload:     payload,
			Header: config.MsgHeader{
				RedDot: 1,
			},
		})
		if err != nil {
			ch.Error("发送消息失败！", zap.Error(err))
			c.ResponseError(errors.New("发送消息失败！"))
			return
		}
	}
	channelReq := config.ChannelReq{
		ChannelID:   channelID,
		ChannelType: channelType,
	}
	err := ch.ctx.SendChannelUpdateWithFromUID(channelReq, channelReq, loginUID)
	if err != nil {
		ch.Warn("发送频道更新命令失败！", zap.Error(err))
	}
	c.ResponseOK()
}

func formatSecondToDisplayTime(second int64) string {
	if second < 60 {
		return fmt.Sprintf("%d秒", second)
	}
	if second < 60*60 {
		return fmt.Sprintf("%d分钟", second/60)
	}
	if second < 60*60*24 {
		return fmt.Sprintf("%d小时", second/60/60)
	}
	if second < 60*60*24*30 {
		return fmt.Sprintf("%d天", second/60/60/24)
	}
	if second < 60*60*24*30*12 {
		return fmt.Sprintf("%d月", second/60/60/24/30)
	}
	return fmt.Sprintf("%d年", second/60/60/24/30/12)
}

type stateResp struct {
	SignalOn    uint8 `json:"signal_on"`    // 是否可以signal加密聊天
	OnlineCount int   `json:"online_count"` // 成员在线数量
}
