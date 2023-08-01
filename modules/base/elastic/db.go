package elastic

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// DB DB
type DB struct {
	session *dbr.Session
}

// NewDB NewDB
func NewDB(session *dbr.Session) *DB {
	return &DB{
		session: session,
	}
}

// Insert Insert
func (d *DB) Insert(model *IndexerErrorModel) error {
	_, err := d.session.InsertInto("indexer_error").Columns(util.AttrToUnderscore(model)...).Record(model).Exec()
	return err
}

// IndexerErrorModel IndexerErrorModel
type IndexerErrorModel struct {
	Index      string
	Action     string
	DocumentID string
	Body       string
	Error      string
	db.BaseModel
}
