package group

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

// Manager 群后台管理api
type Manager struct {
	ctx *config.Context
	log.Log
	managerDB *managerDB
	userDB    *user.DB
	db        *DB
}

// NewManager NewManager
func NewManager(ctx *config.Context) *Manager {
	return &Manager{
		ctx:       ctx,
		Log:       log.NewTLog("groupManager"),
		managerDB: newManagerDB(ctx.DB()),
		userDB:    user.NewDB(ctx),
		db:        NewDB(ctx),
	}
}

// Route 配置路由规则
func (m *Manager) Route(r *wkhttp.WKHttp) {
	auth := r.Group("/v1/manager", m.ctx.AuthMiddleware(r))
	{
		auth.GET("/group/list", m.list)                              // 群列表
		auth.GET("/group/disablelist", m.disablelist)                // 封禁群列表
		auth.PUT("/group/liftban/:groupNo/:status", m.leftbangroup)  // 封禁或解禁某个群
		auth.PUT("/groups/:group_no/forbidden/:on", m.forbidden)     // 群全员禁言
		auth.GET("/groups/:group_no/members", m.members)             // 群成员
		auth.GET("/groups/:group_no/members/blacklist", m.blacklist) // 群黑名单成员
		auth.DELETE("/groups/:group_no/members", m.removeMember)     //移除群成员
	}
}

// 查询群列表
func (m *Manager) list(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	keyword := c.Query("keyword")
	pageIndex, pageSize := c.GetPage()
	var list []*managerGroupModel
	var count int64
	if keyword == "" {
		list, err = m.managerDB.listWithPage(uint64(pageSize), uint64(pageIndex))
		if err != nil {
			m.Error("查询群列表错误", zap.Error(err))
			c.ResponseError(errors.New("查询群列表错误"))
			return
		}
		count, err = m.db.queryGroupCount()
		if err != nil {
			m.Error("查询群数量错误", zap.Error(err))
			c.ResponseError(errors.New("查询群数量错误"))
			return
		}
	} else {
		list, err = m.managerDB.listWithPageAndKeyword(keyword, uint64(pageSize), uint64(pageIndex))
		if err != nil {
			m.Error("查询群列表错误", zap.Error(err))
			c.ResponseError(errors.New("查询群列表错误"))
			return
		}
		count, err = m.managerDB.queryGroupCountWithKeyWord(keyword)
		if err != nil {
			m.Error("查询群数量错误", zap.Error(err))
			c.ResponseError(errors.New("查询群数量错误"))
			return
		}
	}

	result, err := m.getRespList(list)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Response(map[string]interface{}{
		"count": count,
		"list":  result,
	})
}

func (m *Manager) getRespList(list []*managerGroupModel) ([]*managerGroupResp, error) {
	result := make([]*managerGroupResp, 0)
	if len(list) > 0 {
		uids := make([]string, 0)
		groupNos := make([]string, 0)
		for _, group := range list {
			uids = append(uids, group.Creator)
			groupNos = append(groupNos, group.GroupNo)
		}
		users, err := m.userDB.QueryByUIDs(uids)
		if err != nil {
			m.Error("查询群创建者错误", zap.Error(err))
			return result, errors.New("查询群创建者错误")
		}

		memberCounts, err := m.managerDB.queryGroupsMemberCount(groupNos)
		if err != nil {
			m.Error("查询群成员数量错误", zap.Error(err))
			return result, errors.New("查询群成员数量错误")
		}
		for _, group := range list {
			// 得到群主名称
			var createName string
			if len(users) > 0 {
				for _, user := range users {
					if user.UID == group.Creator {
						createName = user.Name
						break
					}
				}
			}
			// 得到群成员数量
			var count int = 0
			if len(memberCounts) > 0 {
				for _, memberCount := range memberCounts {
					if memberCount.GroupNo == group.GroupNo {
						count = memberCount.MemberCount
						break
					}
				}
			}

			result = append(result, &managerGroupResp{
				Name:        group.Name,
				Creator:     group.Creator,
				GroupNo:     group.GroupNo,
				CreateName:  createName,
				CreateAt:    group.CreatedAt.String(),
				Status:      group.Status,
				Forbidden:   group.Forbidden,
				MemberCount: count,
			})
		}
	}
	return result, nil
}

