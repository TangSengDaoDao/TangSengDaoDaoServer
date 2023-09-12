package app

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// DB DB
type DB struct {
	session *dbr.Session
}

func newDB(session *dbr.Session) *DB {
	return &DB{
		session: session,
	}
}

func (d *DB) queryWithAppID(appID string) (*model, error) {
	var m *model
	_, err := d.session.Select("*").From("app").Where("app_id=?", appID).Load(&m)
	return m, err
}

func (d *DB) existWithAppID(appID string) (bool, error) {
	var count int
	_, err := d.session.Select("count(*)").From("app").Where("app_id=?", appID).Load(&count)
	return count > 0, err
}

func (d *DB) insert(m *model) error {
	_, err := d.session.InsertInto("app").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

type model struct {
	AppID   string
	AppKey  string
	AppName string
	AppLogo string
	Status  int
	db.BaseModel
}
