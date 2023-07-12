package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type deviceFlagDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newDeviceFlagDB(ctx *config.Context) *deviceFlagDB {
	return &deviceFlagDB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}

func (d *deviceFlagDB) queryAll() ([]*deviceFlagModel, error) {
	var deviceFlags []*deviceFlagModel
	_, err := d.session.Select("*").From("device_flag").Load(&deviceFlags)
	return deviceFlags, err
}

type deviceFlagModel struct {
	DeviceFlag uint8
	Weight     int
	Remark     string
	db.BaseModel
}
