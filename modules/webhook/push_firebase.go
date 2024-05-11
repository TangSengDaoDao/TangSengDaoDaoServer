package webhook

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	message "firebase.google.com/go/v4/messaging"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"google.golang.org/api/option"
)

// FIREBASEPush 参考代码 https://github.com/firebase/firebase-admin-go/blob/61c6c041bf807c045f6ff3fd0d02fc480f806c9a/snippets/messaging.go#L29-L55
// FIREBASEPush GOOGLE推送
type FIREBASEPush struct {
	jsonPath    string //
	packageName string // android包名
	projectId   string // serviceAccountJson中的project_id值
	channelID   string // 频道id 如果有则填写
	client      message.Client
	log.Log
}

// NewFIREBASEPush NewFIREBASEPush
func NewFIREBASEPush(jsonPath string, packageName string, projectID string, channelID string) *FIREBASEPush {
	// Initialize another app with a different config
	ctx := context.Background()

	c := &firebase.Config{ProjectID: projectID}

	opt := option.WithCredentialsFile(jsonPath)
	app, err := firebase.NewApp(ctx, c, opt)
	if err != nil {
		log.Error("无法初始化firebase: 通过json创建firebase客户端时 ")
		return nil
	}
	// Obtain a messaging.Client from the App.
	client, err := app.Messaging(ctx)
	if err != nil {
		log.Error("通过APP client 创建 message client时:" + err.Error())
		return nil
	}

	return &FIREBASEPush{
		jsonPath:    jsonPath,
		packageName: packageName,
		channelID:   channelID,
		client:      *client,
		projectId:   projectID,
		Log:         log.NewTLog("FIREBASEPush"),
	}
}

// FIREBASEPayload Google Firebase负载
type FIREBASEPayload struct {
	Payload
	notifyID string
}

// NewFIREBASEPayload NewFIREBASEPayload
func NewFIREBASEPayload(payloadInfo *PayloadInfo, notifyID string) *FIREBASEPayload {
	return &FIREBASEPayload{
		Payload:  payloadInfo.toPayload(),
		notifyID: notifyID,
	}
}

// GetPayload 获取推送负载
func (m *FIREBASEPush) GetPayload(msg msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (Payload, error) {
	payloadInfo, err := ParsePushInfo(msg, ctx, toUser)
	if err != nil {
		return nil, err
	}
	return NewFIREBASEPayload(payloadInfo, fmt.Sprintf("%d", msg.MessageSeq)), nil
}

// Push 推送
func (m *FIREBASEPush) Push(deviceToken string, payload Payload) error {
	miPayload := payload.(*FIREBASEPayload)
	ctx := context.Background()
	// 文档 https://firebase.google.com/docs/admin/setup?hl=zh-cn#go_1
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: miPayload.GetTitle(),
			Body:  miPayload.GetContent(),
		},
		Token: deviceToken,
	}

	// Send a message to the device corresponding to the provided
	// registration token.
	response, err := m.client.Send(ctx, message)
	// Response is a message ID string.
	m.Debug("Successfully sent firebase message:" + response)
	if err != nil {
		return err
	}
	return nil
}
