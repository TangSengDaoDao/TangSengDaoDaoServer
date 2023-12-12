package group

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	chservice "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/channel/service"
	common2 "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/file"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/source"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
	"github.com/gocraft/dbr/v2"
	"go.uber.org/zap"
)

// Group 群组相关API
type Group struct {
	ctx *config.Context
	log.Log
	db            *DB
	settingDB     *settingDB
	userDB        *user.DB
	groupService  IService
	fileService   file.IService
	commonService common2.IService
}

// New New
func New(ctx *config.Context) *Group {

	g := &Group{
		ctx:           ctx,
		Log:           log.NewTLog("Group"),
		db:            NewDB(ctx),
		userDB:        user.NewDB(ctx),
		settingDB:     newSettingDB(ctx),
		groupService:  NewService(ctx),
		fileService:   file.NewService(ctx),
		commonService: common2.NewService(ctx),
	}
	g.ctx.AddEventListener(event.EventUserRegister, g.handleRegisterUserEvent)
	g.ctx.AddEventListener(event.GroupMemberAdd, g.handleGroupMemberAddEvent)
	g.ctx.AddEventListener(event.OrgOrDeptCreate, g.handleOrgOrDeptCreateEvent)
	g.ctx.AddEventListener(event.OrgOrDeptEmployeeUpdate, g.handleOrgOrDeptEmployeeUpdate)
	g.ctx.AddEventListener(event.OrgEmployeeExit, g.handleOrgEmployeeExit)
	source.SetGroupMemberProvider(g)
	return g
}

// Route 路由配置
func (g *Group) Route(r *wkhttp.WKHttp) {
	group := r.Group("/v1/group", g.ctx.AuthMiddleware(r))
	{
		group.POST("/create", g.groupCreate)
		group.GET("/my", g.list)                            //我保存的群
		group.GET("/forbidden_times", g.forbiddenTimesList) // 获取禁言时常列表
	}
	groups := r.Group("/v1/groups", g.ctx.AuthMiddleware(r))
	{

		groups.POST("/:group_no/members", g.memberAdd)                                     // 添加群成员
		groups.DELETE("/:group_no/members", g.memberRemove)                                // 移除群成员
		groups.GET("/:group_no/members", g.membersGet)                                     // 获取群成员
		groups.POST("/:group_no/members_delete", g.memberRemove)                           // 移除群成员
		groups.GET("/:group_no/membersync", g.syncMembers)                                 // 同步群成员
		groups.GET("/:group_no", g.groupGet)                                               // 获取群信息
		groups.PUT("/:group_no/setting", g.groupSettingUpdate)                             // 修改群设置
		groups.PUT("/:group_no", g.groupUpdate)                                            // 修改群信息
		groups.PUT("/:group_no/members/:uid", g.memberUpdate)                              // 修改群的群成员信息
		groups.POST("/:group_no/exit", g.groupExit)                                        // 退出群聊
		groups.POST("/:group_no/managers", g.managerAdd)                                   // 添加群管理员
		groups.DELETE("/:group_no/managers", g.managerRemove)                              // 移除群管理员
		groups.POST("/:group_no/forbidden/:on", g.groupForbidden)                          // 群全员禁言
		groups.GET("/:group_no/qrcode", g.groupQRCode)                                     // 获取群二维码信息
		groups.POST("/:group_no/transfer/:to_uid", g.transferGrouper)                      // 群主转让
		groups.POST("/:group_no/member/invite", g.groupMemberInviteAdd)                    // 群成员邀请
		groups.GET("/:group_no/member/h5confirm", g.getToGroupMemberConfirmInviteDetailH5) // 获取确认邀请的h5页面
		groups.POST("/:group_no/blacklist/:action", g.blacklist)                           // 添加或移除黑名单
		groups.POST("/:group_no/forbidden_with_member", g.forbiddenWithGroupMember)        // 禁言或解禁某个群成员
		groups.POST("/:group_no/avatar", g.avatarUpload)                                   // 上传群头像
	}
	openGroups := r.Group("/v1/groups")
	{ // 获取群头像
		openGroups.GET("/:group_no/avatar", g.avatarGet)       // 获取群头像
		openGroups.GET("/:group_no/detail", g.groupDetailGet)  // 群详情
		openGroups.GET("/:group_no/scanjoin", g.groupScanJoin) // 扫码加入群
	}
	openGroup := r.Group("/v1/group")
	{

		openGroup.GET("invites/:invite_no", g.groupMemberInviteDetail) // 获取邀请详情
		openGroup.POST("invite/sure", g.groupMemberInviteSure)         // 确认邀请
	}
	go g.CheckForbiddenLoop()
}

func (g *Group) membersGet(c *wkhttp.Context) {
	keyword := c.Query("keyword")
	groupNo := c.Param("group_no")
	limit, _ := strconv.ParseUint(c.Query("limit"), 10, 64)
	page, _ := strconv.ParseUint(c.Query("page"), 10, 64)
	if page <= 0 {
		page = 1
	}

	if limit <= 0 || limit > 100000 {
		limit = 100
	}
	var members []*MemberDetailModel
	var err error
	members, err = g.db.queryMembersWithKeyword(groupNo, c.GetLoginUID(), keyword, page, limit)
	if err != nil {
		g.Error("查询成员列表失败！", zap.Error(err))
		c.ResponseError(errors.New("查询成员列表失败！"))
		return
	}

	resps := make([]memberDetailResp, 0)
	if len(members) > 0 {
		for _, memberModel := range members {
			resp := memberDetailResp{}
			resps = append(resps, resp.from(memberModel))
		}
	}

	c.Response(resps)
}

