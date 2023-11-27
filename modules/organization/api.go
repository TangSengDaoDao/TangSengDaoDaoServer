package organization

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	common2 "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/file"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

type Organization struct {
	ctx           *config.Context
	commonService common2.IService
	groupService  group.IService
	userService   user.IService
	fileService   file.IService
	log.Log
	db *db
}

func New(ctx *config.Context) *Organization {
	return &Organization{
		ctx:           ctx,
		Log:           log.NewTLog("Organization"),
		commonService: common2.NewService(ctx),
		groupService:  group.NewService(ctx),
		userService:   user.NewService(ctx),
		fileService:   file.NewService(ctx),
		db:            newDB(ctx),
	}
}
func (o *Organization) Route(l *wkhttp.WKHttp) {
	org := l.Group("/v1/organization", o.ctx.AuthMiddleware(l))
	{
		org.POST("", o.createOrg)        // 创建组织
		org.DELETE("", o.deleteOrg)      // 删除组织
		org.GET("/joined", o.joinedOrgs) // 已加入企业
	}
	orgs := l.Group("/v1/organizations", o.ctx.AuthMiddleware(l))
	{
		orgs.POST("/:org_id/logo", o.uploadOrgLogo)             // 上传组织logo
		orgs.GET("/:org_id", o.getOrg)                          // 获取组织信息
		orgs.POST("/:org_id/join", o.join)                      // 加入组织
		orgs.DELETE("/:org_id/quit", o.quit)                    // 退出组织
		orgs.GET("/:org_id/invitecode", o.getInviteCode)        // 获取邀请用户验证码
		orgs.DELETE("/:org_id/employees", o.deleteEmployees)    // 批量删除成员
		orgs.PUT("/:org_id/employees", o.updateEmployeesDept)   // 批量转移成员
		orgs.GET("/:org_id/employees/search", o.searchEmployee) // 搜索组织员工

		orgs.POST("/:org_id/department", o.createDept)            // 创建部门
		orgs.PUT("/:org_id/department/:dept_id", o.updateDept)    // 编辑部门
		orgs.DELETE("/:org_id/department/:dept_id", o.deleteDept) // 删除部门
		orgs.GET("/:org_id/department/:dept_id", o.getDept)       // 获取部门详情

	}
	openOrgs := l.Group("/v1/organizations")
	{
		openOrgs.GET("/:org_id/logo", o.getLogo) // 获取组织头像
	}
}
func (o *Organization) uploadOrgLogo(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	orgId := c.Param("org_id")
	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织信息错误"))
		return
	}
	if org == nil {
		c.ResponseError(errors.New("该组织不存在"))
		return
	}
	if org.Creator != loginUID {
		c.ResponseError(errors.New("此用户无权修改组织头像"))
		return
	}
	if c.Request.MultipartForm == nil {
		err := c.Request.ParseMultipartForm(1024 * 1024 * 20) // 20M
		if err != nil {
			o.Error("数据格式不正确！", zap.Error(err))
			c.ResponseError(errors.New("数据格式不正确！"))
			return
		}
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		o.Error("读取文件失败！", zap.Error(err))
		c.ResponseError(errors.New("读取文件失败！"))
		return
	}
	_, err = o.fileService.UploadFile(fmt.Sprintf("organization/logo/%s.png", orgId), "image/png", func(w io.Writer) error {
		_, err := io.Copy(w, file)
		return err
	})
	defer file.Close()
	if err != nil {
		o.Error("上传文件失败！", zap.Error(err))
		c.ResponseError(errors.New("上传文件失败！"))
		return
	}
	orgEmployees, err := o.db.queryOrgEmployees(orgId)
	if err != nil {
		o.Error("查询员工错误")
		return
	}
	if len(orgEmployees) > 0 {
		uids := make([]string, 0)
		for _, employee := range orgEmployees {
			uids = append(uids, employee.EmployeeUid)
		}
		// 发送头像更新命令
		err = o.ctx.SendCMD(config.MsgCMDReq{
			CMD:         common.CMDOrganizationLogoUpdate,
			Subscribers: uids,
			Param: map[string]interface{}{
				"org_id": orgId,
			},
		})
		if err != nil {
			o.Error("发送组织头像更新命令失败！")
			return
		}
	}
	//更改用户上传头像状态
	org.IsUploadLogo = 1
	err = o.db.updateOrg(org)
	if err != nil {
		o.Error("修改组织是否修改头像错误！", zap.Error(err))
		c.ResponseError(errors.New("修改组织是否修改头像错误！"))
		return
	}
	c.ResponseOK()
}
func (o *Organization) getLogo(c *wkhttp.Context) {
	orgId := c.Param("org_id")
	v := c.Query("v")
	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织信息错误"))
		return
	}
	if org == nil {
		c.ResponseError(errors.New("组织信息不存在"))
		return
	}
	if org.IsUploadLogo == 0 {
		c.Header("Content-Type", "image/jpeg")
		avatarBytes, err := ioutil.ReadFile("assets/assets/org_avatar.jpeg")
		if err != nil {
			o.Error("头像读取失败！", zap.Error(err))
			c.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		c.Writer.Write(avatarBytes)
		return
	}
	path := fmt.Sprintf("/organization/logo/%s.png", orgId)
	downloadUrl, err := o.fileService.DownloadURL(path, "org_avatar.jpeg")
	if err != nil {
		o.Error("获取下载路径失败！", zap.Error(err))
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("%s?%s", downloadUrl, v))
}

