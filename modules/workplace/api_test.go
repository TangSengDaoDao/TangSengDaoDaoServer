package workplace

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

var uid = "10000"
var token = "token122323"

func TestBannerList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = wm.db.insertBanner(&bannerModel{
		BannerNo:    "123b",
		Cover:       "cover1",
		Title:       "悟空IM官网",
		Description: "悟空IM让信息传递更简单",
		JumpType:    0,
		Route:       "http://www.githubim.com",
	})
	assert.NoError(t, err)
	err = wm.db.insertBanner(&bannerModel{
		BannerNo:    "123a",
		Cover:       "cover2",
		Title:       "唐僧叨叨官网",
		Description: "唐僧叨叨让企业轻松拥有自己的即时通讯",
		JumpType:    0,
		Route:       "https://tangsengdaodao.com",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/workplace/banner", nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, true, strings.Contains(w.Body.String(), `"cover":"cover1"`))

}
func TestUserAddApp(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	//wp := New(ctx)
	wm := NewManager(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	var appId = "wukongIM"
	err = wm.db.insertAPP(&appModel{
		AppID:       appId,
		Icon:        "xxxxx",
		Name:        "悟空IM",
		Description: "悟空IM让信息传递更简单",
		JumpType:    0,
		AppRoute:    "http://www.githubim.com",
		WebRoute:    "http://www.githubim.com",
		Status:      1,
		IsPaidApp:   0,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/workplace/user/app", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"app_id": appId,
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserGetApp(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wp := New(ctx)
	wm := NewManager(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	var appId1 = "wukongIM"
	var appId2 = "tsdd"
	err = wm.db.insertAPP(&appModel{
		AppID:       appId1,
		Icon:        "xxxxx",
		Name:        "悟空IM",
		Description: "悟空IM让信息传递更简单",
		JumpType:    0,
		AppRoute:    "http://www.githubim.com",
		WebRoute:    "http://www.githubim.com",
		Status:      1,
		IsPaidApp:   0,
	})
	assert.NoError(t, err)
	err = wm.db.insertAPP(&appModel{
		AppID:       appId2,
		Icon:        "xxxxx",
		Name:        "唐僧叨叨",
		Description: "唐僧叨叨让企业轻松拥有自己的即时通讯",
		JumpType:    0,
		AppRoute:    "http://www.githubim.com",
		WebRoute:    "http://www.githubim.com",
		Status:      1,
		IsPaidApp:   0,
	})
	assert.NoError(t, err)
	err = wp.db.insertUserApp(&userAppModel{
		Uid:     uid,
		SortNum: 1,
		AppID:   appId1,
	})
	assert.NoError(t, err)
	err = wp.db.insertUserApp(&userAppModel{
		Uid:     uid,
		SortNum: 2,
		AppID:   appId2,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/workplace/user/app", nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"悟空IM"`))
}

func TestDeleteUserApp(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wp := New(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	var appId1 = "wukongIM"
	err = wp.db.insertUserApp(&userAppModel{
		Uid:     uid,
		SortNum: 1,
		AppID:   appId1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/workplace/user/app?app_id=%s", appId1), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReorderUserApp(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wp := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	var appId1 = "wukongIM"
	var appId2 = "tsdd"
	err = wp.db.insertUserApp(&userAppModel{
		Uid:     uid,
		SortNum: 1,
		AppID:   appId1,
	})
	assert.NoError(t, err)
	err = wp.db.insertUserApp(&userAppModel{
		Uid:     uid,
		SortNum: 2,
		AppID:   appId2,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/v1/workplace/user/app/reorder", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"app_ids": []string{appId1, appId2},
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCategory(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = wm.db.insertCategory(&categoryModel{
		CategoryNo: "no1",
		Name:       "组织架构",
		SortNum:    1,
	})
	assert.NoError(t, err)
	err = wm.db.insertCategory(&categoryModel{
		CategoryNo: "no2",
		Name:       "审批流程",
		SortNum:    2,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/workplace/category", nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"审批流程"`))
}

func TestGetAppWithCategory(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	wp := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	categoryNo := "no"
	err = wm.db.insertCategory(&categoryModel{
		CategoryNo: categoryNo,
		Name:       "组织架构",
		SortNum:    1,
	})
	assert.NoError(t, err)
	err = wm.db.insertCategoryApp(&categoryAppModel{
		CategoryNo: categoryNo,
		AppId:      "wkim",
		SortNum:    1,
	})
	assert.NoError(t, err)
	err = wm.db.insertCategoryApp(&categoryAppModel{
		CategoryNo: categoryNo,
		AppId:      "tsdd",
		SortNum:    10,
	})
	assert.NoError(t, err)
	err = wp.db.insertUserApp(&userAppModel{
		AppID:   "wkim",
		SortNum: 1,
		Uid:     uid,
	})
	assert.NoError(t, err)
	err = wm.db.insertAPP(&appModel{
		AppID:       "wkim",
		Icon:        "xxxxx",
		Name:        "悟空IM",
		Description: "悟空IM让信息传递更简单",
		JumpType:    0,
		AppRoute:    "http://www.githubim.com",
		WebRoute:    "http://www.githubim.com",
		Status:      1,
		IsPaidApp:   0,
	})
	assert.NoError(t, err)
	err = wm.db.insertAPP(&appModel{
		AppID:       "tsdd",
		Icon:        "xxxxx",
		Name:        "唐僧叨叨",
		Description: "唐僧叨叨让企业轻松拥有自己的即时通讯",
		JumpType:    0,
		AppRoute:    "http://www.githubim.com",
		WebRoute:    "http://www.githubim.com",
		Status:      1,
		IsPaidApp:   0,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/workplace/category/app?category_no=%s", categoryNo), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"app_id":"wkim"`))
}

func TestAddRecord(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	appID := "tsdd"
	err = wm.db.insertAPP(&appModel{
		AppID:  appID,
		Name:   "唐僧叨叨",
		Icon:   "xxx",
		Status: 1,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/workplace/user/app/record", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"app_id": appID,
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetRecord(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wp := New(ctx)
	wm := NewManager(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	appID1 := "wkim"
	appID2 := "tsdd"
	err = wm.db.insertAPP(&appModel{
		AppID:  appID2,
		Name:   "唐僧叨叨",
		Icon:   "xxx",
		Status: 1,
	})
	assert.NoError(t, err)
	err = wm.db.insertAPP(&appModel{
		AppID:  appID1,
		Name:   "悟空IM",
		Icon:   "xxx",
		Status: 1,
	})
	assert.NoError(t, err)
	err = wp.db.insertRecord(&recordModel{
		AppId: appID1,
		Uid:   uid,
		Count: 12,
	})
	assert.NoError(t, err)
	err = wp.db.insertRecord(&recordModel{
		AppId: appID2,
		Uid:   uid,
		Count: 21,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/workplace/user/app/record", nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"唐僧叨叨"`))
}

func TestDeleteRecord(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wp := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	appId := "123"
	err = wp.db.insertRecord(&recordModel{
		AppId: appId,
		Uid:   uid,
		Count: 21,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/workplace/user/app/record?app_id=%s", appId), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
