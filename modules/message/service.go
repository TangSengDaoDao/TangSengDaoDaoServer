package message

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
)

type IService interface {
	DeleteConversation(uid string, channelID string, channelType uint8) error
}

type Service struct {
	ctx *config.Context
	log.Log
}

func NewService(ctx *config.Context) *Service {

	return &Service{
		ctx: ctx,
		Log: log.NewTLog("message.Service"),
	}
}

func (s *Service) DeleteConversation(uid string, channelID string, channelType uint8) error {
	err := s.ctx.IMDeleteConversation(config.DeleteConversationReq{
		ChannelID:   channelID,
		ChannelType: uint8(channelType),
		UID:         uid,
	})
	if err != nil {
		return err
	}
	err = s.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
		CMD:         common.CMDConversationDeleted,
		Param: map[string]interface{}{
			"channel_id":   channelID,
			"channel_type": channelType,
		},
	})
	if err != nil {
		return err
	}

	return nil
}