func (g *Group) avatarGet(c *wkhttp.Context) {
	groupNo := c.Param("group_no")
	v := c.Query("v")
	//是否为系统群
	if groupNo == g.ctx.GetConfig().Account.SystemGroupID {
		c.Header("Content-Type", "image/jpeg")
		avatarBytes, err := ioutil.ReadFile("assets/assets/g_avatar.jpeg")
		if err != nil {
			g.Error("头像读取失败！", zap.Error(err))
			c.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		c.Writer.Write(avatarBytes)
		return
	}
	// 组织群
	if strings.HasPrefix(groupNo, "org_") {
		c.Header("Content-Type", "image/jpeg")
		avatarBytes, err := ioutil.ReadFile("assets/assets/org_avatar.png")
		if err != nil {
			g.Error("头像读取失败！", zap.Error(err))
			c.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		c.Writer.Write(avatarBytes)
		return
	}
	// 部门群
	if strings.HasPrefix(groupNo, "dept_") {
		c.Header("Content-Type", "image/jpeg")
		avatarBytes, err := ioutil.ReadFile("assets/assets/dept_avatar.png")
		if err != nil {
			g.Error("头像读取失败！", zap.Error(err))
			c.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		c.Writer.Write(avatarBytes)
		return
	}
	path := g.ctx.GetConfig().GetGroupAvatarFilePath(groupNo)
	downloadUrl, err := g.fileService.DownloadURL(path, "group_avatar.jpeg")
	if err != nil {
		g.Error("获取下载路径失败！", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("%s?%s", downloadUrl, v))
}

func (g *Group) avatarUpload(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	groupNo := c.Param("group_no")
	if c.Request.MultipartForm == nil {
		err := c.Request.ParseMultipartForm(1024 * 1024 * 20) // 20M
		if err != nil {
			g.Error("数据格式不正确！", zap.Error(err))
			c.ResponseError(errors.New("数据格式不正确！"))
			return
		}
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		g.Error("读取文件失败！", zap.Error(err))
		c.ResponseError(errors.New("读取文件失败！"))
		return
	}

	isCreator, err := g.db.QueryIsGroupCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询群创建者失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群创建者失败！"))
		return
	}
	if !isCreator {
		c.ResponseError(errors.New("只有创建者才能修改头像"))
		return
	}

	groupAvatarPath := g.ctx.GetConfig().GetGroupAvatarFilePath(groupNo)
	_, err = g.fileService.UploadFile(groupAvatarPath, "image/png", func(w io.Writer) error {
		_, err := io.Copy(w, file)
		return err
	})
	defer file.Close()
	if err != nil {
		g.Error("上传文件失败！", zap.Error(err))
		c.ResponseError(errors.New("上传文件失败！"))
		return
	}
	err = g.db.updateAvatar(groupAvatarPath, groupNo)
	if err != nil {
		g.Error("头像修改失败！", zap.String("group_no", groupNo), zap.Error(err))
		c.ResponseError(errors.New("头像修改失败！"))
		return
	}
	// 发送群头像更新命令
	err = g.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupAvatarUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
	if err != nil {
		g.Error("发送群头像更新命令失败！", zap.String("groupNo", groupNo), zap.Error(err))
		c.ResponseError(errors.New("发送群头像更新命令失败！"))
		return
	}
	c.ResponseOK()
}

// 同步群成员
func (g *Group) syncMembers(c *wkhttp.Context) {
	groupNo := c.Param("group_no")

	if g.ctx.GetConfig().IsVisitorChannel(groupNo) {
		c.Request.URL.Path = fmt.Sprintf("/v1/hotline/visitor/channels/%s/members", groupNo)
		g.ctx.GetHttpRoute().HandleContext(c)
		return
	}

	group, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err), zap.String("groupNo", groupNo))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if group == nil {
		g.Error("群不存在不能同步成员！", zap.String("groupNo", groupNo))
		c.ResponseError(errors.New("群不存在不能同步成员！"))
		return
	}
	if group.GroupType == int(GroupTypeSuper) {
		g.Error("超大群不支持同步群成员！", zap.String("groupNo", groupNo))
		c.ResponseError(errors.New("超大群不支持同步群成员！"))
		return
	}

	limit, _ := strconv.ParseUint(c.Query("limit"), 10, 64)
	if limit <= 0 {
		limit = 100
	}
	version, _ := strconv.ParseInt(c.Query("version"), 10, 64)
	memberModels, err := g.db.SyncMembers(groupNo, version, limit)
	if err != nil {
		g.Error("同步成员信息失败！", zap.Error(err), zap.String("groupNo", groupNo))
		c.ResponseError(errors.New("同步成员信息失败！"))
		return
	}
	resps := make([]memberDetailResp, 0)
	for _, memberModel := range memberModels {
		resp := memberDetailResp{}
		resps = append(resps, resp.from(memberModel))
	}
	c.Response(resps)
}

// 获取群详情
func (g *Group) groupGet(c *wkhttp.Context) {
	groupNo := c.Param("group_no")
	// if g.ctx.GetConfig().IsVisitorChannel(groupNo) { // 访客频道
	// 	c.Request.URL.Path = fmt.Sprintf("/v1/hotline/visitor/channel/%s", groupNo)
	// 	g.ctx.Server.GetRoute().HandleContext(c)
	// 	return
	// }
	uid := c.MustGet("uid").(string)

	groupResp, err := g.groupService.GetGroupDetail(groupNo, uid)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Response(groupResp)
}

// 获取群详情
func (g *Group) groupDetailGet(c *wkhttp.Context) {
	groupNo := c.Param("group_no")
	groupModel, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if groupModel == nil {
		c.ResponseError(errors.New("群不存在！"))
		return
	}
	memberCount, err := g.db.QueryMemberCount(groupNo)
	if err != nil {
		g.Error("查询成员数量失败！", zap.Error(err))
		c.ResponseError(errors.New("查询成员数量失败！"))
		return
	}

	c.Response(groupDetailResp{}.from(groupModel, memberCount))
}

// list 我保存的群聊
func (g *Group) list(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	models, err := g.db.querySavedGroups(loginUID)
	if err != nil {
		g.Error("查询我保存的群聊失败", zap.Error(err))
		c.ResponseError(errors.New("查询我保存的群聊失败"))
		return
	}
	resps := make([]*GroupResp, 0)
	for _, model := range models {
		groupResp := &GroupResp{}
		resps = append(resps, groupResp.from(model))
	}
	c.Response(resps)
}

// 创建群
func (g *Group) groupCreate(c *wkhttp.Context) {
	creator := c.MustGet("uid").(string)
	creatorName := c.MustGet("name").(string)
	var req groupReq
	if err := c.BindJSON(&req); err != nil {
		g.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}

	creatorUser, err := g.userDB.QueryByUID(creator)
	if err != nil {
		g.Error("查询创建者信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询创建者信息失败！"))
		return
	}
	if creatorUser == nil {
		g.Error("创建者不存在！", zap.String("creator", creator))
		c.ResponseError(errors.New("创建者不存在！"))
		return
	}

	req.Members = util.RemoveRepeatedElement(append(req.Members, creator)) // 将创建者也加入成员内

	// 查询成员用户信息
	memberUserModels, err := g.userDB.QueryByUIDs(req.Members)
	if err != nil {
		g.Error("查询成员用户信息失败！", zap.Error(err), zap.Strings("members", req.Members))
		c.ResponseError(errors.New("查询成员用户信息失败！"))
		return
	}
	if memberUserModels == nil {
		c.ResponseError(errors.New("成员用户信息不存在！"))
		return
	}
	memberNames := make([]string, 0, len(memberUserModels))
	for _, memberUserModel := range memberUserModels {
		memberNames = append(memberNames, memberUserModel.Name)
	}
	groupName := req.Name
	if groupName == "" {
		groupName = strings.Join(memberNames, "、")
	}

	nameRuns := []rune(groupName)
	if len(nameRuns) > 20 {
		groupName = string(nameRuns[:20])
	}

	groupNo := util.GenerUUID()

	version := g.ctx.GenSeq(common.GroupSeqKey)
	channelServiceObj := register.GetService(ChannelServiceName)
	var channelService chservice.IService
	if channelServiceObj != nil {
		channelService = channelServiceObj.(chservice.IService)
	}
	if channelService != nil {
		if creatorUser != nil && creatorUser.MsgExpireSecond > 0 {
			err = channelService.CreateOrUpdateMsgAutoDelete(groupNo, common.ChannelTypeGroup.Uint8(), creatorUser.MsgExpireSecond)
			if err != nil {
				g.Warn("更新消息自动删除失败！", zap.Error(err))
			}
		}
	}

	tx, err := g.ctx.DB().Begin()
	util.CheckErr(err)
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()

	err = g.db.InsertTx(&Model{
		GroupNo:             groupNo,
		Name:                groupName,
		Creator:             creator,
		Status:              GroupStatusNormal,
		Version:             version,
		AllowViewHistoryMsg: int(common.GroupAllowViewHistoryMsgEnabled),
	}, tx)
	if err != nil {
		g.Error("添加群失败！", zap.Error(err))
		c.ResponseError(errors.New("添加群失败！"))
		tx.RollbackUnlessCommitted()
		return
	}
	realMemberUids := make([]string, 0) // 真实成员uid集合
	userBaseVos := make([]*config.UserBaseVo, 0)
	// 注销用户
	destroyUserBaseVos := make([]*config.UserBaseVo, 0)
	for _, memberUser := range memberUserModels {
		if memberUser.IsDestroy == 1 {
			destroyUserBaseVos = append(destroyUserBaseVos, &config.UserBaseVo{
				UID:  memberUser.UID,
				Name: memberUser.Name,
			})
			continue
		}
		memberVersion := g.ctx.GenSeq(common.GroupMemberSeqKey)
		realMemberUids = append(realMemberUids, memberUser.UID)
		var role = MemberRoleCommon
		if memberUser.UID == creator {
			role = MemberRoleCreator
		}
		err = g.db.InsertMemberTx(&MemberModel{
			GroupNo:   groupNo,
			UID:       memberUser.UID,
			Role:      role,
			Version:   memberVersion,
			InviteUID: creator,
			Robot:     memberUser.Robot,
			Status:    int(common.GroupMemberStatusNormal),
			Vercode:   fmt.Sprintf("%s@%d", util.GenerUUID(), common.GroupMember),
		}, tx)
		if err != nil {
			tx.RollbackUnlessCommitted()
			g.Error("添加成员失败！", zap.Error(err), zap.String("memberUid", memberUser.UID))
			c.ResponseError(errors.New("添加成员失败！"))
			return
		}
		userBaseVos = append(userBaseVos, &config.UserBaseVo{UID: memberUser.UID, Name: memberUser.Name})
	}
	if len(realMemberUids) <= 0 {
		tx.RollbackUnlessCommitted()
		g.Error("群成员不能为空！")
		c.ResponseError(errors.New("群成员不能为空！"))
		return
	}
	// 发布群创建事件
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupCreate,
		Type:  wkevent.Message,
		Data: &config.MsgGroupCreateReq{
			GroupNo:     groupNo,
			Creator:     creator,
			CreatorName: creatorName,
			Members:     userBaseVos,
			Version:     version,
		},
	}, tx)
	if err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}
	var unableAddDestroyAccount int64 = 0
	if len(destroyUserBaseVos) > 0 {
		// 发布无法添加到群聊用户
		unableAddDestroyAccount, err = g.ctx.EventBegin(&wkevent.Data{
			Event: event.GroupUnableAddDestroyAccount,
			Type:  wkevent.Message,
			Data: &config.MsgGroupCreateReq{
				GroupNo:     groupNo,
				Creator:     creator,
				CreatorName: creatorName,
				Members:     destroyUserBaseVos,
				Version:     version,
			},
		}, tx)
		if err != nil {
			tx.RollbackUnlessCommitted()
			g.Error("开启无法添加到群聊事件失败！", zap.Error(err))
			c.ResponseError(errors.New("开启无法添加到群聊事件失败！"))
			return
		}
	}
	groupAvatarEventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupAvatarUpdate,
		Type:  wkevent.CMD,
		Data: &config.CMDGroupAvatarUpdateReq{
			GroupNo: groupNo,
			Members: realMemberUids,
		},
	}, tx)
	if err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("开启群成员头像更新事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启群成员头像更新事件失败！"))
		return
	}

	// 创建IM频道
	err = g.ctx.IMCreateOrUpdateChannel(&config.ChannelCreateReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Subscribers: realMemberUids,
	})
	if err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("创建IM频道失败！", zap.Error(err))
		c.ResponseError(errors.New("创建IM频道失败！"))
		return
	}

	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	g.ctx.EventCommit(eventID)
	g.ctx.EventCommit(groupAvatarEventID)
	if unableAddDestroyAccount != 0 {
		g.ctx.EventCommit(unableAddDestroyAccount)
	}
	groupModel, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	groupResp := &GroupResp{}
	c.Response(groupResp.from(&DetailModel{
		Model:        *groupModel,
		Receipt:      1,
		RevokeRemind: 1,
		Screenshot:   1,
	}))
}

