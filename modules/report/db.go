package report

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	dba "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type db struct {
	session *dbr.Session
	ctx     *config.Context
}

func newDB(ctx *config.Context) *db {
	return &db{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (d *db) queryCategoryAll() ([]*categoryModel, error) {
	var models []*categoryModel
	_, err := d.session.Select("*").From("report_category").Load(&models)
	return models, err
}

func (d *db) insertCategory(m *categoryModel) error {
	_, err := d.session.InsertInto("report_category").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *db) insert(m *model) error {
	_, err := d.session.InsertInto("report").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

type categoryModel struct {
	CategoryNo       string
	CategoryName     string
	CategoryEname    string // 英文分类名称
	ParentCategoryNo string
	dba.BaseModel
}

type model struct {
	UID         string
	CategoryNo  string
	ChannelID   string
	ChannelType uint8
	Imgs        string
	Remark      string
	dba.BaseModel
}
