package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

// VIVO 推送
type VIVOPush struct {
	appID                string
	appKey               string
	appSecret            string
	authTokenCachePrefix string
	log.Log
	ctx *config.Context
}

// NewVIVOPush NewVIVOPush
func NewVIVOPush(appID, appKey, appSecret string, ctx *config.Context) *VIVOPush {
	return &VIVOPush{
		appID:                appID,
		appKey:               appKey,
		appSecret:            appSecret,
		authTokenCachePrefix: "vivo_auth_token",
		Log:                  log.NewTLog("vivopush"),
		ctx:                  ctx,
	}
}

// VIVOPayload VIVO负载
type VIVOPayload struct {
	Payload
	notifyID string
}

// NewVIVOPayload NewVIVOPayload
func NewVIVOPayload(payloadInfo *PayloadInfo, notifyID string) *VIVOPayload {
	return &VIVOPayload{
		Payload:  payloadInfo.toPayload(),
		notifyID: notifyID,
	}
}

// GetPayload GetPayload
func (v *VIVOPush) GetPayload(msg msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (Payload, error) {
	payloadInfo, err := ParsePushInfo(msg, ctx, toUser)
	if err != nil {
		return nil, err
	}
	return NewVIVOPayload(payloadInfo, fmt.Sprintf("%d", msg.MessageSeq)), nil
}

// Push Push
func (v *VIVOPush) Push(deviceToken string, payload Payload) error {
	// 推送文档 https://dev.vivo.com.cn/documentCenter/doc/362
	authToken := v.getAuthToken()
	vivoPayload := payload.(*VIVOPayload)

	resp, err := network.Post("https://api-push.vivo.com.cn/message/send", []byte(util.ToJson(map[string]interface{}{
		"regId":          deviceToken,
		"notifyType":     "4",
		"title":          vivoPayload.GetTitle(),
		"content":        vivoPayload.GetContent(),
		"skipType":       "1",
		"classification": "1",
		"pushMode":       "1",
		"requestId":      util.GenerUUID(),
	})), map[string]string{
		"authToken": authToken,
	})

	if err != nil {
		println("推送错误")
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vivo推送返回错误！-> %s", resp.Body)
	}

	resultMap, err := util.JsonToMap(resp.Body)
	if err != nil {
		return fmt.Errorf("解析vivo推送返回错误！-> %s", resp.Body)
	}

	if resultMap != nil && resultMap["result"] != nil {
		code, _ := resultMap["result"].(json.Number).Int64()
		if code != 0 {
			return errors.New(resultMap["desc"].(string))
		}
	}
	return nil
}

// getAuthToken 获取推送鉴权令牌
func (v *VIVOPush) getAuthToken() string {
	authToken, _ := v.ctx.GetRedisConn().GetString(v.authTokenCachePrefix)
	if authToken != "" {
		return authToken
	}
	timestamp := time.Now().Local().UnixNano() / 1e6
	sign := util.MD5(fmt.Sprintf("%s%s%d%s", v.appID, v.appKey, timestamp, v.appSecret))
	resp, err := network.Post("https://api-push.vivo.com.cn/message/auth", []byte(util.ToJson(map[string]interface{}{
		"appId":     v.appID,
		"appKey":    v.appKey,
		"sign":      sign,
		"timestamp": fmt.Sprintf("%d", timestamp),
	})), nil)
	if err != nil {
		v.Error("获取VIVO推送鉴权错误", zap.Error(err))
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		return authToken
	}

	resultMap, err := util.JsonToMap(resp.Body)
	if err != nil {
		return authToken
	}
	if resultMap != nil && resultMap["result"] != nil {
		code, _ := resultMap["result"].(json.Number).Int64()
		message, _ := resultMap["desc"].(string)
		if code != 0 {
			v.Error("VIVO鉴权返回错误数据", zap.String("错误信息", message))
		} else {
			authToken = resultMap["authToken"].(string)
		}
	}
	if authToken != "" {
		_ = v.ctx.GetRedisConn().SetAndExpire(v.authTokenCachePrefix, authToken, time.Hour*20)
	}
	return authToken
}