// 修改群信息
func (g *Group) groupUpdate(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	loginName := c.MustGet("name").(string)
	groupNo := c.Param("group_no")

	var groupMap map[string]string
	if err := c.BindJSON(&groupMap); err != nil {
		g.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(groupMap) <= 0 {
		c.ResponseError(errors.New("没有需要更新的属性！"))
		return
	}
	// 查询群信息
	group, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if group == nil {
		g.Error("群不存在！", zap.String("group_no", groupNo))
		c.ResponseError(errors.New("群不存在！"))
		return
	}
	// 查询是否是管理者
	isManager, err := g.db.QueryIsGroupManagerOrCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否是群管理者失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否是群管理者失败！"))
		return
	}
	if !isManager {
		c.ResponseError(errors.New("只有群管理者才能修改！"))
		return
	}

	version := g.ctx.GenSeq(common.GroupSeqKey)
	group.Version = version

	// TODO: 这里的写法只支持更新一个属性，如果是多个属性后面需要修改。
	var attrKey string
	for key, value := range groupMap {
		attrKey = key
		switch key {
		case common.GroupAttrKeyName:
			group.Name = value
			break
		case common.GroupAttrKeyNotice:
			group.Notice = value
			break
		case common.GroupAttrKeyInvite:
			invite, _ := strconv.ParseInt(value, 10, 64)
			group.Invite = int(invite)
			break
		}
	}
	tx, err := g.ctx.DB().Begin()
	util.CheckErr(err)

	err = g.db.UpdateTx(group, tx)
	if err != nil {
		tx.Rollback()
		g.Error("更新群信息失败！", zap.Error(err), zap.String("group_no", group.GroupNo), zap.Any("groupMap", groupMap))
		c.ResponseError(errors.New("更新群信息失败！"))
		return
	}
	// 发布群创建事件
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupUpdate,
		Type:  wkevent.Message,
		Data: &config.MsgGroupUpdateReq{
			GroupNo:      groupNo,
			Operator:     loginUID,
			OperatorName: loginName,
			Attr:         attrKey,
			Data:         groupMap,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		g.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	g.ctx.EventCommit(eventID)

	c.ResponseOK()
}

// 添加成员
func (g *Group) memberAdd(c *wkhttp.Context) {
	operator := c.MustGet("uid").(string)
	operatorName := c.MustGet("name").(string)
	var req memberAddReq
	if err := c.BindJSON(&req); err != nil {
		g.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	groupNo := c.Param("group_no")
	/**
	判断群是否存在
	**/
	group, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err), zap.String("groupNo", groupNo))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if group == nil {
		c.ResponseError(errors.New("群不存在！"))
		return
	}
	/**
	判断群是否开启了邀请模式 如果开启了 再判断邀请的人是否是群主或管理员 如果不是则不允许直接添加群成员
	**/
	if group.Invite == 1 {
		creatorOrManager, err := g.db.QueryIsGroupManagerOrCreator(groupNo, operator)
		if err != nil {
			g.Error("查询是否是创建者和管理者失败！", zap.Error(err))
			c.ResponseError(errors.New("查询是否是创建者和管理者失败！"))
			return
		}
		if !creatorOrManager {
			c.ResponseError(errors.New("群开启了邀请模式，不能添加群成员！"))
			return
		}
	}

	err = g.addMembers(req.Members, groupNo, operator, operatorName)
	if err != nil {
		c.ResponseError(err)
		return
	}

	memberCount, err := g.db.QueryMemberCount(groupNo)
	if err != nil {
		g.Error("查询群成员数量失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群成员数量失败！"))
		return
	}

	// 普通群自动升级
	if memberCount >= int64(g.ctx.GetConfig().GroupUpgradeWhenMemberCount) && group.GroupType == int(GroupTypeCommon) {

		var ban = 0
		if group.Status == GroupStatusDisabled {
			ban = 1
		}
		err = g.ctx.IMCreateOrUpdateChannel(&config.ChannelCreateReq{
			ChannelID:   groupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			Ban:         ban,
			Large:       1,
		})
		if err != nil {
			g.Error("更新频道信息失败！", zap.Error(err))
			c.ResponseError(errors.New("更新频道信息失败！"))
			return
		}

		err = g.db.UpdateGroupType(groupNo, GroupTypeSuper)
		if err != nil {
			g.Error("修改群为超级群失败！", zap.Error(err), zap.String("groupNo", groupNo))
			c.ResponseError(errors.New("修改群为超级群失败！"))
			return
		}
		// 发送群升级通知
		err = g.ctx.SendGroupUpgrade(groupNo)
		if err != nil {
			g.Warn("发送群升级通知失败！", zap.Error(err))
		}
	}

	c.ResponseOK()

}

func (g *Group) addMembersTx(members []string, groupNo string, operator, operatorName string, tx *dbr.Tx) (func(), error) {

	/**
	判断操作者是否在群内，如果不在群内是不允许邀请好友的
	**/
	exist, err := g.db.ExistMember(operator, groupNo)
	if err != nil {
		g.Error("查询是否存在群内失败！", zap.Error(err))
		return nil, err
	}
	if !exist {
		return nil, errors.New("群成员不存在群里，不能添加别人！")
	}

	/**
	 获取到真实有效的成员信息
	**/
	tempNewMembers := util.RemoveRepeatedElement(members)
	// 查询用户是否已注销
	userList, err := g.userDB.QueryByUIDs(tempNewMembers)
	if err != nil {
		g.Error("查询添加成员信息错误", zap.Error(err))
		return nil, errors.New("查询添加成员信息错误")
	}
	newMembers := make([]string, 0)
	unableAddMemberVos := make([]*config.UserBaseVo, 0)
	if len(userList) > 0 {
		for _, user := range userList {
			if user.IsDestroy == 1 {
				unableAddMemberVos = append(unableAddMemberVos, &config.UserBaseVo{
					UID:  user.UID,
					Name: user.Name,
				})
			} else {
				newMembers = append(newMembers, user.UID)
			}
		}
	}
	// 如果添加的成员全都已注销则不执行添加到群逻辑
	if len(unableAddMemberVos) == len(tempNewMembers) {
		g.Error("添加用户已注销无法加入群聊", zap.Error(err))
		return nil, errors.New("添加用户已注销无法加入群聊")
	}

	existMembers, err := g.db.QueryMembersWithUids(newMembers, groupNo)
	if err != nil {
		g.Error("查询已在群内存在的成员失败！", zap.Error(err))
		return nil, errors.New("查询已在群内存在的成员失败！")
	}
	// 查询群内黑名单成员
	blacklist, err := g.db.QueryMembersWithStatus(groupNo, int(common.GroupMemberStatusBlacklist))
	if err != nil {
		g.Error("查询群黑名单成员错误", zap.Error(err))
		return nil, errors.New("查询群黑名单成员错误")
	}
	realMembers := make([]string, 0, len(newMembers)) // 真正要添加的群成员
	for _, memberUID := range newMembers {
		exist := false
		for _, existMember := range existMembers {
			if memberUID == existMember.UID {
				exist = true
				break
			}
		}
		if len(blacklist) > 0 {
			for _, blacklistMember := range blacklist {
				if memberUID == blacklistMember.UID {
					exist = true
					break
				}
			}
		}
		if !exist {
			realMembers = append(realMembers, memberUID)
		}
	}
	if len(realMembers) == 0 {
		g.Error("添加的成员已在群内或在群黑名单内", zap.Error(err))
		return nil, errors.New("添加的成员已在群内或在群黑名单内")
	}
	realMemberModels, err := g.userDB.QueryByUIDs(realMembers)
	if err != nil {
		g.Error("查询成员用户信息失败！", zap.Error(err))
		return nil, errors.New("查询成员用户信息失败！")
	}
	memberCount, err := g.db.QueryMemberCount(groupNo)
	if err != nil {
		g.Error("查询群成员数量失败！", zap.Error(err))
		return nil, errors.New("查询群成员数量失败！")
	}
	/**
	 将成员信息存到数据库
	**/
	userBaseVos := make([]*config.UserBaseVo, 0, len(realMembers))
	for _, realMember := range realMemberModels {
		version := g.ctx.GenSeq(common.GroupMemberSeqKey)

		userBaseVos = append(userBaseVos, &config.UserBaseVo{
			UID:  realMember.UID,
			Name: realMember.Name,
		})
		existDelete, err := g.db.ExistMemberDelete(realMember.UID, groupNo)
		if err != nil {
			g.Error("查询是否存在删除成员失败！", zap.Error(err))
			return nil, errors.New("查询是否存在删除成员失败！")
		}
		newMember := &MemberModel{
			GroupNo:   groupNo,
			InviteUID: operator,
			UID:       realMember.UID,
			Vercode:   fmt.Sprintf("%s@%d", util.GenerUUID(), common.GroupMember),
			Version:   version,
			Status:    int(common.GroupMemberStatusNormal),
			Robot:     realMember.Robot,
		}
		if existDelete {
			err = g.db.recoverMemberTx(newMember, tx)
		} else {
			err = g.db.InsertMemberTx(newMember, tx)
		}
		if err != nil {
			g.Error("添加群成员失败！", zap.Error(err))
			return nil, errors.New("添加群成员失败！")
		}
	}

	/**
	发布群成员添加事件
		**/
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupMemberAdd,
		Type:  wkevent.Message,
		Data: &config.MsgGroupMemberAddReq{
			GroupNo:      groupNo,
			Operator:     operator,
			OperatorName: operatorName,
			Members:      userBaseVos,
		},
	}, tx)
	if err != nil {
		g.Error("开启事件失败！", zap.Error(err))
		return nil, errors.New("开启事件失败！")
	}
	var unableAddDestroyAccount int64 = 0
	if len(unableAddMemberVos) > 0 {
		// 发布无法添加到群聊用户
		unableAddDestroyAccount, err = g.ctx.EventBegin(&wkevent.Data{
			Event: event.GroupUnableAddDestroyAccount,
			Type:  wkevent.Message,
			Data: &config.MsgGroupCreateReq{
				GroupNo: groupNo,
				Members: unableAddMemberVos,
			},
		}, tx)
		if err != nil {
			g.Error("开启无法添加到群聊事件失败！", zap.Error(err))
			return nil, errors.New("开启无法添加到群聊事件失败！")
		}
	}
	/**
	 根据目前成员数量判断是否需要发布更新头像事件,如果群主更新过群头像则忽略
	**/
	var groupAvatarEventID int64
	groupIsUploadAvatar, err := g.db.queryGroupAvatarIsUpload(groupNo)
	if err != nil {
		g.Error("查询群头像是否用户上传过失败！", zap.String("group_no", groupNo), zap.Error(err))
	}
	if memberCount < 9 && groupIsUploadAvatar != 1 { // 如果群内已存在群数量小于9且群主未更新过群头像 则需要发布生成群头像的事件

		oldMembers, err := g.db.QueryMembersFirstNine(groupNo)
		if err != nil {
			g.Error("查询先存成员信息失败！", zap.String("group_no", groupNo), zap.Error(err))
			return nil, errors.New("查询先存成员信息失败！")
		}
		ninceMembers := make([]string, 0, 9)
		for _, oldMember := range oldMembers {
			ninceMembers = append(ninceMembers, oldMember.UID)
		}
		if len(ninceMembers)+len(userBaseVos) >= 9 {
			for len(ninceMembers) < 9 {
				for _, userBaseVo := range userBaseVos {
					ninceMembers = append(ninceMembers, userBaseVo.UID)
				}
			}
		} else {
			for _, userBaseVo := range userBaseVos {
				ninceMembers = append(ninceMembers, userBaseVo.UID)
			}
		}

		groupAvatarEventID, err = g.ctx.EventBegin(&wkevent.Data{
			Event: event.GroupAvatarUpdate,
			Type:  wkevent.CMD,
			Data: &config.CMDGroupAvatarUpdateReq{
				GroupNo: groupNo,
				Members: ninceMembers,
			},
		}, tx)
		if err != nil {
			g.Error("开启群成员头像更新事件失败！", zap.Error(err))
			return nil, errors.New("开启群成员头像更新事件失败！")
		}
	}
	// 调用IM的添加订阅者
	err = g.ctx.IMAddSubscriber(&config.SubscriberAddReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Subscribers: realMembers,
	})
	if err != nil {
		g.Error("调用IM的订阅接口失败！", zap.Error(err))
		return nil, errors.New("调用IM的订阅接口失败！")
	}

	return func() {
		// 提交事件
		g.ctx.EventCommit(eventID)
		if groupAvatarEventID != 0 {
			g.ctx.EventCommit(groupAvatarEventID)
		}
		if unableAddDestroyAccount != 0 {
			g.ctx.EventCommit(unableAddDestroyAccount)
		}
	}, nil
}

