package event

import (
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"github.com/gocraft/dbr/v2"
)

// DB 事件的db
type DB struct {
	session *dbr.Session
}

// NewDB 创建DB
func NewDB(session *dbr.Session) *DB {
	return &DB{
		session: session,
	}
}

// InsertTx 插入事件
func (d *DB) InsertTx(m *Model, tx *dbr.Tx) (int64, error) {
	result, err := tx.InsertInto("event").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateStatus 更新事件状态
func (d *DB) UpdateStatus(reason string, status int, versionLock int64, id int64) error {
	_, err := d.session.Update("event").Set("status", status).Set("reason", reason).Where("id=? and version_lock=?", id, versionLock).Exec()
	return err
}

// QueryWithID 根据id查询事件
func (d *DB) QueryWithID(id int64) (*Model, error) {
	var model *Model
	_, err := d.session.Select("*").From("event").Where("id=?", id).Load(&model)
	return model, err
}

// QueryAllWait 查询所有等待事件
func (d *DB) QueryAllWait(limit uint64) ([]*Model, error) {
	var models []*Model
	_, err := d.session.Select("*").From("event").Where("status=? and created_at<?", wkevent.Wait.Int(), util.ToyyyyMMddHHmmss(time.Now().Add(-time.Second*60))).Limit(limit).Load(&models)
	return models, err
}

// ---------- model ----------

// Model 数据库对象
type Model struct {
	Event       string // 事件标示
	Type        int    // 事件类型
	Data        string // 事件数据
	Status      int    // 事件状态 0.待发布 1.已发布 2.发布失败！
	Reason      string // 原因 如果状态为2，则有发布失败的原因
	VersionLock int64  // 乐观锁
	db.BaseModel
}
