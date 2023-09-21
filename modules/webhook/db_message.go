package webhook

import (
	"fmt"
	"hash/crc32"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type messageDB struct {
	ctx *config.Context
	db  *dbr.Session
}

func newMessageDB(ctx *config.Context) *messageDB {

	return &messageDB{
		ctx: ctx,
		db:  ctx.DB(),
	}
}

func (m *messageDB) insertOrUpdateTx(model *messageModel, tx *dbr.Tx) error {
	tbl := m.getTable(model.ChannelID)
	_, err := tx.InsertBySql(fmt.Sprintf("insert into %s(message_id,message_seq,client_msg_no,header,setting,`signal`,from_uid,channel_id,channel_type,expire,expire_at,timestamp,payload,is_deleted) values(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE payload=payload", tbl), model.MessageID, model.MessageSeq, model.ClientMsgNo, model.Header, model.Setting, model.Signal, model.FromUID, model.ChannelID, model.ChannelType, model.Expire, model.ExpireAt, model.Timestamp, model.Payload, model.IsDeleted).Exec()
	return err
}

// 通过频道ID获取表
func (m *messageDB) getTable(channelID string) string {
	tableIndex := crc32.ChecksumIEEE([]byte(channelID)) % uint32(m.ctx.GetConfig().TablePartitionConfig.MessageTableCount)
	if tableIndex == 0 {
		return "message"
	}
	return fmt.Sprintf("message%d", tableIndex)
}

type messageModel struct {
	MessageID   string
	MessageSeq  int64
	ClientMsgNo string
	Header      string
	Setting     uint8
	Signal      uint8 // 是否signal加密
	FromUID     string
	ChannelID   string
	ChannelType uint8
	Expire      uint32
	ExpireAt    uint32
	Timestamp   int32
	Payload     string
	IsDeleted   int
	db.BaseModel
}
