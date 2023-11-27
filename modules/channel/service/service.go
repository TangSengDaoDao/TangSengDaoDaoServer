package service

type IService interface {
	// 获取频道设置集合
	GetChannelSettings(channelIDs []string) ([]*ChannelSettingResp, error)
	// 创建或更新频道消息自动删除时间
	CreateOrUpdateMsgAutoDelete(channelID string, channelType uint8, msgAutoDelete int64) error
}

type ChannelSettingResp struct {
	ChannelID         string
	ChannelType       uint8
	ParentChannelID   string
	ParentChannelType uint8
}
