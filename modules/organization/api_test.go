package organization

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

var uid = "10000"
var token = "token122323"

// 创建组织
func TestCreateOrg(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	// org := New(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/organization", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name": "唐僧叨叨网络科技有限公司",
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 获取组织
func TestGetOrg(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "111"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId,
		Name:         "唐僧叨叨",
		ShortNo:      "11",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/organizations/%s", orgId), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"唐僧叨叨"`))
}

// 已加入组织
func TestJoinedOrgs(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId1 := "111"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId1,
		Name:         "唐僧叨叨",
		ShortNo:      "11",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	orgId2 := "222"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId2,
		Name:         "悟空IM",
		ShortNo:      "22",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId1,
		EmployeeUid: uid,
		Role:        1,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId2,
		EmployeeUid: uid,
		Role:        1,
		Status:      1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/organization/joined", nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"唐僧叨叨"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"悟空IM"`))
}

// 获取邀请码
func TestGetInvitecode(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId1 := "111"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId1,
		Name:         "唐僧叨叨",
		ShortNo:      "11",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId1,
		EmployeeUid: uid,
		Role:        OrgEmployeeAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/organizations/%s/invitecode", orgId1), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
}

// 加入组织
func TestJoinOrg(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId1 := "111"
	adminUID := "admin111"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId1,
		Name:         "唐僧叨叨",
		ShortNo:      "11",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId1,
		EmployeeUid: adminUID,
		Role:        OrgEmployeeAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.userService.AddUser(&user.AddUserReq{
		UID:   adminUID,
		Name:  "111",
		Phone: "13000000000",
		Zone:  "0086",
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: orgId1,
		Name:    "唐僧叨叨",
	})
	assert.NoError(t, err)
	code := fmt.Sprintf("tsddno1@%s", adminUID)
	key := fmt.Sprintf("%s_%s", orgId1, adminUID)
	err = org.ctx.GetRedisConn().SetAndExpire(key, code, time.Hour*1)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/organizations/%s/join", orgId1), bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"code": code,
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 退出组织
func TestQuitOrg(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "111"
	adminUID := "admin111"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId,
		Name:         "唐僧叨叨",
		ShortNo:      "11",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: adminUID,
		Role:        OrgEmployeeSuperAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: uid,
		Role:        OrgEmployeeNormal,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: orgId,
		Name:    "唐僧叨叨",
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   orgId,
		MemberUID: uid,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   orgId,
		MemberUID: adminUID,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/organizations/%s/quit", orgId), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 批量删除员工
func TestDeleteEmployee(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "111"
	u1 := "u1"
	u2 := "u2"
	u3 := "u3"
	g1 := "g1"
	g2 := "g2"
	g3 := "g3"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId,
		Name:         "唐僧叨叨",
		ShortNo:      "11",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: uid,
		Role:        OrgEmployeeSuperAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: u1,
		Role:        OrgEmployeeNormal,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: u2,
		Role:        OrgEmployeeNormal,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: u3,
		Role:        OrgEmployeeNormal,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: orgId,
		Name:    "唐僧叨叨",
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: g1,
		Name:    "g1",
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: g2,
		Name:    "g2",
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: g3,
		Name:    "g3",
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   orgId,
		MemberUID: u1,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   orgId,
		MemberUID: u2,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   orgId,
		MemberUID: uid,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   g1,
		MemberUID: u1,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   g2,
		MemberUID: u2,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   g3,
		MemberUID: u3,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:          orgId,
		DeptId:         g1,
		IsCreatedGroup: 1,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:          orgId,
		DeptId:         g2,
		IsCreatedGroup: 1,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:          orgId,
		DeptId:         g3,
		IsCreatedGroup: 1,
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      g1,
		EmployeeUid: u1,
		EmployeeId:  "1",
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      g2,
		EmployeeUid: u2,
		EmployeeId:  "2",
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      g3,
		EmployeeUid: u3,
		EmployeeId:  "3",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/organizations/%s/employees", orgId), bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uids": []string{u1, u2, u3},
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 批量转移员工
func TestUpdateDeptEmployee(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "111"
	u1 := "u1"
	u2 := "u2"
	u3 := "u3"
	g1 := "g1"
	g2 := "g2"
	g3 := "g3"
	err = org.db.insertOrg(&orgModel{
		OrgId:        orgId,
		Name:         "唐僧叨叨",
		ShortNo:      "11",
		Creator:      uid,
		IsUploadLogo: 0,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: uid,
		Role:        OrgEmployeeSuperAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: u1,
		Role:        OrgEmployeeNormal,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: u2,
		Role:        OrgEmployeeNormal,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: u3,
		Role:        OrgEmployeeNormal,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: orgId,
		Name:    "唐僧叨叨",
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: g1,
		Name:    "g1",
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: g2,
		Name:    "g2",
	})
	assert.NoError(t, err)
	err = org.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: g3,
		Name:    "g3",
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   g1,
		MemberUID: u1,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   g2,
		MemberUID: u2,
	})
	assert.NoError(t, err)
	err = org.groupService.AddMember(&group.AddMemberReq{
		GroupNo:   g3,
		MemberUID: u3,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:          orgId,
		DeptId:         g1,
		IsCreatedGroup: 1,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:          orgId,
		DeptId:         g2,
		IsCreatedGroup: 1,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:          orgId,
		DeptId:         g3,
		IsCreatedGroup: 1,
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      g1,
		EmployeeUid: u1,
		EmployeeId:  "1",
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      g2,
		EmployeeUid: u2,
		EmployeeId:  "2",
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      g3,
		EmployeeUid: u3,
		EmployeeId:  "3",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/v1/organizations/%s/employees", orgId), bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uids":     []string{u1, u2, u3},
		"dept_ids": []string{g1},
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 创建部门
func TestCreateDeptament(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "org1"
	err = org.db.insertOrg(&orgModel{
		OrgId:   orgId,
		Name:    "唐僧叨叨",
		ShortNo: "s1",
		Creator: uid,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: uid,
		Role:        OrgEmployeeSuperAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/organizations/%s/department", orgId), bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name":                 "技术部",
		"short_no":             "",
		"parent_id":            orgId,
		"is_create_dept_group": 1,
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 修改部门
func TestUpdateDept(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "org1"
	err = org.db.insertOrg(&orgModel{
		OrgId:   orgId,
		Name:    "唐僧叨叨",
		ShortNo: "s1",
		Creator: uid,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: uid,
		Role:        OrgEmployeeSuperAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	deptId := "1"
	err = org.db.insertDept(&deptModel{
		DeptId:   deptId,
		OrgId:    orgId,
		Name:     "技术部",
		ParentId: "p12",
		ShortNo:  "1",
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		DeptId:   "dkdk",
		OrgId:    orgId,
		Name:     "22",
		ParentId: "p12",
		ShortNo:  "2",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/v1/organizations/%s/department/%s", orgId, deptId), bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name":      "技术部1",
		"short_no":  "23",
		"parent_id": orgId,
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteDept(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "org1"
	err = org.db.insertOrg(&orgModel{
		OrgId:   orgId,
		Name:    "唐僧叨叨",
		ShortNo: "s1",
		Creator: uid,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: uid,
		Role:        OrgEmployeeSuperAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	deptId := "1"
	err = org.db.insertDept(&deptModel{
		DeptId:   deptId,
		OrgId:    orgId,
		Name:     "技术部",
		ParentId: "p12",
		ShortNo:  "1",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/organizations/%s/department/%s", orgId, deptId), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// 部门详情
func TestGetDeptDetail(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "org1"
	deptId := "11"
	err = org.db.insertOrg(&orgModel{
		OrgId:   orgId,
		Name:    "唐僧叨叨",
		ShortNo: "s1",
		Creator: uid,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:       orgId,
		EmployeeUid: uid,
		Role:        OrgEmployeeSuperAdmin,
		Status:      1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:        orgId,
		EmployeeUid:  "u1",
		EmployeeName: "唐僧叨叨管理员",
		Role:         OrgEmployeeAdmin,
		Status:       1,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:        orgId,
		EmployeeUid:  "u2",
		EmployeeName: "唐僧叨叨员工",
		Role:         OrgEmployeeNormal,
		Status:       1,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:   orgId,
		DeptId:  deptId,
		Name:    "技术部",
		ShortNo: "1",
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:    orgId,
		DeptId:   "dd",
		Name:     "技术部11",
		ShortNo:  "12",
		ParentId: deptId,
	})
	assert.NoError(t, err)
	err = org.db.insertDept(&deptModel{
		OrgId:    orgId,
		DeptId:   "dd1",
		Name:     "技术部112",
		ShortNo:  "123",
		ParentId: deptId,
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      deptId,
		EmployeeUid: "u1",
	})
	assert.NoError(t, err)
	err = org.db.insertDeptEmployee(&deptEmployeeModel{
		OrgId:       orgId,
		DeptId:      deptId,
		EmployeeUid: "u2",
	})
	assert.NoError(t, err)

	err = org.userService.AddUser(&user.AddUserReq{
		UID:  "u2",
		Name: "我是普通用户",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/organizations/%s/department/%s", orgId, deptId), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"技术部"`))
}

// 搜索员工
func TestSearchEmployee(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	org := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	orgId := "org1"
	err = org.db.insertOrg(&orgModel{
		OrgId:   orgId,
		Name:    "唐僧叨叨",
		ShortNo: "s1",
		Creator: uid,
	})
	assert.NoError(t, err)
	err = org.db.insertOrgEmployee(&orgEmployeeModel{
		OrgId:        orgId,
		EmployeeUid:  uid,
		EmployeeName: "sdklsklsd",
		Role:         OrgEmployeeSuperAdmin,
		Status:       1,
	})
	assert.NoError(t, err)

	err = org.userService.AddUser(&user.AddUserReq{
		UID:  uid,
		Name: "sd",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/organizations/%s/employees/search?keyword=sd", orgId), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"sd"`))
}