func (o *Organization) searchEmployee(c *wkhttp.Context) {
	orgId := c.Param("org_id")
	keyword := c.Query("keyword")
	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	if keyword == "" {
		c.ResponseError(errors.New("搜索关键字不能为空"))
		return
	}
	employees, err := o.db.searchEmployee(orgId, keyword)
	if err != nil {
		o.Error("搜索员工错误", zap.Error(err))
		c.ResponseError(errors.New("搜索员工错误"))
		return
	}
	uids := make([]string, 0)
	if len(employees) > 0 {
		for _, m := range employees {
			uids = append(uids, m.EmployeeUid)
		}
	}
	users, err := o.userService.GetUsers(uids)
	if err != nil {
		o.Error("查询用户资料错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户资料错误"))
		return
	}
	list := make([]*searchResp, 0)
	if len(employees) > 0 {
		for _, m := range employees {
			username := ""
			if len(users) > 0 {
				for _, user := range users {
					if user.UID == m.EmployeeUid {
						username = user.Name
						break
					}
				}
			}
			list = append(list, &searchResp{
				EmployeeUid:  m.EmployeeUid,
				EmployeeName: m.EmployeeName,
				Role:         m.Role,
				Username:     username,
			})
		}
	}
	c.Response(list)
}

func (o *Organization) joinedOrgs(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	employees, err := o.db.queryOrgEmployeesWithUid(loginUID)
	if err != nil {
		o.Error("查询已加入组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询已加入组织信息错误"))
		return
	}
	list := make([]*orgResp, 0)
	if len(employees) > 0 {
		orgIds := make([]string, 0)
		for _, m := range employees {
			orgIds = append(orgIds, m.OrgId)
		}
		orgs, err := o.db.queryOrgsWithOrgIds(orgIds)
		if err != nil {
			o.Error("查询组织信息错误", zap.Error(err))
			c.ResponseError(errors.New("查询组织信息错误"))
			return
		}
		if len(orgs) > 0 {
			for _, org := range orgs {
				list = append(list, &orgResp{
					OrgId:        org.OrgId,
					Creator:      org.Creator,
					Name:         org.Name,
					ShortNo:      org.ShortNo,
					IsUploadLogo: org.IsUploadLogo,
				})
			}
		}
	}

	c.Response(list)
}