func (g *Group) addMembers(members []string, groupNo string, operator, operatorName string) error {
	tx, _ := g.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	commitCallback, err := g.addMembersTx(members, groupNo, operator, operatorName, tx)
	if err != nil {
		tx.RollbackUnlessCommitted()
		return err
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		g.Error("提交事务失败！", zap.Error(err))
		return errors.New("提交事务失败！")
	}
	if commitCallback != nil {
		commitCallback()
	}

	return nil
}

// 添加管理员
func (g *Group) managerAdd(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	var memberUIDs []string
	if err := c.BindJSON(&memberUIDs); err != nil {
		g.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(memberUIDs) <= 0 {
		c.ResponseError(errors.New("请选择需要添加的成员！"))
		return
	}
	for _, memberUID := range memberUIDs {
		if memberUID == loginUID {
			c.ResponseError(errors.New("不能将自己设置为管理员！"))
			return
		}
	}
	groupNo := c.Param("group_no")
	isCreator, err := g.db.QueryIsGroupCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否是创建者失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否是创建者失败！"))
		return
	}
	if !isCreator {
		c.ResponseError(errors.New("只有创建者才能设置管理员！"))
		return
	}

	groupModel, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if groupModel == nil {
		c.ResponseError(errors.New("群不存在！"))
		return
	}

	version := g.ctx.GenSeq(common.GroupMemberSeqKey)

	err = g.db.UpdateMembersToManager(groupNo, memberUIDs, version)
	if err != nil {
		g.Error("更新成员为管理员失败！", zap.Any("memberUIDs", memberUIDs), zap.Error(err))
		c.ResponseError(errors.New("更新成员为管理员失败！"))
		return
	}

	if groupModel.Forbidden == 1 { // 如果是禁言状态，则重置管理员白名单
		err = g.setIMWhitelistForGroupManager(groupModel.GroupNo)
		if err != nil {
			c.ResponseError(errors.New("设置白名单失败！"))
			g.Error("设置白名单失败！", zap.Error(err))
			return
		}
	}

	err = g.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupMemberUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
	if err != nil {
		g.Error("发送命令消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送命令消息失败！"))
		return
	}
	c.ResponseOK()
}

// 移除管理员
func (g *Group) managerRemove(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	var memberUIDs []string
	if err := c.BindJSON(&memberUIDs); err != nil {
		g.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(memberUIDs) <= 0 {
		c.ResponseError(errors.New("请选择需要添加的成员！"))
		return
	}
	for _, memberUID := range memberUIDs {
		if memberUID == loginUID {
			c.ResponseError(errors.New("不能将自己移除管理员！"))
			return
		}
	}
	groupNo := c.Param("group_no")

	isCreator, err := g.db.QueryIsGroupCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否是创建者失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否是创建者失败！"))
		return
	}
	if !isCreator {
		c.ResponseError(errors.New("只有创建者才能设置管理员！"))
		return
	}

	groupModel, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if groupModel == nil {
		c.ResponseError(errors.New("群不存在！"))
		return
	}

	version := g.ctx.GenSeq(common.GroupMemberSeqKey)

	err = g.db.UpdateManagersToMember(groupNo, memberUIDs, version)
	if err != nil {
		g.Error("更新成员为管理员失败！", zap.Any("memberUIDs", memberUIDs), zap.Error(err))
		c.ResponseError(errors.New("更新成员为管理员失败！"))
		return
	}

	if groupModel.Forbidden == 1 { // 如果是禁言状态，则重置管理员白名单
		err = g.setIMWhitelistForGroupManager(groupModel.GroupNo)
		if err != nil {
			c.ResponseError(errors.New("设置白名单失败！"))
			g.Error("设置白名单失败！", zap.Error(err))
			return
		}
	}

	err = g.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupMemberUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
	if err != nil {
		g.Error("发送命令消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送命令消息失败！"))
		return
	}
	c.ResponseOK()
}

// 群全员禁言
func (g *Group) groupForbidden(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	loginName := c.MustGet("name").(string)
	groupNo := c.Param("group_no")
	on := c.Param("on")
	isCreatorOrManager, err := g.db.QueryIsGroupManagerOrCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否是创建者失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否是创建者失败！"))
		return
	}
	if !isCreatorOrManager {
		c.ResponseError(errors.New("只有创建者或管理员才能禁言！"))
		return
	}
	groupModel, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err))
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
		managerOrCreaterUIDs, err := g.db.QueryGroupManagerOrCreatorUIDS(groupNo)
		if err != nil {
			c.ResponseErrorf("查询管理者们的uid失败！", err)
			return
		}
		whitelistUIDs = managerOrCreaterUIDs
	}
	// 重置白名单
	err = g.resetIMWhitelist(whitelistUIDs, groupNo)

	if err != nil {
		g.Error("设置禁言失败！", zap.Error(err))
		c.ResponseError(errors.New(err.Error()))
		return
	}

	tx, err := g.ctx.DB().Begin()
	util.CheckErr(err)

	err = g.db.UpdateTx(groupModel, tx)
	if err != nil {
		tx.Rollback()
		g.Error("更新群信息失败！", zap.Error(err), zap.String("group_no", groupModel.GroupNo))
		c.ResponseError(errors.New("更新群信息失败！"))
		return
	}
	// 发布群信息更新事件
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupUpdate,
		Type:  wkevent.Message,
		Data: &config.MsgGroupUpdateReq{
			GroupNo:      groupNo,
			Operator:     loginUID,
			OperatorName: loginName,
			Attr:         common.GroupAttrKeyForbidden,
			Data: map[string]string{
				common.GroupAttrKeyForbidden: on,
			},
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		g.Error("开启群更新事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启群更新事件失败！"))
		return
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	g.ctx.EventCommit(eventID)

	c.ResponseOK()
}

// 设置群管理员（包含创建者）列表作为群白名单
func (g *Group) setIMWhitelistForGroupManager(groupNo string) error {
	managerOrCreaterUIDs, err := g.db.QueryGroupManagerOrCreatorUIDS(groupNo)
	if err != nil {
		return err
	}
	return g.resetIMWhitelist(managerOrCreaterUIDs, groupNo)
}

// 重新设置群管理的白名单
func (g *Group) resetIMWhitelist(whitelist []string, groupNo string) error {
	// 群全员禁言
	err := g.ctx.IMWhitelistSet(config.ChannelWhitelistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   groupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
		},
		UIDs: whitelist,
	})
	if err != nil {
		g.Error("设置白名单失败！", zap.Error(err))
		return err
	}
	return nil

}