// 封禁群列表
func (m *Manager) disablelist(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	pageIndex, pageSize := c.GetPage()
	list, err := m.managerDB.queryGroupsWithStatus(GroupStatusDisabled, uint64(pageSize), uint64(pageIndex))
	if err != nil {
		m.Error("查询群列表错误", zap.Error(err))
		c.ResponseError(errors.New("查询群列表错误"))
		return
	}
	result, err := m.getRespList(list)
	if err != nil {
		c.ResponseError(err)
		return
	}
	count, err := m.managerDB.queryGroupCountWithStatus(GroupStatusDisabled)
	if err != nil {
		m.Error("查询群总数错误", zap.Error(err))
		c.ResponseError(errors.New("查询群总数错误"))
		return
	}
	c.Response(map[string]interface{}{
		"count": count,
		"list":  result,
	})
}

// 封禁或解禁某个群
func (m *Manager) leftbangroup(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	groupNo := c.Param("groupNo")
	status := c.Param("status")
	if groupNo == "" {
		c.ResponseError(errors.New("操作群ID不能为空"))
		return
	}
	if status == "" {
		c.ResponseError(errors.New("操作状态不能为空"))
		return
	}
	group, err := m.db.QueryWithGroupNo(groupNo)
	if err != nil {
		m.Error("查询群信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询群信息错误"))
		return
	}
	if group == nil {
		c.ResponseError(errors.New("操作的群不存在"))
		return
	}
	groupStatus, _ := strconv.Atoi(status)
	if groupStatus != GroupStatusNormal && groupStatus != GroupStatusDisabled {
		c.ResponseError(errors.New("未知操作类型"))
		return
	}

	if groupStatus == group.Status {
		c.ResponseOK()
		return
	}
	var ban = 0
	if groupStatus == GroupStatusDisabled {
		ban = 1
	}
	err = m.ctx.IMCreateOrUpdateChannelInfo(&config.ChannelInfoCreateReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Ban:         ban,
		Large:       group.GroupType,
	})
	if err != nil {
		m.Error("调用IM修改channel信息服务失败！", zap.Error(err))
		c.ResponseError(errors.New("调用IM修改channel信息服务失败！"))
		return
	}
	group.Status = groupStatus
	//通知群成员更新群资料
	// todo
	tx, err := m.ctx.DB().Begin()
	util.CheckErr(err)
	groupMap := make(map[string]string)
	groupMap["status"] = strconv.Itoa(groupStatus)
	err = m.db.UpdateTx(group, tx)
	if err != nil {
		tx.Rollback()
		m.Error("更新群信息失败！", zap.Error(err), zap.String("group_no", group.GroupNo), zap.Any("groupMap", groupMap))
		c.ResponseError(errors.New("更新群信息失败！"))
		return
	}
	// 发布群创建事件
	eventID, _ := m.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupUpdate,
		Type:  wkevent.Message,
		Data: &config.MsgGroupUpdateReq{
			GroupNo:      groupNo,
			Operator:     c.GetLoginUID(),
			OperatorName: c.GetLoginName(),
			Attr:         common.GroupAttrKeyStatus,
			Data:         groupMap,
		},
	}, tx)
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		m.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	m.ctx.EventCommit(eventID)

	c.ResponseOK()
}

// 禁言
func (m *Manager) forbidden(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	groupNo := c.Param("group_no")
	on := c.Param("on")
	if groupNo == "" {
		c.ResponseError(errors.New("群编号不能为空"))
		return
	}
	groupModel, err := m.db.QueryWithGroupNo(groupNo)
	if err != nil {
		m.Error("查询群信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if groupModel == nil {
		c.ResponseError(errors.New("群不存在！"))
		return
	}
	forbidden, _ := strconv.ParseInt(on, 10, 64)
	groupModel.Forbidden = int(forbidden)

	whitelistUIDs := make([]string, 0)
	if forbidden == 1 {
		managerOrCreaterUIDs, err := m.db.QueryGroupManagerOrCreatorUIDS(groupNo)
		if err != nil {
			c.ResponseErrorf("查询管理者们的uid失败！", err)
			return
		}
		whitelistUIDs = managerOrCreaterUIDs
	}
	// 群全员禁言
	err = m.ctx.IMWhitelistSet(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   groupModel.GroupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
		},
		UIDs: whitelistUIDs,
	})
	if err != nil {
		m.Error("设置禁言失败！", zap.Error(err))
		c.ResponseError(errors.New(err.Error()))
		return
	}

	tx, err := m.ctx.DB().Begin()
	util.CheckErr(err)

	err = m.db.UpdateTx(groupModel, tx)
	if err != nil {
		tx.Rollback()
		m.Error("更新群信息失败！", zap.Error(err), zap.String("group_no", groupModel.GroupNo))
		c.ResponseError(errors.New("更新群信息失败！"))
		return
	}
	// 发布群信息更新事件
	eventID, err := m.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupUpdate,
		Type:  wkevent.Message,
		Data: &config.MsgGroupUpdateReq{
			GroupNo:      groupNo,
			Operator:     c.GetLoginUID(),
			OperatorName: c.GetLoginName(),
			Attr:         common.GroupAttrKeyForbidden,
			Data: map[string]string{
				common.GroupAttrKeyForbidden: on,
			},
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		m.Error("开启群更新事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启群更新事件失败！"))
		return
	}

	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		m.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	m.ctx.EventCommit(eventID)

	c.ResponseOK()
}

