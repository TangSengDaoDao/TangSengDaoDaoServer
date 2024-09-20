package message

import (
	"sort"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type messageExtraDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newMessageExtraDB(ctx *config.Context) *messageExtraDB {
	return &messageExtraDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (m *messageExtraDB) insertTx(md *messageExtraModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("message_extra").Columns(util.AttrToUnderscore(md)...).Record(md).Exec()
	return err
}

func (m *messageExtraDB) updateTx(md *messageExtraModel, tx *dbr.Tx) error {
	_, err := tx.Update("message_extra").SetMap(map[string]interface{}{
		"readed_count": md.ReadedCount,
		"version":      md.Version,
		"revoke":       md.Revoke,
		"revoker":      md.Revoker,
	}).Where("message_id=?", md.MessageID).Exec()
	return err
}

func (m *messageExtraDB) insertOrUpdateContentEditTx(md *messageExtraModel, tx *dbr.Tx) error {
	_, err := tx.InsertBySql("INSERT INTO message_extra (message_id,message_seq,channel_id,channel_type,content_edit,content_edit_hash,edited_at,version) VALUES (?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE content_edit=VALUES(content_edit),content_edit_hash=VALUES(content_edit_hash),edited_at=VALUES(edited_at),version=VALUES(version)", md.MessageID, md.MessageSeq, md.ChannelID, md.ChannelType, md.ContentEdit, md.ContentEditHash, md.EditedAt, md.Version).Exec()
	return err
}

func (m *messageExtraDB) insertOrUpdatePinnedTx(md *messageExtraModel, tx *dbr.Tx) error {
	_, err := tx.InsertBySql("INSERT INTO message_extra (message_id,message_seq,channel_id,channel_type,is_pinned,version) VALUES (?,?,?,?,?,?) ON DUPLICATE KEY UPDATE is_pinned=VALUES(is_pinned),version=VALUES(version)", md.MessageID, md.MessageSeq, md.ChannelID, md.ChannelType, md.IsPinned, md.Version).Exec()
	return err
}

func (m *messageExtraDB) insertOrUpdateDeleted(md *messageExtraModel) error {
	_, err := m.session.InsertBySql("INSERT INTO message_extra (message_id,message_seq,channel_id,channel_type,is_deleted,version) VALUES (?,?,?,?,?,?) ON DUPLICATE KEY UPDATE is_deleted=VALUES(is_deleted),version=VALUES(version)", md.MessageID, md.MessageSeq, md.ChannelID, md.ChannelType, md.IsDeleted, md.Version).Exec()
	return err
}

// 是否存在相同编辑内容
func (m *messageExtraDB) existContentEdit(messageID string, contentEditHash string) (bool, error) {
	var count int
	err := m.session.Select("count(*)").From("message_extra").Where("message_id=? and content_edit_hash=?", messageID, contentEditHash).LoadOne(&count)
	return count > 0, err
}

func (m *messageExtraDB) queryWithMessageIDsAndUID(messageIDs []string, loginUID string) ([]*messageExtraDetailModel, error) {
	if len(messageIDs) <= 0 {
		return nil, nil
	}
	var models []*messageExtraDetailModel
	_, err := m.session.Select("message_extra.*,(select count(*) from member_readed where member_readed.message_id=message_extra.message_id and member_readed.uid='"+loginUID+"') readed,(select created_at from member_readed where member_readed.message_id=message_extra.message_id and member_readed.uid='"+loginUID+"') readed_at").From("message_extra").Where("message_id in ?", messageIDs).Load(&models)
	return models, err
}

func (m *messageExtraDB) queryWithMessageIDs(messageIDs []string) ([]*messageExtraModel, error) {
	var list []*messageExtraModel
	_, err := m.session.Select("*").From("message_extra").Where("message_id in ?", messageIDs).Load(&list)
	return list, err
}

func (m *messageExtraDB) queryWithMessageID(messageID string) (*messageExtraModel, error) {
	var model *messageExtraModel
	_, err := m.session.Select("*").From("message_extra").Where("message_id=?", messageID).Load(&model)
	return model, err
}

func (m *messageExtraDB) sync(version int64, channelID string, channelType uint8, limit uint64, loginUID string) ([]*messageExtraDetailModel, error) {
	var models []*messageExtraDetailModel
	selectSql := "message_extra.*,(select count(*) from member_readed where member_readed.message_id=message_extra.message_id and member_readed.uid='" + loginUID + "') readed,(select created_at from member_readed where member_readed.message_id=message_extra.message_id and member_readed.uid='" + loginUID + "') readed_at"
	builder := m.session.Select(selectSql).From("message_extra")
	var err error
	if version == 0 {
		builder = builder.Where("channel_id=? and channel_type=?", channelID, channelType).OrderDesc("version").Limit(limit)
		_, err = builder.Load(&models)
		newModels := messageExtraDetailModelSlice(models)
		sort.Sort(newModels)
		models = newModels
	} else {
		builder = builder.Where("channel_id=? and channel_type=? and version>?", channelID, channelType, version).OrderAsc("version").Limit(limit)
		_, err = builder.Load(&models)
	}

	return models, err
}

type messageExtraDetailModelSlice []*messageExtraDetailModel

func (m messageExtraDetailModelSlice) Len() int {
	return len(m)
}
func (m messageExtraDetailModelSlice) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m messageExtraDetailModelSlice) Less(i, j int) bool { return m[i].Version < m[j].Version }

type messageExtraDetailModel struct {
	messageExtraModel
	Readed   int          // 是否已读（针对于自己）
	ReadedAt dbr.NullTime // 已读时间

}

type messageExtraModel struct {
	MessageID       string
	MessageSeq      uint32
	FromUID         string
	ChannelID       string
	ChannelType     uint8
	Revoke          int
	Revoker         string // 消息撤回者的uid
	CloneNo         string
	ReadedCount     int            // 已读数量
	ContentEdit     dbr.NullString // 编辑后的正文
	ContentEditHash string
	EditedAt        int // 编辑时间 时间戳（秒）
	IsDeleted       int
	Version         int64 // 数据版本
	IsPinned        int   // 是否置顶
	db.BaseModel
}
