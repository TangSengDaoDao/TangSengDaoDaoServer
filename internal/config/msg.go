package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/sendgrid/rest"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// DeviceLevel 设备等级
type DeviceLevel uint8

const (
	// DeviceLevelSlave 从设备
	DeviceLevelSlave DeviceLevel = 0
	// DeviceLevelMaster 主设备
	DeviceLevelMaster DeviceLevel = 1
)

// DeviceFlag 设备类型
type DeviceFlag uint8

const (
	// APP APP
	APP DeviceFlag = iota
	// Web Web
	Web
	// PC在线
	PC
)

type Channel struct {
	ChannelID   string
	ChannelType uint8
}

func (d DeviceFlag) Uint8() uint8 {
	return uint8(d)
}

// UpdateIMTokenReq 更新IM token的请求
type UpdateIMTokenReq struct {
	UID         string
	Token       string
	DeviceFlag  DeviceFlag
	DeviceLevel DeviceLevel
}

type UpdateTokenStatus int

const (
	UpdateTokenStatusSuccess UpdateTokenStatus = 200
	UpdateTokenStatusBan     UpdateTokenStatus = 19
)

// UpdateIMTokenResp 更新IM Token的返回参数
type UpdateIMTokenResp struct {
	Status UpdateTokenStatus `json:"status"` // 状态
}

// UpdateIMToken 更新IM的token
func (c *Context) UpdateIMToken(req UpdateIMTokenReq) (*UpdateIMTokenResp, error) {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/user/token", []byte(util.ToJson(map[string]interface{}{
		"uid":          req.UID,
		"token":        req.Token,
		"device_level": req.DeviceLevel,
		"device_flag":  req.DeviceFlag,
	})), nil)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return nil, err
	}
	var result *UpdateIMTokenResp
	if err := util.ReadJsonByByte([]byte(resp.Body), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// 退出用户指定的设备 deviceFlag -1 表示退出用户所有的设备
func (c *Context) QuitUserDevice(uid string, deviceFlag int) error {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/user/device_quit", []byte(util.ToJson(map[string]interface{}{
		"uid":         uid,
		"device_flag": deviceFlag,
	})), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		c.Error("IM服务错误！", zap.Error(err))
		return fmt.Errorf("IM服务返回状态[%d]失败！", resp.StatusCode)
	}
	return nil
}

// SendMessageBatch 给一批用户发送消息
func (c *Context) SendMessageBatch(req *MsgSendBatch) error {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/message/sendbatch", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			resultMap, err := util.JsonToMap(resp.Body)
			if err != nil {
				return err
			}
			if resultMap != nil && resultMap["msg"] != nil {
				return fmt.Errorf("IM服务[SendMessageBatch]失败！ -> %s", resultMap["msg"])
			}
		}
		return fmt.Errorf("IM服务[SendMessageBatch]返回状态[%d]失败！", resp.StatusCode)
	}
	return nil

}

// SendMessage 发送消息
func (c *Context) SendMessage(req *MsgSendReq) error {
	_, err := c.SendMessageWithResult(req)
	return err
}

// SendMessage 发送消息
func (c *Context) SendMessageWithResult(req *MsgSendReq) (*MsgSendResp, error) {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/message/send", []byte(util.ToJson(req)), nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			resultMap, err := util.JsonToMap(resp.Body)
			if err != nil {
				return nil, err
			}
			if resultMap != nil && resultMap["msg"] != nil {
				return nil, fmt.Errorf("IM服务[SendMessage]失败！ -> %s", resultMap["msg"])
			}
		}
		return nil, fmt.Errorf("IM服务[SendMessage]返回状态[%d]失败！", resp.StatusCode)
	} else {
		dataResult := gjson.Get(resp.Body, "data")

		messageID := dataResult.Get("message_id").Int()
		messageSeq := dataResult.Get("message_seq").Int()
		clientMsgNo := dataResult.Get("client_msg_no").String()
		return &MsgSendResp{
			MessageID:   messageID,
			MessageSeq:  uint32(messageSeq),
			ClientMsgNo: clientMsgNo,
		}, nil
	}
}