// 获取群二维码信息
func (g *Group) groupQRCode(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	groupNo := c.Param("group_no")

	exist, err := g.db.ExistMember(loginUID, groupNo)
	if err != nil {
		g.Error("查询是否存在群内失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否存在群内失败！"))
		return
	}
	if !exist {
		c.ResponseError(errors.New("只有群内用户才能生成二维码！"))
		return
	}

	uuid := util.GenerUUID()
	err = g.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", common.QRCodeCachePrefix, uuid), util.ToJson(common.NewQRCodeModel(common.QRCodeTypeGroup, map[string]interface{}{
		"group_no":  groupNo,
		"generator": loginUID, // 生成者
	})), time.Hour*24*7)
	if err != nil {
		g.Error("设置缓存失败！", zap.Error(err))
		c.ResponseError(errors.New("设置缓存失败！"))
		return
	}
	c.Response(gin.H{
		"day":    7,
		"qrcode": fmt.Sprintf("%s/%s", g.ctx.GetConfig().External.BaseURL, strings.ReplaceAll(g.ctx.GetConfig().QRCodeInfoURL, ":code", uuid)),
		"expire": time.Now().Add(time.Hour * 24 * 7).Format("01月02日"),
	})

}

// 加入群
func (g *Group) groupScanJoin(c *wkhttp.Context) {
	authCode := c.Query("auth_code")
	groupNo := c.Param("group_no")
	authInfo, err := g.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", common.AuthCodeCachePrefix, authCode))
	if err != nil {
		g.Error("获取认证信息数据失败！", zap.Error(err))
		c.ResponseError(errors.New("获取认证信息数据失败！"))
		return
	}
	if authInfo == "" {
		c.ResponseError(errors.New("认证信息不存在或已失效！"))
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
	if authType != string(common.AuthCodeTypeJoinGroup) {
		c.ResponseError(errors.New("授权码不是入群授权码！"))
		return
	}
	authGroupNo := authMap["group_no"].(string)
	if authGroupNo != groupNo {
		c.ResponseError(errors.New("此授权码非此群的！"))
		return
	}
	generator := authMap["generator"].(string)
	if strings.TrimSpace(generator) == "" {
		c.ResponseError(errors.New("没有二维码生成信息！"))
		return
	}
	scaner := authMap["scaner"].(string)
	if strings.TrimSpace(scaner) == "" {
		c.ResponseError(errors.New("没有二维码扫码信息！"))
		return
	}
	existMember, err := g.db.ExistMember(scaner, groupNo)
	if err != nil {
		g.Error("查询是否存在群内时失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否存在群内时失败！"))
		return
	}
	if existMember {
		c.ResponseError(errors.New("已经在群内，不能再加入！"))
		return
	}
	// 查询生成二维码信息
	generatorInfo, err := g.userDB.QueryByUID(generator)
	if err != nil {
		g.Error("获取生成二维码的用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("获取生成二维码的用户信息失败！"))
		return
	}
	if generatorInfo == nil {
		c.ResponseError(errors.New("生成二维码的用户信息不存在！"))
		return
	}
	// 查询扫码者用户信息
	scanerInfo, err := g.userDB.QueryByUID(scaner)
	if err != nil {
		g.Error("查询扫码者用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询扫码者用户信息失败！"))
		return
	}
	if scanerInfo == nil {
		c.ResponseError(errors.New("扫码者信息不存在！"))
		return
	}

	memberCount, err := g.db.QueryMemberCount(groupNo)
	if err != nil {
		g.Error("查询成员数量！", zap.Error(err))
		c.ResponseError(errors.New("查询成员数量！"))
		return
	}

	version := g.ctx.GenSeq(common.GroupMemberSeqKey)

	memberModel := &MemberModel{
		GroupNo:   groupNo,
		UID:       scaner,
		Role:      MemberRoleCommon,
		Version:   version,
		Status:    int(common.GroupMemberStatusNormal),
		InviteUID: generator,
	}

	tx, _ := g.db.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupMemberScanJoin,
		Type:  wkevent.Message,
		Data: config.MsgGroupMemberScanJoin{
			GroupNo:       groupNo,
			Generator:     generatorInfo.UID,
			GeneratorName: generatorInfo.Name,
			Scaner:        scanerInfo.UID,
			ScanerName:    scanerInfo.Name,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		g.Error("开启事件事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件事务失败！"))
		return
	}
	var groupAvatarEventID int64

	groupIsUploadAvatar, err := g.db.queryGroupAvatarIsUpload(groupNo)
	if err != nil {
		g.Error("查询群头像是否用户上传过失败！", zap.String("group_no", groupNo), zap.Error(err))
	}

	if memberCount < 9 && groupIsUploadAvatar != 1 {
		oldMembers, err := g.db.QueryMembersFirstNine(groupNo)
		if err != nil {
			tx.Rollback()
			g.Error("查询先存成员信息失败！", zap.String("group_no", groupNo), zap.Error(err))
			c.ResponseError(errors.New("查询先存成员信息失败！"))
			return
		}
		members := make([]string, 0, len(oldMembers)+1)
		for _, oldMember := range oldMembers {
			members = append(members, oldMember.UID)
		}
		members = append(members, scanerInfo.UID)

		groupAvatarEventID, err = g.ctx.EventBegin(&wkevent.Data{
			Event: event.GroupAvatarUpdate,
			Type:  wkevent.CMD,
			Data: &config.CMDGroupAvatarUpdateReq{
				GroupNo: groupNo,
				Members: members,
			},
		}, tx)
		if err != nil {
			tx.Rollback()
			g.Error("开启群成员头像更新事件失败！", zap.Error(err))
			c.ResponseError(errors.New("开启群成员头像更新事件失败！"))
			return
		}
	}

	existDelete, err := g.db.ExistMemberDelete(scaner, groupNo)
	if err != nil {
		tx.Rollback()
		g.Error("查询是否存在删除成员失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否存在删除成员失败！"))
		return
	}
	if existDelete {
		err = g.db.recoverMemberTx(memberModel, tx)
	} else {
		err = g.db.InsertMemberTx(memberModel, tx)
	}
	if err != nil {
		tx.Rollback()
		g.Error("添加群成员失败！", zap.Error(err))
		c.ResponseError(errors.New("添加群成员失败！"))
		return
	}
	// 调用IM的添加订阅者
	err = g.ctx.IMAddSubscriber(&config.SubscriberAddReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Subscribers: []string{scaner},
	})
	if err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("调用IM的订阅接口失败！", zap.Error(err))
		c.ResponseError(errors.New("调用IM的订阅接口失败！"))
		return
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	g.ctx.EventCommit(eventID)
	if groupAvatarEventID != 0 {
		g.ctx.EventCommit(groupAvatarEventID)
	}

	c.ResponseOK()
}

// 群主转让
func (g *Group) transferGrouper(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	loginName := c.MustGet("name").(string)
	toUID := c.Param("to_uid")
	groupNo := c.Param("group_no")

	/**
	查询转让者用户信息
	**/
	toUser, err := g.userDB.QueryByUID(toUID)
	if err != nil {
		g.Error("查询转让用户失败！", zap.Error(err))
		c.ResponseError(errors.New("查询转让用户失败！"))
		return
	}
	if toUser == nil || toUser.IsDestroy == 1 {
		c.ResponseError(errors.New("转让用户不存在或已注销！"))
		return
	}

	/**
	判断转让的用户是否在群内,只有在群内才能转让
	**/
	// exist, err := g.db.ExistMember(toUID, groupNo)
	// if err != nil {
	// 	g.Error("查询是否存在成员失败！", zap.Error(err))
	// 	c.ResponseError(errors.New("查询是否存在成员失败！"))
	// 	return
	// }
	// if !exist {
	// 	c.ResponseError(errors.New("转让的用户没在群内！"))
	// 	return
	// }
	toMember, err := g.db.QueryMemberWithUID(toUID, groupNo)
	if err != nil {
		g.Error("查询是否存在成员失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否存在成员失败！"))
		return
	}
	if toMember == nil {
		c.ResponseError(errors.New("转让的用户没在群内！"))
		return
	}
	forbiddenExpirTime := toMember.ForbiddenExpirTime
	/**
	判断当前请求转让的用户是否是群主，只有群主才能把群主的位置转让给别人
	**/
	isCreator, err := g.db.QueryIsGroupCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否是群主失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否是群主失败！"))
		return
	}
	if !isCreator {
		c.ResponseError(errors.New("不是群主，不能转让"))
		return
	}

	groupModel, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if groupModel == nil {
		c.ResponseError(errors.New("群不存在！"))
		return
	}

	version := g.ctx.GenSeq(common.GroupMemberSeqKey)
	/**
	修改群主为普通成员，修改转让用户为群主
	**/
	tx, _ := g.db.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupMemberTransferGrouper,
		Type:  wkevent.Message,
		Data: config.MsgGroupTransferGrouper{
			GroupNo:        groupNo,
			OldGrouper:     loginUID,
			OldGrouperName: loginName,
			NewGrouper:     toUID,
			NewGrouperName: toUser.Name,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		g.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}
	err = g.db.UpdateMemberRoleTx(groupNo, loginUID, MemberRoleCommon, version, tx)
	if err != nil {
		tx.Rollback()
		g.Error("更新成普通成员失败！", zap.Error(err))
		c.ResponseError(errors.New("更新成普通成员失败！"))
		return
	}
	err = g.db.UpdateMemberRoleTx(groupNo, toUID, MemberRoleCreator, version, tx)
	if err != nil {
		tx.Rollback()
		g.Error("更新成创建者失败！", zap.Error(err))
		c.ResponseError(errors.New("更新成创建者失败！"))
		return
	}
	// 修改普通成员禁言时长
	err = g.db.updateMemberForbiddenExpirTimeTx(groupNo, toUID, 0, version, tx)
	if err != nil {
		tx.Rollback()
		g.Error("修改成员禁言时长失败！", zap.Error(err))
		c.ResponseError(errors.New("修改成员禁言时长失败！"))
		return
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	g.ctx.EventCommit(eventID)

	if groupModel.Forbidden == 1 { // 如果是禁言状态，则重置管理员白名单
		err = g.setIMWhitelistForGroupManager(groupModel.GroupNo)
		if err != nil {
			tx.Rollback()
			c.ResponseError(errors.New("设置白名单失败！"))
			g.Error("设置白名单失败！", zap.Error(err))
			return
		}
	}
	if forbiddenExpirTime > 0 {
		toUIDs := make([]string, 0)
		toUIDs = append(toUIDs, toUID)
		err = g.ctx.IMBlacklistRemove(config.ChannelBlacklistReq{
			ChannelReq: config.ChannelReq{
				ChannelID:   groupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
			},
			UIDs: toUIDs,
		})
		if err != nil {
			tx.Rollback()
			c.ResponseError(errors.New("新群主添加白名单失败！"))
			g.Error("新群主添加白名单失败！", zap.Error(err))
			return
		}
	}

	c.ResponseOK()

}

