package common

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	dbs "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type shortnoDB struct {
	ctx *config.Context
	db  *dbr.Session
}

func newShortnoDB(ctx *config.Context) *shortnoDB {
	return &shortnoDB{
		ctx: ctx,
		db:  ctx.DB(),
	}
}

func (s *shortnoDB) inserts(shortnos []string) error {
	if len(shortnos) == 0 {
		return nil
	}
	tx, _ := s.db.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	for _, st := range shortnos {
		_, err := tx.InsertBySql("insert into shortno(shortno) values(?)", st).Exec()
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}
	return nil

}

func (s *shortnoDB) queryVail() (*shortnoModel, error) {
	var m *shortnoModel
	_, err := s.db.Select("*").From("shortno").Where("used=0 and hold=0 and locked=0").Limit(1).Load(&m)
	return m, err
}

func (s *shortnoDB) updateLock(shortno string, lock int) error {
	_, err := s.db.Update("shortno").Set("locked", lock).Where("shortno=?", shortno).Exec()
	return err
}

func (s *shortnoDB) updateUsed(shortno string, used int, business string) error {
	_, err := s.db.Update("shortno").Set("used", used).Set("business", business).Where("shortno=?", shortno).Exec()
	return err
}
func (s *shortnoDB) updateHold(shortno string, hold int) error {
	_, err := s.db.Update("shortno").Set("hold", hold).Where("shortno=?", shortno).Exec()
	return err
}

func (s *shortnoDB) queryVailCount() (int64, error) {
	var cn int64
	_, err := s.db.Select("count(*)").From("shortno").Where("used=0 and hold=0 and locked=0").Load(&cn)
	return cn, err
}

type shortnoModel struct {
	Shortno  string
	Used     int
	Hold     int
	Locked   int
	Business string
	dbs.BaseModel
}
