package webhook

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

type harmonyOSTestTransport func(*http.Request) (*http.Response, error)

func (f harmonyOSTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newTestHarmonyOSPush(t *testing.T) *HarmonyOSPush {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	data, err := json.Marshal(harmonyOSServiceAccount{
		ProjectID: "test-project", KeyID: "test-key", SubAccount: "test-account",
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
	})
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "service-account.json")
	require.NoError(t, os.WriteFile(path, data, 0600))
	pusher, err := NewHarmonyOSPush(config.HarmonyOSPush{
		BundleID: "com.example.app", ServiceAccountFile: path, Category: "IM", TestMessage: true,
	})
	require.NoError(t, err)
	return pusher
}

func TestHarmonyOSRequest(t *testing.T) {
	pusher := newTestHarmonyOSPush(t)
	for _, badge := range []int{-1, 0, 12, 99, 120} {
		pusher.client.Transport = harmonyOSTestTransport(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "https://push-api.cloud.huawei.com/v3/test-project/messages:send", r.URL.String())
			require.Equal(t, "0", r.Header.Get("push-type"))
			require.Equal(t, "application/json; charset=UTF-8", r.Header.Get("Content-Type"))
			auth := r.Header.Get("Authorization")
			require.True(t, strings.HasPrefix(auth, "Bearer "))
			token, err := jwt.Parse(strings.TrimPrefix(auth, "Bearer "), func(token *jwt.Token) (interface{}, error) {
				require.Equal(t, "PS256", token.Method.Alg())
				require.Equal(t, "test-key", token.Header["kid"])
				require.Equal(t, "JWT", token.Header["typ"])
				return &pusher.privateKey.PublicKey, nil
			}, jwt.WithValidMethods([]string{"PS256"}))
			require.NoError(t, err)
			require.True(t, token.Valid)
			claims := token.Claims.(jwt.MapClaims)
			require.Equal(t, "test-account", claims["iss"])
			require.Equal(t, harmonyOSTokenAudience, claims["aud"])
			require.Equal(t, float64(3600), claims["exp"].(float64)-claims["iat"].(float64))
			data, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &body))
			require.NotContains(t, body, "message") // 不发送 Android HMS 载荷。
			require.Equal(t, []interface{}{"test-device-token"}, body["target"].(map[string]interface{})["token"])
			require.Equal(t, true, body["pushOptions"].(map[string]interface{})["testMessage"])
			notification := body["payload"].(map[string]interface{})["notification"].(map[string]interface{})
			require.Equal(t, "群名称", notification["title"])
			require.Equal(t, "用户：你好", notification["body"])
			require.Equal(t, "IM", notification["category"])
			require.Equal(t, float64(0), notification["clickAction"].(map[string]interface{})["actionType"])
			wantBadge := badge
			if wantBadge < 0 {
				wantBadge = 0
			}
			if wantBadge > 99 {
				wantBadge = 99
			}
			require.Equal(t, map[string]interface{}{"setNum": float64(wantBadge)}, notification["badge"])
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":"80000000","msg":"Success"}`))}, nil
		})
		payload := &BasePayload{title: "群名称", content: "用户：你好", badge: badge}
		require.NoError(t, pusher.Push("test-device-token", payload))
		require.Equal(t, badge, payload.GetBadge())
	}
}

