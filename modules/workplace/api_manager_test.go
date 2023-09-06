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

func TestAddBanner(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	//m := NewManager(ctx)
	//m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	req, _ := http.NewRequest("POST", "/v1/manager/workplace/banner", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"cover":       "https://api.botgate.cn/v1/users/admin/avatar",
		"title":       "横幅title",
		"description": "横幅介绍",
		"jump_type":   0,
		"route":       "https://element-plus.gitee.io/zh-CN/",
	}))))
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestGetBanners(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	bannerNo := "no1"
	err = wm.db.insertBanner(&bannerModel{
		BannerNo:    bannerNo,
		Cover:       "cover_1122",
		Title:       "",
		Description: "ddd",
		JumpType:    1,
		Route:       "moment",
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("GET", "/v1/manager/workplace/banner", nil)
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, true, strings.Contains(w.Body.String(), `"cover":"cover_1122"`))
}
func TestUpdateBanner(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	bannerNo := "no1"
	err = wm.db.insertBanner(&bannerModel{
		BannerNo:    bannerNo,
		Cover:       "cover_1122",
		Title:       "",
		Description: "ddd",
		JumpType:    1,
		Route:       "moment",
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("PUT", "/v1/manager/workplace/banner", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"banner_no":   bannerNo,
		"cover":       "cover_1122u",
		"title":       "u",
		"description": "dddu",
		"jump_type":   0,
		"route":       "https://githubim.com",
	}))))
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteBanner(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	bannerNo := "no1"
	err = wm.db.insertBanner(&bannerModel{
		BannerNo:    bannerNo,
		Cover:       "cover_1122",
		Title:       "",
		Description: "ddd",
		JumpType:    1,
		Route:       "moment",
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/manager/workplace/banner?banner_no=%s", bannerNo), nil)
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAddCategory(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	req, _ := http.NewRequest("POST", "/v1/manager/workplace/category", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name": "组织架构",
	}))))
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetManagerCategory(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
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
		Name:       "日程安排",
		SortNum:    2,
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("GET", "/v1/manager/workplace/category", nil)
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"日程安排"`))
}

func TestSortCategory(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	no1 := "no1"
	no2 := "no2"
	err = wm.db.insertCategory(&categoryModel{
		CategoryNo: no1,
		Name:       "组织架构",
		SortNum:    1,
	})
	assert.NoError(t, err)
	err = wm.db.insertCategory(&categoryModel{
		CategoryNo: no2,
		Name:       "日程安排",
		SortNum:    2,
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("PUT", "/v1/manager/workplace/category/reorder", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"category_nos": []string{no1, no2},
	}))))
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAddAPP(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	no := "no"
	err = wm.db.insertCategory(&categoryModel{
		CategoryNo: no,
		Name:       "日程安排",
		SortNum:    2,
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("POST", "/v1/manager/workplace/app", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"category_no":  no,
		"icon":         "xxx",
		"name":         "组织架构",
		"description":  "平面化组织架构",
		"app_category": "bot",
		"jump_type":    0,
		"route":        "https://www.githubim.com",
		"is_paid_app":  0,
	}))))
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateAPP(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	appId := "wkim"
	categoryNo := "im"
	err = wm.db.insertAPP(&appModel{
		AppID:       appId,
		CategoryNo:  categoryNo,
		Icon:        "xxxxx",
		Name:        "悟空IM",
		Description: "悟空IM让信息传递更简单",
		JumpType:    0,
		Route:       "http://www.githubim.com",
		Status:      1,
		IsPaidApp:   0,
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("PUT", "/v1/manager/workplace/app", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"app_id":       appId,
		"category_no":  categoryNo,
		"icon":         "xxxxxu",
		"name":         "悟空IMu",
		"description":  "悟空IM让信息传递更简单u",
		"app_category": "bot",
		"jump_type":    0,
		"route":        "https://www.githubim.com",
		"is_paid_app":  0,
	}))))
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteAPP(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	wm := NewManager(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	appId := "wkim"
	categoryNo := "im"
	err = wm.db.insertAPP(&appModel{
		AppID:       appId,
		CategoryNo:  categoryNo,
		Icon:        "xxxxx",
		Name:        "悟空IM",
		Description: "悟空IM让信息传递更简单",
		JumpType:    0,
		Route:       "http://www.githubim.com",
		Status:      1,
		IsPaidApp:   0,
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/manager/workplace/app?app_id=%s&category_no=%s", appId, categoryNo), bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"app_id": appId,
	}))))
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