// SendFriendApply 发送好友申请请求
func (c *Context) SendFriendApply(req *MsgFriendApplyReq) error {

	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    0,
			SyncOnce:  1, // 只同步一次
		},
		ChannelID:   req.ToUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"apply_uid":  req.ApplyUID,
			"apply_name": req.ApplyName,
			"to_uid":     req.ToUID,
			"remark":     req.Remark,
			"token":      req.Token,
			"type":       common.FriendApply,
		})),
	})
}

// SendFriendSure 发送好友确认请求
func (c *Context) SendFriendSure(req *MsgFriendSureReq) error {
	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: 0,
			RedDot:    0,
			SyncOnce:  1, // 只同步一次
		},
		ChannelID:   req.ToUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"sure_uid":  req.FromUID,
			"sure_name": req.FromName,
			"to_uid":    req.ToUID,
			"content":   "你们已经是好友了，可以愉快的聊天了！",
			"type":      common.FriendSure,
		})),
	})
}

func (c *Context) SendFriendDelete(req *MsgFriendDeleteReq) error {
	return c.SendCMD(MsgCMDReq{
		ChannelID:   req.FromUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
		CMD:         common.CMDFriendDeleted,
		Param: map[string]interface{}{
			"uid": req.ToUID,
		},
	})
}

