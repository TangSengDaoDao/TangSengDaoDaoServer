package channel

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/api/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/api/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/wkhttp"
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
		auth.GET("/channels/:channel_id/:channel_type", ch.channelGet) // 获取频道信息
	}
}

func (ch *Channel) channelGet(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	channelID := c.Param("channel_id")
	channelTypeI64, _ := strconv.ParseInt(c.Param("channel_type"), 10, 64)
	channelType := uint8(channelTypeI64)

	var channelResp *channelResp
	if channelType == common.ChannelTypePerson.Uint8() {
		userDetailResp, err := ch.userService.GetUserDetail(channelID, loginUID)
		if err != nil {
			c.ResponseError(err)
			return
		}
		if userDetailResp == nil {
			ch.Error("用户不存在！", zap.String("channel_id", channelID))
			c.ResponseError(errors.New("用户不存在！"))
		}
		channelResp = newChannelRespWithUserDetailResp(userDetailResp)
	} else if channelType == common.ChannelTypeGroup.Uint8() {
		groupResp, err := ch.groupService.GetGroupDetail(channelID, loginUID)
		if err != nil {
			c.ResponseError(err)
			return
		}
		if groupResp == nil {
			ch.Error("群不存在！", zap.String("channel_id", channelID))
			c.ResponseError(errors.New("群不存在！"))
			return
		}
		channelResp = newChannelRespWithGroupResp(groupResp)
	} else {
		ch.Error("不支持的频道类型", zap.Uint8("channelType", channelType))
		c.ResponseError(fmt.Errorf("不支持的频道类型[%d]", channelType))
		return
	}
	channelSettingM, err := ch.channelSettingDB.queryWithChannel(channelID, channelType)
	if err != nil {
		ch.Error("查询频道设置失败！", zap.Error(err))
		c.ResponseError(errors.New("查询频道设置失败！"))
		return
	}
	if channelSettingM != nil {
		channelResp.ParentChannel = &struct {
			ChannelID   string `json:"channel_id"`
			ChannelType uint8  `json:"channel_type"`
		}{
			ChannelID:   channelSettingM.ParentChannelID,
			ChannelType: channelSettingM.ParentChannelType,
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

type stateResp struct {
	SignalOn    uint8 `json:"signal_on"`    // 是否可以signal加密聊天
	OnlineCount int   `json:"online_count"` // 成员在线数量
}
