package channel

import "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"

type IService interface {
	// 获取频道设置集合
	GetChannelSettings(channelIDs []string) ([]*ChannelSettingResp, error)
}

type service struct {
	ctx              *config.Context
	channelSettingDB *channelSettingDB
}

func NewService(ctx *config.Context) IService {
	return &service{
		ctx:              ctx,
		channelSettingDB: newChannelSettingDB(ctx),
	}
}

func (s *service) GetChannelSettings(channelIDs []string) ([]*ChannelSettingResp, error) {
	channelSettingModels, err := s.channelSettingDB.queryWithChannelIDs(channelIDs)
	if err != nil {
		return nil, err
	}
	channelSettingResps := make([]*ChannelSettingResp, 0, len(channelSettingModels))
	if len(channelSettingModels) > 0 {
		for _, channelSettingM := range channelSettingModels {
			channelSettingResps = append(channelSettingResps, newChannelSettingResp(channelSettingM))
		}
	}
	return channelSettingResps, nil
}

type ChannelSettingResp struct {
	ChannelID         string
	ChannelType       uint8
	ParentChannelID   string
	ParentChannelType uint8
}

func newChannelSettingResp(m *channelSettingModel) *ChannelSettingResp {

	return &ChannelSettingResp{
		ChannelID:         m.ChannelID,
		ChannelType:       m.ChannelType,
		ParentChannelID:   m.ParentChannelID,
		ParentChannelType: m.ParentChannelType,
	}
}
