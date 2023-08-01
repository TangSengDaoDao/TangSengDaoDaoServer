package group

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// InsertInviteTx 添加邀请信息
func (d *DB) InsertInviteTx(model *InviteModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("group_invite").Columns(util.AttrToUnderscore(model)...).Record(model).Exec()
	return err
}

// InsertInviteItemTx 添加邀请项
func (d *DB) InsertInviteItemTx(model *InviteItemModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("invite_item").Columns(util.AttrToUnderscore(model)...).Record(model).Exec()
	return err
}

// QueryInviteDetail 查询邀请详情
func (d *DB) QueryInviteDetail(inviteNo string) (*InviteDetailModel, error) {
	var model *InviteDetailModel
	_, err := d.session.Select("group_invite.*,IFNULL(user.name,'') inviter_name").From("group_invite").LeftJoin("user", "group_invite.inviter=user.uid").Where("invite_no=?", inviteNo).Load(&model)
	return model, err
}

// QueryInviteItemDetail 查询邀请item详情
func (d *DB) QueryInviteItemDetail(inviteNo string) ([]*InviteItemDetailModel, error) {
	var items []*InviteItemDetailModel
	_, err := d.session.Select("invite_item.*,IFNULL(user.name,'') name").From("invite_item").LeftJoin("user", "invite_item.uid=user.uid").Where("invite_item.invite_no=?", inviteNo).Load(&items)
	return items, err

}

// UpdateInviteStatusTx 更新邀请信息状态
func (d *DB) UpdateInviteStatusTx(allower string, status int, inviteNo string, tx *dbr.Tx) error {
	_, err := tx.Update("group_invite").Set("allower", allower).Set("status", status).Where("invite_no=?", inviteNo).Exec()
	return err
}

// UpdateInviteItemStatusTx 更新邀请信息状态
func (d *DB) UpdateInviteItemStatusTx(status int, inviteNo string, tx *dbr.Tx) error {
	_, err := tx.Update("invite_item").Set("status", status).Where("invite_no=?", inviteNo).Exec()
	return err
}

// InviteModel InviteModel
type InviteModel struct {
	InviteNo string `json:"invite_no"` // 邀请唯一编号
	GroupNo  string `json:"group_no"`  // 群唯一编号
	Inviter  string `json:"inviter"`   // 邀请者
	Remark   string `json:"remark"`    // 邀请备注
	Status   int    `json:"status"`    // 状态 0.未确认 1.已确认
	Allower  string `json:"allower"`   // 确认者
	db.BaseModel
}

// InviteDetailModel 邀请者详情
type InviteDetailModel struct {
	InviteModel
	InviterName string `json:"inviter_name"` // 邀请者名称

}

// InviteItemDetailModel item详情
type InviteItemDetailModel struct {
	InviteItemModel
	Name string // 被邀请者名称
}

// InviteItemModel InviteItemModel
type InviteItemModel struct {
	InviteNo string `json:"invite_no"` // 邀请唯一编号
	GroupNo  string `json:"group_no"`  // 群唯一编号
	Inviter  string `json:"inviter"`   // 邀请者
	UID      string `json:"uid"`       // 被邀请uid
	Status   int    `json:"status"`    // 状态 0.未确认 1.已确认
	db.BaseModel
}
