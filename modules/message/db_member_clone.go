package message

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type memberCloneDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newMemberCloneDB(ctx *config.Context) *memberCloneDB {
	return &memberCloneDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (m *memberCloneDB) insertTx(model *memberCloneModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("member_clone").Columns(util.AttrToUnderscore(model)...).Record(model).Exec()
	return err
}

func (m *memberCloneDB) queryWithCloneNo(cloneNo string) ([]*memberCloneModel, error) {
	var models []*memberCloneModel
	_, err := m.session.Select("*").From("member_clone").Where("clone_no=?", cloneNo).Load(&models)
	return models, err
}

// 查询未读列表
func (m *memberCloneDB) queryUnreadWithMessageIDAndPage(cloneNo string, fromUID string, messageID int64, pIndex, pSize uint64) ([]*memberUnreadModel, error) {
	var models []*memberUnreadModel
	_, err := m.session.Select("*").From("member_clone").Where("clone_no=? and uid<>? and uid not in (select member_readed.uid  from member_readed where message_id=?)", cloneNo, fromUID, messageID).Limit(pSize).Offset((pIndex - 1) * pSize).Load(&models)
	return models, err
}

type memberCloneModel struct {
	CloneNo     string
	ChannelID   string
	ChannelType uint8
	UID         string
	db.BaseModel
}

type memberUnreadModel struct {
	CloneNo     string
	ChannelID   string
	ChannelType uint8
	UID         string
}
