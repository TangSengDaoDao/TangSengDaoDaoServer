package config

import (
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
)

// IMStreamStart 消息流开始
// 返回流编号
func (c *Context) IMStreamStart(req MessageStreamStartReq) (string, error) {

	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/streammessage/start", []byte(util.ToJson(req)), nil)
	if err != nil {
		return "", err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return "", err
	}
	var resultMap map[string]interface{}
	err = util.ReadJsonByByte([]byte(resp.Body), &resultMap)
	if err != nil {
		return "", err
	}
	if resultMap == nil {
		return "", errors.New("result is nil")
	}
	return resultMap["stream_no"].(string), nil
}

func (c *Context) IMStreamEnd(req MessageStreamEndReq) error {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/streammessage/end", []byte(util.ToJson(req)), nil)
	if err != nil {
		return err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return err
	}
	return nil

}

type MessageStreamStartReq struct {
	Header      MsgHeader `json:"header"`        // 消息头
	ClientMsgNo string    `json:"client_msg_no"` // 客户端消息编号（相同编号，客户端只会显示一条）
	FromUID     string    `json:"from_uid"`      // 发送者UID
	ChannelID   string    `json:"channel_id"`    // 频道ID
	ChannelType uint8     `json:"channel_type"`  // 频道类型
	Payload     []byte    `json:"payload"`       // 消息内容
}

type MessageStreamEndReq struct {
	StreamNo    string `json:"stream_no"`    // 消息流编号
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
}