func TestHarmonyOSResponseFailures(t *testing.T) {
	pusher := newTestHarmonyOSPush(t)
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, "private response"},
		{"throttled", 503, "private response"},
		{"redirect", 302, ""},
		{"business failure", 200, `{"code":"80100003","msg":"test-device-token"}`},
		{"partial failure", 200, `{"code":"80100000"}`},
		{"missing code", 200, `{}`},
		{"wrong code type", 200, `{"code":80000000}`},
		{"null", 200, `null`},
		{"malformed", 200, `<html>error</html>`},
		{"trailing data", 200, `{"code":"80000000"} invalid`},
		{"oversized", 200, strings.Repeat(" ", 64*1024) + `{"code":"80000000"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			pusher.client.Transport = harmonyOSTestTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			err := pusher.Push("test-device-token", &BasePayload{title: "标题", content: "内容"})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "test-device-token")
			require.NotContains(t, err.Error(), "private response")
			require.Equal(t, 1, calls) // 未确认送达结果时不盲目重发，避免重复通知。
			if tc.status == http.StatusUnauthorized {
				require.Empty(t, pusher.token)
			}
		})
	}
	pusher.client.Transport = harmonyOSTestTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network unavailable")
	})
	require.ErrorContains(t, pusher.Push("token", &BasePayload{}), "network unavailable")
	require.Error(t, pusher.Push("", &BasePayload{}))
	require.Error(t, pusher.Push("token", nil))
	require.ErrorContains(t, pusher.Push("token", &BaseRTCPayload{}), "VoIP")
}

func TestHarmonyOSJWTCache(t *testing.T) {
	pusher := newTestHarmonyOSPush(t)
	now := time.Now()
	first, err := pusher.accessToken(now)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := pusher.accessToken(now.Add(58 * time.Minute))
			if err != nil || token != first {
				t.Error("concurrent callers must reuse valid JWT")
			}
		}()
	}
	wg.Wait()
	refreshed, err := pusher.accessToken(now.Add(59 * time.Minute))
	require.NoError(t, err)
	require.NotEqual(t, first, refreshed)
}

func TestHarmonyOSInvalidConfig(t *testing.T) {
	_, err := NewHarmonyOSPush(config.HarmonyOSPush{})
	require.Error(t, err)
	path := filepath.Join(t.TempDir(), "service-account.json")
	for _, body := range []string{`invalid`, `{}`, `{"project_id":"p","key_id":"k","sub_account":"s","private_key":"bad"}`} {
		require.NoError(t, os.WriteFile(path, []byte(body), 0600))
		_, err = NewHarmonyOSPush(config.HarmonyOSPush{BundleID: "com.example.app", ServiceAccountFile: path})
		require.Error(t, err)
	}
}

func TestHarmonyOSRegistration(t *testing.T) {
	pusher := newTestHarmonyOSPush(t)
	hms := NewHMSPush("", "", "android.bundle")
	w := &Webhook{pushMap: map[common.DeviceType]map[string]Push{
		common.DeviceTypeHMS: {"android.bundle": hms},
	}}
	cfg := config.New()
	require.NoError(t, w.configureHarmonyOSPush(cfg.Push.HARMONYOS))
	require.NotContains(t, w.pushMap, DeviceTypeHarmonyOS)
	cfg.Push.HARMONYOS.BundleID = pusher.config.BundleID
	require.Error(t, w.configureHarmonyOSPush(cfg.Push.HARMONYOS))
	require.NotContains(t, w.pushMap, DeviceTypeHarmonyOS)
	cfg.Push.HARMONYOS.ServiceAccountFile = pusher.config.ServiceAccountFile
	require.NoError(t, w.configureHarmonyOSPush(cfg.Push.HARMONYOS))
	require.IsType(t, &HarmonyOSPush{}, w.pushMap[DeviceTypeHarmonyOS]["com.example.app"])
	require.Nil(t, w.pushMap[DeviceTypeHarmonyOS]["android.bundle"])
	require.Same(t, hms, w.pushMap[common.DeviceTypeHMS]["android.bundle"])
}

func TestHarmonyOSDefaultNotificationOptions(t *testing.T) {
	pusher := newTestHarmonyOSPush(t)
	pusher.config.Category = ""
	pusher.config.TestMessage = false
	pusher.client.Transport = harmonyOSTestTransport(func(r *http.Request) (*http.Response, error) {
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		notification := body["payload"].(map[string]interface{})["notification"].(map[string]interface{})
		require.NotContains(t, notification, "category")
		require.Equal(t, false, body["pushOptions"].(map[string]interface{})["testMessage"])
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":"80000000"}`))}, nil
	})
	require.NoError(t, pusher.Push("token", &BasePayload{title: "标题", content: "内容"}))
}