func (o *Organization) updateEmployeesDept(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	loginName := c.GetLoginName()
	orgId := c.Param("org_id")
	type reqVO struct {
		Uids    []string `json:"uids"`
		DeptIds []string `json:"dept_ids"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("请求数据格式有误！", err)
		return
	}

	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	if len(req.DeptIds) == 0 {
		c.ResponseError(errors.New("部门id不能为空"))
		return
	}
	if len(req.Uids) == 0 {
		c.ResponseError(errors.New("成员id不能为空"))
		return
	}
	err := o.checkRole(loginUID, orgId)
	if err != nil {
		c.ResponseError(err)
		return
	}
	users, err := o.userService.GetUsers(req.Uids)
	if err != nil {
		o.Error("查询成员信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询成员信息错误"))
		return
	}
	orgDemployees, err := o.db.queryOrgEmployees(orgId)
	if err != nil {
		o.Error("查询组织成员信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织成员信息错误"))
		return
	}
	if len(orgDemployees) == 0 {
		c.ResponseError(errors.New("组织下无成员信息"))
		return
	}
	// 不属于该组织的用户
	absentUids := make([]string, 0)
	for _, uid := range req.Uids {
		isAdd := true
		for _, employee := range orgDemployees {
			if employee.EmployeeUid == uid {
				isAdd = false
				break
			}
		}
		if isAdd {
			absentUids = append(absentUids, uid)
		}
	}
	if len(absentUids) > 0 {
		users, err := o.userService.GetUsers(absentUids)
		if err != nil {
			o.Error("查询一批用户信息错误", zap.Error(err))
			c.ResponseError(errors.New("查询一批用户信息错误"))
			return
		}
		var responseContent = ""
		if len(users) > 0 {
			var names bytes.Buffer
			for index, user := range users {
				if index != 0 {
					names.WriteString("、")
				}
				names.WriteString(fmt.Sprintf("`%s`", user.Name))
			}
			responseContent = fmt.Sprintf("用户 %s 不在该组织内", names.String())
		} else {
			responseContent = "选择用户不在组织内"
		}
		c.ResponseError(errors.New(responseContent))
		return
	}
	// 获取涉及到的部门
	deptIds := make([]string, 0)
	employees, err := o.db.queryDeptEmployeesWithOrgIdAndUids(orgId, req.Uids)
	if err != nil {
		o.Error("查询用户所在部门错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户所在部门错误"))
		return
	}
	if len(employees) > 0 {
		for _, m := range employees {
			deptIds = append(deptIds, m.DeptId)
		}
	}
	deptIds = append(deptIds, req.DeptIds...)
	depts, err := o.db.queryDeptsWithIds(deptIds)
	if err != nil {
		o.Error("批量查询部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("批量查询部门信息错误"))
		return
	}
	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	type tempDeptEmployee struct {
		Uid    string
		DeptId string
	}
	allList := make([]*tempDeptEmployee, 0)
	addList := make([]*tempDeptEmployee, 0)
	deleteList := make([]*tempDeptEmployee, 0)
	for _, deptId := range req.DeptIds {
		for _, uid := range req.Uids {
			allList = append(allList, &tempDeptEmployee{
				Uid:    uid,
				DeptId: deptId,
			})
		}
	}
	if len(employees) == 0 {
		for _, m := range allList {
			addList = append(addList, &tempDeptEmployee{
				Uid:    m.Uid,
				DeptId: m.DeptId,
			})
		}
	} else {
		// 得到需要删除的部门成员
		for _, employee := range employees {
			isAdd := true
			for _, md := range allList {
				if md.DeptId == employee.DeptId && md.Uid == employee.EmployeeUid {
					isAdd = false
					break
				}
			}
			if isAdd {
				deleteList = append(deleteList, &tempDeptEmployee{
					DeptId: employee.DeptId,
					Uid:    employee.EmployeeUid,
				})
			}
		}
		// 得到需要添加的部门成员
		for _, md := range allList {
			isAdd := true
			for _, employee := range employees {
				if employee.EmployeeUid == md.Uid && employee.DeptId == md.DeptId {
					isAdd = false
					break
				}
			}
			if isAdd {
				addList = append(addList, &tempDeptEmployee{
					Uid:    md.Uid,
					DeptId: md.DeptId,
				})
			}
		}
	}
	if len(deleteList) > 0 {
		for _, md := range deleteList {
			err = o.db.deleteDeptEmployeeTx(md.Uid, md.DeptId, tx)
			if err != nil {
				tx.Rollback()
				o.Error("删除部门成员错误", zap.Error(err))
				c.ResponseError(errors.New("删除部门成员错误"))
				return
			}
		}
	}
	if len(addList) > 0 {
		for _, md := range addList {
			err = o.db.insertDeptEmployeeTx(&deptEmployeeModel{
				OrgId:         orgId,
				DeptId:        md.DeptId,
				EmployeeUid:   md.Uid,
				WorkforceType: WorkforceTypes[0],
			}, tx)
			if err != nil {
				tx.Rollback()
				o.Error("新增部门成员错误", zap.Error(err))
				c.ResponseError(errors.New("新增部门成员错误"))
				return
			}
		}
	}

	// 更新对应群成员信息
	list := make([]*config.OrgOrDeptEmployeeVO, 0)
	if len(addList) > 0 {
		for _, employee := range addList {
			isAdd := true
			for _, dept := range depts {
				if dept.DeptId == employee.DeptId && dept.IsCreatedGroup == 0 {
					isAdd = false
					break
				}
			}
			if !isAdd {
				continue
			}
			employeeName := employee.Uid
			if len(users) > 0 {
				for _, user := range users {
					if employee.Uid == user.UID {
						employeeName = user.Name
						break
					}
				}
			}
			list = append(list, &config.OrgOrDeptEmployeeVO{
				Operator:     loginUID,
				OperatorName: loginName,
				EmployeeUid:  employee.Uid,
				EmployeeName: employeeName,
				GroupNo:      employee.DeptId,
				Action:       "add",
			})
		}
	}

	if len(deleteList) > 0 {
		for _, employee := range deleteList {
			isAdd := true
			for _, dept := range depts {
				if dept.DeptId == employee.DeptId && dept.IsCreatedGroup == 0 {
					isAdd = false
					break
				}
			}
			if !isAdd {
				continue
			}
			employeeName := employee.Uid
			if len(users) > 0 {
				for _, user := range users {
					if employee.Uid == user.UID {
						employeeName = user.Name
						break
					}
				}
			}
			list = append(list, &config.OrgOrDeptEmployeeVO{
				Operator:     loginUID,
				OperatorName: loginName,
				EmployeeUid:  employee.Uid,
				EmployeeName: employeeName,
				GroupNo:      employee.DeptId,
				Action:       "delete",
			})
		}
	}
	// 发布加入或退出部门群事件
	eventID, err := o.ctx.EventBegin(&wkevent.Data{
		Event: event.OrgOrDeptEmployeeUpdate,
		Type:  wkevent.None,
		Data: &config.MsgOrgOrDeptEmployeeUpdateReq{
			Members: list,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}
	if err = tx.Commit(); err != nil {
		tx.Rollback()
		o.Error("提交事物错误", zap.Error(err))
		c.ResponseError(errors.New("提交事物错误"))
		return
	}
	o.ctx.EventCommit(eventID)
	c.ResponseOK()
}

func (o *Organization) deleteEmployees(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	loginName := c.GetLoginName()
	orgId := c.Param("org_id")
	type reqVO struct {
		Uids []string `json:"uids"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("请求数据格式有误！", err)
		return
	}

	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	if len(req.Uids) == 0 {
		c.ResponseError(errors.New("删除成员列表不能为空"))
		return
	}
	err := o.checkRole(loginUID, orgId)
	if err != nil {
		c.ResponseError(err)
		return
	}
	employees, err := o.db.queryDeptEmployeesWithOrgIdAndUids(orgId, req.Uids)
	if err != nil {
		o.Error("查询组织下成员错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织下成员错误"))
		return
	}
	deptIds := make([]string, 0)
	if len(employees) > 0 {
		for _, m := range employees {
			deptIds = append(deptIds, m.DeptId)
		}
	}
	depts, err := o.db.queryDeptsWithIds(deptIds)
	if err != nil {
		o.Error("查询部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询部门信息错误"))
		return
	}
	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err = o.db.deleteOrgEmployeesTx(orgId, req.Uids, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除组织下成员错误", zap.Error(err))
		c.ResponseError(errors.New("删除组织下成员错误"))
		return
	}
	err = o.db.deleteDeptEmployeesWithOrgIdAndUidsTx(orgId, req.Uids, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除部门下成员错误", zap.Error(err))
		c.ResponseError(errors.New("删除部门下成员错误"))
		return
	}

	list := make([]*config.OrgOrDeptEmployeeVO, 0)
	// 移除全员群
	for _, uid := range req.Uids {
		list = append(list, &config.OrgOrDeptEmployeeVO{
			Operator:     loginUID,
			OperatorName: loginName,
			EmployeeUid:  uid,
			EmployeeName: uid,
			GroupNo:      orgId,
			Action:       "delete",
		})
	}
	if len(employees) > 0 {
		for _, employee := range employees {
			isAdd := false
			if len(depts) > 0 {
				for _, dept := range depts {
					if dept.DeptId == employee.DeptId && dept.IsCreatedGroup == 1 {
						isAdd = true
						break
					}
				}
			}
			if isAdd {
				list = append(list, &config.OrgOrDeptEmployeeVO{
					Operator:     loginUID,
					OperatorName: loginName,
					EmployeeUid:  employee.EmployeeUid,
					EmployeeName: employee.EmployeeUid,
					GroupNo:      employee.DeptId,
					Action:       "delete",
				})
			}
		}
	}
	// 发布退出组织和相关部门事件
	eventID, err := o.ctx.EventBegin(&wkevent.Data{
		Event: event.OrgOrDeptEmployeeUpdate,
		Type:  wkevent.Message,
		Data: &config.MsgOrgOrDeptEmployeeUpdateReq{
			Members: list,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		o.Error("提交事物错误", zap.Error(err))
		c.ResponseError(errors.New("提交事物错误"))
		return
	}
	o.ctx.EventCommit(eventID)

	c.ResponseOK()
}

func (o *Organization) getDept(c *wkhttp.Context) {
	deptId := c.Param("dept_id")
	orgId := c.Param("org_id")
	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	if deptId == "" {
		c.ResponseError(errors.New("部门ID不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织信息错误"))
		return
	}
	if org == nil {
		c.ResponseError(errors.New("组织不存在"))
		return
	}
	dept, err := o.db.queryDeptWithId(deptId)
	if err != nil {
		o.Error("查询部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询部门信息错误"))
		return
	}
	if dept == nil {
		c.ResponseError(errors.New("该部门不存在"))
		return
	}
	if org.OrgId != dept.OrgId {
		c.ResponseError(errors.New("该部门不在此组织下"))
		return
	}
	// 查询子部门
	childDepts, err := o.db.queryDeptsWithParentId(deptId)
	if err != nil {
		o.Error("查询子部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询子部门信息错误"))
		return
	}
	// 查询员工
	employees, err := o.db.queryDeptEmployeesWithDeptId(deptId)
	if err != nil {
		o.Error("查询员工错误", zap.Error(err))
		c.ResponseError(errors.New("查询员工错误"))
		return
	}
	uids := make([]string, 0)
	if len(employees) > 0 {
		for _, m := range employees {
			uids = append(uids, m.EmployeeUid)
		}
	}
	var orgEmployees []*orgEmployeeModel
	if len(uids) > 0 {
		// 查询组织成员
		orgEmployees, err = o.db.queryOrgEmployeesWithOrgIdAndUids(orgId, uids)
		if err != nil {
			o.Error("查询组织内员工信息错误", zap.Error(err))
			c.ResponseError(errors.New("查询组织内员工信息错误"))
			return
		}
	}
	// 查询用户资料
	users, err := o.userService.GetUsers(uids)
	if err != nil {
		o.Error("查询用户资料错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户资料错误"))
		return
	}
	deptDeptResp := &deptDetailResp{}
	deptDeptResp.DeptId = deptId
	deptDeptResp.IsCreatedGroup = dept.IsCreatedGroup
	deptDeptResp.Name = dept.Name
	deptDeptResp.ParentId = dept.ParentId
	deptDeptResp.OrgId = orgId
	employeeRespList := make([]*deptEmployeeResp, 0)
	if len(employees) > 0 {
		for _, m := range employees {
			role := OrgEmployeeNormal
			userName := ""
			employeeName := ""
			if len(orgEmployees) > 0 {
				for _, orgM := range orgEmployees {
					if orgM.EmployeeUid == m.EmployeeUid {
						role = orgM.Role
						employeeName = orgM.EmployeeName
						break
					}
				}
			}
			if len(users) > 0 {
				for _, user := range users {
					if m.EmployeeUid == user.UID {
						userName = user.Name
						break
					}
				}
			}
			employeeRespList = append(employeeRespList, &deptEmployeeResp{
				OrgId:         m.OrgId,
				DeptId:        m.DeptId,
				EmployeeId:    m.EmployeeId,
				EmployeeUid:   m.EmployeeUid,
				JobTitle:      m.JobTitle,
				WorkforceType: m.WorkforceType,
				EmployeeName:  employeeName,
				Username:      userName,
				Role:          role,
			})
		}
	}
	childDeptRespList := make([]*deptResp, 0)
	if len(childDepts) > 0 {
		for _, dept := range childDepts {
			childDeptRespList = append(childDeptRespList, &deptResp{
				DeptId:         dept.DeptId,
				Name:           dept.Name,
				OrgId:          dept.OrgId,
				ParentId:       dept.ParentId,
				IsCreatedGroup: dept.IsCreatedGroup,
				ShortNo:        dept.ShortNo,
			})
		}
	}
	deptDeptResp.ChildDepts = childDeptRespList
	deptDeptResp.Employees = employeeRespList
	c.Response(deptDeptResp)
}