// IMCreateOrUpdateChannelInfo 修改或创建channel信息
func (c *Context) IMCreateOrUpdateChannelInfo(req *ChannelInfoCreateReq) error {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/info", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMCreateOrUpdateChannel 请求IM创建或更新频道
func (c *Context) IMCreateOrUpdateChannel(req *ChannelCreateReq) error {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMBlacklistAdd 添加黑名单
func (c *Context) IMBlacklistAdd(req ChannelBlacklistReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/blacklist_add", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMBlacklistSet 设置黑名单
func (c *Context) IMBlacklistSet(req ChannelBlacklistReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/blacklist_set", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMBlacklistRemove 移除黑名单
func (c *Context) IMBlacklistRemove(req ChannelBlacklistReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/blacklist_remove", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMWhitelistAdd 添加白名单
func (c *Context) IMWhitelistAdd(req ChannelWhitelistReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/whitelist_add", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMWhitelistSet 白名单设置（覆盖旧的数据）
func (c *Context) IMWhitelistSet(req ChannelWhitelistReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/whitelist_set", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMWhitelistRemove 移除白名单
func (c *Context) IMWhitelistRemove(req ChannelWhitelistReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/whitelist_remove", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMAddSubscriber 请求IM创建频道
func (c *Context) IMAddSubscriber(req *SubscriberAddReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/subscriber_add", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IM服务[IMAddSubscriber]返回状态[%d]失败！", resp.StatusCode)
	}
	return nil
}

// IMRemoveSubscriber 请求IM创建频道
func (c *Context) IMRemoveSubscriber(req *SubscriberRemoveReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/subscriber_remove", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMGetConversations 获取用户最近会话列表
func (c *Context) IMGetConversations(uid string) ([]*ConversationResp, error) {

	resp, err := network.Get(c.cfg.WuKongIM.APIURL+"/conversations", map[string]string{
		"uid": uid,
	}, nil)
	if err != nil {
		return nil, err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return nil, err
	}
	var resps []*ConversationResp
	err = util.ReadJsonByByte([]byte(resp.Body), &resps)
	if err != nil {
		return nil, err
	}
	return resps, nil
}

// IMClearConversationUnread 清除用户某个频道的未读数
func (c *Context) IMClearConversationUnread(req ClearConversationUnreadReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/conversations/setUnread", []byte(util.ToJson(req)), nil)
	if err != nil {
		return nil
	}
	return c.handlerIMError(resp)
}

// IMDeleteConversation 删除最近会话
func (c *Context) IMDeleteConversation(req DeleteConversationReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/conversations/delete", []byte(util.ToJson(req)), nil)
	if err != nil {
		return nil
	}
	return c.handlerIMError(resp)
}

// IMSyncUserConversation 同步用户会话数据
func (c *Context) IMSyncUserConversation(uid string, version int64, msgCount int64, lastMsgSeqs string, larges []*Channel) ([]*SyncUserConversationResp, error) {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/conversation/sync", []byte(util.ToJson(map[string]interface{}{
		"uid":           uid,
		"version":       version,
		"last_msg_seqs": lastMsgSeqs,
		"msg_count":     msgCount,
		"larges":        larges,
	})), nil)
	if err != nil {
		return nil, err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return nil, err
	}
	var conversations []*SyncUserConversationResp
	err = util.ReadJsonByByte([]byte(resp.Body), &conversations)
	if err != nil {
		return nil, err
	}
	return conversations, nil
}

// IMSyncUserConversationAck 同步用户会话数据回执
// func (c *Context) IMSyncUserConversationAck(uid string, cmdVersion int64) error {

// 	resp, err := network.Post(c.cfg.IMExtendURL+"/conversation/syncack", []byte(util.ToJson(map[string]interface{}{
// 		"uid":         uid,
// 		"cmd_version": cmdVersion,
// 	})), nil)
// 	if err != nil {
// 		return err
// 	}
// 	return c.handlerIMError(resp)

// }

// IMSyncChannelMessage 同步频道消息
func (c *Context) IMSyncChannelMessage(req SyncChannelMessageReq) (*SyncChannelMessageResp, error) {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/channel/messagesync", []byte(util.ToJson(req)), nil)
	if err != nil {
		return nil, err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return nil, err
	}
	var syncChannelMessageResp *SyncChannelMessageResp
	err = util.ReadJsonByByte([]byte(resp.Body), &syncChannelMessageResp)
	if err != nil {
		return nil, err
	}
	return syncChannelMessageResp, nil
}

// IMSyncMessage 同步IM消息
func (c *Context) IMSyncMessage(req *MsgSyncReq) ([]*MessageResp, error) {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/message/sync", []byte(util.ToJson(req)), nil)
	if err != nil {
		return nil, err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return nil, err
	}
	var resps []*MessageResp
	err = util.ReadJsonByByte([]byte(resp.Body), &resps)
	if err != nil {
		return nil, err
	}
	return resps, nil
}

// IMSyncMessageAck 同步IM消息回执
func (c *Context) IMSyncMessageAck(req *SyncackReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/message/syncack", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// IMDeleteMessage 删除IM消息
// func (c *Context) IMDeleteMessage(req *MessageDeleteReq) error {

// 	resp, err := network.Post(c.cfg.IMExtendURL+"/message/delete", []byte(util.ToJson(req)), nil)
// 	if err != nil {
// 		return err
// 	}
// 	return c.handlerIMError(resp)
// }

// IMRevokeMessage 撤回IM消息
func (c *Context) IMRevokeMessage(req *MessageRevokeReq) error {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/message/revoke", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// SendRevoke 发送撤回消息
func (c *Context) SendRevoke(req *MsgRevokeReq) error {

	return c.SendCMD(MsgCMDReq{
		FromUID:     req.FromUID,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		CMD:         "messageRevoke",
		Param: map[string]interface{}{
			"message_id": fmt.Sprintf("%d", req.MessageID),
		},
	})
}

// SendCMD 发送CMD消息
func (c *Context) SendCMD(req MsgCMDReq) error {

	contentMap := map[string]interface{}{
		"cmd":  req.CMD,
		"type": common.CMD,
	}
	if req.Param != nil {
		contentMap["param"] = req.Param
	}
	var noPersist = 0
	if req.NoPersist {
		noPersist = 1
	}
	setting := Setting{
		NoUpdateConversation: true,
	}

	contentBytes := []byte(util.ToJson(contentMap))

	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			NoPersist: noPersist,
			RedDot:    0,
			SyncOnce:  1,
		},
		Setting:     setting.ToUint8(),
		FromUID:     req.FromUID,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
		Subscribers: req.Subscribers,
		Payload:     contentBytes,
	})
}

func (c *Context) SendTyping(channelID string, channelType uint8, fromUID string) error {
	// 发送输入中的命令
	err := c.SendCMD(MsgCMDReq{
		NoPersist:   true,
		CMD:         common.CMDTyping,
		ChannelID:   channelID,
		ChannelType: channelType,
		Param: map[string]interface{}{
			"from_uid":     fromUID,
			"channel_id":   channelID,
			"channel_type": channelType,
		},
	})
	return err
}

func (c *Context) handlerIMError(resp *rest.Response) error {
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			resultMap, err := util.JsonToMap(resp.Body)
			if err != nil {
				return err
			}
			if resultMap != nil && resultMap["msg"] != nil {
				return fmt.Errorf("IM服务失败！ -> %s", resultMap["msg"])
			}
		}
		return fmt.Errorf("IM服务返回状态[%d]失败！", resp.StatusCode)
	}
	return nil
}