// 修改群里群成员信息
func (g *Group) memberUpdate(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	memberUID := c.Param("uid")
	groupNo := c.Param("group_no")
	var memberUpdateMap map[string]interface{}
	if err := c.BindJSON(&memberUpdateMap); err != nil {
		g.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	isManager, err := g.db.QueryIsGroupManagerOrCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否是群管理者失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否是群管理者失败！"))
		return
	}
	if !isManager && loginUID != memberUID {
		g.Error("只有管理员才能修改其他人的成员信息！")
		c.ResponseError(errors.New("只有管理员才能修改其他人的成员信息！"))
		return
	}
	memberModel, err := g.db.QueryMemberWithUID(memberUID, groupNo)
	if err != nil {
		g.Error("查询成员信息失败！", zap.Error(err), zap.String("groupNo", groupNo), zap.String("memberUID", memberUID))
		c.ResponseError(errors.New("查询成员信息失败！"))
		return
	}
	if memberModel == nil {
		c.ResponseError(errors.New("成员信息不存在！"))
		return
	}
	for key, value := range memberUpdateMap {
		switch key {
		case "remark":
			memberModel.Remark = value.(string)
			break
		}
	}
	memberModel.Version = g.ctx.GenSeq(common.GroupMemberSeqKey)
	err = g.db.UpdateMember(memberModel)
	if err != nil {
		g.Error("更新群成员信息失败！", zap.Error(err))
		c.ResponseError(errors.New("更新群成员信息失败！"))
		return
	}
	err = g.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupMemberUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
	if err != nil {
		g.Error("发送命令消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送命令消息失败！"))
		return
	}

	c.ResponseOK()
}

// 移除群成员
func (g *Group) memberRemove(c *wkhttp.Context) {
	operator := c.GetLoginUID()
	operatorName := c.GetLoginName()
	var req memberRemoveReq
	if err := c.BindJSON(&req); err != nil {
		g.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	groupNo := c.Param("group_no")
	req.Members = util.RemoveRepeatedElement(req.Members)

	// 判断群是否存在
	group, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息失败！", zap.Error(err), zap.String("groupNo", groupNo))
		c.ResponseError(errors.New("查询群信息失败！"))
		return
	}
	if group == nil {
		c.ResponseError(errors.New("群不存在！"))
		return
	}
	// 查询操作者身份
	if c.CheckLoginRole() != nil {
		member, err := g.db.QueryMemberWithUID(operator, groupNo)
		if err != nil {
			g.Error("查询操作者群成员信息错误", zap.Error(err))
			c.ResponseError(errors.New("查询操作者群成员信息错误"))
			return
		}
		if member == nil {
			c.ResponseError(errors.New("操作者不再此群"))
			return
		}
		if member.Role != int(common.GroupMemberRoleCreater) && member.Role != int(common.GroupMemberRoleManager) {
			c.ResponseError(errors.New("普通成员无法删除群成员"))
			return
		}
	}
	realDeleteMemberModels, err := g.userDB.QueryByUIDs(req.Members)
	if err != nil {
		g.Error("查询成员用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询成员用户信息失败！"))
		return
	}
	memberCount, err := g.db.QueryMemberCount(groupNo)
	if err != nil {
		g.Error("查询群成员数量失败！", zap.Error(err))
		c.ResponseError(errors.New("查询群成员数量失败！"))
		return
	}

	userBaseVos := make([]*config.UserBaseVo, 0, len(realDeleteMemberModels))
	for _, realMember := range realDeleteMemberModels {
		userBaseVos = append(userBaseVos, &config.UserBaseVo{
			UID:  realMember.UID,
			Name: realMember.Name,
		})
	}
	nowMemberCount := int(memberCount) - len(userBaseVos) // 当前成员数量

	needGenGroupAvatar := false // 是否需要生成头像

	if nowMemberCount < 9 && nowMemberCount > 0 {
		needGenGroupAvatar = true
	}
	if !needGenGroupAvatar {
		needGenGroupAvatar, err = g.db.membersInFirstNine(groupNo, req.Members)
		if err != nil {
			g.Error("查询最早加入的成员信息失败！", zap.Error(err))
			c.ResponseError(errors.New("查询最早加入的成员信息失败！"))
			return
		}
	}
	groupIsUploadAvatar, err := g.db.queryGroupAvatarIsUpload(groupNo)
	if err != nil {
		g.Error("查询群头像是否用户上传过失败！", zap.String("group_no", groupNo), zap.Error(err))
	}
	if groupIsUploadAvatar == 1 {
		needGenGroupAvatar = false
	}

	tx, err := g.db.session.Begin()
	util.CheckErr(err)
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()

	for _, realMember := range realDeleteMemberModels {

		version := g.ctx.GenSeq(common.GroupMemberSeqKey)
		err = g.db.DeleteMemberTx(groupNo, realMember.UID, version, tx)
		if err != nil {
			tx.RollbackUnlessCommitted()
			g.Error("删除群成员失败！", zap.Error(err))
			c.ResponseError(errors.New("删除群成员失败！"))
			return
		}
	}

	// 发布群成员删除事件
	groupMemberRemoveReq := &config.MsgGroupMemberRemoveReq{
		GroupNo:      groupNo,
		Operator:     operator,
		OperatorName: operatorName,
		Members:      userBaseVos,
	}
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.GroupMemberRemove,
		Type:  wkevent.Message,
		Data:  groupMemberRemoveReq,
	}, tx)
	if err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}

	var groupAvatarEventID int64
	if needGenGroupAvatar {
		nineMemberUIDs := make([]string, 0, 9)
		nownineMembers, err := g.db.QueryMembersFirstNineExclude(groupNo, req.Members)
		if err != nil {
			tx.Rollback()
			g.Error("查询先存成员信息失败！", zap.String("group_no", groupNo), zap.Error(err))
			c.ResponseError(errors.New("查询先存成员信息失败！"))
			return
		}
		if len(nownineMembers) > 0 {
			for _, nowninceMember := range nownineMembers {
				nineMemberUIDs = append(nineMemberUIDs, nowninceMember.UID)
			}
		}
		if len(nineMemberUIDs) > 0 {
			groupAvatarEventID, err = g.ctx.EventBegin(&wkevent.Data{
				Event: event.GroupAvatarUpdate,
				Type:  wkevent.CMD,
				Data: &config.CMDGroupAvatarUpdateReq{
					GroupNo: groupNo,
					Members: nineMemberUIDs,
				},
			}, tx)
			if err != nil {
				tx.Rollback()
				g.Error("开启群成员头像更新事件失败！", zap.Error(err))
				c.ResponseError(errors.New("开启群成员头像更新事件失败！"))
				return
			}
		}
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	// 提交事件
	g.ctx.EventCommit(eventID)
	if groupAvatarEventID != 0 {
		g.ctx.EventCommit(groupAvatarEventID)
	}

	// 调用IM的移除订阅者
	err = g.ctx.IMRemoveSubscriber(&config.SubscriberRemoveReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Subscribers: req.Members,
	})
	if err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("调用IM的移除订阅者接口失败！", zap.Error(err))
		c.ResponseError(errors.New("调用IM的移除订阅者接口失败！"))
		return
	}

	//给被踢的成员发送被踢消息
	err = g.ctx.SendGroupMemberBeRemove(groupMemberRemoveReq)
	if err != nil {
		g.Warn("发送群成员被踢消息失败！", zap.Error(err))
	}

	c.ResponseOK()
}

