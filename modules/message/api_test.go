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
	println("执行完毕")
	assert.Equal(t, http.StatusOK, w.Code)
	time.Sleep(time.Second * 30)
}
