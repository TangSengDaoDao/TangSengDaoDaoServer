package message

import (
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type deviceOffsetDB struct {
	session *dbr.Session
}

func newDeviceOffsetDB(session *dbr.Session) *deviceOffsetDB {
	return &deviceOffsetDB{
		session: session,
	}
}

func (d *deviceOffsetDB) insertOrUpdateTx(tx *dbr.Tx, model *deviceOffsetModel) error {
	sq := fmt.Sprintf("INSERT INTO device_offset (uid,device_uuid,channel_id,channel_type,message_seq) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE message_seq=IF(message_seq<VALUES(message_seq),VALUES(message_seq),message_seq)")
	_, err := tx.InsertBySql(sq, model.UID, model.DeviceUUID, model.ChannelID, model.ChannelType, model.MessageSeq).Exec()
	return err
}

func (d *deviceOffsetDB) queryWithUIDAndDeviceUUID(uid string, deviceUUID string) ([]*deviceOffsetModel, error) {
	var models []*deviceOffsetModel
	_, err := d.session.Select("*").From("device_offset").Where("uid=? and device_uuid=?", uid, deviceUUID).Load(&models)
	return models, err
}

func (d *deviceOffsetDB) queryMessageSeq(uid string, deviceUUID string, channelID string, channelType uint8) (int64, error) {
	var messageSeq int64
	_, err := d.session.Select("IFNULL(message_seq,0)").From("device_offset").Where("uid=? and device_uuid=? and channel_id=? and channel_type=?", uid, deviceUUID, channelID, channelType).Limit(1).Load(&messageSeq)
	return messageSeq, err
}

type deviceOffsetModel struct {
	UID         string
	DeviceUUID  string
	ChannelID   string
	ChannelType uint8
	MessageSeq  int64
	db.BaseModel
}