func (o *Organization) deleteDept(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	deptId := c.Param("dept_id")
	orgId := c.Param("org_id")
	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	if deptId == "" {
		c.ResponseError(errors.New("部门ID不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织信息错误"))
		return
	}
	if org == nil {
		c.ResponseError(errors.New("组织不存在"))
		return
	}
	dept, err := o.db.queryDeptWithId(deptId)
	if err != nil {
		o.Error("查询部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询部门信息错误"))
		return
	}
	if dept == nil {
		c.ResponseError(errors.New("该部门不存在"))
		return
	}
	if dept.OrgId != orgId {
		c.ResponseError(errors.New("该部门不在此组织下"))
		return
	}
	err = o.checkRole(loginUID, dept.OrgId)
	if err != nil {
		c.ResponseError(err)
		return
	}

	employees, err := o.db.queryDeptEmployeesWithDeptId(deptId)
	if err != nil {
		o.Error("查询部门下成员错误", zap.Error(err))
		c.ResponseError(errors.New("查询部门下成员错误"))
		return
	}
	if len(employees) > 0 {
		c.ResponseError(errors.New("请先移除部门内所有人员再进行删除部门操作"))
		return
	}
	err = o.db.deleteDeptWithId(deptId)
	if err != nil {
		o.Error("删除部门错误", zap.Error(err))
		c.ResponseError(errors.New("删除部门错误"))
		return
	}
	c.ResponseOK()
}

