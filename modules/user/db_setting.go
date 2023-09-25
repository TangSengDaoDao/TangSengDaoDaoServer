package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// SettingDB 设置db
type SettingDB struct {
	session *dbr.Session
}

// NewSettingDB NewDB
func NewSettingDB(session *dbr.Session) *SettingDB {
	return &SettingDB{
		session: session,
	}
}

// InsertUserSettingModel 插入用户设置
func (d *SettingDB) InsertUserSettingModel(setting *SettingModel) error {
	_, err := d.session.InsertInto("user_setting").Columns(util.AttrToUnderscore(setting)...).Record(setting).Exec()
	return err
}

// InsertUserSettingModelTx 插入用户设置
func (d *SettingDB) InsertUserSettingModelTx(setting *SettingModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("user_setting").Columns(util.AttrToUnderscore(setting)...).Record(setting).Exec()
	return err
}

// QueryUserSettingModel 查询用户设置
func (d *SettingDB) QueryUserSettingModel(uid, loginUID string) (*SettingModel, error) {
	var model *SettingModel
	_, err := d.session.Select("*").From("user_setting").Where("uid=? and to_uid=?", loginUID, uid).Load(&model)
	if err != nil {
		return nil, err
	}
	return model, nil
}

// QueryTwoUserSettingModel 查询双方用户设置
func (d *SettingDB) QueryTwoUserSettingModel(uid, loginUID string) ([]*SettingModel, error) {
	var models []*SettingModel
	_, err := d.session.Select("*").From("user_setting").Where("(uid=? and to_uid=?) or (uid=? and to_uid=?)", loginUID, uid, uid, loginUID).Load(&models)
	if err != nil {
		return nil, err
	}
	return models, nil
}

func (d *SettingDB) QueryWithUidsAndToUID(uids []string, toUID string) ([]*SettingModel, error) {
	var models []*SettingModel
	_, err := d.session.Select("*").From("user_setting").Where("uid in ? and to_uid=?", uids, toUID).Load(&models)
	return models, err
}

func (d *SettingDB) QueryUserSettings(uids []string, loginUID string) ([]*SettingModel, error) {
	var models []*SettingModel
	_, err := d.session.Select("*").From("user_setting").Where("uid=? and to_uid in ?", loginUID, uids).Load(&models)
	if err != nil {
		return nil, err
	}
	return models, nil
}

// updateUserSettingModel 更新用户设置
func (d *SettingDB) updateUserSettingModelWithToUIDTx(setting *SettingModel, uid string, toUID string, tx *dbr.Tx) error {
	_, err := tx.Update("user_setting").SetMap(map[string]interface{}{
		"mute":          setting.Mute,
		"top":           setting.Top,
		"blacklist":     setting.Blacklist,
		"chat_pwd_on":   setting.ChatPwdOn,
		"screenshot":    setting.Screenshot,
		"revoke_remind": setting.RevokeRemind,
		"receipt":       setting.Receipt,
		"flame":         setting.Flame,
		"flame_second":  setting.FlameSecond,
		"remark":        setting.Remark,
	}).Where("uid=? and to_uid=?", uid, toUID).Exec()
	return err
}

// UpdateUserSettingModel 更新用户设置
func (d *SettingDB) UpdateUserSettingModel(setting *SettingModel) error {
	_, err := d.session.Update("user_setting").SetMap(map[string]interface{}{
		"mute":          setting.Mute,
		"top":           setting.Top,
		"version":       setting.Version,
		"chat_pwd_on":   setting.ChatPwdOn,
		"screenshot":    setting.Screenshot,
		"revoke_remind": setting.RevokeRemind,
		"receipt":       setting.Receipt,
		"flame":         setting.Flame,
		"flame_second":  setting.FlameSecond,
		"remark":        setting.Remark,
	}).Where("id=?", setting.Id).Exec()
	return err
}

func (d *SettingDB) querySettingByUIDAndToUID(uid, toUID string) (*SettingModel, error) {
	var setting *SettingModel
	_, err := d.session.Select("*").From("user_setting").Where("uid=? and to_uid=?", uid, toUID).Load(&setting)
	return setting, err
}

// ------------ model ------------

// SettingModel 用户设置
type SettingModel struct {
	UID          string // 用户UID
	ToUID        string // 对方uid
	Mute         int    // 免打扰
	Top          int    // 置顶
	ChatPwdOn    int    // 是否开启聊天密码
	Screenshot   int    //截屏通知
	RevokeRemind int    //撤回提醒
	Blacklist    int    //黑名单
	Receipt      int    //消息是否回执
	Flame        int    // 是否开启阅后即焚
	FlameSecond  int    // 阅后即焚秒数
	Version      int64  // 版本
	Remark       string // 备注
	db.BaseModel
}

func newDefaultSettingModel() *SettingModel {
	return &SettingModel{
		Screenshot:   1,
		RevokeRemind: 1,
		Receipt:      1,
	}
}
