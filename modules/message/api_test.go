package message

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/server"
	"github.com/stretchr/testify/assert"
)

var uid = "10000"

// var friendUID = "10001"
var token = "token122323"

func newTestServer() (*server.Server, *config.Context) {
	os.Remove("test.db")
	cfg := config.New()
	cfg.Test = true
	ctx := config.NewContext(cfg)
	ctx.Event = event.New(ctx)
	err := ctx.Cache().Set(cfg.Cache.TokenCachePrefix+token, uid+"@test")
	if err != nil {
		panic(err)
	}
	// 创建server
	s := server.New(ctx)
	return s, ctx

}

func TestMessageSync(t *testing.T) {
	s, ctx := newTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/message/sync", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uid":             uid,
		"max_message_seq": 100,
		"limit":           100,
	}))))
	req.Header.Set("token", token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(time.Millisecond * 200)

}

func TestMessageSyncack(t *testing.T) {
	s, ctx := newTestServer()
	f := New(ctx)
	f.Route(s.GetRoute())

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/message/syncack/111", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uid":              uid,
		"last_message_seq": 100,
	}))))
	req.Header.Set("token", token)
	assert.NoError(t, err)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(time.Millisecond * 200)

}
