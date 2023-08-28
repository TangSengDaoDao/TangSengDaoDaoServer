package user

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

var token = "token122323"

func TestUser_Register(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	// u := New(ctx)
	// u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/register", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"code":     "123456",
		"zone":     "0086",
		"phone":    "13600000002",
		"password": "1234567",
		"device": map[string]interface{}{
			"device_id":    "device_id1",
			"device_name":  "device_name1",
			"device_model": "device_model1",
		},
	}))))

	s.GetRoute().ServeHTTP(w, req)
	fmt.Println(w.Body.String())
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"token":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"username":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"sex"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"category"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"short_no":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"zone":"0086"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"phone":"13600000002"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"setting":{"search_by_phone":1,"search_by_short":1,"new_msg_notice":1,"msg_show_detail":1,"voice_on":1,"shock_on":1}`))
}
func TestUser_Login(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)

	err := u.db.Insert(&Model{
		UID:           testutil.UID,
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		DeviceLock:    0,
		Status:        1,
		Zone:          "0086",
		Phone:         "13600000001",
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/login", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"username": "admin",
		"password": "123456",
		"device": map[string]interface{}{
			"device_id":    "device_id3",
			"device_name":  "device_name1",
			"device_model": "device_model1",
		},
	}))))
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, true, strings.Contains(w.Body.String(), `"token":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), fmt.Sprintf(`"uid":"%s"`, testutil.UID)))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"username":"admin"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"admin"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"sex":1`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"category":"客服"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"short_no":"uid_xxx1"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"zone":"0086"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"phone":"13600000001"`))

	time.Sleep(2 * time.Second)
}

func TestUser_Search(t *testing.T) {
	s, ctx := testutil.NewTestServer()

	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = u.db.Insert(&Model{
		UID:           "1234",
		Zone:          "0086",
		Phone:         "13600000001",
		Username:      "008613600000001",
		Password:      util.MD5(util.MD5("123456")),
		Name:          "tt",
		ShortNo:       "wukongchat_001",
		SearchByPhone: 1,
		SearchByShort: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/user/search?keyword=wukongchat_001", nil)
	s.GetRoute().ServeHTTP(w, req)

	fmt.Println(w.Body.String())

	assert.Equal(t, true, strings.Contains(w.Body.String(), `"exist":1`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":"1234"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"tt"`))

}

func TestUserGet(t *testing.T) {
	s, ctx := testutil.NewTestServer()

	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = u.db.Insert(&Model{
		UID:      "1234",
		Username: "admin",
		Password: util.MD5(util.MD5("123456")),
		Name:     "tt",
		Category: "客服",
		Sex:      1,
		ShortNo:  "test11",
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/users/1234", nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)

	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":"1234"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"tt"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"category":"客服"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"sex":1`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"short_no":"test11"`))

}

func TestUserUpdateInfo(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = u.db.Insert(&Model{
		UID:           testutil.UID,
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13600000001",
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/v1/user/current", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"name": "张丹丹",
	}))))
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserSetting(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = u.db.Insert(&Model{
		UID:           testutil.UID,
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13600000001",
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/v1/user/my/setting", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"device_lock": 1,
	}))))
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestAddBlackList(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = u.db.Insert(&Model{
		UID:           "adminuid1",
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13600000001",
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/blacklist/adminuid1", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBlacklists(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	err = u.db.Insert(&Model{
		UID:           "adminuid2",
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13600000001",
	})
	assert.NoError(t, err)

	err = u.settingDB.InsertUserSettingModel(&SettingModel{
		UID:       testutil.UID,
		ToUID:     "adminuid2",
		Blacklist: 1,
	})
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/user/blacklists", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":"adminuid2"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"admin"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"usename":"admin"`))
}

func TestSetChatPwd(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = u.db.Insert(&Model{
		UID:           testutil.UID,
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13600000001",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/chatpwd", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"login_pwd": "123456",
		"chat_pwd":  "111111",
	}))))
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
func TestSendLoginCheckPhoneCode(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = u.db.Insert(&Model{
		UID:           testutil.UID,
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13781388696",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/sms/login_check_phone", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uid": testutil.UID,
	}))))
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginCheckPhone(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	w := httptest.NewRecorder()
	err := u.db.Insert(&Model{
		UID:           testutil.UID,
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "客服",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13781388696",
	})
	assert.NoError(t, err)
	req, _ := http.NewRequest("POST", "/v1/user/login/check_phone", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uid":  testutil.UID,
		"code": "3346",
	}))))
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"token":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"username":"admin"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"admin"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"sex":1`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"category":"客服"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"short_no":"uid_xxx1"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"zone":"0086"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"phone":"13781388696"`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"setting":{"search_by_phone":1,"search_by_short":1,"new_msg_notice":1,"msg_show_detail":1,"voice_on":1,"shock_on":1}`))
}
func TestCustomerservices(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	err := u.db.Insert(&Model{
		UID:           testutil.UID,
		Name:          "admin",
		Username:      "admin",
		Sex:           1,
		Password:      util.MD5(util.MD5("123456")),
		Category:      "service",
		ShortNo:       "uid_xxx1",
		SearchByPhone: 1,
		SearchByShort: 1,
		NewMsgNotice:  1,
		MsgShowDetail: 1,
		VoiceOn:       1,
		ShockOn:       1,
		Zone:          "0086",
		Phone:         "13781388696",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/user/customerservices", nil)
	req.Header.Set("token", testutil.Token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":`))
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"name":"admin"`))
}

func TestUploadAvatar(t *testing.T) {
	path := "../../../assets/assets/avatar.png"
	file, err := os.Open(path)
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", path)
	if err != nil {
		writer.Close()
		t.Error(err)
	}
	io.Copy(part, file)
	writer.Close()

	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	u.Route(s.GetRoute())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/users/%s/avatar", testutil.UID), body)
	req.Header.Set("token", testutil.Token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
