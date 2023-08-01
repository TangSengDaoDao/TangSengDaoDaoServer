package message

import (
	"fmt"
	"hash/crc32"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type messageUserExtraDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newMessageUserExtraDB(ctx *config.Context) *messageUserExtraDB {
	return &messageUserExtraDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

// 插入或更新消息为已删除
func (m *messageUserExtraDB) insertOrUpdateDeleted(md *messageUserExtraModel) error {
	sq := fmt.Sprintf("INSERT INTO %s (uid,message_id,message_seq,channel_id,channel_type,message_is_deleted) VALUES (?,?,?,?,?,?) ON DUPLICATE KEY UPDATE  message_is_deleted=VALUES(message_is_deleted)", m.getTable(md.UID))
	_, err := m.session.InsertBySql(sq, md.UID, md.MessageID, md.MessageSeq, md.ChannelID, md.ChannelType, md.MessageIsDeleted).Exec()
	return err
}
func (m *messageUserExtraDB) insertOrUpdateDeletedTx(md *messageUserExtraModel, tx *dbr.Tx) error {
	sq := fmt.Sprintf("INSERT INTO %s (uid,message_id,message_seq,channel_id,channel_type,message_is_deleted) VALUES (?,?,?,?,?,?) ON DUPLICATE KEY UPDATE  message_is_deleted=VALUES(message_is_deleted)", m.getTable(md.UID))
	_, err := tx.InsertBySql(sq, md.UID, md.MessageID, md.MessageSeq, md.ChannelID, md.ChannelType, md.MessageIsDeleted).Exec()
	return err
}

// 插入或更新消息语音已读状态
func (m *messageUserExtraDB) insertOrUpdateVoiceRead(md *messageUserExtraModel) error {
	sq := fmt.Sprintf("INSERT INTO %s (uid,message_id,message_seq,channel_id,channel_type,voice_readed) VALUES (?,?,?,?,?,?) ON DUPLICATE KEY UPDATE  voice_readed=VALUES(voice_readed)", m.getTable(md.UID))
	_, err := m.session.InsertBySql(sq, md.UID, md.MessageID, md.MessageSeq, md.ChannelID, md.ChannelType, md.VoiceReaded).Exec()
	return err
}

// 通过消息id集合和消息拥有者uid查询编辑消息
func (m *messageUserExtraDB) queryWithMessageIDsAndUID(messageIDs []string, uid string) ([]*messageUserExtraModel, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}
	var models []*messageUserExtraModel
	_, err := m.session.Select("*").From(m.getTable(uid)).Where("uid=? and message_id in ?", uid, messageIDs).Load(&models)
	return models, err
}

func (m *messageUserExtraDB) getTable(uid string) string {
	tableIndex := crc32.ChecksumIEEE([]byte(uid)) % uint32(m.ctx.GetConfig().TablePartitionConfig.MessageUserEditTableCount)
	if tableIndex == 0 {
		return "message_user_extra"
	}
	return fmt.Sprintf("message_user_extra%d", tableIndex)
}

type messageUserExtraModel struct {
	UID              string
	MessageID        string
	MessageSeq       uint32
	ChannelID        string
	ChannelType      uint8
	VoiceReaded      int
	MessageIsDeleted int
	db.BaseModel
}
