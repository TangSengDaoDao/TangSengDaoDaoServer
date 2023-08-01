package user

import (
	"context"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type deviceDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newDeviceDB(ctx *config.Context) *deviceDB {
	return &deviceDB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}

// 添加或更新设备
func (d *deviceDB) insertOrUpdateDevice(m *deviceModel) error {
	_, err := d.session.InsertBySql("insert into device(uid,device_id,device_name,device_model,last_login) values(?,?,?,?,?) ON DUPLICATE KEY UPDATE device_name=VALUES(device_name),device_model=VALUES(device_model),last_login=VALUES(last_login)", m.UID, m.DeviceID, m.DeviceName, m.DeviceModel, m.LastLogin).Exec()
	return err
}
func (d *deviceDB) insertOrUpdateDeviceCtx(ctx context.Context, m *deviceModel) error {
	span, _ := d.ctx.Tracer().StartSpanFromContext(ctx, "insertOrUpdateDevice")
	defer span.Finish()
	return d.insertOrUpdateDevice(m)
}

// 添加或更新设备
func (d *deviceDB) insertOrUpdateDeviceTx(m *deviceModel, tx *dbr.Tx) error {
	_, err := tx.InsertBySql("insert into device(uid,device_id,device_name,device_model,last_login) values(?,?,?,?,?) ON DUPLICATE KEY UPDATE device_name=VALUES(device_name),device_model=VALUES(device_model),last_login=VALUES(last_login)", m.UID, m.DeviceID, m.DeviceName, m.DeviceModel, m.LastLogin).Exec()
	return err
}

// 获取用户设备列表
func (d *deviceDB) queryDeviceWithUID(uid string) ([]*deviceModel, error) {
	var devices []*deviceModel
	_, err := d.session.Select("*").From("device").Where("uid=?", uid).OrderDir("last_login", false).Load(&devices)
	return devices, err
}

// 是否存在指定用户的指定设备
func (d *deviceDB) existDeviceWithDeviceIDAndUID(deviceID, uid string) (bool, error) {
	var count int
	_, err := d.session.Select("count(*)").From("device").Where("device_id=? and uid=?", deviceID, uid).Load(&count)
	return count > 0, err
}

// 是否存在指定用户的指定设备
func (d *deviceDB) existDeviceWithDeviceIDAndUIDCtx(ctx context.Context, deviceID, uid string) (bool, error) {
	span, _ := d.ctx.Tracer().StartSpanFromContext(ctx, "existDeviceWithDeviceIDAndUID")
	defer span.Finish()
	return d.existDeviceWithDeviceIDAndUID(deviceID, uid)
}

// 更新设备最后一次登录时间
func (d *deviceDB) updateDeviceLastLogin(lastLogin int64, deviceID, uid string) error {
	_, err := d.session.Update("device").SetMap(map[string]interface{}{
		"last_login": lastLogin,
	}).Where("device_id=? and uid=?", deviceID, uid).Exec()
	return err
}

// 更新设备最后一次登录时间
func (d *deviceDB) updateDeviceLastLoginCtx(ctx context.Context, lastLogin int64, deviceID, uid string) error {
	span, _ := d.ctx.Tracer().StartSpanFromContext(ctx, "updateDeviceLastLogin")
	defer span.Finish()
	return d.updateDeviceLastLogin(lastLogin, deviceID, uid)
}

// 通过设备ID删除设备
func (d *deviceDB) deleteDeviceWithDeviceIDAndUID(deviceID string, uid string) error {
	_, err := d.session.DeleteFrom("device").Where("device_id=? and uid=?", deviceID, uid).Exec()
	return err
}

// 查询最后一次登录的设备
// func (d *deviceDB) queryDeviceLastLogin(uid string) (*deviceModel, error) {
// 	var m *deviceModel
// 	_, err := d.session.Select("*").From("device").OrderDir("last_login", false).Where("uid=?", uid).Limit(1).Load(&m)
// 	return m, err
// }

// 查询一批最后一次登录的设备信息
func (d *deviceDB) queryDeviceLastLoginWithUids(uids []string) ([]*deviceModel, error) {

	var list []*deviceModel
	_, err := d.session.SelectBySql("select * from device where id in ( select max(id) from device group by uid having uid in ?)", uids).Load(&list)
	//_, err := d.session.Select("*").From("device").Where("uid in ?", uids).OrderDir("last_login", false).Limit(1).Load(&list)
	return list, err
}

type deviceModel struct {
	UID         string // 设备属于用户的uid
	DeviceID    string // 设备唯一ID
	DeviceName  string // 设备名称
	DeviceModel string // 设备型号
	LastLogin   int64  // 最后一次登录时间
	db.BaseModel
}
