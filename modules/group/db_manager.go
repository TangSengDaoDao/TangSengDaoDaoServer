package group

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

// managerDB managerDB
type managerDB struct {
	session *dbr.Session
}

// newManagerDB
func newManagerDB(session *dbr.Session) *managerDB {
	return &managerDB{
		session: session,
	}
}

// 查询群列表
func (m *managerDB) listWithPage(pageSize, page uint64) ([]*managerGroupModel, error) {
	var list []*managerGroupModel
	_, err := m.session.Select("*").From("`group`").Offset((page-1)*pageSize).Limit(pageSize).OrderDir("created_at", false).Load(&list)
	return list, err
}

// 模糊查询群列表
func (m *managerDB) listWithPageAndKeyword(keyword string, pageSize, page uint64) ([]*managerGroupModel, error) {
	var list []*managerGroupModel
	_, err := m.session.Select("*").From("`group`").Where("name like ? or group_no like ?", "%"+keyword+"%", "%"+keyword+"%").Offset((page-1)*pageSize).Limit(pageSize).OrderDir("created_at", false).Load(&list)
	return list, err
}

// 通过关键字查询群总数
func (m *managerDB) queryGroupCountWithKeyWord(keyword string) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("`group`").Where("name like ? or group_no like ?", "%"+keyword+"%", "%"+keyword+"%").Load(&count)
	return count, err
}

// 查询群成员数量
func (m *managerDB) queryGroupsMemberCount(groupNos []string) ([]*managerGroupCountModel, error) {
	var list []*managerGroupCountModel
	_, err := m.session.SelectBySql("select  *,(select count(*) from group_member m where m.group_no=g.group_no) member_count from `group` g where group_no in ?", groupNos).Load(&list)
	return list, err
}

// 查询群总数
func (m *managerDB) queryGroupCountWithStatus(status int) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("`group`").Where("status=?", status).Load(&count)
	return count, err
}

// 通过status查询群列表
func (m *managerDB) queryGroupsWithStatus(status int, pageSize, pageIndex uint64) ([]*managerGroupModel, error) {
	var list []*managerGroupModel
	_, err := m.session.Select("*").From("`group`").Where("status=?", status).Offset((pageIndex-1)*pageSize).Limit(pageSize).OrderDir("created_at", false).Load(&list)
	return list, err
}

// 查询某个区间的注册数量
func (m *managerDB) queryRegisterCountWithDateSpace(startDate, endDate string) ([]*managerGroupModel, error) {
	var models []*managerGroupModel
	_, err := m.session.Select("*").From("`group`").Where("date_format(created_at,'%Y-%m-%d')>=? and date_format(created_at,'%Y-%m-%d')<=?", startDate, endDate).OrderDir("created_at", false).Load(&models)
	return models, err
}

// 群成员
func (m *managerDB) queryGroupMembers(groupNo string, pageSize, pageIndex uint64) ([]*managerMemberModel, error) {
	var list []*managerMemberModel
	_, err := m.session.Select("group_member.*,user.name").From("group_member").LeftJoin("user", "user.uid=group_member.uid").Where("group_member.group_no=? and group_member.is_deleted=0 and group_member.status=1", groupNo).Offset((pageIndex - 1) * pageSize).Limit(pageSize).OrderBy("group_member.role=1 desc,group_member.role=2 desc,group_member.created_at asc").Load(&list)
	return list, err
}

// 模糊查询群成员
func (m *managerDB) queryGroupMembersWithKeyWord(groupNo string, keyword string, pageSize, pageIndex uint64) ([]*managerMemberModel, error) {
	var list []*managerMemberModel
	_, err := m.session.Select("group_member.*,user.name").From("group_member").Join("user", "user.uid=group_member.uid").Where("group_member.group_no=? and group_member.is_deleted=0 and group_member.status=1 and user.name like ?", groupNo, "%"+keyword+"%").Offset((pageIndex - 1) * pageSize).Limit(pageSize).OrderBy("group_member.role=1 desc,group_member.role=2 desc,group_member.created_at asc").Load(&list)
	return list, err
}

// 查询某个群成员数量
func (m *managerDB) queryGroupMemberCount(groupNo string) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("group_member").Where("group_no=? and is_deleted=0 and status=1", groupNo).Load(&count)
	return count, err
}

// 模糊查询群成员数量
func (m *managerDB) queryGroupMemberCountWithKeyword(groupNo, keyword string) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("group_member").LeftJoin("user", "group_member.uid=user.uid").Where("group_member.group_no=? and group_member.is_deleted=0 and group_member.status=1 and user.name like ?", groupNo, "%"+keyword+"%").Load(&count)
	return count, err
}

// 通过status查询群成员
func (m *managerDB) queryGroupMembersWithStatus(groupNo string, status int, pageSize, pageIndex uint64) ([]*managerMemberModel, error) {
	var list []*managerMemberModel
	_, err := m.session.Select("group_member.*,user.name").From("group_member").LeftJoin("user", "user.uid=group_member.uid").Where("group_member.group_no=? and group_member.is_deleted=0 and group_member.status=?", groupNo, status).Offset((pageIndex - 1) * pageSize).Limit(pageSize).OrderBy("group_member.role=1 desc,group_member.role=2 desc,group_member.created_at asc").Load(&list)
	return list, err
}

// 通过status查询群成员数量
func (m *managerDB) queryGroupMemberCountWithStatus(groupNo string, status int) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("group_member").Where("group_no=? and status=?", groupNo, status).Load(&count)
	return count, err
}

type managerGroupModel struct {
	GroupNo            string // 群编号
	Name               string // 群名称
	Notice             string // 群公告
	Creator            string // 创建者uid
	Status             int    // 群状态
	Version            int64  // 版本号
	Forbidden          int    // 是否全员禁言
	Invite             int    // 是否开启邀请确认 0.否 1.是
	ForbiddenAddFriend int    //群内禁止加好友
	db.BaseModel
}
type managerGroupCountModel struct {
	MemberCount int    // 群成员数量
	GroupNo     string // 群ID
}

// managerMemberModel 成员model
type managerMemberModel struct {
	GroupNo   string // 群编号
	Name      string // 用户名称
	UID       string // 成员uid
	Remark    string // 成员备注
	Role      int    // 成员角色
	Version   int64
	Vercode   string //验证码
	IsDeleted int    // 是否删除
	db.BaseModel
}
