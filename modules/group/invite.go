package group

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 群邀请添加
func (g *Group) groupMemberInviteAdd(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	loginName := c.MustGet("name").(string)
	groupNo := c.Param("group_no")
	var req InviteReq
	if err := c.BindJSON(&req); err != nil {
		g.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}

	creatorOrManagerUIDS, err := g.db.QueryGroupManagerOrCreatorUIDS(groupNo)
	if err != nil {
		g.Error("查询创建者或管理员的uid失败！", zap.String("group_no", groupNo), zap.Error(err))
		c.ResponseError(errors.New("查询创建者或管理员的uid失败！"))
		return
	}

	inviteNo := util.GenerUUID()

	tx, _ := g.db.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupMemberInviteRequest,
		Type:  wkevent.Message,
		Data: config.MsgGroupMemberInviteReq{
			GroupNo:     groupNo,
			InviteNo:    inviteNo,
			Inviter:     loginUID,
			InviterName: loginName,
			Num:         len(req.UIDS),
			Subscribers: creatorOrManagerUIDS,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		g.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}
	inviteModel := &InviteModel{
		InviteNo: inviteNo,
		GroupNo:  groupNo,
		Inviter:  loginUID,
		Remark:   req.Remark,
		Status:   InviteStatusWait,
	}
	err = g.db.InsertInviteTx(inviteModel, tx)
	if err != nil {
		tx.Rollback()
		g.Error("添加邀请数据失败！", zap.Error(err))
		c.ResponseError(errors.New("添加邀请数据失败！"))
		return
	}
	for _, uid := range req.UIDS {
		item := &InviteItemModel{
			InviteNo: inviteNo,
			GroupNo:  groupNo,
			Inviter:  loginUID,
			UID:      uid,
			Status:   InviteStatusWait,
		}
		err := g.db.InsertInviteItemTx(item, tx)
		if err != nil {
			tx.Rollback()
			g.Error("添加邀请项失败！", zap.Error(err))
			c.ResponseError(errors.New("添加邀请项失败！"))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	g.ctx.EventCommit(eventID)
	c.ResponseOK()
}

// 获取群成员邀请详情的h5
func (g *Group) getToGroupMemberConfirmInviteDetailH5(c *wkhttp.Context) {
	groupNo := c.Param("group_no")
	inviteNo := c.Query("invite_no")
	loginUID := c.MustGet("uid").(string)

	managerOrCreator, err := g.db.QueryIsGroupManagerOrCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否管理者或创建者失败！")
		c.ResponseError(errors.New("查询是否管理者或创建者失败！"))
		return
	}
	if !managerOrCreator {
		c.ResponseError(errors.New("你不是群主或管理员！"))
		return
	}
	authCode := util.GenerUUID()
	err = g.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", common.AuthCodeCachePrefix, authCode), util.ToJson(map[string]interface{}{
		"group_no":  groupNo,  // 群编号
		"invite_no": inviteNo, // 邀请编号
		"allower":   loginUID, // 通过者
		"type":      common.AuthCodeTypeGroupMemberInvite,
	}), time.Minute*5)

	h5URL := fmt.Sprintf("%s/invite_detail.html?invite_no=%s&auth_code=%s", g.ctx.GetConfig().External.H5BaseURL, inviteNo, authCode)
	c.JSON(http.StatusOK, gin.H{
		"url": h5URL,
	})
}

