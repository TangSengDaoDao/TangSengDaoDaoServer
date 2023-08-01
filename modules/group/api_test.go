package group

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGroupCreate(t *testing.T) {

	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	err := f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/group/create", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name":    "群组1",
		"members": []string{"10009", "10010"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"群组1"`)
	time.Sleep(time.Millisecond * 200)
}

func TestGroupGet(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo:            "1",
		Name:               "test",
		Creator:            testutil.UID,
		Version:            1,
		Status:             1,
		ForbiddenAddFriend: 1,
	})
	assert.NoError(t, err)
	err = f.settingDB.InsertSetting(&Setting{
		GroupNo:         "1",
		UID:             "10000",
		Mute:            1,
		Save:            1,
		ShowNick:        1,
		Top:             1,
		ChatPwdOn:       1,
		JoinGroupRemind: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/groups/1", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"group_no":"1"`, `"name":"test"`, `"chat_pwd":1`, `"mute":1`, `"top":1`, `"show_nick":1`, `"save":1`)

	time.Sleep(time.Millisecond * 200)
}

func TestGroupMemberAdd(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/groups/1/members", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"members": []string{"10009", "10010"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

}

func TestGroupMemberRemove(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/groups/1/members", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"members": []string{"10009", "10010"},
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

}

func TestSyncMembers(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.userDB.Insert(&user.Model{
		UID:  "10009",
		Name: "张九",
	})
	assert.NoError(t, err)
	err = f.userDB.Insert(&user.Model{
		UID:  "10010",
		Name: "李十",
	})
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Status:  1,
	})
	assert.NoError(t, err)

	err = f.db.InsertMember(&MemberModel{
		GroupNo: "1",
		UID:     "10009",
		Version: 2,
	})
	assert.NoError(t, err)
	err = f.db.InsertMember(&MemberModel{
		GroupNo: "1",
		UID:     "10010",
		Version: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/groups/1/membersync?version=1", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	b := w.Body.String()
	assert.Contains(t, b, `"uid":"10009"`)
	assert.NotContains(t, b, `"uid":"10010"`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGroupSettingUpdate(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Version: 1,
		Status:  1,
	})
	assert.NoError(t, err)
	err = f.db.InsertMember(&MemberModel{
		UID:     testutil.UID,
		GroupNo: "1",
		Role:    1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/groups/1/setting", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"mute":      1,
		"top":       1,
		"save":      1,
		"show_nick": 1,
		"chat_pwd":  1,
		"forbidden": 1,
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGroupUpdate(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Version: 1,
		Status:  1,
	})
	assert.NoError(t, err)
	err = f.db.InsertMember(&MemberModel{
		GroupNo: "1",
		UID:     testutil.UID,
		Role:    MemberRoleCreator,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/groups/1", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name": "test2",
	}))))
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

}

func TestList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	// 先清空旧数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = f.db.Insert(&Model{
		GroupNo: "1",
		Name:    "test",
		Creator: testutil.UID,
		Version: 1,
		Status:  1,
	})
	assert.NoError(t, err)
	err = f.settingDB.InsertSetting(&Setting{
		UID:     testutil.UID,
		GroupNo: "1",
		Save:    1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/group/my", nil)
	req.Header.Set("token", testutil.Token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"group_no":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":`))

}
