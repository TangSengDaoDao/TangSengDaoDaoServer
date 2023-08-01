package message

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type userLastOffsetDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newUserLastOffsetDB(ctx *config.Context) *userLastOffsetDB {

	return &userLastOffsetDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (d *userLastOffsetDB) insertOrUpdateTx(tx *dbr.Tx, model *userLastOffsetModel) error {
	sq := "INSERT INTO user_last_offset (uid,channel_id,channel_type,message_seq) VALUES (?,?,?,?) ON DUPLICATE KEY UPDATE message_seq=IF(message_seq<VALUES(message_seq),VALUES(message_seq),message_seq)"
	_, err := tx.InsertBySql(sq, model.UID, model.ChannelID, model.ChannelType, model.MessageSeq).Exec()
	return err
}

func (d *userLastOffsetDB) queryWithUID(uid string) ([]*userLastOffsetModel, error) {
	var models []*userLastOffsetModel
	_, err := d.session.Select("*").From("user_last_offset").Where("uid=?", uid).Load(&models)
	return models, err
}

type userLastOffsetModel struct {
	UID         string
	ChannelID   string
	ChannelType uint8
	MessageSeq  int64
	db.BaseModel
}
