package common

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAddVersion(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	model := &appVersionReq{
		AppVersion:  "1.0",
		OS:          "android",
		DownloadURL: "http://www.githubim.com/download/test.apk",
		IsForce:     1,
		UpdateDesc:  "发布新版本",
	}
	req, _ := http.NewRequest("POST", "/v1/common/appversion", bytes.NewReader([]byte(util.ToJson(model))))
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetNewVersion(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	f := New(ctx)
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	_, err = f.db.insertAppVersion(&appVersionModel{
		AppVersion:  "1.0",
		OS:          "android",
		DownloadURL: "http://www.githubim.com",
		IsForce:     1,
		UpdateDesc:  "发布新版本",
	})
	assert.NoError(t, err)

	_, err = f.db.insertAppVersion(&appVersionModel{
		AppVersion:  "1.2",
		OS:          "android",
		DownloadURL: "http://www.githubim.com",
		IsForce:     1,
		UpdateDesc:  "发布新版本",
	})
	assert.NoError(t, err)

	f.Route(s.GetRoute())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/common/appversion/android/1.2", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"app_version":1.0`))
}
