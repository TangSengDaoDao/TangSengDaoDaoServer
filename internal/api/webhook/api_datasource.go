package webhook

import (
	"errors"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/api/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/api/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/wkhttp"
	"go.uber.org/zap"
)

// 数据源
func (w *Webhook) datasource(c *wkhttp.Context) {
	var cmdReq struct {
		CMD  string                 `json:"cmd"`
		Data map[string]interface{} `json:"data"`
	}
	if err := c.BindJSON(&cmdReq); err != nil {
		w.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if strings.TrimSpace(cmdReq.CMD) == "" {
		c.ResponseError(errors.New("cmd不能为空！"))
		return
	}
	w.Debug("请求数据源", zap.Any("cmd", cmdReq))
	var result interface{}
	var err error
	switch cmdReq.CMD {
	case "getChannelInfo":
		result, err = w.getChannelInfo(cmdReq.Data)
	case "getSubscribers":
		result, err = w.getSubscribers(cmdReq.Data)
	case "getBlacklist":
		result, err = w.getBlacklist(cmdReq.Data)
	case "getWhitelist":
		result, err = w.getWhitelist(cmdReq.Data)
	case "getSystemUIDs":
		result, err = w.getSystemUIDs()
	}

	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Response(result)
}

func (w *Webhook) getChannelInfo(data map[string]interface{}) (interface{}, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}
	channelInfoMap := map[string]interface{}{}
	if channelReq.ChannelType == common.ChannelTypeGroup.Uint8() {
		groupInfo, err := w.groupService.GetGroupWithGroupNo(channelReq.ChannelID)
		if err != nil {
			w.Error("获取群信息失败！", zap.Error(err))
			return nil, err
		}
		if groupInfo != nil {
			if groupInfo.Status == group.GroupStatusDisabled {
				channelInfoMap["ban"] = 1
			}
			if groupInfo.GroupType == group.GroupTypeSuper {
				channelInfoMap["large"] = 1
			}

		}
	}

	return channelInfoMap, nil
}

func (w *Webhook) getSubscribers(data map[string]interface{}) ([]string, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}

	if channelReq.ChannelType == common.ChannelTypePerson.Uint8() {
		return make([]string, 0), nil
	}

	subscribers := make([]string, 0)
	if channelReq.ChannelType == common.ChannelTypeCommunityTopic.Uint8() {
		return make([]string, 0), nil
	}

	mebmers, err := w.groupService.GetMembers(channelReq.ChannelID)
	if err != nil {
		return nil, err
	}

	if len(mebmers) > 0 {
		for _, member := range mebmers {
			subscribers = append(subscribers, member.UID)
		}
	}
	return subscribers, nil

}

func (w *Webhook) getBlacklist(data map[string]interface{}) ([]string, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}
	if channelReq.ChannelType == uint8(common.ChannelTypeGroup) {
		return w.groupService.GetBlacklistMemberUIDs(channelReq.ChannelID)
	}

	if channelReq.ChannelType == uint8(common.ChannelTypePerson) && common.IsFakeChannel(channelReq.ChannelID) {
		uids := strings.Split(channelReq.ChannelID, "@")
		exist, err := w.userService.ExistBlacklist(uids[0], uids[1])
		if err != nil {
			return nil, err
		}
		if exist {
			return uids, nil
		}
	}
	return make([]string, 0), nil
}

func (w *Webhook) getWhitelist(data map[string]interface{}) ([]string, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}

	if channelReq.ChannelType == uint8(common.ChannelTypeGroup) {
		groupInfo, err := w.groupService.GetGroupWithGroupNo(channelReq.ChannelID)
		if err != nil {
			w.Error("获取群信息失败！", zap.Error(err))
			return nil, err
		}
		if groupInfo.Forbidden == 1 {
			return w.groupService.GetMemberUIDsOfManager(channelReq.ChannelID)
		}
		return make([]string, 0), nil
	}
	friends, err := w.userService.GetFriends(channelReq.ChannelID)
	if err != nil {
		return nil, err
	}
	firendUIDs := make([]string, 0, len(friends))
	if len(friends) > 0 {
		for _, friend := range friends {
			if friend.IsAlone == 0 {
				firendUIDs = append(firendUIDs, friend.UID)
			}
		}
	}
	return firendUIDs, nil
}

func (w *Webhook) getSystemUIDs() ([]string, error) {
	users, err := w.userService.GetUsersWithCategory(user.CategoryService)
	if err != nil {
		return nil, err
	}
	uids := make([]string, 0, len(users))
	if len(users) > 0 {
		for _, user := range users {
			uids = append(uids, user.UID)
		}
	}
	return uids, nil
}

type ChannelReq struct {
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}
