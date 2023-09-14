package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"go.uber.org/zap"
)

// OPPO 推送
type OPPOPush struct {
	appID                string
	appKey               string
	appSecret            string
	masterSecret         string // 服务端密钥
	authTokenCachePrefix string
	log.Log
	ctx *config.Context
}

// NewOPPOPush NewOPPOPush
func NewOPPOPush(appID, appKey, appSecret, masterSecret string, ctx *config.Context) *OPPOPush {
	return &OPPOPush{
		appID:                appID,
		appKey:               appKey,
		appSecret:            appSecret,
		masterSecret:         masterSecret,
		authTokenCachePrefix: "oppo_auth_token",
		Log:                  log.NewTLog("oppopush"),
		ctx:                  ctx,
	}
}

// OPPOPayload oppo负载
type OPPOPayload struct {
	Payload
	notifyID string
}

// NewOPPOPayload NewOPPOPayload
func NewOPPOPayload(payloadInfo *PayloadInfo, notifyID string) *OPPOPayload {
	return &OPPOPayload{
		Payload:  payloadInfo.toPayload(),
		notifyID: notifyID,
	}
}

// GetPayload GetPayload
func (o *OPPOPush) GetPayload(msg msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (Payload, error) {
	payloadInfo, err := ParsePushInfo(msg, ctx, toUser)
	if err != nil {
		return nil, err
	}
	return NewOPPOPayload(payloadInfo, fmt.Sprintf("%d", msg.MessageSeq)), nil
}

// Push Push
func (o *OPPOPush) Push(deviceToken string, payload Payload) error {
	// 推送文档 https://open.oppomobile.com/new/developmentDoc/info?id=11238
	authToken := o.getAuthToken()
	oppoPayload := payload.(*OPPOPayload)
	message := map[string]interface{}{
		"target_type":  2,
		"target_value": deviceToken,
		"notification": map[string]string{
			"title":   oppoPayload.GetTitle(),
			"content": oppoPayload.GetContent(),
		},
	}
	dataType, _ := json.Marshal(message)
	dataString := string(dataType)
	resp, err := network.PostForWWWForm("https://api.push.oppomobile.com/server/v1/message/notification/unicast", map[string]string{
		"auth_token": authToken,
		"message":    dataString,
	}, nil)

	if err != nil {
		println("推送错误")
		return err
	}
	if resp != nil && resp["code"] != nil {
		code, _ := resp["code"].(json.Number).Int64()
		if code != 0 {
			return errors.New(resp["message"].(string))
		}
	}
	return nil
}

// getAuthToken 获取推送鉴权令牌
func (o *OPPOPush) getAuthToken() string {
	authToken, _ := o.ctx.GetRedisConn().GetString(o.authTokenCachePrefix)
	if authToken != "" {
		return authToken
	}
	timestamp := time.Now().Local().UnixNano() / 1e6
	sign := o.SHA256(fmt.Sprintf("%s%d%s", o.appKey, timestamp, o.masterSecret))
	resp, err := network.PostForWWWForm("https://api.push.oppomobile.com/server/v1/auth", map[string]string{
		"app_key":   o.appKey,
		"sign":      sign,
		"timestamp": fmt.Sprintf("%d", timestamp),
	}, nil)
	if err != nil {
		o.Error("获取OPPO推送鉴权错误", zap.Error(err))
		return ""
	}

	if resp != nil && resp["code"] != nil {
		code, _ := resp["code"].(json.Number).Int64()
		message, _ := resp["message"].(string)
		if code != 0 {
			o.Error("OPPO鉴权返回错误数据", zap.String("错误信息", message))
		} else {
			data, _ := resp["data"].(map[string]interface{})
			authToken = data["auth_token"].(string)
		}
	}
	if authToken != "" {
		_ = o.ctx.GetRedisConn().SetAndExpire(o.authTokenCachePrefix, authToken, time.Hour*20)
	}
	return authToken
}

func (o *OPPOPush) SHA256(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	res := hex.EncodeToString(hash.Sum(nil))
	fmt.Println(len(res))
	return res
}
