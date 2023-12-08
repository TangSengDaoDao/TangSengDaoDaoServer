package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type managerDB struct {
	session *dbr.Session
	ctx     *config.Context
}

// newManagerDB
func newManagerDB(ctx *config.Context) *managerDB {
	return &managerDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

// 通过账号和密码查询用户信息
func (m *managerDB) queryUserInfoWithNameAndPwd(username string) (*managerLoginModel, error) {
	var model *managerLoginModel
	_, err := m.session.Select("*").From("user").Where("username=?", username).Load(&model)
	return model, err
}

// 获取用户列表
func (m *managerDB) queryUserListWithPage(pageSize, page uint64, onelineStatus int) ([]*managerUserModel, error) {
	// var users []*managerUserModel
	// _, err := m.session.Select("*").From("user").Offset((page-1)*pageSize).Limit(pageSize).OrderDir("created_at", false).Load(&users)
	// return users, err

	var users []*managerUserModel
	selectStm := m.session.Select("user.uid,user.name,user.username,user.status,user.phone,user.short_no,user.sex,user.is_destroy,user.created_at,user.gitee_uid,user.github_uid,user.wx_openid,max(user_online.online) online").From("user").LeftJoin("user_online", "user.uid=user_online.uid")
	if onelineStatus != -1 {
		selectStm = selectStm.Where("user_online.online=?", onelineStatus)
	}
	selectStm = selectStm.GroupBy("user.uid,user.name,user.username,user.status,user.phone,user.short_no,user.sex,user.is_destroy,user.created_at,user.gitee_uid,user.github_uid,user.wx_openid")

	// select  from user left join user_online on user.uid=user_online.uid where user_online.online=1  group by user.uid,user.name,user.status,user.phone,user.short_no,user.sex,user.is_destroy,user.created_at  limit 100
	_, err := selectStm.Offset((page-1)*pageSize).Limit(pageSize).OrderDir("user.created_at", false).Load(&users)
	return users, err
}

// 模糊查询用户列表
// onelineStatus 在线状态 -1 为所有 0. 离线 1. 在线
func (m *managerDB) queryUserListWithPageAndKeyword(keyword string, onelineStatus int, pageSize, page uint64) ([]*managerUserModel, error) {
	var users []*managerUserModel
	selectStm := m.session.Select("user.uid,user.name,user.username,user.status,user.phone,user.short_no,user.sex,user.is_destroy,user.created_at,user.gitee_uid,user.github_uid,user.wx_openid,max(user_online.online) online").From("user").LeftJoin("user_online", "user.uid=user_online.uid").Where("user.name like ? or user.uid like ? or user.phone like ? or user.short_no like ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	if onelineStatus != -1 {
		selectStm = selectStm.Where("user_online.online=?", onelineStatus)
	}
	selectStm = selectStm.GroupBy("user.uid,user.name,user.username,user.status,user.phone,user.short_no,user.sex,user.is_destroy,user.created_at,user.gitee_uid,user.github_uid,user.wx_openid")

	// select  from user left join user_online on user.uid=user_online.uid where user_online.online=1  group by user.uid,user.name,user.status,user.phone,user.short_no,user.sex,user.is_destroy,user.created_at  limit 100
	_, err := selectStm.Offset((page-1)*pageSize).Limit(pageSize).OrderDir("user.created_at", false).Load(&users)
	return users, err
}

// 模糊查询用户数量
func (m *managerDB) queryUserCountWithKeyWord(keyword string) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("user").Where("name like ? or uid like ? or phone like ? or short_no like ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").Load(&count)
	return count, err
}

// queryUserBlacklist 查询某个用户的黑名单
func (m *managerDB) queryUserBlacklists(uid string) ([]*managerUserBlacklistModel, error) {
	var users []*managerUserBlacklistModel
	_, err := m.session.Select("`user`.*,IFNULL(user_setting.updated_at,'') ").From("`user`").LeftJoin(`user_setting`, "user.uid=user_setting.to_uid and user_setting.blacklist=1").Where("`user_setting`.uid=?", uid).Load(&users)
	return users, err
}

// 通过status查询用户列表
func (m *managerDB) queryUserListWithStatus(status int, pageSize, page uint64) ([]*managerUserModel, error) {
	var users []*managerUserModel
	_, err := m.session.Select("*").From("user").Where("status=?", status).Offset((page-1)*pageSize).Limit(pageSize).OrderDir("updated_at", false).Load(&users)
	return users, err
}

// 通过status查询用户数量
func (m *managerDB) queryUserCountWithStatus(status int) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("user").Where("status=?", status).Load(&count)
	return count, err
}

func (m *managerDB) queryUserOnline(uid string) ([]*userOnline, error) {
	var list []*userOnline
	_, err := m.session.Select("*").From("user_online").Where("uid=?", uid).Load(&list)
	return list, err
}

func (m *managerDB) queryUserWithNameAndRole(username string, role string) (*managerUserModel, error) {
	var user *managerUserModel
	_, err := m.session.Select("*").From("user").Where("username=? and role=?", username, role).Load(&user)
	return user, err
}

func (m *managerDB) queryUsersWithRole(role string) ([]*managerUserModel, error) {
	var list []*managerUserModel
	_, err := m.session.Select("*").From("user").Where("role=?", role).Load(&list)
	return list, err
}
func (m *managerDB) deleteUserWithUIDAndRole(uid, role string) error {
	_, err := m.session.DeleteFrom("user").Where("uid=? and role=?", uid, role).Exec()
	return err
}

type managerLoginModel struct {
	Username string
	UID      string
	Name     string
	Password string
	Role     string
}

type managerUserModel struct {
	Username  string
	Name      string
	UID       string
	Status    int
	Phone     string
	ShortNo   string
	WXOpenid  string // 微信openid
	GiteeUID  string // gitee uid
	GithubUID string // github uid
	Sex       int
	IsDestroy int
	db.BaseModel
}

type managerUserBlacklistModel struct {
	Name string
	UID  string
	db.BaseModel
}

type userOnline struct {
	UID         string
	DeviceFlag  uint8 // 设备标记 0. APP 1.web
	LastOnline  int   // 最后一次在线时间
	LastOffline int   // 最后一次离线时间
	Online      int
	Version     int64 // 数据版本
	db.BaseModel
}