// 移除群成员
func (m *Manager) removeMember(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Request.URL.Path = fmt.Sprintf("/v1/groups/%s/members", c.Param("group_no"))
	m.ctx.GetHttpRoute().HandleContext(c)
}

// 群成员
func (m *Manager) members(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	groupNo := c.Param("group_no")
	pageIndex, pageSize := c.GetPage()
	if groupNo == "" {
		c.ResponseError(errors.New("群编号不能为空"))
		return
	}
	keyword := c.Query("keyword")
	groupModel, err := m.db.QueryWithGroupNo(groupNo)
	if err != nil {
		m.Error("查询群信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询群信息错误"))
		return
	}
	if groupModel == nil {
		c.ResponseError(errors.New("操作的群不存在"))
		return
	}
	var list []*managerMemberModel
	var count int64
	if keyword == "" {
		list, err = m.managerDB.queryGroupMembers(groupNo, uint64(pageSize), uint64(pageIndex))
		if err != nil {
			m.Error("查询群成员错误", zap.Error(err))
			c.ResponseError(errors.New("查询群成员错误"))
			return
		}
		count, err = m.managerDB.queryGroupMemberCount(groupNo)
		if err != nil {
			m.Error("查询群成员总数错误", zap.Error(err))
			c.ResponseError(errors.New("查询群成员总数错误"))
			return
		}
	} else {
		list, err = m.managerDB.queryGroupMembersWithKeyWord(groupNo, keyword, uint64(pageSize), uint64(pageIndex))
		if err != nil {
			m.Error("查询群成员错误", zap.Error(err))
			c.ResponseError(errors.New("查询群成员错误"))
			return
		}
		count, err = m.managerDB.queryGroupMemberCountWithKeyword(groupNo, keyword)
		if err != nil {
			m.Error("查询群成员总数错误", zap.Error(err))
			c.ResponseError(errors.New("查询群成员总数错误"))
			return
		}
	}

	c.Response(map[string]interface{}{
		"count": count,
		"list":  m.from(list),
	})
}

// 群黑名单成员
func (m *Manager) blacklist(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	groupNo := c.Param("group_no")
	pageIndex, pageSize := c.GetPage()
	if groupNo == "" {
		c.ResponseError(errors.New("群编号不能为空"))
		return
	}
	list, err := m.managerDB.queryGroupMembersWithStatus(groupNo, int(common.GroupMemberStatusBlacklist), uint64(pageSize), uint64(pageIndex))
	if err != nil {
		m.Error("查询群成员错误", zap.Error(err))
		c.ResponseError(errors.New("查询群成员错误"))
		return
	}
	count, err := m.managerDB.queryGroupMemberCountWithStatus(groupNo, int(common.GroupMemberStatusBlacklist))
	if err != nil {
		m.Error("查询群成员总数错误", zap.Error(err))
		c.ResponseError(errors.New("查询群成员总数错误"))
		return
	}
	c.Response(map[string]interface{}{
		"count": count,
		"list":  m.from(list),
	})
}

func (m *Manager) from(list []*managerMemberModel) []*managerMemberResp {
	result := make([]*managerMemberResp, 0)
	for _, model := range list {
		result = append(result, &managerMemberResp{
			Name:      model.Name,
			Remark:    model.Remark,
			Role:      model.Role,
			CreatedAt: model.CreatedAt.String(),
			UID:       model.UID,
		})
	}
	return result
}

type managerGroupResp struct {
	Name        string `json:"name"`
	GroupNo     string `json:"group_no"`
	Creator     string `json:"creator"`
	CreateName  string `json:"create_name"`
	CreateAt    string `json:"create_at"`
	Status      int    `json:"status"`
	MemberCount int    `json:"member_count"`
	Forbidden   int    `json:"forbidden"`
}

type managerMemberResp struct {
	Name      string `json:"name"`
	UID       string `json:"uid"`
	Role      int    `json:"role"` // 成员角色
	CreatedAt string `json:"created_at"`
	Remark    string `json:"remark"` // 成员备注
}

// type managerMemberRemoveReq struct {
// 	Members []string `json:"members"` // 成员uid
// }

// func (m managerMemberRemoveReq) Check() error {
// 	if len(m.Members) <= 0 {
// 		return errors.New("群成员不能为空！")
// 	}
// 	return nil
// }
