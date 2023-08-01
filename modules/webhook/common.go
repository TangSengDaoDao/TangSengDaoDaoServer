package webhook

import (
	"encoding/json"
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"go.uber.org/zap"
)

// EventMsgOffline 离线消息
const EventMsgOffline = "msg.offline"

// EventOnlineStatus 在线状态
const EventOnlineStatus = "user.onlinestatus"

// EventMsgNotify 消息通知 (所有消息)
const EventMsgNotify = "msg.notify"

const (
	nameCachePrefix      string = "name:"
	groupNameCachePrefix string = "groupName:"
)

type PayloadInfo struct {
	Title   string
	Content string
	Badge   int

	// ------ 以下是rtc推送需要 ------
	IsVideoCall bool   // 是否是rtc消息
	FromUID     string // rtc消息需要
	CallType    common.RTCCallType
	Operation   string
}

func (p *PayloadInfo) toPayload() Payload {
	var payload Payload
	var basePayload = BasePayload{
		title:   p.Title,
		content: p.Content,
		badge:   p.Badge,
	}
	if p.IsVideoCall {
		payload = &BaseRTCPayload{
			BasePayload: basePayload,
			fromUID:     p.FromUID,
			operation:   p.Operation,
			callType:    p.CallType,
		}
	} else {
		payload = &basePayload
	}
	return payload
}