func (o *Organization) updateDept(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	deptId := c.Param("dept_id")
	orgId := c.Param("org_id")
	type reqVO struct {
		Name     string `json:"name"`
		ParentId string `json:"parent_id"`
		ShortNo  string `json:"short_no"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("请求数据格式有误！", err)
		return
	}
	if deptId == "" {
		c.ResponseError(errors.New("部门Id不能为空"))
		return
	}
	if req.Name == "" {
		c.ResponseError(errors.New("部门名称不能为空"))
		return
	}
	if req.ParentId == "" {
		c.ResponseError(errors.New("上级部门ID不能为空"))
	}
	err := o.checkRole(loginUID, orgId)
	if err != nil {
		c.ResponseError(err)
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织信息错误"))
		return
	}
	dept, err := o.db.queryDeptWithId(deptId)
	if err != nil {
		o.Error("查询部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询部门信息错误"))
		return
	}
	if dept == nil {
		c.ResponseError(errors.New("该部门不存在或已解散"))
		return
	}
	if req.ShortNo == "" {
		req.ShortNo = dept.ShortNo
	} else {
		tempDept, err := o.db.queryDeptWithShortNo(req.ShortNo)
		if err != nil {
			o.Error("通过短号查询部门信息错误", zap.Error(err))
			c.ResponseError(errors.New("通过短号查询部门信息错误"))
			return
		}
		if tempDept != nil && tempDept.DeptId != dept.DeptId {
			c.ResponseError(errors.New("该短号已存在"))
			return
		}
	}

	parentDept, err := o.db.queryDeptWithId(req.ParentId)
	if err != nil {
		o.Error("查询上级部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询上级部门信息错误"))
		return
	}
	if org == nil && parentDept == nil {
		c.ResponseError(errors.New("上级部门部门不存在"))
		return
	}
	dept.Name = req.Name
	dept.ParentId = req.ParentId
	dept.ShortNo = req.ShortNo
	err = o.db.updateDept(dept)
	if err != nil {
		o.Error("修改部门信息错误", zap.Error(err))
		c.ResponseError(errors.New("修改部门信息错误"))
		return
	}
	c.ResponseOK()
}

func (o *Organization) getOrg(c *wkhttp.Context) {
	orgId := c.Param("org_id")
	if orgId == "" {
		c.ResponseError(errors.New("组织id不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织错误"))
		return
	}
	c.Response(&orgResp{
		Name:         org.Name,
		Creator:      org.Creator,
		OrgId:        org.OrgId,
		ShortNo:      org.ShortNo,
		IsUploadLogo: org.IsUploadLogo,
	})
}
func (o *Organization) deleteOrg(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	// loginName := c.GetLoginName()
	orgId := c.Query("org_id")
	if orgId == "" {
		c.ResponseError(errors.New("组织id不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织错误"))
		return
	}
	if org == nil || org.Creator != loginUID {
		c.ResponseError(errors.New("用户无权执行此操作"))
		return
	}
	// depts, err := o.db.queryDeptsWithOrgId(orgId)
	// if err != nil {
	// 	o.Error("查询组织下部门错误", zap.Error(err))
	// 	c.ResponseError(errors.New("查询组织下部门错误"))
	// 	return
	// }
	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err = o.db.deleteOrgTx(orgId, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除组织错误", zap.Error(err))
		c.ResponseError(errors.New("删除组织错误"))
		return
	}
	err = o.db.deleteOrgEmployeeWithOrgIdTx(orgId, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除组织成员错误", zap.Error(err))
		c.ResponseError(errors.New("删除组织成员错误"))
		return
	}
	err = o.db.deleteDeptWithOrgIdTx(orgId, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除部门错误", zap.Error(err))
		c.ResponseError(errors.New("删除部门错误"))
		return
	}
	err = o.db.deleteDeptEmployeesWithOrgidTx(orgId, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除部门成员错误", zap.Error(err))
		c.ResponseError(errors.New("删除部门成员错误"))
		return
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		o.Error("事物提交错误", zap.Error(err))
		c.ResponseError(errors.New("事物提交错误"))
		return
	}
	// todo 解散群
	// list := make([]*config.OrgOrDeptVO, 0)
	// list = append(list, &config.OrgOrDeptVO{})
	c.ResponseOK()
}

func (o *Organization) quit(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	orgId := c.Param("org_id")
	if orgId == "" {
		c.ResponseError(errors.New("组织id不能为空"))
		return
	}
	orgEmployee, err := o.db.queryOrgEmployee(loginUID, orgId)
	if err != nil {
		o.Error("查询组织成员错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织成员错误"))
		return
	}
	if orgEmployee == nil {
		c.ResponseError(errors.New("该用户不在此组织内"))
		return
	}
	if orgEmployee.Role == OrgEmployeeSuperAdmin {
		c.ResponseError(errors.New("超级管理员无法退出组织"))
		return
	}
	employees, err := o.db.queryDeptEmployeesWithOrgIdAndUid(orgId, loginUID)
	if err != nil {
		o.Error("查询加入的部门错误", zap.Error(err))
		c.ResponseError(errors.New("查询加入的部门错误"))
		return
	}
	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err = o.db.deleteOrgEmployeeTx(loginUID, orgId, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除组织成员错误", zap.Error(err))
		c.ResponseError(errors.New("删除组织成员错误"))
		return
	}
	err = o.db.deleteDeptEmployeesWithOrgidAndUidTx(orgId, loginUID, tx)
	if err != nil {
		tx.Rollback()
		o.Error("删除部门成员错误", zap.Error(err))
		c.ResponseError(errors.New("删除部门成员错误"))
		return
	}

	groupNos := make([]string, 0)
	groupNos = append(groupNos, orgId)

	if len(employees) > 0 {
		for _, employee := range employees {
			groupNos = append(groupNos, employee.DeptId)
		}
	}
	// 发布退出组织和相关部门群事件
	eventID, err := o.ctx.EventBegin(&wkevent.Data{
		Event: event.OrgEmployeeExit,
		Type:  wkevent.None,
		Data: &config.OrgEmployeeExitReq{
			Operator: loginUID,
			GroupNos: groupNos,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}

	if err = tx.Commit(); err != nil {
		o.Error("数据库事物提交失败", zap.Error(err))
		c.ResponseError(errors.New("数据库事物提交失败"))
		tx.Rollback()
		return
	}
	o.ctx.EventCommit(eventID)

	c.ResponseOK()
}

func (o *Organization) join(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	loginName := c.GetLoginName()
	orgId := c.Param("org_id")
	type reqVO struct {
		Code string `json:"code"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("请求数据格式有误！", err)
		return
	}
	if orgId == "" {
		c.ResponseError(errors.New("组织id不能为空"))
		return
	}
	if req.Code == "" {
		c.ResponseError(errors.New("邀请码不能为空"))
		return
	}
	if !strings.Contains(req.Code, "@") {
		c.ResponseError(errors.New("邀请码错误"))
		return
	}
	strs := strings.Split(req.Code, "@")
	inviteUid := ""
	inviteName := ""
	if len(strs) > 0 {
		inviteUid = strs[1]
	}
	if inviteUid == "" {
		c.ResponseError(errors.New("邀请码错误"))
		return
	}
	key := fmt.Sprintf("%s_%s", orgId, inviteUid)
	code, err := o.ctx.GetRedisConn().GetString(key)
	if err != nil {
		o.Error("获取缓存邀请码错误", zap.Error(err))
		c.ResponseError(errors.New("获取缓存邀请码错误"))
		return
	}
	if code != req.Code {
		c.ResponseError(errors.New("邀请码不存在或已失效"))
		return
	}
	userInfo, err := o.userService.GetUser(inviteUid)
	if err != nil {
		o.Error("查询邀请人信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询邀请人信息错误"))
		return
	}
	if userInfo == nil {
		c.ResponseError(errors.New("邀请人不存在或已注销"))
		return
	}
	inviteName = userInfo.Name
	orgEmployee, err := o.db.queryOrgEmployee(inviteUid, orgId)
	if err != nil {
		o.Error("查询组织成员错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织成员错误"))
		return
	}
	if orgEmployee == nil || orgEmployee.Role == OrgEmployeeNormal {
		c.ResponseError(errors.New("邀请码已失效"))
		return
	}
	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	err = o.db.insertOrgEmployeeTx(&orgEmployeeModel{
		EmployeeUid:  loginUID,
		EmployeeName: loginName,
		OrgId:        orgId,
		Role:         OrgEmployeeNormal,
		Status:       1,
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("新增组织成员错误", zap.Error(err))
		c.ResponseError(errors.New("新增组织成员错误"))
		return
	}
	err = o.db.insertDeptEmployeeTx(&deptEmployeeModel{
		EmployeeUid:   loginUID,
		OrgId:         orgId,
		DeptId:        orgId,
		WorkforceType: WorkforceTypes[0],
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("新增部门成员错误", zap.Error(err))
		c.ResponseError(errors.New("新增部门成员错误"))
		return
	}

	list := make([]*config.OrgOrDeptEmployeeVO, 0)
	list = append(list, &config.OrgOrDeptEmployeeVO{
		Operator:     inviteUid,
		OperatorName: inviteName,
		EmployeeUid:  loginUID,
		GroupNo:      orgId,
		EmployeeName: loginName,
		Action:       "add",
	})
	// 发布加入组织事件
	eventID, err := o.ctx.EventBegin(&wkevent.Data{
		Event: event.OrgOrDeptEmployeeUpdate,
		Type:  wkevent.None,
		Data: &config.MsgOrgOrDeptEmployeeUpdateReq{
			Members: list,
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}
	if tx.Commit(); err != nil {
		tx.Rollback()
		o.Error("数据库事物提交失败", zap.Error(err))
		c.ResponseError(errors.New("数据库事物提交失败"))
		return
	}
	o.ctx.EventCommit(eventID)
	c.ResponseOK()
}

func (o *Organization) getInviteCode(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	orgId := c.Param("org_id")
	code := util.GenerUUID()
	vercode := fmt.Sprintf("%s@%s", code, loginUID)
	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织信息错误"))
		return
	}
	if org == nil {
		c.ResponseError(errors.New("该组织不存在或被解散"))
		return
	}
	err = o.checkRole(loginUID, orgId)
	if err != nil {
		c.ResponseError(err)
		return
	}
	key := fmt.Sprintf("%s_%s", orgId, loginUID)
	err = o.ctx.GetRedisConn().SetAndExpire(key, vercode, time.Hour*24*7)
	if err != nil {
		o.Error("缓存邀请认证码错误", zap.Error(err))
		c.ResponseError(errors.New("缓存邀请认证码错误"))
		return
	}
	c.Response(map[string]interface{}{
		"code": vercode,
	})
}

func (o *Organization) checkRole(uid, orgId string) error {
	employee, err := o.db.queryOrgEmployee(uid, orgId)
	if err != nil {
		o.Error("查询组织成员错误", zap.Error(err))
		return errors.New("查询组织成员错误")
	}
	if employee == nil || employee.Role == OrgEmployeeNormal {
		return errors.New("用户无权执行此操作")
	}
	return nil
}

func (o *Organization) createDept(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	loginName := c.GetLoginName()
	orgId := c.Param("org_id")
	type reqVO struct {
		Name              string `json:"name"`
		ParentId          string `json:"parent_id"`
		ShortNo           string `json:"short_no"`
		IsCreateDeptGroup int    `json:"is_create_dept_group"` // 是否创建部门群
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("请求数据格式有误！", err)
		return
	}
	if orgId == "" {
		c.ResponseError(errors.New("组织ID不能为空"))
		return
	}
	if req.Name == "" {
		c.ResponseError(errors.New("部门名字不能为空"))
		return
	}
	if req.ParentId == "" {
		c.ResponseError(errors.New("上级部门不能为空"))
		return
	}
	org, err := o.db.queryOrg(orgId)
	if err != nil {
		o.Error("查询组织信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询组织信息错误"))
		return
	}
	if org == nil {
		c.ResponseError(errors.New("该组织不存在或被解散"))
		return
	}
	err = o.checkRole(loginUID, orgId)
	if err != nil {
		c.ResponseError(err)
		return
	}
	if req.ShortNo == "" {
		req.ShortNo = util.Ten2Hex(time.Now().UnixNano())
	} else {
		dept, err := o.db.queryDeptWithShortNo(req.ShortNo)
		if err != nil {
			o.Error("查询部门信息错误", zap.Error(err))
			c.ResponseError(errors.New("查询部门信息错误"))
			return
		}
		if dept != nil {
			c.ResponseError(errors.New("该部门编号已经存在"))
			return
		}
	}
	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	deptId := fmt.Sprintf("dept_%s", util.GenerUUID())
	if req.IsCreateDeptGroup == 1 {
		err = o.db.insertDeptTx(&deptModel{
			Name:           req.Name,
			DeptId:         deptId,
			ParentId:       req.ParentId,
			OrgId:          orgId,
			IsCreatedGroup: 1,
			ShortNo:        req.ShortNo,
		}, tx)
		if err != nil {
			tx.Rollback()
			o.Error("创建部门错误", zap.Error(err))
			c.ResponseError(errors.New("创建部门错误"))
			return
		}
		err = o.db.insertDeptEmployeeTx(&deptEmployeeModel{
			OrgId:         orgId,
			DeptId:        deptId,
			EmployeeUid:   loginUID,
			WorkforceType: WorkforceTypes[0],
		}, tx)
		if err != nil {
			tx.Rollback()
			o.Error("新增部门成员错误", zap.Error(err))
			c.ResponseError(errors.New("新增部门成员错误"))
			return
		}

		// 发布创建部门事件
		eventID, err := o.ctx.EventBegin(&wkevent.Data{
			Event: event.OrgOrDeptCreate,
			Type:  wkevent.None,
			Data: &config.MsgOrgOrDeptCreateReq{
				GroupNo:       deptId,
				Name:          req.Name,
				Operator:      loginUID,
				OperatorName:  loginName,
				GroupCategory: "department",
			},
		}, tx)
		if err != nil {
			tx.Rollback()
			o.Error("开启事件失败！", zap.Error(err))
			c.ResponseError(errors.New("开启事件失败！"))
			return
		}
		if err = tx.Commit(); err != nil {
			tx.Rollback()
			o.Error("提交事件失败！", zap.Error(err))
			c.ResponseError(errors.New("提交事件失败！"))
			return
		}
		o.ctx.EventCommit(eventID)
	} else {
		err = o.db.insertDept(&deptModel{
			Name:           req.Name,
			DeptId:         deptId,
			ParentId:       req.ParentId,
			OrgId:          orgId,
			ShortNo:        req.ShortNo,
			IsCreatedGroup: 0,
		})
		if err != nil {
			o.Error("创建部门错误", zap.Error(err))
			c.ResponseError(errors.New("创建部门错误"))
			return
		}
	}

	c.ResponseOK()
}

func (o *Organization) createOrg(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	loginName := c.GetLoginName()
	type reqVO struct {
		Name string `json:"name"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("请求数据格式有误！", err)
		return
	}
	if req.Name == "" {
		c.ResponseError(errors.New("名字不能为空"))
		return
	}
	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	shortNo := fmt.Sprintf("T%s", util.Ten2Hex(time.Now().UnixNano()))
	orgId := fmt.Sprintf("org_%s", util.GenerUUID())
	err := o.db.insertOrgTx(&orgModel{
		Name:    req.Name,
		Creator: loginUID,
		OrgId:   orgId,
		ShortNo: shortNo,
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("创建企业错误", zap.Error(err))
		c.ResponseError(errors.New("创建企业错误"))
		return
	}
	err = o.db.insertOrgEmployeeTx(&orgEmployeeModel{
		EmployeeUid:  loginUID,
		EmployeeName: loginName,
		Role:         OrgEmployeeSuperAdmin,
		OrgId:        orgId,
		Status:       1,
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("添加组织成员错误", zap.Error(err))
		c.ResponseError(errors.New("添加组织成员错误"))
		return
	}
	err = o.db.insertDeptEmployeeTx(&deptEmployeeModel{
		EmployeeUid:   loginUID,
		OrgId:         orgId,
		WorkforceType: WorkforceTypes[0],
		DeptId:        orgId,
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("添加部门成员错误", zap.Error(err))
		c.ResponseError(errors.New("添加部门成员错误"))
		return
	}

	// 发布创建组织事件
	eventID, err := o.ctx.EventBegin(&wkevent.Data{
		Event: event.OrgOrDeptCreate,
		Type:  wkevent.None,
		Data: &config.MsgOrgOrDeptCreateReq{
			GroupNo:       orgId,
			Operator:      loginUID,
			OperatorName:  loginName,
			Name:          req.Name,
			GroupCategory: "organization",
		},
	}, tx)
	if err != nil {
		tx.Rollback()
		o.Error("开启事件失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事件失败！"))
		return
	}
	if err := tx.Commit(); err != nil {
		tx.RollbackUnlessCommitted()
		o.Error("提交事务失败！", zap.Error(err))
		c.ResponseError(errors.New("提交事务失败！"))
		return
	}
	o.ctx.EventCommit(eventID)
	c.ResponseOK()
}

type orgResp struct {
	Name         string `json:"name"`           // 组织名称
	OrgId        string `json:"org_id"`         // 组织ID
	ShortNo      string `json:"short_no"`       // 短号
	Creator      string `json:"creator"`        // 创建者
	IsUploadLogo int    `json:"is_upload_logo"` // 是否已上传组织头像 1.是
}

type deptResp struct {
	Name           string `json:"name"`             // 部门名
	OrgId          string `json:"org_id"`           // 所属组织
	DeptId         string `json:"dept_id"`          // 部门ID
	ParentId       string `json:"parent_id"`        // 父部门ID
	ShortNo        string `json:"short_no"`         // 短号
	IsCreatedGroup int    `json:"is_created_group"` // 是否创建部门群 1.是
}

type deptDetailResp struct {
	deptResp
	ChildDepts []*deptResp         `json:"child_depts"` // 子部门
	Employees  []*deptEmployeeResp `json:"employees"`   // 员工
}

type deptEmployeeResp struct {
	OrgId         string `json:"org_id"`         // 组织ID
	DeptId        string `json:"dept_id"`        // 部门ID
	EmployeeId    string `json:"employee_id"`    // 工号
	WorkforceType string `json:"workforce_type"` // 人员类型 ‘正式’|‘编外’|‘顾问’
	JobTitle      string `json:"job_title"`      // 职务
	EmployeeUid   string `json:"employee_uid"`   // 员工uid
	EmployeeName  string `json:"employee_name"`  // 员工名称
	Username      string `json:"username"`       // 平台名称
	Role          int    `json:"role"`           // 组织内角色 1.管理员 2.子管理员 0.普通员工
}
type searchResp struct {
	EmployeeUid  string `json:"employee_uid"`  // 员工uid
	EmployeeName string `json:"employee_name"` // 员工名称
	Username     string `json:"username"`      // 平台名称
	Role         int    `json:"role"`          // 组织内角色 1.管理员 2.子管理员 0.普通员工
}