// 修改群设置
func (g *Group) groupSettingUpdate(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string) // 登录用户
	loginName := c.GetLoginName()
	groupNo := c.Param("group_no")

	var resultMap map[string]interface{}
	if err := c.BindJSON(&resultMap); err != nil {
		g.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if len(resultMap) == 0 {
		c.ResponseOK()
		return
	}

	getSettingFnc := func() (*Setting, bool, error) {
		setting, err := g.settingDB.QuerySetting(groupNo, loginUID)
		if err != nil {
			g.Error("查询群设置信息失败！", zap.Error(err))
			return nil, false, err
		}
		insert := false // 是否是插入操作
		version := g.ctx.GenSeq(common.GroupSettingSeqKey)
		if setting == nil { // 不存在设置信息
			insert = true
			setting = newDefaultSetting()
			setting.GroupNo = groupNo
			setting.UID = loginUID
			setting.Version = version
		} else {
			setting.Version = version
		}
		return setting, insert, nil
	}

	getGroupFnc := func() (*Model, error) {
		group, err := g.db.QueryWithGroupNo(groupNo)
		if err != nil {
			g.Error("查询群信息失败", zap.Error(err))
			return nil, err
		}
		if group == nil {
			g.Error("修改的群不存在", zap.Error(err))
			return nil, errors.New("修改的群不存在")
		}
		return group, nil
	}

	for key, value := range resultMap {
		settingActionFnc := settingActionMap[key]
		if settingActionFnc != nil {
			setting, newSetting, err := getSettingFnc()
			if err != nil {
				g.Error("获取设置信息失败！", zap.Error(err))
				c.ResponseError(errors.New("获取设置信息失败！"))
				return
			}
			ctx := &settingContext{
				loginUID:     loginUID,
				loginName:    c.GetLoginName(),
				groupSetting: setting,
				newSetting:   newSetting,
				g:            g,
			}
			err = settingActionFnc(ctx, value)
			if err != nil {
				g.Error("修改群设置信息错误", zap.Error(err))
				c.ResponseError(err)
				return
			}
			continue
		}
		groupUpdateActionFnc := groupUpdateActionMap[key]
		if groupUpdateActionFnc != nil {
			group, err := getGroupFnc()
			if err != nil {
				g.Error("获取群信息失败！", zap.Error(err))
				c.ResponseError(err)
				return
			}
			ctx := &groupUpdateContext{
				loginUID:   loginUID,
				loginName:  loginName,
				groupModel: group,
				g:          g,
			}
			err = groupUpdateActionFnc(ctx, value)
			if err != nil {
				g.Error("修改群设置信息错误", zap.Error(err))
				c.ResponseError(err)
				return
			}
			continue
		}
	}

	c.ResponseOK()
}

// 退出群聊
func (g *Group) groupExit(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	groupNo := c.Param("group_no")

	// 调用IM的移除订阅者
	err := g.ctx.IMRemoveSubscriber(&config.SubscriberRemoveReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		Subscribers: []string{loginUID},
	})
	if err != nil {
		g.Error("移除订阅者失败！", zap.Error(err))
		c.ResponseError(errors.New("移除订阅者失败！"))
		return
	}
	loginMember, err := g.db.QueryMemberWithUID(loginUID, groupNo)
	if err != nil {
		g.Error("查询是否存在群成员失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否存在群成员失败！"))
		return
	}
	if loginMember == nil {
		c.ResponseError(errors.New("群成员不存在群内！"))
		return
	}
	/**
	如果退出的人是群主，则选择第二个入群的人作为群主。
	**/
	var newGrouper *MemberModel // 新群主
	if loginMember.Role == MemberRoleCreator {
		// 查询第二老成员
		newGrouper, err = g.db.QuerySecondOldestMember(groupNo)
		if err != nil {
			g.Error("查询第二元老成员失败！", zap.Error(err))
			c.ResponseError(errors.New("查询第二元老成员失败！"))
			return
		}
	}
	/**
	如果退出的人是普通成员，则直接删除就行
	**/
	version := g.ctx.GenSeq(common.GroupMemberSeqKey)

	tx, err := g.db.session.Begin()
	if err != nil {
		tx.Rollback()
		g.Error("开启数据库事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启数据库事务失败！"))
		return
	}
	eventID, err := g.ctx.EventBegin(&wkevent.Data{
		Event: event.ConversationDelete,
		Type:  wkevent.CMD,
		Data: &config.DeleteConversationReq{
			ChannelID:   groupNo,
			ChannelType: common.ChannelTypeGroup.Uint8(),
			UID:         loginUID,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		g.Error("开启事件事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件事务失败！"))
		return
	}
	if newGrouper != nil {
		err = g.db.UpdateMemberRoleTx(groupNo, newGrouper.UID, MemberRoleCreator, version, tx)
		if err != nil {
			tx.Rollback()
			g.Error("更换新的群主失败！", zap.Error(err))
			c.ResponseError(errors.New("更换新的群主失败！"))
			return
		}
	}
	err = g.db.DeleteMemberTx(groupNo, loginUID, version, tx)
	if err != nil {
		tx.Rollback()
		g.Error("删除群成员失败！", zap.Error(err))
		c.ResponseError(errors.New("删除群成员失败！"))
		return
	}
	groupSetting, err := g.settingDB.querySettingWithTx(groupNo, loginUID, tx)
	if err != nil {
		tx.Rollback()
		g.Error("查询用户群设置错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户群设置错误"))
		return
	}
	if groupSetting != nil && groupSetting.Save == 1 {
		// 清除保存设置
		groupSetting.Save = 0
		err = g.settingDB.UpdateSettingWithTx(groupSetting, tx)
		if err != nil {
			tx.Rollback()
			g.Error("修改群设置信息错误", zap.Error(err))
			c.ResponseError(errors.New("修改群设置信息错误"))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		g.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	g.ctx.EventCommit(eventID)
	// 发送群成员更新命令
	err = g.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupMemberUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
	if err != nil {
		g.Error("发送群更新命令失败！", zap.Error(err), zap.String("groupNo", groupNo))
		c.ResponseError(errors.New("发送群更新命令失败！"))
		return
	}
	var showName = loginMember.Remark
	if showName == "" {
		showName = c.GetLoginName()
	}

	// 发送群成员退出群聊消息
	err = g.ctx.SendGroupExit(groupNo, loginUID, showName)
	if err != nil {
		g.Error("发送成员退出群聊错误", zap.Error(err))
	}

	c.ResponseOK()

}

// 添加或移除黑名单
func (g *Group) blacklist(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	groupNo := c.Param("group_no")
	action := c.Param("action")
	var req blacklistReq
	if err := c.BindJSON(&req); err != nil {
		g.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if len(req.Uids) == 0 {
		c.ResponseError(errors.New("群成员不能为空"))
		return
	}
	if groupNo == "" {
		c.ResponseError(errors.New("群编号不能为空"))
		return
	}
	if action == "" {
		c.ResponseError(errors.New("操作类型不能为空"))
		return
	}
	group, err := g.db.QueryDetailWithGroupNo(groupNo, loginUID)
	if err != nil {
		g.Error("查询群详情错误", zap.Error(err))
		c.ResponseError(errors.New("查询群详情错误"))
		return
	}
	if group == nil {
		g.Error("群不存在", zap.Error(err))
		c.ResponseError(errors.New("群不存在"))
		return
	}
	// 查询是否是管理者
	isManager, err := g.db.QueryIsGroupManagerOrCreator(groupNo, loginUID)
	if err != nil {
		g.Error("查询是否是群管理者失败！", zap.Error(err))
		c.ResponseError(errors.New("查询是否是群管理者失败！"))
		return
	}
	if !isManager {
		c.ResponseError(errors.New("只有群管理者才能修改！"))
		return
	}
	status := 0
	if action == "add" {
		status = int(common.GroupMemberStatusBlacklist)
	} else {
		status = int(common.GroupMemberStatusNormal)
	}

	version := g.ctx.GenSeq(common.GroupMemberSeqKey)
	err = g.db.updateMembersStatus(version, groupNo, status, req.Uids)
	if err != nil {
		g.Error("添加或移除群成员黑名单错误", zap.Error(err))
		c.ResponseError(errors.New("添加或移除群成员黑名单错误！"))
		return
	}
	if status == int(common.GroupMemberStatusBlacklist) {
		err = g.setGroupBlacklist(groupNo, req.Uids, status == int(common.GroupMemberStatusBlacklist))
		if err != nil {
			g.Error("添加IM黑名单错误", zap.Error(err))
			c.ResponseError(errors.New("添加IM黑名单错误"))
			return
		}
	} else {
		members, err := g.db.QueryMembersWithUids(req.Uids, groupNo)
		if err != nil {
			g.Error("查询移除黑名单成员错误", zap.Error(err))
			c.ResponseError(errors.New("查询移除黑名单成员错误"))
			return
		}
		if members == nil || len(members) == 0 {
			c.ResponseError(errors.New("移除成员不存在"))
			return
		}
		removeUIDs := make([]string, 0)
		for _, member := range members {
			if member.ForbiddenExpirTime == 0 {
				removeUIDs = append(removeUIDs, member.UID)
			}
		}
		if len(removeUIDs) > 0 {
			err = g.setGroupBlacklist(groupNo, req.Uids, false)
			if err != nil {
				g.Error("移除IM黑名单错误", zap.Error(err))
				c.ResponseError(errors.New("移除IM黑名单错误"))
				return
			}
		}
	}
	// 发送群成员更新命令
	err = g.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupMemberUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
	if err != nil {
		g.Error("发送更新群成员消息错误", zap.Error(err))
		c.ResponseError(errors.New("发送更新群成员消息错误！"))
		return
	}
	c.ResponseOK()
}

// 禁言时长列表
func (g *Group) forbiddenTimesList(c *wkhttp.Context) {
	type forbiddenTime struct {
		Text string `json:"text"`
		Key  int    `json:"key"`
	}
	list := []*forbiddenTime{
		{
			Text: "1分钟",
			Key:  1,
		},
		{
			Text: "10分钟",
			Key:  2,
		},
		{
			Text: "1小时",
			Key:  3,
		},
		{
			Text: "1天",
			Key:  4,
		},
		{
			Text: "1周",
			Key:  5,
		},
		{
			Text: "1个月",
			Key:  6,
		},
	}
	c.Response(list)
}

// 禁言某个群成员
func (g *Group) forbiddenWithGroupMember(c *wkhttp.Context) {
	type forbiddenWithGroupMemberReq struct {
		MemberUID string `json:"member_uid"`
		Action    int    `json:"action"` // 0.解禁1.禁言
		Key       int    `json:"key"`
	}
	var req forbiddenWithGroupMemberReq
	if err := c.BindJSON(&req); err != nil {
		g.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	loginUID := c.GetLoginUID()
	groupNo := c.Param("group_no")
	if groupNo == "" {
		c.ResponseError(errors.New("群编号不能为空"))
		return
	}
	if req.MemberUID == "" {
		c.ResponseError(errors.New("群成员ID不能为空"))
		return
	}

	if req.Action != 0 && req.Action != 1 {
		c.ResponseError(errors.New("操作类型错误"))
		return
	}
	group, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询群信息错误"))
		return
	}
	if group == nil {
		c.ResponseError(errors.New("操作群不存在"))
		return
	}
	loginGroupMember, err := g.db.QueryMemberWithUID(loginUID, group.GroupNo)
	if err != nil {
		g.Error("查询登录用户群内信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询登录用户群内信息错误"))
		return
	}
	if loginGroupMember == nil {
		c.ResponseError(errors.New("登录用户不在本群内无法操作"))
		return
	}
	member, err := g.db.QueryMemberWithUID(req.MemberUID, group.GroupNo)
	if err != nil {
		g.Error("查询成员信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询成员信息错误"))
		return
	}
	if member == nil {
		c.ResponseError(errors.New("该成员不在群内"))
		return
	}
	if loginGroupMember.Role == MemberRoleCommon || member.Role == MemberRoleCreator || loginGroupMember.Role == member.Role {
		c.ResponseError(errors.New("操作用户权限不够"))
		return
	}
	member.Version = g.ctx.GenSeq(common.GroupMemberSeqKey)
	if req.Action == 0 {
		// 解禁
		member.ForbiddenExpirTime = 0
		err := g.db.UpdateMember(member)
		if err != nil {
			g.Error("解除用户禁言错误", zap.Error(err))
			c.ResponseError(errors.New("解除用户禁言错误"))
			return
		}
	} else {
		expirationTime := time.Now().Unix()
		switch req.Key {
		case 1:
			expirationTime += 60
		case 2:
			expirationTime += 60 * 10
		case 3:
			expirationTime += 60 * 60
		case 4:
			expirationTime += 60 * 60 * 24
		case 5:
			expirationTime += 60 * 60 * 24 * 7
		case 6:
			expirationTime += 60 * 60 * 24 * 30
		default:
			expirationTime = 0
		}
		if expirationTime == 0 {
			c.ResponseError(errors.New("禁言成员时长参数错误"))
			return
		}
		member.ForbiddenExpirTime = expirationTime
		err = g.db.UpdateMember(member)
		if err != nil {
			g.Error("禁言用户错误", zap.Error(err))
			c.ResponseError(errors.New("禁言用户错误"))
			return
		}
	}

	// 加入talk黑名单
	uids := make([]string, 0)
	uids = append(uids, req.MemberUID)
	err = g.setGroupBlacklist(groupNo, uids, req.Action == 1)
	if err != nil {
		c.ResponseError(errors.New("设置IM黑名单错误"))
		return
	}
	err = g.ctx.SendCMD(config.MsgCMDReq{
		ChannelID:   groupNo,
		ChannelType: common.ChannelTypeGroup.Uint8(),
		CMD:         common.CMDGroupMemberUpdate,
		Param: map[string]interface{}{
			"group_no": groupNo,
		},
	})
	if err != nil {
		g.Error("发送命令消息失败！", zap.Error(err))
		c.ResponseError(errors.New("发送命令消息失败！"))
		return
	}
	c.ResponseOK()
}

