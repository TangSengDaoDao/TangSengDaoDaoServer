package message

import (
	"errors"
	"strings"
)

type deleteReq struct {
	MessageID   string `json:"message_id"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	MessageSeq  uint32 `json:"message_seq"`
}

func (d *deleteReq) check() error {
	if strings.TrimSpace(d.MessageID) == "" {
		return errors.New("消息ID不能为空！")
	}
	if strings.TrimSpace(d.ChannelID) == "" {
		return errors.New("频道ID不能为空！")
	}
	if d.ChannelType == 0 {
		return errors.New("频道类型不能为空！")
	}
	if d.MessageSeq == 0 {
		return errors.New("消息序号不能为空！")
	}
	return nil
}

type voiceReadedReq struct {
	deleteReq
}
