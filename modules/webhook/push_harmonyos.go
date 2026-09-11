package webhook

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/golang-jwt/jwt/v4"
)

const DeviceTypeHarmonyOS common.DeviceType = "HARMONYOS"

const harmonyOSTokenAudience = "https://oauth-login.cloud.huawei.com/oauth2/v3/token"

func (w *Webhook) configureHarmonyOSPush(cfg config.HarmonyOSPush) error {
	cfg.BundleID = strings.TrimSpace(cfg.BundleID)
	cfg.ServiceAccountFile = strings.TrimSpace(cfg.ServiceAccountFile)
	cfg.Category = strings.TrimSpace(cfg.Category)
	if cfg.BundleID == "" {
		return nil
	}
	pusher, err := NewHarmonyOSPush(cfg)
	if err != nil {
		return err
	}
	w.pushMapMu.Lock()
	defer w.pushMapMu.Unlock()
	w.pushMap[DeviceTypeHarmonyOS] = map[string]Push{cfg.BundleID: pusher}
	return nil
}

type harmonyOSServiceAccount struct {
	ProjectID  string `json:"project_id"`
	KeyID      string `json:"key_id"`
	PrivateKey string `json:"private_key"`
	SubAccount string `json:"sub_account"`
}

// HarmonyOSPush 通过 V3 场景化消息接口发送普通通知。
type HarmonyOSPush struct {
	config     config.HarmonyOSPush
	projectID  string
	keyID      string
	subAccount string
	privateKey *rsa.PrivateKey
	client     *http.Client
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
}

var _ Push = (*HarmonyOSPush)(nil)

func NewHarmonyOSPush(cfg config.HarmonyOSPush) (*HarmonyOSPush, error) {
	if cfg.BundleID == "" || cfg.ServiceAccountFile == "" {
		return nil, errors.New("HarmonyOS push requires bundleID and serviceAccountFile")
	}
	data, err := os.ReadFile(cfg.ServiceAccountFile)
	if err != nil {
		return nil, fmt.Errorf("read HarmonyOS service account: %w", err)
	}
	var account harmonyOSServiceAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, errors.New("invalid HarmonyOS service account JSON")
	}
	if strings.TrimSpace(account.ProjectID) == "" || strings.TrimSpace(account.KeyID) == "" || strings.TrimSpace(account.SubAccount) == "" {
		return nil, errors.New("HarmonyOS service account requires project_id, key_id and sub_account")
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(account.PrivateKey))
	if err != nil {
		return nil, errors.New("invalid HarmonyOS service account RSA private_key")
	}
	return &HarmonyOSPush{
		config: cfg, projectID: account.ProjectID, keyID: account.KeyID,
		subAccount: account.SubAccount, privateKey: key,
		client: &http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (h *HarmonyOSPush) GetPayload(msg msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (Payload, error) {
	info, err := ParsePushInfo(msg, ctx, toUser)
	if err != nil {
		return nil, err
	}
	return info.toPayload(), nil
}

func (h *HarmonyOSPush) accessToken(now time.Time) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.token != "" && now.Add(time.Minute).Before(h.expiresAt) {
		return h.token, nil
	}
	expiresAt := now.Add(time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, jwt.MapClaims{
		"iss": h.subAccount, "aud": harmonyOSTokenAudience,
		"iat": now.Unix(), "exp": expiresAt.Unix(),
	})
	token.Header["kid"] = h.keyID
	signed, err := token.SignedString(h.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign HarmonyOS push JWT: %w", err)
	}
	h.token, h.expiresAt = signed, expiresAt
	return signed, nil
}

func (h *HarmonyOSPush) Push(deviceToken string, payload Payload) error {
	if strings.TrimSpace(deviceToken) == "" || payload == nil {
		return errors.New("HarmonyOS push requires device token and payload")
	}
	// wkpush 当前没有 VoIP 消费能力，不能将通话控制消息伪装为普通通知。
	if payload.GetRTCPayload() != nil {
		return errors.New("HarmonyOS VoIP push is not supported by the client")
	}
	badge := payload.GetBadge()
	if badge < 0 {
		badge = 0
	} else if badge > 99 {
		badge = 99 // REST API setNum 只接受 [0, 100)，不修改服务端真实未读数。
	}
	notification := map[string]interface{}{
		"title": payload.GetTitle(), "body": payload.GetContent(),
		"badge":       map[string]int{"setNum": badge},
		"clickAction": map[string]int{"actionType": 0},
	}
	if h.config.Category != "" {
		notification["category"] = h.config.Category
	}
	body, err := json.Marshal(map[string]interface{}{
		"payload":     map[string]interface{}{"notification": notification},
		"target":      map[string]interface{}{"token": []string{deviceToken}},
		"pushOptions": map[string]interface{}{"testMessage": h.config.TestMessage},
	})
	if err != nil {
		return err
	}
	token, err := h.accessToken(time.Now())
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://push-api.cloud.huawei.com/v3/"+url.PathEscape(h.projectID)+"/messages:send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("push-type", "0")
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send HarmonyOS push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		h.mu.Lock()
		if h.token == token {
			h.token = "" // 下一条消息重新签发；不自动重发当前通知。
		}
		h.mu.Unlock()
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HarmonyOS push HTTP status %d", resp.StatusCode)
	}
	var result struct {
		Code string `json:"code"`
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024+1))
	if err != nil {
		return fmt.Errorf("read HarmonyOS push response: %w", err)
	}
	if len(data) > 64*1024 || json.Unmarshal(data, &result) != nil || result.Code == "" {
		return errors.New("invalid HarmonyOS push response")
	}
	if result.Code != "80000000" {
		// 不记录原始响应，云端错误信息可能回显设备 Token 或通知正文。
		return fmt.Errorf("HarmonyOS push rejected, code=%s", result.Code)
	}
	return nil
}
