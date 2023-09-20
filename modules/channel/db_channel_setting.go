package channel

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type channelSettingDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newChannelSettingDB(ctx *config.Context) *channelSettingDB {
	return &channelSettingDB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}

func (c *channelSettingDB) queryWithChannel(channelID string, channelType uint8) (*channelSettingModel, error) {
	var m *channelSettingModel
	_, err := c.session.Select("*").From("channel_setting").Where("channel_id=? and channel_type=?", channelID, channelType).Load(&m)
	return m, err
}

func (c *channelSettingDB) queryWithChannelIDs(channelIDs []string) ([]*channelSettingModel, error) {
	var models []*channelSettingModel
	_, err := c.session.Select("*").From("channel_setting").Where("channel_id in ?", channelIDs).Load(&models)
	return models, err
}

func (c *channelSettingDB) insertOrAddMsgAutoDelete(channelID string, channelType uint8, msgAutoDelete int64) error {
	_, err := c.session.InsertBySql("insert into channel_setting (channel_id, channel_type, msg_auto_delete) values (?, ?, ?) ON DUPLICATE KEY UPDATE msg_auto_delete=VALUES(msg_auto_delete)", channelID, channelType, msgAutoDelete).Exec()
	return err
}

type channelSettingModel struct {
	ChannelID         string
	ChannelType       uint8
	ParentChannelID   string
	ParentChannelType uint8
	MsgAutoDelete     int64
	db.BaseModel
}
