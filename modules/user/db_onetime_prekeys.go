package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type onetimePrekeysDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newOnetimePrekeysDB(ctx *config.Context) *onetimePrekeysDB {
	return &onetimePrekeysDB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}

func (o *onetimePrekeysDB) insertTx(m *onetimePrekeysModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("signal_onetime_prekeys").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (o *onetimePrekeysDB) delete(uid string, keyID int) error {
	_, err := o.session.DeleteFrom("signal_onetime_prekeys").Where("uid=? and key_id=?", uid, keyID).Exec()
	return err
}

func (o *onetimePrekeysDB) deleteWithUID(uid string) error {
	_, err := o.session.DeleteFrom("signal_onetime_prekeys").Where("uid=?", uid).Exec()
	return err
}

// 查询用户最小的onetimePreKey
func (o *onetimePrekeysDB) queryMinWithUID(uid string) (*onetimePrekeysModel, error) {
	var m *onetimePrekeysModel
	_, err := o.session.Select("*").From("signal_onetime_prekeys").Where("uid=?", uid).OrderAsc("key_id").Limit(1).Load(&m)
	return m, err
}

func (o *onetimePrekeysDB) queryCount(uid string) (int, error) {
	var cn int
	err := o.session.Select("count(*)").From("signal_onetime_prekeys").Where("uid=?", uid).LoadOne(&cn)
	return cn, err
}

type onetimePrekeysModel struct {
	UID    string
	KeyID  int
	Pubkey string
	db.BaseModel
}
