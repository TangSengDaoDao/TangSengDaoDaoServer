package user

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/testutil"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/wkhttp"
	"github.com/stretchr/testify/assert"
)

func TestAddUser(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()

	req, _ := http.NewRequest("POST", "/v1/manager/adduser", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name":     "张三",
		"zone":     "0086",
		"phone":    "13600000002",
		"password": "1234567",
	}))))
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogin(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	err = m.userDB.Insert(&Model{
		UID:      "xxx",
		Username: "superAdmin",
		Name:     "超级管理员",
		Password: util.MD5(util.MD5("admiN123456")),
		Role:     string(wkhttp.SuperAdmin),
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("POST", "/v1/manager/login", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"username": "superAdmin",
		"password": "admiN123456",
	}))))
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":"xxx"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"超级管理员"`))
}

func TestBlacklist(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = m.userDB.Insert(&Model{
		UID:      "xxx",
		Username: "111",
		Name:     "111",
		Password: util.MD5(util.MD5("111")),
	})
	assert.NoError(t, err)
	err = m.userDB.Insert(&Model{
		UID:      "sss",
		Username: "222",
		Name:     "222",
		Password: util.MD5(util.MD5("222")),
	})
	assert.NoError(t, err)
	m.userSettingDB.InsertUserSettingModel(&SettingModel{
		UID:       "xxx",
		ToUID:     "sss",
		Blacklist: 1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/manager/blacklist?uid=xxx", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
}

func TestUpdatePwd(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = m.userDB.Insert(&Model{
		UID:      testutil.UID,
		Username: "111",
		Name:     "111",
		Password: util.MD5(util.MD5("111")),
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/manager/user/updatepassword", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"new_password": "333333",
		"password":     "111",
	}))))
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
