package config

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"go.uber.org/zap"
)

func (c *Context) SendChannelUpdateWithFromUID(channel ChannelReq, updateChannel ChannelReq, fromUID string) error {
	// 发送一个频道更新命令 发给自己的其他设备，如果其他设备在线的话
	err := c.SendCMD(MsgCMDReq{
		ChannelID:   channel.ChannelID,
		ChannelType: channel.ChannelType,
		FromUID:     fromUID,
		CMD:         common.CMDChannelUpdate,
		Param: map[string]interface{}{
			"channel_id":   updateChannel.ChannelID,
			"channel_type": updateChannel.ChannelType,
		},
	})
	if err != nil {
		c.Error("发送频道更新命令失败！", zap.Error(err))
		return err
	}
	return nil
}

// SendChannelUpdate 发送频道更新命令
func (c *Context) SendChannelUpdate(channel ChannelReq, updateChannel ChannelReq) error {

	return c.SendChannelUpdateWithFromUID(channel, updateChannel, "")
}
func (c *Context) SendChannelUpdateToGroup(groupNo string) error {
	channelReq := ChannelReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
	}
	return c.SendChannelUpdateWithFromUID(channelReq, channelReq, "")
}

func (c *Context) SendChannelUpdateToUser(uid string, channel ChannelReq) error {

	return c.SendChannelUpdateWithFromUID(ChannelReq{
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
	}, channel, "")
}
