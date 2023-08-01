package message

import (
	"fmt"
	"hash/crc32"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

// DB DB
type DB struct {
	session *dbr.Session
	ctx     *config.Context
}

// NewDB NewDB
func NewDB(ctx *config.Context) *DB {
	return &DB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}

// InsertTx 添加消息
// func (d *DB) InsertTx(m *Model, tx *dbr.Tx) error {
// 	_, err := tx.InsertInto("message").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
// 	return err
// }

func (d *DB) queryMessageWithMessageID(channelID string, channelType uint8, messageID string) (*messageModel, error) {
	var m *messageModel
	_, err := d.session.Select("*").From(d.getTable(channelID)).Where("message_id=?", messageID).Load(&m)
	return m, err
}

func (d *DB) queryMessagesWithMessageIDs(channelID string, channelType uint8, messageIDs []string) ([]*messageModel, error) {
	if len(messageIDs) <= 0 {
		return nil, nil
	}
	var models []*messageModel
	_, err := d.session.Select("*").From(d.getTable(channelID)).Where("message_id in ?", messageIDs).Load(&models)
	return models, err
}

func (d *DB) queryMaxMessageSeq(channelID string, channelType uint8) (uint32, error) {
	var maxMessageSeq uint32
	err := d.session.Select("IFNULL(max(message_seq),0)").From(d.getTable(channelID)).Where("channel_id=? and channel_type=?", channelID, channelType).LoadOne(&maxMessageSeq)
	return maxMessageSeq, err
}

func (d *DB) queryMessagesWithChannelClientMsgNo(channelID string, channelType uint8, clientMsgNo string) ([]*messageModel, error) {
	var models []*messageModel
	_, err := d.session.Select("*").From(d.getTable(channelID)).Where("channel_id=? and channel_type=? and client_msg_no=?", channelID, channelType, clientMsgNo).Load(&models)
	return models, err
}
func (d *DB) queryProhibitWordsWithVersion(version int64) ([]*ProhibitWordModel, error) {
	var list []*ProhibitWordModel
	_, err := d.session.Select("*").From("prohibit_words").Where("`version` > ?", version).Load(&list)
	return list, err
}

// 通过频道ID获取表
func (d *DB) getTable(channelID string) string {
	tableIndex := crc32.ChecksumIEEE([]byte(channelID)) % uint32(d.ctx.GetConfig().TablePartitionConfig.MessageTableCount)
	if tableIndex == 0 {
		return "message"
	}
	return fmt.Sprintf("message%d", tableIndex)
}

// ProhibitWordModel 违禁词model
type ProhibitWordModel struct {
	Content   string
	IsDeleted int
	Version   int64
	db.BaseModel
}

// Model 消息model
type messageModel struct {
	MessageID   int64
	MessageSeq  uint32
	ClientMsgNo string
	Header      string
	Setting     uint8
	FromUID     string
	ChannelID   string
	ChannelType uint8
	Timestamp   int64
	Type        int
	Payload     []byte
	IsDeleted   int
	Signal      int
	db.BaseModel
}
