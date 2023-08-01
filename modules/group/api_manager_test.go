package group

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGroupList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = m.db.Insert(&Model{
		GroupNo: "xxxx",
		Name:    "gxxx",
		Creator: "1111",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/manager/group/list?pageIndex=1&pageSize=10&keyword=gx", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"group_no":`))
}

func TestGroupCount(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = m.db.Insert(&Model{
		GroupNo: "111",
		Name:    "sss",
		Creator: "xxx",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/manager/group/count", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
}
func TestDisableList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = m.db.Insert(&Model{
		GroupNo: "111",
		Name:    "sss",
		Creator: "xxx",
		Status:  GroupStatusDisabled,
	})
	assert.NoError(t, err)
	err = m.userDB.Insert(&user.Model{
		UID:  "xxx",
		Name: "001",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/manager/group/disablelist?page_size=1&page_index=1", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
}

func TestBlackList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = m.db.Insert(&Model{
		GroupNo: "111",
		Name:    "sss",
		Creator: "xxx",
	})
	assert.NoError(t, err)
	err = m.userDB.Insert(&user.Model{
		UID:  "xxx",
		Name: "001",
	})
	assert.NoError(t, err)
	err = m.db.InsertMember(&MemberModel{
		UID:     "xxx",
		GroupNo: "111",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/manager/groups/111/members/blacklist?page_size=1&page_index=1", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
}
