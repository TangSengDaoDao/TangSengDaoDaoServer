package webhook

import (
	"errors"
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
)

// IOSPayload iOS负载
type IOSPayload struct {
	Payload
}

// NewIOSPayload NewIOSPayload
func NewIOSPayload(payloadInfo *PayloadInfo) Payload {

	return &IOSPayload{
		Payload: payloadInfo.toPayload(),
	}
}

// IOSPush IOSPush
type IOSPush struct {
	client      *apns2.Client
	topic       string
	password    string
	p12FilePath string
	dev         bool // 是否是开发环境
	log.Log
}

// NewIOSPush NewIOSPush
func NewIOSPush(topic string, dev bool, p12FilePath string, password string) *IOSPush {
	return &IOSPush{
		topic:       topic,
		dev:         dev,
		p12FilePath: p12FilePath,
		password:    password,
		Log:         log.NewTLog("IOSPush"),
	}
}

func (p *IOSPush) createClient() (*apns2.Client, error) {
	cert, err := certificate.FromP12File(p.p12FilePath, p.password)
	if err != nil {
		return nil, err
	}
	var client *apns2.Client
	if p.dev {
		client = apns2.NewClient(cert).Development()
	} else {
		client = apns2.NewClient(cert).Production()
	}
	return client, nil
}

// GetPayload 获取推送负载
func (p *IOSPush) GetPayload(msg msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (Payload, error) {
	pushInfo, err := ParsePushInfo(msg, ctx, toUser)
	if err != nil {
		return nil, err
	}
	return NewIOSPayload(pushInfo), nil
}

// Push iOS推送
func (p *IOSPush) Push(deviceToken string, payload Payload) error {
	notification := &apns2.Notification{}
	notification.DeviceToken = deviceToken
	notification.Topic = p.topic

	rtcPayload := payload.GetRTCPayload()
	if rtcPayload != nil {
		fmt.Println("音视频推送。。。。。")
		notification.Payload = []byte(util.ToJson(map[string]interface{}{
			"aps": map[string]interface{}{
				"content-available": 1,
				"alert":             "",
				"badge":             payload.GetBadge(),
				"sound":             "default",
			},
			"content":   payload.GetContent(),
			"call_type": rtcPayload.GetCallType(),
			"from_uid":  rtcPayload.GetFromUID(),
		}))
	} else {
		fmt.Println("普通推送。。。。。")
		notification.Payload = []byte(util.ToJson(map[string]interface{}{
			"aps": map[string]interface{}{
				"alert": map[string]interface{}{
					"title": payload.GetTitle(),
					"body":  payload.GetContent(),
				},
				"badge": payload.GetBadge(),
				"sound": "default",
			},
		}))
	}

	var err error
	if p.client == nil {
		p.client, err = p.createClient()
		if err != nil {
			return err
		}
	}
	res, err := p.client.Push(notification)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.New(res.Reason)
	}
	return nil
}