// ParsePushInfo 解析推送信息 获得title,content,badge
func ParsePushInfo(msgResp msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (*PayloadInfo, error) {
	toUID := toUser.UID
	fromName, err := getFromName(msgResp, ctx)
	if err != nil {
		return nil, err
	}

	// 红点
	badge, err := getUserBadge(toUID, ctx)
	if err != nil {
		log.Warn("获取用户红点失败", zap.Error(err), zap.String("uid", toUID))
	}

	payloadInfo := &PayloadInfo{
		Badge: badge,
	}

	content, err := getMessageAlert(msgResp, toUser, ctx)
	if err != nil {
		return nil, err
	}

	if msgResp.ChannelType == common.ChannelTypePerson.Uint8() {
		payloadInfo.Title = fromName
	} else {
		var groupName string
		groupName, err = getAndCacheGroupName(msgResp, ctx)
		if err != nil {
			log.Error("获取群名失败！", zap.Error(err), zap.String("group_no", msgResp.ChannelID))
			return nil, err
		}
		payloadInfo.Title = groupName
		content = fmt.Sprintf("%s：%s", fromName, content)
	}
	payloadInfo.Content = content

	return payloadInfo, nil
}

func getFromName(msgResp msgOfflineNotify, ctx *config.Context) (string, error) {
	fromName, err := getAndCacheShowNameForFromUID(msgResp, ctx)
	if err != nil {
		log.Error("获取fromUID对应的名称失败！", zap.Error(err))
		return "", err
	}
	return fromName, nil
}

func getMessageAlert(msg msgOfflineNotify, toUser *user.Resp, ctx *config.Context) (string, error) {
	setting := config.SettingFromUint8(msg.Setting)
	if msg.PayloadMap == nil || setting.Signal || !ctx.GetConfig().Push.ContentDetailOn || toUser.MsgShowDetail == 1 {
		return "您有一条新的消息", nil
	}

	var alert string
	contentTypeInt64, _ := msg.PayloadMap["type"].(json.Number).Int64()
	contentType := common.ContentType(contentTypeInt64)
	switch contentType {
	case common.Text:
		if msg.PayloadMap["content"] != nil {
			alert = msg.PayloadMap["content"].(string)
		}
	case common.Image:
		alert = "[图片]"
	case common.GIF:
		alert = "[GIF]"
	case common.Voice:
		alert = "[语音]"
	case common.Video:
		alert = "[视频]"
	case common.Card:
		alert = "[名片]"
	case common.File:
		alert = "[文件]"
	case common.Location:
		alert = "[位置]"
	case common.RedPacket:
		alert = "[红包]"
	case common.Transfer:
		alert = "[转账]"
	case common.VectorSticker:
		alert = "[动画表情]"
	case common.EmojiSticker:
		alert = "[emoji表情]"
	case common.MultipleForward:
		alert = "[聊天记录]"
	}
	return alert, nil
}

var webhookDB *DB

// 获取和缓存发送者的显示名称
func getAndCacheShowNameForFromUID(msgResp msgOfflineNotify, ctx *config.Context) (string, error) {
	if webhookDB == nil {
		webhookDB = NewDB(ctx.DB())
	}

	var name, // 发送者常用名
		remark, // 接收者对发送者的备注
		nameInGroup string // 如果是群聊则发送者在群里的备注
	if msgResp.ChannelType == common.ChannelTypePerson.Uint8() {
		key := fmt.Sprintf("%s%s-%s", nameCachePrefix, msgResp.FromUID, msgResp.ToUID)
		nameMap, err := ctx.GetRedisConn().Hgetall(key)
		if err != nil {
			log.Error("从缓存中获取名字失败！", zap.Error(err))
			return "", err
		}
		if len(nameMap) > 0 { // 存在缓存，直接取出
			name = nameMap["name"]
			remark = nameMap["remark"]
		} else { // 不存在缓存，从DB获取，然后再缓存
			name, remark, _, err = webhookDB.GetThirdName(msgResp.FromUID, msgResp.ToUID, "")
			if err != nil {
				return "", err
			}
			err = ctx.GetRedisConn().Hmset(key, "name", name, "remark", remark)
			if err != nil {
				log.Error("缓存名字失败！", zap.Error(err))
				return "", err
			}
			err = ctx.GetRedisConn().Expire(key, ctx.GetConfig().Cache.NameCacheExpire)
			if err != nil {
				log.Error("设置过期时间失败！", zap.String("key", key), zap.Error(err))
				return "", err
			}
		}
	} else {
		key := fmt.Sprintf("%s%s-%s@%s", nameCachePrefix, msgResp.FromUID, msgResp.ToUID, msgResp.ChannelID)
		nameMap, err := ctx.GetRedisConn().Hgetall(key)
		if err != nil {
			log.Error("从缓存中获取名字失败！", zap.Error(err))
			return "", err
		}
		if len(nameMap) > 0 { // 存在缓存，直接取出
			name = nameMap["name"]
			remark = nameMap["remark"]
			nameInGroup = nameMap["name_in_group"]
		} else { // 不存在缓存，从DB获取，然后再缓存
			name, remark, nameInGroup, err = webhookDB.GetThirdName(msgResp.FromUID, msgResp.ToUID, msgResp.ChannelID)
			if err != nil {
				return "", err
			}
			err = ctx.GetRedisConn().Hmset(key, "name", name, "remark", remark, "name_in_group", nameInGroup)
			if err != nil {
				log.Error("缓存名字失败！", zap.Error(err))
				return "", err
			}
			err = ctx.GetRedisConn().Expire(key, ctx.GetConfig().Cache.NameCacheExpire)
			if err != nil {
				log.Error("设置过期时间失败！", zap.String("key", key), zap.Error(err))
				return "", err
			}
		}

	}
	if remark != "" { // 优先返回备注
		return remark, nil
	}
	if nameInGroup != "" {
		return nameInGroup, nil
	}
	return name, nil
}

// 获取和缓存群名
func getAndCacheGroupName(msgResp msgOfflineNotify, ctx *config.Context) (string, error) {
	if webhookDB == nil {
		webhookDB = NewDB(ctx.DB())
	}

	key := fmt.Sprintf("%s%s", groupNameCachePrefix, msgResp.ChannelID)
	groupName, err := ctx.GetRedisConn().GetString(key)
	if err != nil {
		return "", err
	}
	if groupName == "" {
		groupName, err = webhookDB.GetGroupName(msgResp.ChannelID)
		if err != nil {
			return "", err
		}
		err = ctx.GetRedisConn().Set(key, groupName)
		if err != nil {
			return "", err
		}
		err = ctx.GetRedisConn().Expire(key, ctx.GetConfig().Cache.NameCacheExpire)
		if err != nil {
			log.Error("设置群名过期时间失败！", zap.String("key", key), zap.Error(err))
			return "", err
		}
	}
	return groupName, nil
}

func getUserBadge(uid string, ctx *config.Context) (int, error) {
	badge, err := ctx.GetRedisConn().Hincrby(common.UserDeviceBadgePrefix, uid, 1)
	if err != nil {
		log.Error("获取红点数失败！", zap.Error(err))
		return 0, err
	}
	return int(badge), nil
}
