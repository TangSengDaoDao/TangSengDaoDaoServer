package user

import (
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// DB DB
type friendDB struct {
	session *dbr.Session
	ctx     *config.Context
}

// NewDB NewDB
func newFriendDB(ctx *config.Context) *friendDB {
	return &friendDB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}

// InsertTx 插入好友信息
func (d *friendDB) InsertTx(m *FriendModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("friend").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	if err != nil {
		return err
	}
	friendKey := fmt.Sprintf("%s%s", CacheKeyFriends, m.UID)
	err = d.ctx.GetRedisConn().SAdd(friendKey, m.ToUID)
	return err
}

// Insert 插入好友信息
func (d *friendDB) Insert(m *FriendModel) error {
	_, err := d.session.InsertInto("friend").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	if err != nil {
		return err
	}
	friendKey := fmt.Sprintf("%s%s", CacheKeyFriends, m.UID)
	err = d.ctx.GetRedisConn().SAdd(friendKey, m.ToUID)
	return err
}

// IsFriend 是否是好友
func (d *friendDB) IsFriend(uid, toUID string) (bool, error) {
	var m *FriendModel
	_, err := d.session.Select("*").From("friend").Where("uid=? and to_uid=?", uid, toUID).Load(&m)
	if err != nil {
		return false, err
	}
	var isFriend = false
	if m != nil && m.IsDeleted == 0 {
		isFriend = true
	}
	return isFriend, nil
}

// 修改好友关系
func (d *friendDB) updateRelationshipTx(uid, toUID string, isDeleted, isAlone int, sourceVercode string, version int64, tx *dbr.Tx) error {
	_, err := tx.Update("friend").SetMap(map[string]interface{}{
		"is_deleted":     isDeleted,
		"is_alone":       isAlone,
		"source_vercode": sourceVercode,
		"version":        version,
	}).Where("uid=? and to_uid=?", uid, toUID).Exec()
	if err != nil {
		return err
	}
	friendKey := fmt.Sprintf("%s%s", CacheKeyFriends, uid)
	if isDeleted == 1 {
		err = d.ctx.GetRedisConn().SRem(friendKey, toUID)
	} else {
		err = d.ctx.GetRedisConn().SAdd(friendKey, toUID)
	}

	return err
}

func (d *friendDB) updateRelationship2Tx(uid, toUID string, isDeleted, isAlone int, version int64, tx *dbr.Tx) error {
	_, err := tx.Update("friend").SetMap(map[string]interface{}{
		"is_deleted": isDeleted,
		"is_alone":   isAlone,
		"version":    version,
	}).Where("uid=? and to_uid=?", uid, toUID).Exec()
	if err != nil {
		return err
	}
	friendKey := fmt.Sprintf("%s%s", CacheKeyFriends, uid)
	if isDeleted == 1 {
		err = d.ctx.GetRedisConn().SRem(friendKey, toUID)
	} else {
		err = d.ctx.GetRedisConn().SAdd(friendKey, toUID)
	}

	return err
}

// 修改好友单项关系
func (d *friendDB) updateAloneTx(uid, toUID string, isAlone int, tx *dbr.Tx) error {
	_, err := tx.Update("friend").Set("is_alone", isAlone).Where("uid=? and to_uid=?", uid, toUID).Exec()
	return err
}

// 删除好友
// func (d *friendDB) delete(uid, toUID string) error {
// 	_, err := d.session.DeleteFrom("friend").Where("uid=? and to_uid=?", uid, toUID).Exec()
// 	if err != nil {
// 		return err
// 	}
// 	friendKey := fmt.Sprintf("%s%s", CacheKeyFriends, uid)
// 	err = d.ctx.GetRedisConn().SRem(friendKey, toUID)
// 	return err
// }

// 删除好友
// func (d *friendDB) deleteTx(uid, toUID string, tx *dbr.Tx) error {
// 	_, err := tx.Update("friend").SetMap(map[string]interface{}{
// 		"is_deleted": 1,
// 		"is_alone":   1,
// 	}).Where("uid=? and to_uid=?", uid, toUID).Exec()

// 	//_, err := tx.DeleteFrom("friend").Where("uid=? and to_uid=?", uid, toUID).Exec()
// 	if err != nil {
// 		return err
// 	}
// 	friendKey := fmt.Sprintf("%s%s", CacheKeyFriends, uid)
// 	err = d.ctx.GetRedisConn().SRem(friendKey, toUID)
// 	return err
// }

// 通过vercode查询好友信息
func (d *friendDB) queryWithVercode(vercode string) (*FriendModel, error) {
	var friend *FriendModel
	_, err := d.session.Select("*").From("friend").Where("vercode=?", vercode).Load(&friend)
	return friend, err
}

// 通过vercode查询好友信息
func (d *friendDB) queryWithVercodes(vercodes []string) ([]*FriendDetailModel, error) {
	var friends []*FriendDetailModel
	_, err := d.session.Select("friend.*,IFNULL(user.name,'') name").From("friend").LeftJoin("user", "friend.uid=user.uid").Where("friend.vercode in ?", vercodes).Load(&friends)
	return friends, err
}

// 查询某个好友
func (d *friendDB) queryWithUID(uid, toUID string) (*FriendModel, error) {
	var friend *FriendModel
	_, err := d.session.Select("*").From("friend").Where("uid=? and to_uid=?", uid, toUID).Load(&friend)
	return friend, err
}

// 查询双方好友
func (d *friendDB) queryTwoWithUID(uid, toUID string) ([]*FriendModel, error) {
	var friends []*FriendModel
	_, err := d.session.Select("*").From("friend").Where("(uid=? and to_uid=?) or (uid=? and to_uid=?)", uid, toUID, toUID, uid).Load(&friends)
	return friends, err
}

// 查询指定用户uid的在toUids范围内的好友
func (d *friendDB) queryWithToUIDsAndUID(toUids []string, uid string) ([]*FriendModel, error) {
	var friends []*FriendModel
	_, err := d.session.Select("*").From("friend").Where("uid=? and to_uid in ?", uid, toUids).Load(&friends)
	return friends, err
}

// 查询uids范围内的用户与toUID是好友的数据
func (d *friendDB) queryWithToUIDAndUIDs(toUID string, uids []string) ([]*FriendModel, error) {
	var friends []*FriendModel
	_, err := d.session.Select("*").From("friend").Where("to_uid=? and uid in ?", toUID, uids).Load(&friends)
	return friends, err
}

// QueryFriendsWithKeyword 通过关键字查询自己的好友
func (d *friendDB) QueryFriendsWithKeyword(uid string, keyword string) ([]*DetailModel, error) {
	var details []*DetailModel
	builder := d.session.Select("friend.id,friend.to_uid,IFNULL(user.name,'') to_name,friend.is_deleted,friend.created_at,friend.updated_at,IFNULL(user_setting.mute,0) mute,IFNULL(user_setting.top,0) top,IFNULL(user_setting.version,0)+friend.version version").From("friend").LeftJoin("user", "friend.to_uid=user.uid").LeftJoin("user_setting", "user.uid=user_setting.to_uid and user_setting.uid=friend.uid").Where("friend.uid=?", uid).OrderDir("friend.version + IFNULL(user_setting.version,0)", true)
	if keyword != "" {
		builder = builder.Where("user.name like ?", "%"+keyword+"%")
	}
	_, err := builder.Load(&details)
	return details, err
}

// SyncFriendsOfDeprecated 同步好友
// Deprecated 已废弃，用SyncFriends方法。
func (d *friendDB) SyncFriendsOfDeprecated(version int64, uid string, limit uint64) ([]*DetailModel, error) {
	var details []*DetailModel
	builder := d.session.Select("friend.id,IFNULL(friend.vercode,'') vercode,friend.to_uid,IFNULL(user.name,'') to_name,IFNULL(user.category,'') to_category,IFNULL(user.robot,0) robot,IFNULL(user.short_no,'') short_no,IFNULL(friend.remark,'') remark,friend.is_deleted,friend.created_at,friend.updated_at,IFNULL(user_setting.mute,0) mute,IFNULL(user_setting.chat_pwd_on,0) chat_pwd_on,IFNULL(user_setting.blacklist,0) blacklist,IFNULL(user_setting.top,0) top,IFNULL(user_setting.receipt,0) receipt,friend.version + IFNULL(user_setting.version,0) version").From("friend").LeftJoin("user", "friend.to_uid=user.uid").LeftJoin("user_setting", "user.uid=user_setting.to_uid and user_setting.uid=friend.uid").Where("friend.uid=?", uid).OrderDir("friend.version + IFNULL(user_setting.version,0)", true)
	var err error
	if version <= 0 {
		_, err = builder.Limit(limit).Load(&details)
	} else {
		_, err = builder.Where("IFNULL(user_setting.version,0) + friend.version > ?", version).Limit(limit).Load(&details)
	}
	return details, err
}

func (d *friendDB) SyncFriends(version int64, uid string, limit uint64) ([]*FriendModel, error) {
	var models []*FriendModel
	builder := d.session.Select("*").From("friend").Where("friend.uid=?", uid).OrderDir("friend.version", true)
	_, err := builder.Where("friend.version > ?", version).Limit(limit).Load(&models)
	return models, err
}

// QueryFriends 查询用户的所有好友
func (d *friendDB) QueryFriends(uid string) ([]*DetailModel, error) {
	var details []*DetailModel
	_, err := d.session.Select("friend.*,IFNULL(user.name,'') to_name").From("friend").LeftJoin("user", "user.uid=friend.to_uid").Where("friend.uid=? and friend.is_deleted=0", uid).Load(&details)
	return details, err
}

// QueryFriendsWithUIDs 通过用户id查询好友
func (d *friendDB) QueryFriendsWithUIDs(uid string, toUIDs []string) ([]*FriendDetailModel, error) {
	var friends []*FriendDetailModel
	_, err := d.session.Select("friend.*,IFNULL(user.name,'') to_name").From("friend").LeftJoin("user", "user.uid=friend.to_uid").Where("friend.uid=? and friend.is_deleted=0 and friend.to_uid in ?", uid, toUIDs).Load(&friends)
	return friends, err
}

func (d *friendDB) updateVersionTx(version int64, uid string, toUID string, tx *dbr.Tx) error {
	_, err := tx.Update("friend").Set("version", version).Where("uid=? and to_uid=?", uid, toUID).Exec()
	return err
}

func (d *friendDB) existBlacklist(uid string, toUID string) (bool, error) {
	var cn int
	_, err := d.session.Select("count(*)").From("user_setting").Where("((uid=? and to_uid=?) or (uid=? and to_uid=?)) and blacklist=1", uid, toUID, toUID, uid).Load(&cn)
	return cn > 0, err
}
func (d *friendDB) insertApplyTx(m *FriendApplyModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("friend_apply_record").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *friendDB) insertApply(m *FriendApplyModel) error {
	_, err := d.session.InsertInto("friend_apply_record").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *friendDB) queryApplysWithPage(uid string, pageSize, page uint64) ([]*FriendApplyModel, error) {
	var list []*FriendApplyModel
	_, err := d.session.Select("*").From("friend_apply_record").Where("uid=?", uid).Offset((page-1)*pageSize).Limit(pageSize).OrderDir("created_at", false).Load(&list)
	return list, err
}

func (d *friendDB) deleteApplyWithUidAndToUid(uid, toUid string) error {
	_, err := d.session.DeleteFrom("friend_apply_record").Where("uid=? and to_uid=?", uid, toUid).Exec()
	return err
}
func (d *friendDB) queryApplyWithUidAndToUid(uid, toUid string) (*FriendApplyModel, error) {
	var apply *FriendApplyModel
	_, err := d.session.Select("*").From("friend_apply_record").Where("uid=? and to_uid=?", uid, toUid).Load(&apply)
	return apply, err
}

func (d *friendDB) updateApply(apply *FriendApplyModel) error {
	_, err := d.session.Update("friend_apply_record").SetMap(map[string]interface{}{
		"status": apply.Status,
	}).Where("id=?", apply.Id).Exec()
	return err
}

func (d *friendDB) updateApplyTx(apply *FriendApplyModel, tx *dbr.Tx) error {
	_, err := tx.Update("friend_apply_record").SetMap(map[string]interface{}{
		"status": apply.Status,
	}).Where("id=?", apply.Id).Exec()
	return err
}

// DetailModel 好友详情
type DetailModel struct {
	Remark     string //好友备注
	ToUID      string // 好友uid
	ToName     string // 好友名字
	ToCategory string // 用户分类
	Mute       int    // 免打扰
	Top        int    // 置顶
	Version    int64  // 版本
	Vercode    string // 验证码 加好友需要
	IsDeleted  int    // 是否删除
	IsAlone    int    // 是否为单项好友
	ShortNo    string //短编号
	ChatPwdOn  int    // 是否开启聊天密码
	Blacklist  int    //是否在黑名单
	Receipt    int    //消息是否回执
	Robot      int    // 机器人0.否1.是
	db.BaseModel
}

// FriendModel 好友对象
type FriendModel struct {
	UID           string
	ToUID         string
	Flag          int
	Version       int64
	IsDeleted     int
	IsAlone       int // 是否为单项好友
	Vercode       string
	SourceVercode string //来源验证码
	Initiator     int    //1:发起方
	db.BaseModel
}

// FriendDetailModel 好友资料
type FriendDetailModel struct {
	FriendModel
	Name   string // 用户名称
	ToName string //对方用户名称
}

// FriendApplyModel 好友申请记录
type FriendApplyModel struct {
	UID    string
	ToUID  string
	Remark string
	Token  string
	Status int // 状态 0.未处理 1.通过 2.拒绝
	db.BaseModel
}
