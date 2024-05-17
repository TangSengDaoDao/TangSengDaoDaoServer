package message

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type pinnedDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newPinnedDB(ctx *config.Context) *pinnedDB {
	return &pinnedDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}
func (d *pinnedDB) queryWithMessageId(channelID string, channelType uint8, messageId string) (*pinnedMessageModel, error) {
	var model *pinnedMessageModel
	_, err := d.session.Select("*").From("pinned_message").Where("channel_id=? and channel_type=? and message_id=?", channelID, channelType, messageId).Load(&model)
	return model, err
}

func (d *pinnedDB) queryWithMessageIds(channelID string, channelType uint8, messageIds []string) ([]*pinnedMessageModel, error) {
	var list []*pinnedMessageModel
	_, err := d.session.Select("*").From("pinned_message").Where("channel_id=? and channel_type=? and message_id in ?", channelID, channelType, messageIds).Load(&list)
	return list, err
}
func (d *pinnedDB) insert(m *pinnedMessageModel) error {
	_, err := d.session.InsertInto("pinned_message").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *pinnedDB) update(m *pinnedMessageModel) error {
	_, err := d.session.Update("pinned_message").SetMap(map[string]interface{}{
		"is_deleted": m.IsDeleted,
		"version":    m.Version,
	}).Where("message_id=?", m.MessageId).Exec()
	return err
}

func (d *pinnedDB) updateTx(m *pinnedMessageModel, tx *dbr.Tx) error {
	_, err := tx.Update("pinned_message").SetMap(map[string]interface{}{
		"is_deleted": m.IsDeleted,
		"version":    m.Version,
	}).Where("message_id=?", m.MessageId).Exec()
	return err
}
func (d *pinnedDB) queryWithUnDeletedMessage(channelID string, channelType uint8) ([]*pinnedMessageModel, error) {
	var list []*pinnedMessageModel
	_, err := d.session.Select("*").From("pinned_message").Where("channel_id=? and channel_type=? and is_deleted=0", channelID, channelType).Load(&list)
	return list, err
}
func (d *pinnedDB) queryWithChannelIDAndVersion(channelID string, channelType uint8, version int64) ([]*pinnedMessageModel, error) {
	var list []*pinnedMessageModel
	_, err := d.session.Select("*").From("pinned_message").Where("channel_id=? and channel_type=? and version>?", channelID, channelType, version).Load(&list)
	return list, err
}

func (d *pinnedDB) queryCountWithChannel(channelID string, channelType uint8) (int64, error) {
	var cn int64
	_, err := d.session.Select("count(*)").From("pinned_message").Where("channel_id=? and channel_type=? and is_deleted=0", channelID, channelType).Load(&cn)
	return cn, err
}

type pinnedMessageModel struct {
	MessageId   string
	ChannelID   string
	ChannelType uint8
	MessageSeq  uint32
	IsDeleted   int8
	Version     int64
	db.BaseModel
}