func (g *Group) CheckForbiddenLoop() {
	var limit int64 = 100
	var errSleep = time.Second * 1
	var noDataSleep = time.Second * 15
	for {
		models, err := g.db.queryForbiddenExpirationTimeMembers(limit)
		if err != nil {
			g.Warn("查询禁言成员信息错误", zap.Error(err))
			time.Sleep(errSleep)
			continue
		}
		if len(models) <= 0 {
			time.Sleep(noDataSleep)
			continue
		}
		for _, model := range models {
			model.Version = g.ctx.GenSeq(common.GroupMemberSeqKey)
			model.ForbiddenExpirTime = 0
			err = g.db.UpdateMember(model)
			if err != nil {
				g.Warn("更新禁言成员新消息错误", zap.Error(err))
				continue
			}
			uids := make([]string, 0)
			uids = append(uids, model.UID)
			if model.Status != int(common.GroupMemberStatusBlacklist) {
				err = g.setGroupBlacklist(model.GroupNo, uids, false)
				if err != nil {
					g.Warn("更新禁言成员新消息错误", zap.Error(err))
					continue
				}
			}
			err = g.ctx.SendCMD(config.MsgCMDReq{
				ChannelID:   model.GroupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
				CMD:         common.CMDGroupMemberUpdate,
				Param: map[string]interface{}{
					"group_no": model.GroupNo,
				},
			})
			if err != nil {
				g.Error("发送命令消息失败！", zap.Error(err))
				continue
			}
		}
	}
}

// 设置talk黑名单
func (g *Group) setGroupBlacklist(groupNo string, uids []string, isAdd bool) error {
	var err error
	if isAdd {
		err = g.ctx.IMBlacklistAdd(config.ChannelBlacklistReq{
			ChannelReq: config.ChannelReq{
				ChannelID:   groupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
			}, UIDs: uids})
	} else {
		println("移除黑名单--->")
		err = g.ctx.IMBlacklistRemove(config.ChannelBlacklistReq{
			ChannelReq: config.ChannelReq{
				ChannelID:   groupNo,
				ChannelType: common.ChannelTypeGroup.Uint8(),
			}, UIDs: uids})
	}
	if err != nil {
		g.Error("设置群黑名单错误", zap.Error(err))
		return err
	}
	return nil
}

// ---------- vo ----------

type groupDetailResp struct {
	GroupNo     string `json:"group_no"`  // 群编号
	Name        string `json:"name"`      // 群名称
	Notice      string `json:"notice"`    // 群公告
	Forbidden   int    `json:"forbidden"` // 是否全员禁言
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	MemberCount int64  `json:"member_count"` // 成员数量
	Version     int64  `json:"version"`      // 群数据版本
}

func (g groupDetailResp) from(model *Model, memberCount int64) groupDetailResp {
	return groupDetailResp{
		GroupNo:     model.GroupNo,
		Name:        model.Name,
		Notice:      model.Notice,
		Version:     model.Version,
		Forbidden:   model.Forbidden,
		MemberCount: memberCount,
		CreatedAt:   model.CreatedAt.String(),
		UpdatedAt:   model.UpdatedAt.String(),
	}
}

// 成员详情model
type memberDetailResp struct {
	ID                 uint64 `json:"id"`
	UID                string `json:"uid"`                  // 成员uid
	GroupNo            string `json:"group_no"`             // 群唯一编号
	Name               string `json:"name"`                 // 群成员名称
	Remark             string `json:"remark"`               // 成员备注
	Role               int    `json:"role"`                 // 成员角色
	Version            int64  `json:"version"`              // 版本号
	IsDeleted          int    `json:"is_deleted"`           // 是否删除
	Status             int    `json:"status"`               //成员状态0:正常，2:黑名单
	Vercode            string `json:"vercode"`              // 验证码
	InviteUID          string `json:"invite_uid"`           // 邀请人
	Robot              int    `json:"robot"`                // 机器人
	ForbiddenExpirTime int64  `json:"forbidden_expir_time"` // 禁言时长
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

func (r memberDetailResp) from(model *MemberDetailModel) memberDetailResp {
	return memberDetailResp{
		ID:                 uint64(model.Id),
		UID:                model.UID,
		GroupNo:            model.GroupNo,
		Name:               model.Name,
		Remark:             model.Remark,
		Role:               model.Role,
		Version:            model.Version,
		IsDeleted:          model.IsDeleted,
		Status:             model.Status,
		Vercode:            model.Vercode,
		InviteUID:          model.InviteUID,
		Robot:              model.Robot,
		ForbiddenExpirTime: model.ForbiddenExpirTime,
		CreatedAt:          model.CreatedAt.String(),
		UpdatedAt:          model.UpdatedAt.String(),
	}
}

type groupReq struct {
	Name    string   `json:"name"`    // 群名
	Members []string `json:"members"` // 成员uid
}

func (g groupReq) Check() error {
	if len(g.Members) <= 0 {
		return errors.New("群成员不能为空！")
	}
	return nil
}

type memberUpdateReq struct {
	UID    string `json:"uid"`
	Remark string `json:"remark"`
}

func (m memberUpdateReq) Check() error {
	if strings.TrimSpace(m.UID) == "" {
		return errors.New("uid不能为空！")
	}
	return nil
}

// 添加或移除黑名单
type blacklistReq struct {
	Uids []string `json:"uids"` //成员uid
}
type memberAddReq struct {
	Members []string `json:"members"` // 成员uid
}

func (m memberAddReq) Check() error {
	if len(m.Members) <= 0 {
		return errors.New("群成员不能为空！")
	}
	return nil
}

type memberRemoveReq struct {
	Members []string `json:"members"` // 成员uid
}

func (m memberRemoveReq) Check() error {
	if len(m.Members) <= 0 {
		return errors.New("群成员不能为空！")
	}
	return nil
}