// ---------- req  ----------

type PullMode int

const (
	PullModeDown PullMode = iota
	PullModeUp
)

// SyncChannelMessageReq 同步频道消息请求
type SyncChannelMessageReq struct {
	LoginUID        string   `json:"login_uid"`
	DeviceUUID      string   `json:"device_uuid"`
	ChannelID       string   `json:"channel_id"`
	ChannelType     uint8    `json:"channel_type"`
	StartMessageSeq uint32   `json:"start_message_seq"` // 开始序列号
	EndMessageSeq   uint32   `json:"end_message_seq"`   // 结束序列号
	Limit           int      `json:"limit"`             // 每次同步数量限制
	PullMode        PullMode `json:"pull_mode"`         // 拉取模式
}

// SyncChannelMessageResp 同步频道消息返回
type SyncChannelMessageResp struct {
	StartMessageSeq uint32         `json:"start_message_seq"` // 开始序列号
	EndMessageSeq   uint32         `json:"end_message_seq"`   // 结束序列号
	PullMode        PullMode       `json:"pull_mode"`         // 拉取模式
	Messages        []*MessageResp `json:"messages"`          // 消息数据
}

// ClearConversationUnreadReq 清除用户某个频道未读数请求
type ClearConversationUnreadReq struct {
	UID         string `json:"uid"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	Unread      int    `json:"unread"`
	MessageSeq  uint32 `json:"message_seq"`
}

// SyncackReq 同步回执请求
type SyncackReq struct {
	// 用户uid
	UID string `json:"uid"`
	// 最后一次同步的message_seq
	LastMessageSeq uint32 `json:"last_message_seq"`
}

func (s SyncackReq) String() string {
	return fmt.Sprintf("UID: %s LastMessageSeq: %d", s.UID, s.LastMessageSeq)
}

// Check 检查参数输入
func (s SyncackReq) Check() error {
	if strings.TrimSpace(s.UID) == "" {
		return errors.New("用户UID不能为空！")
	}
	if s.LastMessageSeq == 0 {
		return errors.New("最后一次messageSeq不能为0！")
	}
	return nil
}

// IMSOnlineStatus 获取指定用户的在线状态
func (c *Context) IMSOnlineStatus(uids []string) ([]*OnlinestatusResp, error) {
	if c.cfg.Test {
		c.Info("获取指定用户的在线状态", zap.String("req", util.ToJson(uids)))
		return nil, nil
	}
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/user/onlinestatus", []byte(util.ToJson(uids)), nil)
	if err != nil {
		return nil, err
	}
	if err := c.handlerIMError(resp); err != nil {
		return nil, err
	}
	var resps []*OnlinestatusResp
	err = util.ReadJsonByByte([]byte(resp.Body), &resps)
	if err != nil {
		return nil, err
	}
	return resps, nil
}

// OnlinestatusResp 在线状态返回
type OnlinestatusResp struct {
	UID         string `json:"uid"`          // 在线用户uid
	DeviceFlag  uint8  `json:"device_flag"`  // 设备标记 0. APP 1.web
	LastOffline int    `json:"last_offline"` // 最后一次离线时间
	Online      int    `json:"online"`       // 是否在线
}

// MessageDeleteReq 删除消息请求
type MessageDeleteReq struct {
	UID         string   `json:"uid"`          // 频道ID
	ChannelID   string   `json:"channel_id"`   // 频道ID
	ChannelType uint8    `json:"channel_type"` // 频道类型
	MessageIDs  []uint64 `json:"message_ids"`  // 消息ID集合 （如果all=1 则此字段无效）
}

// MessageRevokeReq 消息撤回请求
type MessageRevokeReq struct {
	ChannelID   string   `json:"channel_id"`   // 频道ID
	ChannelType uint8    `json:"channel_type"` // 频道类型
	MessageIDs  []uint64 `json:"message_ids"`  // 指定需要撤回的消息
}

// MsgRevokeReq 撤回消息请求
type MsgRevokeReq struct {
	FromUID      string `json:"from_uid"`
	Operator     string `json:"operator"`      // 操作者uid
	OperatorName string `json:"operator_name"` // 操作者名称
	ChannelID    string `json:"channel_id"`    // 频道ID
	ChannelType  uint8  `json:"channel_type"`  // 频道类型
	MessageID    int64  `json:"message_id"`    // 消息ID
}

// UserBaseVo 用户基础信息
type UserBaseVo struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

// MsgSendReq 发送消息请求
type MsgSendReq struct {
	Header      MsgHeader `json:"header"`       // 消息头
	Setting     uint8     `json:"setting"`      // setting
	FromUID     string    `json:"from_uid"`     // 模拟发送者的UID
	ChannelID   string    `json:"channel_id"`   // 频道ID
	ChannelType uint8     `json:"channel_type"` // 频道类型
	StreamNo    string    `json:"stream_no"`    // 消息流号
	Subscribers []string  `json:"subscribers"`  // 订阅者 如果此字段有值，表示消息只发给指定的订阅者
	Payload     []byte    `json:"payload"`      // 消息内容
}

type MsgSendResp struct {
	MessageID   int64  `json:"message_id"`    // 消息ID
	ClientMsgNo string `json:"client_msg_no"` // 客户端消息唯一编号
	MessageSeq  uint32 `json:"message_seq"`   // 消息序号
}

// MsgSendBatch 给一批用户发送消息请求
type MsgSendBatch struct {
	Header      MsgHeader `json:"header"`      // 消息头
	FromUID     string    `json:"from_uid"`    // 模拟发送者的UID
	Subscribers []string  `json:"subscribers"` // 订阅者 如果此字段有值，表示消息只发给指定的订阅者
	Payload     []byte    `json:"payload"`     // 消息内容
}

func (m *MsgSendReq) String() string {
	return fmt.Sprintf("ChannelID:%s ChannelType:%d Payload:%s", m.ChannelID, m.ChannelType, string(m.Payload))
}

// MsgFriendApplyReq 好友申请
type MsgFriendApplyReq struct {
	ApplyUID  string `json:"apply_uid"`  // 发起申请人的uid
	ApplyName string `json:"apply_name"` // 发起申请人的名字
	ToUID     string `json:"to_uid"`     // 接收者
	Remark    string `json:"remark"`     // 申请备注
	Token     string `json:"token"`      // 凭证
}

// MsgFriendSureReq 确认好友申请
type MsgFriendSureReq struct {
	ToUID    string `json:"to_uid"`    // 接收好友申请的人uid
	FromUID  string `json:"from_uid"`  // 发起申请人的uid
	FromName string `json:"from_name"` // 发起申请人的名字
}

// MsgFriendDeleteReq 好友删除
type MsgFriendDeleteReq struct {
	FromUID string `json:"from_uid"` // 删除人的uid
	ToUID   string `json:"to_uid"`   // 被删除的好友uid
}

// MsgGroupMemberAddReq 添加群成员
type MsgGroupMemberAddReq struct {
	Operator     string        `json:"operator"`      // 操作者uid
	OperatorName string        `json:"operator_name"` // 操作者名称
	GroupNo      string        `json:"group_no"`      // 群编号
	Members      []*UserBaseVo `json:"members"`       // 邀请成员
}

// CMDGroupAvatarUpdateReq 群头像更新请求
type CMDGroupAvatarUpdateReq struct {
	GroupNo string   `json:"group_no"` // 群编号
	Members []string `json:"members"`  // 成员uids
}

// MsgCMDReq CMD消息请求
type MsgCMDReq struct {
	NoPersist   bool                   `json:"-"`            // 是否需要存储
	FromUID     string                 `json:"from_uid"`     // 模拟发送者的UID
	ChannelID   string                 `json:"channel_id"`   // 频道ID
	ChannelType uint8                  `json:"channel_type"` // 频道类型
	Subscribers []string               `json:"subscribers"`  // 订阅者 如果此字段有值，表示消息只发给指定的订阅者
	CMD         string                 `json:"cmd"`          // 操命令
	Param       map[string]interface{} `json:"param"`        // 命令参数
}

// MsgSyncReq 消息同步请求
type MsgSyncReq struct {
	UID        string `json:"uid"`         // 谁的消息
	MessageSeq uint32 `json:"message_seq"` // 客户端最大消息序列号
	Limit      int    `json:"limit"`       // 消息数量限制
}

// MsgHeader 消息头
type MsgHeader struct {
	NoPersist int `json:"no_persist"` // 是否不持久化
	RedDot    int `json:"red_dot"`    // 是否显示红点
	SyncOnce  int `json:"sync_once"`  // 此消息只被同步或被消费一次(1表示消息将走写模式) ，特别注意：sync_once=1表示写扩散 sync_once=0表示读扩散 写扩散的messageSeq和读扩散messageSeq来源不一样
}

func (h MsgHeader) String() string {
	return fmt.Sprintf("NoPersist:%d RedDot:%d SyncOnce:%d", h.NoPersist, h.RedDot, h.SyncOnce)
}

// SyncUserConversationResp 最近会话离线返回
type SyncUserConversationResp struct {
	ChannelID       string         `json:"channel_id"`         // 频道ID
	ChannelType     uint8          `json:"channel_type"`       // 频道类型
	Unread          int            `json:"unread"`             // 未读消息
	Timestamp       int64          `json:"timestamp"`          // 最后一次会话时间
	LastMsgSeq      int64          `json:"last_msg_seq"`       // 最后一条消息seq
	LastClientMsgNo string         `json:"last_client_msg_no"` // 最后一条客户端消息编号
	OffsetMsgSeq    int64          `json:"offset_msg_seq"`     // 偏移位的消息seq
	Version         int64          `json:"version"`            // 数据版本
	Recents         []*MessageResp `json:"recents"`            // 最近N条消息
}

// SyncUserConversationRespWrap SyncUserConversationRespWrap
type SyncUserConversationRespWrap struct {
	Conversations []*SyncUserConversationResp `json:"conversations"`
	CMDVersion    int64                       `json:"cmd_version"` // 最新cmd版本号
	CMDs          []*CMDResp                  `json:"cmds"`        // cmd集合
}

// CMDResp CMDResp
type CMDResp struct {
	CMD   string      `json:"cmd"`
	Param interface{} `json:"param"`
}

// Setting Setting
type Setting struct {
	Receipt              bool // 消息已读回执，此标记表示，此消息需要已读回执
	NoUpdateConversation bool // 不更新最近会话
	Signal               bool // 是否signal加密
}

// ToUint8 ToUint8
func (s Setting) ToUint8() uint8 {
	return uint8(encodeBool(s.Receipt)<<7 | encodeBool(s.NoUpdateConversation)<<6 | encodeBool(s.Signal)<<5)
}

// SettingFromUint8 SettingFromUint8
func SettingFromUint8(v uint8) Setting {
	s := Setting{}
	s.Receipt = (v >> 7 & 0x01) > 0
	s.NoUpdateConversation = (v >> 6 & 0x01) > 0
	s.Signal = (v >> 5 & 0x01) > 0
	return s
}
func encodeBool(b bool) (i int) {
	if b {
		i = 1
	}
	return
}

// MessageResp 消息
type MessageResp struct {
	Header      MsgHeader         `json:"header"`              // 消息头
	Setting     uint8             `json:"setting"`             // 设置
	MessageID   int64             `json:"message_id"`          // 服务端的消息ID(全局唯一)
	MessageSeq  uint32            `json:"message_seq"`         // 消息序列号 （用户唯一，有序递增）
	ClientMsgNo string            `json:"client_msg_no"`       // 客户端消息唯一编号
	FromUID     string            `json:"from_uid"`            // 发送者UID
	ToUID       string            `json:"to_uid"`              // 接受者uid
	ChannelID   string            `json:"channel_id"`          // 频道ID
	ChannelType uint8             `json:"channel_type"`        // 频道类型
	Timestamp   int32             `json:"timestamp"`           // 服务器消息时间戳(10位，到秒)
	Payload     []byte            `json:"payload"`             // 消息内容
	StreamNo    string            `json:"stream_no,omitempty"` // 流编号
	Streams     []*StreamItemResp `json:"streams,omitempty"`   // 消息流
	// ReplyCount    int            `json:"reply_count,omitempty"`     // 回复集合
	// ReplyCountSeq string         `json:"reply_count_seq,omitempty"` // 回复数量seq
	// ReplySeq      string         `json:"reply_seq,omitempty"`       // 回复seq
	// Reactions     []ReactionResp `json:"reactions,omitempty"`       // 回应数据
	IsDeleted   int `json:"is_deleted"`   // 是否已删除
	VoiceStatus int `json:"voice_status"` // 语音状态 0.未读 1.已读

	payloadMap map[string]interface{}
}

// GetPayloadMap GetPayloadMap
func (m *MessageResp) GetPayloadMap() (map[string]interface{}, error) {
	if m.payloadMap == nil {
		var payloadMap map[string]interface{}
		if err := util.ReadJsonByByte(m.Payload, &payloadMap); err != nil {
			return nil, err
		}
		m.payloadMap = payloadMap
	}
	return m.payloadMap, nil
}

// GetContentType 消息正文类型
func (m *MessageResp) GetContentType() int {
	payloadMap, err := m.GetPayloadMap()
	if err != nil {
		return 0
	}
	contentTypeInt64, _ := payloadMap["type"].(json.Number).Int64()
	return int(contentTypeInt64)
}

type StreamItemResp struct {
	StreamSeq   uint32 `json:"stream_seq"`    // 流序号
	ClientMsgNo string `json:"client_msg_no"` // 客户端消息唯一编号
	Blob        []byte `json:"blob"`          // 消息内容
}

// ReactionResp 回应返回
type ReactionResp struct {
	Seq   string     `json:"seq"`   // 回复序列号
	Users []UserResp `json:"users"` // 回应用户集合
	Emoji string     `json:"emoji"` // 回应的emoji
	Count int        `json:"count"` // 回应数量
}

// UserResp UserResp
type UserResp struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

// ConversationResp 最近会话返回数据
type ConversationResp struct {
	ChannelID   string       `json:"channel_id"`   // 频道ID
	ChannelType uint8        `json:"channel_type"` // 频道类型
	Unread      int64        `json:"unread"`       // 未读数
	Timestamp   int64        `json:"timestamp"`    // 最后一次会话时间戳
	LastMessage *MessageResp `json:"last_message"` // 最后一条消息
}

// ChannelCreateReq 频道创建请求
type ChannelCreateReq struct {
	ChannelID   string   `json:"channel_id"`   // 频道ID
	ChannelType uint8    `json:"channel_type"` // 频道类型
	Ban         int      `json:"ban"`          // 是否被封禁（一般建议500或1000成员以上设置为超大群，超大群，注意：超大群不会维护最近会话数据）
	Large       int      `json:"large"`        // 是否是超大群（被封后 任何人都不能发消息，包括创建者）
	Subscribers []string `json:"subscribers"`  // 订阅者
}

// ChannelInfoCreateReq
type ChannelInfoCreateReq struct {
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
	Ban         int    `json:"ban"`          // 是否封禁
	Large       int    `json:"large"`        // 是否大群
}

// ChannelReq ChannelReq
type ChannelReq struct {
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
}

// DeleteConversationReq  DeleteConversationReq
type DeleteConversationReq struct {
	UID         string `json:"uid"`
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
}

// ChannelBlacklistReq 黑名单
type ChannelBlacklistReq struct {
	ChannelReq
	UIDs []string `json:"uids"` // 黑名单用户
}

// ChannelWhitelistReq 白名单
type ChannelWhitelistReq struct {
	ChannelReq
	UIDs []string `json:"uids"` // 白名单用户
}

// SubscriberAddReq 添加订阅请求
type SubscriberAddReq struct {
	ChannelID   string   `json:"channel_id"`
	ChannelType uint8    `json:"channel_type"`
	Reset       int      `json:"reset"` // 是否重置订阅者 （0.不重置 1.重置），选择重置，将删除原来的所有成员
	Subscribers []string `json:"subscribers"`
}

// SubscriberRemoveReq 移除订阅请求
type SubscriberRemoveReq struct {
	ChannelID   string   `json:"channel_id"`
	ChannelType uint8    `json:"channel_type"`
	Subscribers []string `json:"subscribers"`
}
