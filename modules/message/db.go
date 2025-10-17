package message

import (
	"fmt"
	"hash/crc32"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
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

func (d *DB) queryMaxMessageSeq(channelID string, channelType uint8) (uint32, error) {
	var maxMessageSeq uint32
	err := d.session.Select("IFNULL(max(message_seq),0)").From(d.getTable(channelID)).Where("channel_id=? and channel_type=?", channelID, channelType).LoadOne(&maxMessageSeq)
	return maxMessageSeq, err
}

func (d *DB) queryProhibitWordsWithVersion(version int64) ([]*ProhibitWordModel, error) {
	var list []*ProhibitWordModel
	_, err := d.session.Select("*").From("prohibit_words").Where("`version` > ?", version).Load(&list)
	return list, err
}

// 新增消息
func (d *DB) insertMessage(m *messageModel) error {
	_, err := d.session.InsertInto(d.getTable(m.ChannelID)).Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
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
	// Type        int
	Payload   []byte
	IsDeleted int
	Signal    int
	Expire    uint32
	db.BaseModel
}
