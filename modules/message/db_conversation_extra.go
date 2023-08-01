package message

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type conversationExtraDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newConversationExtraDB(ctx *config.Context) *conversationExtraDB {

	return &conversationExtraDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (c *conversationExtraDB) insertOrUpdate(model *conversationExtraModel) error {
	_, err := c.session.InsertBySql("INSERT INTO conversation_extra (uid,channel_id,channel_type,browse_to,keep_message_seq,keep_offset_y,draft,version) VALUES (?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE browse_to=IF(VALUES(browse_to)>browse_to,VALUES(browse_to),browse_to),`keep_message_seq`=VALUES(`keep_message_seq`),keep_offset_y=VALUES(keep_offset_y),draft=VALUES(draft),version=VALUES(version)", model.UID, model.ChannelID, model.ChannelType, model.BrowseTo, model.KeepMessageSeq, model.KeepOffsetY, model.Draft, model.Version).Exec()
	return err
}

func (c *conversationExtraDB) sync(uid string, version int64) ([]*conversationExtraModel, error) {
	var models []*conversationExtraModel
	_, err := c.session.Select("*").From("conversation_extra").Where("uid=? and version>?", uid, version).Load(&models)
	return models, err
}

func (c *conversationExtraDB) queryWithChannelIDs(uid string, channelIDs []string) ([]*conversationExtraModel, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	var models []*conversationExtraModel
	_, err := c.session.Select("*").From("conversation_extra").Where("uid=? and channel_id in ?", uid, channelIDs).Load(&models)
	return models, err
}

type conversationExtraModel struct {
	UID            string
	ChannelID      string
	ChannelType    uint8
	BrowseTo       uint32
	KeepMessageSeq uint32
	KeepOffsetY    int
	Draft          string // 草稿
	Version        int64
	db.BaseModel
}
