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
func TestUsernameRegister(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	username := "userone123123"
	password := "123123"
	u.db.Insert(&Model{
		UID:      "123",
		Username: username,
		Password: util.MD5(util.MD5(password)),
		Name:     username,
		ShortNo:  "123",
		Status:   1,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/usernameregister", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"username": "skldkdlskds",
		"password": password,
		"device": map[string]interface{}{
			"device_id":    "device_id3",
			"device_name":  "device_name1",
			"device_model": "device_model1",
		},
	}))))
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"status":110`))
}
func TestUsernameLogin(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	username := "userone123123"
	password := "123123"
	u.db.Insert(&Model{
		UID:           "123",
		Username:      username,
		Password:      util.MD5(util.MD5(password)),
		Name:          username,
		ShortNo:       "123",
		Status:        1,
		Web3PublicKey: "03af80b90d25145da28c583359beb47b21796b2fe1a23c1511e443e7a64dfdb27d",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/usernamelogin", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"username": username,
		"password": password,
		"device": map[string]interface{}{
			"device_id":    "device_id3",
			"device_name":  "device_name1",
			"device_model": "device_model1",
		},
	}))))
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"username":userone123123`))
}
func TestUploadWeb3PublicKey(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	username := "userone"
	password := "123123"
	uid := "123"
	u.db.Insert(&Model{
		UID:      uid,
		Username: username,
		Password: util.MD5(util.MD5(password)),
		Name:     username,
		ShortNo:  "123",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/web3publickey", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"uid":             uid,
		"web3_public_key": "03af80b90d25145da28c583359beb47b21796b2fe1a23c1511e443e7a64dfdb27d",
	}))))
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"uid":123`))
}
func TestGetVerifyText(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	uid := "123"
	err = u.db.Insert(&Model{
		UID:           uid,
		Username:      "123",
		ShortNo:       "123",
		Status:        1,
		Web3PublicKey: "03af80b90d25145da28c583359beb47b21796b2fe1a23c1511e443e7a64dfdb27d",
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/user/web3verifytext?uid=%s", uid), nil)
	s.GetRoute().ServeHTTP(w, req)
	panic(w.Body)
}
func TestUpdatePassword(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	username := "userone"
	password := "123123"
	u.db.Insert(&Model{
		UID:           testutil.UID,
		Username:      username,
		Password:      util.MD5(util.MD5(password)),
		Name:          username,
		ShortNo:       "123",
		Web3PublicKey: "03af80b90d25145da28c583359beb47b21796b2fe1a23c1511e443e7a64dfdb27d",
	})
	w := httptest.NewRecorder()

	req, _ := http.NewRequest("PUT", "/v1/user/updatepassword", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"new_password": "new_pwd_123",
		"password":     password,
	}))))
	req.Header.Set("token", testutil.Token)

	s.GetRoute().ServeHTTP(w, req)
	panic(w.Body)
	// assert.Equal(t, http.StatusOK, w.Code)
}
func TestResetPwd(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)
	username := "userone"
	password := "123123"
	u.db.Insert(&Model{
		UID:           "123",
		Username:      username,
		Password:      util.MD5(util.MD5(password)),
		Name:          username,
		ShortNo:       "123",
		Web3PublicKey: "03af80b90d25145da28c583359beb47b21796b2fe1a23c1511e443e7a64dfdb27d",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/user/pwdforget_web3", bytes.NewReader([]byte(util.ToJson(map[string]interface{}{
		"username":    username,
		"password":    "new_pwd_123",
		"verify_text": "hello123",
		"sign_text":   "44459fd9146290dcd913350bae6fe79fd48050d39b3c1c315e8f032af3b555d41af6f2c07d4d0f1d8d5dd041af8175e657ae981cf47e58aa5547ab08fc7066e401",
	}))))
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
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

func TestGetUserRedDot(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	//u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = u.db.insertUserRedDot(&userRedDotModel{
		UID:      testutil.UID,
		Count:    1,
		IsDot:    0,
		Category: UserRedDotCategoryFriendApply,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/v1/user/reddot/%s", UserRedDotCategoryFriendApply), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, true, strings.Contains(w.Body.String(), `"count":1`))
	// assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteUserRedDot(t *testing.T) {
	s, ctx := testutil.NewTestServer()
	u := New(ctx)
	//u.Route(s.GetRoute())
	//清除数据
	err := testutil.CleanAllTables(ctx)
	assert.NoError(t, err)

	err = u.db.insertUserRedDot(&userRedDotModel{
		UID:      testutil.UID,
		Count:    1,
		IsDot:    0,
		Category: UserRedDotCategoryFriendApply,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1/user/reddot/%s", UserRedDotCategoryFriendApply), nil)
	req.Header.Set("token", token)
	s.GetRoute().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