// 群邀请确认
func (g *Group) groupMemberInviteSure(c *wkhttp.Context) {
	authCode := c.Query("auth_code")
	authInfo, err := g.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", common.AuthCodeCachePrefix, authCode))
	if err != nil {
		g.Error("获取授权信息失败！", zap.Error(err))
		c.ResponseError(errors.New("获取授权信息失败！"))
		return
	}
	var authMap map[string]interface{}
	err = util.ReadJsonByByte([]byte(authInfo), &authMap)
	if err != nil {
		g.Error("解码认证信息的JSON数据失败！", zap.Error(err))
		c.ResponseError(errors.New("解码认证信息的JSON数据失败！"))
		return
	}
	authType := authMap["type"].(string)
	if authType != string(common.AuthCodeTypeGroupMemberInvite) {
		c.ResponseError(errors.New("授权码不是确认邀请！"))
		return
	}
	inviteNo := authMap["invite_no"].(string)
	/**
	判断邀请信息是否有效
	**/
	inviteDetailModel, err := g.db.QueryInviteDetail(inviteNo)
	if err != nil {
		g.Error("查询邀请详情失败！", zap.Error(err))
		c.ResponseError(errors.New("查询邀请详情失败！"))
		return
	}
	if inviteDetailModel == nil {
		c.ResponseError(errors.New("没有查询到邀请信息！"))
		return
	}
	if inviteDetailModel.Status != InviteStatusWait {
		c.ResponseError(errors.New("邀请信息不是待邀请状态！"))
		return
	}
	/**
	查询邀请成员详情
	**/
	inviteItemDetilModels, err := g.db.QueryInviteItemDetail(inviteNo)
	if err != nil {
		g.Error("查询邀请详情失败！", zap.Error(err))
		c.ResponseError(errors.New("查询邀请详情失败！"))
		return
	}
	if inviteItemDetilModels == nil || len(inviteItemDetilModels) <= 0 {
		c.ResponseError(errors.New("没有查到邀请信息！"))
		return
	}
	members := make([]string, 0, len(inviteItemDetilModels))
	groupNo := inviteItemDetilModels[0].GroupNo
	inviter := inviteItemDetilModels[0].Inviter
	for _, inviteItemDetilModel := range inviteItemDetilModels {
		members = append(members, inviteItemDetilModel.UID)
	}
	/**
	添加成员
	**/
	inviterUser, err := g.userDB.QueryByUID(inviter)
	if err != nil {
		g.Error("查询邀请者的用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询邀请者的用户信息失败！"))
		return
	}
	if inviterUser == nil {
		g.Error("没有查到邀请者的用户信息！")
		c.ResponseError(errors.New("没有查到邀请者的用户信息！"))
		return

	}
	allower := authMap["allower"].(string)
	tx, _ := g.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err = g.db.UpdateInviteStatusTx(allower, InviteStatusOK, inviteNo, tx)
	if err != nil {
		tx.Rollback()
		g.Error("更新邀请信息状态失败！", zap.Error(err))
		c.ResponseError(errors.New("更新邀请信息状态失败！"))
		return
	}
	err = g.db.UpdateInviteItemStatusTx(InviteStatusOK, inviteNo, tx)
	if err != nil {
		tx.Rollback()
		g.Error("更新邀请信息项状态失败！", zap.Error(err))
		c.ResponseError(errors.New("更新邀请信息项状态失败！"))
		return
	}
	commitCallback, err := g.addMembersTx(members, groupNo, inviterUser.UID, inviterUser.Name, tx)
	if err != nil {
		tx.Rollback()
		g.Error("添加成员失败！", zap.Error(err))
		c.ResponseError(errors.New("添加成员失败！"))
		return
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}

	commitCallback()
	c.ResponseOK()

}

// groupMemberInviteDetail 获取群成员邀请详情
func (g *Group) groupMemberInviteDetail(c *wkhttp.Context) {
	inviteNo := c.Param("invite_no")
	inviteDetilModel, err := g.db.QueryInviteDetail(inviteNo)
	if err != nil {
		g.Error("查询邀请详情失败！", zap.Error(err))
		c.ResponseError(errors.New("查询邀请详情失败！"))
		return
	}
	if inviteDetilModel == nil {
		c.ResponseError(errors.New("没有查到邀请信息！"))
		return
	}
	inviteItems, err := g.db.QueryInviteItemDetail(inviteNo)
	if err != nil {
		g.Error("获取邀请项失败！", zap.Error(err))
		c.ResponseError(errors.New("获取邀请项失败！"))
		return
	}
	g.Debug("inviteItems-", zap.Int("len", len(inviteItems)))
	c.Response(InviteDetailResp{}.From(inviteDetilModel, inviteItems))
}

// InviteReq 群邀请
type InviteReq struct {
	UIDS   []string `json:"uids"`
	Remark string   `json:"remark"`
}

// Check Check
func (i InviteReq) Check() error {
	if len(i.UIDS) <= 0 {
		return errors.New("被邀请者不能为空！")
	}
	return nil
}

// InviteDetailResp 邀请详情返回
type InviteDetailResp struct {
	InviteNo    string                 `json:"invite_no"`    // 邀请唯一编号
	GroupNo     string                 `json:"group_no"`     // 群唯一编号
	Inviter     string                 `json:"inviter"`      // 邀请者
	Remark      string                 `json:"remark"`       // 邀请备注
	InviterName string                 `json:"inviter_name"` // 邀请者名称
	Status      int                    `json:"status"`       // 状态 0.未确认 1.已确认
	Items       []InviteItemDetailResp `json:"items"`        // 邀请项详情
}

// From From
func (i InviteDetailResp) From(model *InviteDetailModel, items []*InviteItemDetailModel) InviteDetailResp {
	resp := InviteDetailResp{}
	resp.InviteNo = model.InviteNo
	resp.GroupNo = model.GroupNo
	resp.Inviter = model.Inviter
	resp.Remark = model.Remark
	resp.InviterName = model.InviterName
	resp.Status = model.Status
	if len(items) > 0 {
		itemResps := make([]InviteItemDetailResp, 0, len(items))
		for _, item := range items {
			itemResps = append(itemResps, InviteItemDetailResp{
				UID:  item.UID,
				Name: item.Name,
			})
		}
		resp.Items = itemResps
	}
	fmt.Println(resp)
	return resp
}

// InviteItemDetailResp 邀请item
type InviteItemDetailResp struct {
	UID  string `json:"uid"`  // 被邀请uid
	Name string `json:"name"` // 被邀请者名称
}
