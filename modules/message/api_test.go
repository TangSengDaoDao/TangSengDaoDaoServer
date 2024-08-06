package message

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	_ "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/webhook"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/module"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/server"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
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

// 消息已读
func TestReadedMessage(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	msg := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	messageIds := make([]string, 0)
	messageIds = append(messageIds, "1")
	messageIds = append(messageIds, "2")
	messageIds = append(messageIds, "3")
	channelID := "c1"
	channelType := common.ChannelTypePerson.Uint8()
	payload, err := json.Marshal(map[string]interface{}{
		"type":    1,
		"content": "1",
	})
	assert.NoError(t, err)
	err = msg.db.insertMessage(&messageModel{
		MessageID:   1,
		MessageSeq:  1,
		FromUID:     common.GetFakeChannelIDWith(channelID, uid),
		ChannelID:   channelID,
		ChannelType: channelType,
		IsDeleted:   0,
		Payload:     payload,
	})
	assert.NoError(t, err)
	err = msg.db.insertMessage(&messageModel{
		MessageID:   2,
		MessageSeq:  2,
		FromUID:     common.GetFakeChannelIDWith(channelID, uid),
		ChannelID:   channelID,
		ChannelType: channelType,
		IsDeleted:   0,
		Payload:     payload,
	})
	assert.NoError(t, err)
	err = msg.db.insertMessage(&messageModel{
		MessageID:   3,
		MessageSeq:  3,
		FromUID:     common.GetFakeChannelIDWith(channelID, uid),
		ChannelID:   channelID,
		ChannelType: channelType,
		IsDeleted:   0,
		Payload:     payload,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/message/readed", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"channel_id":   channelID,
		"channel_type": channelType,
		"message_ids":  messageIds,
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(time.Second * 30)
}

func TestPinMessage(t *testing.T) {
	s, _ := NewTestServer1()
	// msg := New(ctx)
	// err := testutil.CleanAllTables(ctx)
	// assert.NoError(t, err)
	channelID := "c1"
	channelType := common.ChannelTypePerson.Uint8()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/message/pinned", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"channel_id":   channelID,
		"channel_type": channelType,
		"message_id":   "1",
		"message_seq":  1,
	}))))
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestSyncPindMessage(t *testing.T) {
	s, ctx := NewTestServer1()
	msg := New(ctx)
	// err := testutil.CleanAllTables(ctx)
	// assert.NoError(t, err)
	channelID := "54a266c29cbc45d0a66883c8fc2974cd"
	channelType := common.ChannelTypePerson.Uint8()
	err := msg.pinnedDB.insert(&pinnedMessageModel{
		MessageId:   "1788882133855502336",
		ChannelID:   common.GetFakeChannelIDWith(channelID, UID),
		ChannelType: channelType,
		IsDeleted:   0,
		Version:     11,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/message/pinned/sync", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"channel_id":   channelID,
		"channel_type": channelType,
		"version":      0,
	}))))
	req.Header.Set("token", Token)
	s.GetRoute().ServeHTTP(w, req)
	panic(w.Body)
	// assert.Equal(t, http.StatusOK, w.Code)
}

// UID 测试用户ID
var UID = "beb714efd08a4530a5881ebd7f2fde38"

// Token 测试用户token
var Token = "token122323"

// NewTestServer 创建一个测试服务器
func NewTestServer1(args ...string) (*server.Server, *config.Context) {
	cfg := config.New()
	cfg.Test = true
	// cfg.TracingOn = true
	// cfg.TracerAddr = "49.235.106.135:6831"
	cfg.DB.MySQLAddr = "root:demo@tcp(127.0.0.1)/test?charset=utf8mb4&parseTime=true"
	cfg.DB.Migration = false
	ctx := config.NewContext(cfg)
	// ctx.Event = event.New(ctx)
	// 先清空旧数据
	// err := CleanAllTables(ctx)
	// if err != nil {
	// 	panic(err)
	// }

	// ctx.Event = event.New(ctx)
	err := ctx.Cache().Set(cfg.Cache.TokenCachePrefix+Token, UID+"@test")
	if err != nil {
		panic(err)
	}

	// _, err = ctx.DB().InsertBySql("insert into `app`(app_id,app_key,status) VALUES('wukongchat',substring(MD5(RAND()),1,20),1)").Exec()
	// if err != nil {
	// 	panic(err)
	// }

	// 创建server
	s := server.New(ctx)
	// ctx.Server = s
	s.GetRoute().UseGin(ctx.Tracer().GinMiddle())
	ctx.SetHttpRoute(s.GetRoute())
	err = module.Setup(ctx)
	if err != nil {
		panic(err)
	}

	return s, ctx

}
