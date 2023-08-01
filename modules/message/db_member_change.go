package message

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type memberChangeDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newMemberChangeDB(ctx *config.Context) *memberChangeDB {
	return &memberChangeDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

// 查询频道成员最大版本号
func (m *memberChangeDB) queryMaxVersion(channelID string, channelType uint8) (*memberChangeModel, error) {
	var model *memberChangeModel
	_, err := m.session.Select("*").From("member_change").Where("channel_id=? and channel_type=?", channelID, channelType).OrderDesc("max_version").Limit(1).Load(&model)
	return model, err
}

func (m *memberChangeDB) insertTx(model *memberChangeModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("member_change").Columns(util.AttrToUnderscore(model)...).Record(model).Exec()
	return err
}

func (m *memberChangeDB) updateMaxVersion(maxVersion int64, channelID string, channelType uint8) error {
	_, err := m.session.Update("member_change").Set("max_version", maxVersion).Where("channel_id=? and channel_type=?", channelID, channelType).Exec()
	return err
}

type memberChangeModel struct {
	CloneNo     string
	ChannelID   string
	ChannelType uint8
	MaxVersion  int64
	db.BaseModel
}
