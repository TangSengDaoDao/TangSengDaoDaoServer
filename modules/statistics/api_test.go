package statistics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCountNum(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := NewStatistics(ctx)
	u.Route(s.GetRoute())
	err := u.userService.AddUser(&user.AddUserReq{
		Name: "ss",
		UID:  "sss",
	})
	assert.NoError(t, err)
	err = u.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: "xxx",
		Name:    "sxx",
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/statistics/countnum?date=2021-03-02", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"user_total_count":1`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"register_count":"1"`))
}

func TestRegisterUserListWithDateSpace(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := NewStatistics(ctx)
	u.Route(s.GetRoute())
	err := u.userService.AddUser(&user.AddUserReq{
		Name: "ss",
		UID:  "sss",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/statistics/registeruser/2021-03-03/2021-03-03", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)

}

func TestGroupWithDateSpace(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := NewStatistics(ctx)
	u.Route(s.GetRoute())
	err := u.groupService.AddGroup(&group.AddGroupReq{
		GroupNo: "xxx",
		Name:    "sxx",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/statistics/createdgroup/2021-03-03/2021-03-03", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	panic(w.Body)
}
