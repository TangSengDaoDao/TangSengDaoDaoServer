package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

// HMSPayload 华为负载
type HMSPayload struct {
	Payload
	accessToken string
}

// NewHMSPayload NewHMSPayload
func NewHMSPayload(payloadInfo *PayloadInfo, accessToken string) *HMSPayload {
	return &HMSPayload{
		Payload:     payloadInfo.toPayload(),
		accessToken: accessToken,
	}
}

// HMSPush 华为推送
type HMSPush struct {
	appID       string // 华为app id
	appSecret   string // 华为app secret
	packageName string // android包名
	log.Log
	hmsAccessTokenCachePrefix string
}

// NewHMSPush NewHMSPush
func NewHMSPush(appID string, appSecret string, packageName string) *HMSPush {
	return &HMSPush{
		appID:                     appID,
		appSecret:                 appSecret,
		packageName:               packageName,
		hmsAccessTokenCachePrefix: "hms_accesstoken",
		Log:                       log.NewTLog("HMSPush"),
	}
}

// GetPayload 获取推送负载
func (h *HMSPush) GetPayload(msg msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (Payload, error) {
	payloadInfo, err := ParsePushInfo(msg, ctx, toUser)
	if err != nil {
		log.Warn("推送失败！", zap.Error(err))
		return nil, err
	}
	accessToken, err := ctx.GetRedisConn().GetString(h.hmsAccessTokenCachePrefix)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(accessToken) == "" {
		var expire time.Duration
		accessToken, expire, err = h.GetHMSAccessToken()
		if err != nil {
			return nil, err
		}
		err = ctx.GetRedisConn().SetAndExpire(h.hmsAccessTokenCachePrefix, accessToken, expire)
		if err != nil {
			return nil, err
		}
	}
	return NewHMSPayload(payloadInfo, accessToken), nil
}

// Push 推送
func (h *HMSPush) Push(deviceToken string, payload Payload) error {
	hmsPayload := payload.(*HMSPayload)
	channelID := "wukongchat_new_msg_notification"
	sound := "/raw/newmsg"
	category := "IM"
	if hmsPayload.GetRTCPayload() != nil && hmsPayload.GetRTCPayload().GetOperation() != "cancel" {
		channelID = "wukongchat_new_rtc_notification"
		sound = "/raw/newrtc"
		category = "VOIP"
	}
	resp, err := network.Post(fmt.Sprintf("https://push-api.cloud.huawei.com/v1/%s/messages:send", h.appID), []byte(util.ToJson(map[string]interface{}{
		"validate_only": false,
		"message": map[string]interface{}{
			"token": []string{deviceToken},
			"android": map[string]interface{}{
				"category": category,
				"notification": map[string]interface{}{
					"visibility":    "PUBLIC",
					"title":         payload.GetTitle(),
					"body":          payload.GetContent(),
					"sound":         sound,
					"importance":    "NORMAL",
					"default_sound": true,
					"channel_id":    channelID,
					"click_action": map[string]interface{}{
						"type": 3,
					},
					"badge": map[string]interface{}{
						"add_num": 1,
						"class":   fmt.Sprintf("%s%s", h.packageName, ".MainActivity"),
					},
				},
			},
		},
	})), map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", hmsPayload.accessToken),
	})
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("华为推送返回错误！-> %s", resp.Body)
	}
	h.Debug("返回", zap.String("body", resp.Body))
	resultMap, err := util.JsonToMap(resp.Body)
	if err != nil {
		return err
	}
	if resultMap != nil && resultMap["code"] != nil {
		code := resultMap["code"].(string)
		if code != "80000000" {
			return errors.New(resultMap["msg"].(string))
		}
	}
	return nil
}

// GetHMSAccessToken 获取华为的访问Token
func (h *HMSPush) GetHMSAccessToken() (string, time.Duration, error) {

	resultMap, err := network.PostForWWWForm("https://oauth-login.cloud.huawei.com/oauth2/v2/token", map[string]string{
		"grant_type":    "client_credentials",
		"client_secret": h.appSecret,
		"client_id":     h.appID,
	}, nil)
	if err != nil {
		return "", 0, err
	}
	if resultMap != nil {
		accessToken := resultMap["access_token"].(string)
		expiresIn, _ := resultMap["expires_in"].(json.Number).Int64()
		if expiresIn <= 0 {
			expiresIn = 3600
		}
		return accessToken, time.Duration(expiresIn) * time.Second, nil
	}
	return "", 0, nil

}
