package search

import (
	"errors"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

type Search struct {
	ctx *config.Context
	log.Log
	userService  user.IService
	groupService group.IService
}

func New(ctx *config.Context) *Search {
	s := &Search{
		ctx:          ctx,
		Log:          log.NewTLog("search"),
		userService:  user.NewService(ctx),
		groupService: group.NewService(ctx),
	}
	return s
}

func (s *Search) Route(r *wkhttp.WKHttp) {
	searchs := r.Group("/v1/search", s.ctx.AuthMiddleware(r))
	{
		searchs.GET("/gobal", s.gobal)     // 全局搜索
		searchs.POST("/message", s.search) // 搜索消息
	}
}

func (s *Search) gobal(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	keyword := c.Query("keyword")
	if len(strings.TrimSpace(keyword)) == 0 {
		c.ResponseError(errors.New("关键字不能为空"))
		return
	}

	// 查询消息
	msgResp, err := s.ctx.IMSearchUserMessages(&config.SearchUserMessageReq{
		UID:            loginUID,
		PayloadContent: keyword,
		Limit:          10,
		Page:           1,
	})
	if err != nil {
		s.Error("查询悟空IM消息错误", zap.Error(err))
		c.ResponseError(errors.New("查询悟空IM消息错误"))
		return
	}

	groupIds := make([]string, 0)
	uids := make([]string, 0)
	if msgResp != nil && len(msgResp.Messages) > 0 {
		for _, m := range msgResp.Messages {
			if m.ChannelType == common.ChannelTypeGroup.Uint8() {
				groupIds = append(groupIds, m.ChannelID)
			} else if m.ChannelType == common.ChannelTypePerson.Uint8() {
				uids = append(uids, m.ChannelID)
			}
		}
	}
	joinedGroups, err := s.groupService.GetGroupsWithMemberUID(loginUID)
	if err != nil {
		s.Error("查询加入的群列表错误", zap.Error(err))
		c.ResponseError(errors.New("查询加入的群列表错误"))
		return
	}
	if len(joinedGroups) > 0 {
		for _, group := range joinedGroups {
			groupIds = append(groupIds, group.GroupNo)
		}
	}
	var groups []*group.GroupResp
	var users []*user.UserDetailResp
	if len(groupIds) > 0 {
		groups, err = s.groupService.GetGroupDetails(groupIds, loginUID)
		if err != nil {
			s.Error("查询群列表错误", zap.Error(err))
			c.ResponseError(errors.New("查询群列表错误"))
			return
		}
	}
	if len(uids) > 0 {
		users, err = s.userService.GetUserDetails(uids, loginUID)
		if err != nil {
			s.Error("查询用户列表错误", zap.Error(err))
			c.ResponseError(errors.New("查询用户列表错误"))
			return
		}
	}

	groupResps := make([]*channelResp, 0)
	if len(joinedGroups) > 0 {
		for _, g := range joinedGroups {
			isAdd := false
			remark := ""
			if strings.Contains(g.Name, keyword) {
				isAdd = true
			}
			if len(groups) > 0 {
				for _, group := range groups {
					if group.GroupNo == g.GroupNo {
						remark = group.Remark
						if strings.Contains(group.Remark, keyword) {
							isAdd = true
						}
						break
					}
				}
			}
			if isAdd {
				groupResps = append(groupResps, &channelResp{
					ChannelID:     g.GroupNo,
					ChannelType:   common.ChannelTypeGroup.Uint8(),
					ChannelName:   g.Name,
					ChannelRemark: remark,
				})
			}
		}
	}

	// 查询好友
	friends, err := s.userService.SearchFriendsWithKeyword(loginUID, keyword)
	if err != nil {
		s.Error("查询好友错误", zap.Error(err))
		c.ResponseError(err)
		return
	}
	friendResps := make([]*channelResp, 0)
	if len(friends) > 0 {
		for _, friend := range friends {
			friendResps = append(friendResps, &channelResp{
				ChannelID:     friend.UID,
				ChannelName:   friend.Name,
				ChannelType:   common.ChannelTypePerson.Uint8(),
				ChannelRemark: friend.Remark,
			})
		}
	}
	messagesResp := make([]*messageResp, 0)
	if len(msgResp.Messages) > 0 {
		for _, msg := range msgResp.Messages {
			var isDeleted int8 = 0
			setting := config.SettingFromUint8(msg.Setting)
			var payloadMap map[string]interface{}
			if setting.Signal {
				payloadMap = map[string]interface{}{
					"type": common.SignalError.Int(),
				}
			} else {
				err := util.ReadJsonByByte(msg.Payload, &payloadMap)
				if err != nil {
					log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(msg.Payload)))
				}
				if len(payloadMap) > 0 {
					visibles := payloadMap["visibles"]
					if visibles != nil {
						visiblesArray := visibles.([]interface{})
						if len(visiblesArray) > 0 {
							isDeleted = 1
							for _, limitUID := range visiblesArray {
								if limitUID == loginUID {
									isDeleted = 0
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

			var tempChannel *channelResp
			if msg.ChannelType == common.ChannelTypePerson.Uint8() {
				for _, user := range users {
					if user.UID == msg.ChannelID {
						tempChannel = &channelResp{
							ChannelID:     user.UID,
							ChannelType:   common.ChannelTypePerson.Uint8(),
							ChannelRemark: user.Remark,
							ChannelName:   user.Name,
						}
						break
					}
				}
			}
			if msg.ChannelType == common.ChannelTypeGroup.Uint8() {
				for _, group := range groups {
					if group.GroupNo == msg.ChannelID {
						tempChannel = &channelResp{
							ChannelID:     group.GroupNo,
							ChannelType:   common.ChannelTypeGroup.Uint8(),
							ChannelName:   group.Name,
							ChannelRemark: group.Remark,
						}
						break
					}
				}
			}
			messagesResp = append(messagesResp, &messageResp{
				MessageIDStr: msg.MessageIDStr,
				MessageID:    msg.MessageID,
				MessageSeq:   msg.MessageSeq,
				FromUID:      msg.FromUID,
				Timestamp:    msg.Timestamp,
				Payload:      payloadMap,
				ClientMsgNo:  msg.ClientMsgNo,
				Channel:      tempChannel,
				IsDeleted:    isDeleted,
			})
		}
	}
	c.Response(map[string]interface{}{
		"friends":  friendResps,
		"groups":   groupResps,
		"messages": messagesResp,
	})
}

func (s *Search) search(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	var req struct {
		ContentType []int  `json:"content_type"` // 消息类型
		Keyword     string `json:"keyword"`      // 搜索关键字
		FromUID     string `json:"from_uid"`     // 发送者uid
		ChannelID   string `json:"channel_id"`   // 频道ID
		ChannelType uint8  `json:"channel_type"` // 频道类型
		Topic       string `json:"topic"`        // 根据topic搜索
		Limit       int    `json:"limit"`        // 查询限制数量
		Page        int    `json:"page"`         // 页码，分页使用，默认为1
		StartTime   int64  `json:"start_time"`   //  消息时间（开始）
		EndTime     int64  `json:"end_time"`     // 消息时间（结束，结果不包含end_time）
	}
	if err := c.BindJSON(&req); err != nil {
		s.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	msgResp, err := s.ctx.IMSearchUserMessages(&config.SearchUserMessageReq{
		UID:            loginUID,
		PayloadContent: req.Keyword,
		PayloadTypes:   req.ContentType,
		Limit:          req.Limit,
		Page:           req.Page,
		FromUID:        req.FromUID,
		ChannelID:      req.ChannelID,
		ChannelType:    req.ChannelType,
		Topic:          req.Topic,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
	})
	if err != nil {
		s.Error("查询悟空IM消息错误", zap.Error(err))
		c.ResponseError(errors.New("查询悟空IM消息错误"))
		return
	}
	messages := make([]*messageResp, 0)
	if msgResp == nil || len(msgResp.Messages) == 0 {
		c.Response(messages)
		return
	}
	groupIds := make([]string, 0)
	uids := make([]string, 0)
	for _, m := range msgResp.Messages {
		if m.ChannelType == common.ChannelTypeGroup.Uint8() {
			groupIds = append(groupIds, m.ChannelID)
		} else if m.ChannelType == common.ChannelTypePerson.Uint8() {
			uids = append(uids, m.ChannelID)
		}
	}
	var groups []*group.GroupResp
	var users []*user.UserDetailResp
	if len(groupIds) > 0 {
		groups, err = s.groupService.GetGroupDetails(groupIds, loginUID)
		if err != nil {
			s.Error("查询群列表错误", zap.Error(err))
			c.ResponseError(errors.New("查询群列表错误"))
			return
		}
	}
	if len(uids) > 0 {
		users, err = s.userService.GetUserDetails(uids, loginUID)
		if err != nil {
			s.Error("查询用户列表错误", zap.Error(err))
			c.ResponseError(errors.New("查询用户列表错误"))
			return
		}
	}

	for _, msg := range msgResp.Messages {
		var isDeleted int8 = 0
		setting := config.SettingFromUint8(msg.Setting)
		var payloadMap map[string]interface{}
		if setting.Signal {
			payloadMap = map[string]interface{}{
				"type": common.SignalError.Int(),
			}
		} else {
			err := util.ReadJsonByByte(msg.Payload, &payloadMap)
			if err != nil {
				log.Warn("负荷数据不是json格式！", zap.Error(err), zap.String("payload", string(msg.Payload)))
			}
			if len(payloadMap) > 0 {
				visibles := payloadMap["visibles"]
				if visibles != nil {
					visiblesArray := visibles.([]interface{})
					if len(visiblesArray) > 0 {
						isDeleted = 1
						for _, limitUID := range visiblesArray {
							if limitUID == loginUID {
								isDeleted = 0
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

		var tempChannel *channelResp
		if msg.ChannelType == common.ChannelTypePerson.Uint8() {
			for _, user := range users {
				if user.UID == msg.ChannelID {
					tempChannel = &channelResp{
						ChannelID:     user.UID,
						ChannelType:   common.ChannelTypePerson.Uint8(),
						ChannelRemark: user.Remark,
						ChannelName:   user.Name,
					}
					break
				}
			}
		}
		if msg.ChannelType == common.ChannelTypeGroup.Uint8() {
			for _, group := range groups {
				if group.GroupNo == msg.ChannelID {
					tempChannel = &channelResp{
						ChannelID:     group.GroupNo,
						ChannelType:   common.ChannelTypeGroup.Uint8(),
						ChannelName:   group.Name,
						ChannelRemark: group.Remark,
					}
					break
				}
			}
		}
		messages = append(messages, &messageResp{
			MessageIDStr: msg.MessageIDStr,
			MessageID:    msg.MessageID,
			MessageSeq:   msg.MessageSeq,
			FromUID:      msg.FromUID,
			Timestamp:    msg.Timestamp,
			Payload:      payloadMap,
			ClientMsgNo:  msg.ClientMsgNo,
			Channel:      tempChannel,
			IsDeleted:    isDeleted,
		})
	}
	c.Response(messages)
}

type channelResp struct {
	ChannelID     string `json:"channel_id"`
	ChannelType   uint8  `json:"channel_type"`
	ChannelRemark string `json:"channel_remark"`
	ChannelName   string `json:"channel_name"`
}

type messageResp struct {
	Setting      uint8                  `json:"setting"`          // 设置
	MessageID    int64                  `json:"message_id"`       // 服务端的消息ID(全局唯一)
	MessageIDStr string                 `json:"message_idstr"`    // 服务端的消息ID(全局唯一)字符串形式
	MessageSeq   uint32                 `json:"message_seq"`      // 消息序列号 （用户唯一，有序递增）
	ClientMsgNo  string                 `json:"client_msg_no"`    // 客户端消息唯一编号
	FromUID      string                 `json:"from_uid"`         // 发送者UID
	Expire       uint32                 `json:"expire,omitempty"` // expire
	Timestamp    int32                  `json:"timestamp"`        // 服务器消息时间戳(10位，到秒)
	Payload      map[string]interface{} `json:"payload"`          // 消息内容
	IsDeleted    int8                   `json:"is_deleted"`       // 是否已删除
	Channel      *channelResp           `json:"channel"`          // 消息所属channel
}
