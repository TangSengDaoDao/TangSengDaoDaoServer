package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// LoginLogDB 登录日志DB
type LoginLogDB struct {
	session *dbr.Session
}

// NewLoginLogDB NewDB
func NewLoginLogDB(session *dbr.Session) *LoginLogDB {
	return &LoginLogDB{
		session: session,
	}
}

// insert 添加登录日志
func (l *LoginLogDB) insert(m *LoginLogModel) error {
	_, err := l.session.InsertInto("login_log").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

// queryLastLoginIP 查询最后一次登录日志
func (l *LoginLogDB) queryLastLoginIP(uid string) (*LoginLogModel, error) {
	var model *LoginLogModel
	_, err := l.session.Select("*").From("login_log").Where("uid=?", uid).OrderDir("created_at", false).Limit(1).Load(&model)
	if err != nil {
		return nil, err
	}
	return model, nil
}

// LoginLogModel 登录日志
type LoginLogModel struct {
	LoginIP string //登录IP
	UID     string
	db.BaseModel
}
