package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type maillistDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newMaillistDB(ctx *config.Context) *maillistDB {
	return &maillistDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (d *maillistDB) insertTx(m *maillistModel, tx *dbr.Tx) error {
	_, err := tx.InsertBySql("INSERT INTO user_maillist (zone,phone,name,vercode,uid) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE `phone`=VALUES(`phone`)", m.Zone, m.Phone, m.Name, m.Vercode, m.UID).Exec()
	return err
}

func (d *maillistDB) queryWitchVercode(vercode string) (*maillistModel, error) {
	var model *maillistModel
	_, err := d.session.Select("*").From("user_maillist").Where("vercode=? ", vercode).Load(&model)
	return model, err
}
func (d *maillistDB) query(uid string) ([]*maillistModel, error) {
	var models []*maillistModel
	_, err := d.session.Select("*").From("user_maillist").Where("uid=?", uid).Load(&models)
	return models, err
}

type maillistModel struct {
	UID     string
	Phone   string
	Zone    string
	Name    string
	Vercode string
	db.BaseModel
}
