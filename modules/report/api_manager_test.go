package report

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestReportList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	m := NewManager(ctx)
	m.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = m.db.insertCategory(&categoryModel{
		CategoryNo:   "11",
		CategoryName: "其他信息",
	})
	assert.NoError(t, err)
	err = m.db.insert(&model{
		UID:         "uid1111",
		CategoryNo:  "11",
		ChannelID:   "channelID111",
		ChannelType: 2,
		Remark:      "不当言论",
		Imgs:        "xxxxx,xxxxx",
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("GET", "/v1/manager/reportlist?pageSize=12&pageIndex=1", nil)
	w := httptest.NewRecorder()
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	panic(w.Body)
}
