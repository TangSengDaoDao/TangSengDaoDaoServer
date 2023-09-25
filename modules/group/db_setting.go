package group

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// DB DB
type settingDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newSettingDB(ctx *config.Context) *settingDB {
	return &settingDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

// QuerySetting 查询设置
func (s *settingDB) QuerySetting(groupNo, uid string) (*Setting, error) {
	var setting *Setting
	_, err := s.session.Select("*").From("group_setting").Where("group_no=? and uid=?", groupNo, uid).Load(&setting)
	return setting, err
}

func (s *settingDB) querySettingWithTx(groupNo, uid string, tx *dbr.Tx) (*Setting, error) {
	var setting *Setting
	_, err := tx.Select("*").From("group_setting").Where("group_no=? and uid=?", groupNo, uid).Load(&setting)
	return setting, err
}

func (s *settingDB) QuerySettings(groupNos []string, uid string) ([]*Setting, error) {
	var settings []*Setting
	_, err := s.session.Select("*").From("group_setting").Where("group_no in ? and uid=?", groupNos, uid).Load(&settings)
	return settings, err
}
func (s *settingDB) QuerySettingsWithUIDs(groupNo string, uids []string) ([]*Setting, error) {
	var settings []*Setting
	_, err := s.session.Select("*").From("group_setting").Where("group_no=? and uid in ?", groupNo, uids).Load(&settings)
	return settings, err
}

// InsertSetting 添加设置
func (s *settingDB) InsertSetting(setting *Setting) error {
	_, err := s.session.InsertInto("group_setting").Columns(util.AttrToUnderscore(setting)...).Record(setting).Exec()
	return err
}

// InsertSettingTx 添加设置
func (s *settingDB) InsertSettingTx(setting *Setting, tx *dbr.Tx) error {
	_, err := tx.InsertInto("group_setting").Columns(util.AttrToUnderscore(setting)...).Record(setting).Exec()
	return err
}

// UpdateSetting 更新设置
func (s *settingDB) UpdateSetting(setting *Setting) error {
	_, err := s.session.Update("group_setting").SetMap(map[string]interface{}{
		"chat_pwd_on":       setting.ChatPwdOn,
		"mute":              setting.Mute,
		"top":               setting.Top,
		"save":              setting.Save,
		"show_nick":         setting.ShowNick,
		"group_no":          setting.GroupNo,
		"uid":               setting.UID,
		"version":           setting.Version,
		"revoke_remind":     setting.RevokeRemind,
		"join_group_remind": setting.JoinGroupRemind,
		"screenshot":        setting.Screenshot,
		"receipt":           setting.Receipt,
		"flame":             setting.Flame,
		"flame_second":      setting.FlameSecond,
		"remark":            setting.Remark,
	}).Where("id=?", setting.Id).Exec()
	return err
}

// UpdateSetting 更新设置
func (s *settingDB) UpdateSettingWithTx(setting *Setting, tx *dbr.Tx) error {
	_, err := tx.Update("group_setting").SetMap(map[string]interface{}{
		"chat_pwd_on":       setting.ChatPwdOn,
		"mute":              setting.Mute,
		"top":               setting.Top,
		"save":              setting.Save,
		"show_nick":         setting.ShowNick,
		"group_no":          setting.GroupNo,
		"uid":               setting.UID,
		"version":           setting.Version,
		"revoke_remind":     setting.RevokeRemind,
		"join_group_remind": setting.JoinGroupRemind,
		"screenshot":        setting.Screenshot,
		"receipt":           setting.Receipt,
		"flame":             setting.Flame,
		"flame_second":      setting.FlameSecond,
		"remark":            setting.Remark,
	}).Where("id=?", setting.Id).Exec()
	return err
}

// Setting 群设置
type Setting struct {
	UID             string // 用户uid
	GroupNo         string // 群编号
	Mute            int    // 免打扰
	Top             int    // 置顶
	ShowNick        int    // 显示昵称
	Save            int    // 是否保存
	ChatPwdOn       int    //是否开启聊天密码
	Screenshot      int    //截屏通知
	RevokeRemind    int    //撤回通知
	JoinGroupRemind int    //进群提醒
	Receipt         int    //消息是否回执
	Flame           int    // 是否开启阅后即焚
	FlameSecond     int    // 阅后即焚秒数
	Remark          string // 群备注
	Version         int64  // 版本
	db.BaseModel
}

func newDefaultSetting() *Setting {
	return &Setting{
		RevokeRemind: 1,
		Screenshot:   1,
		Receipt:      1,
	}
}
