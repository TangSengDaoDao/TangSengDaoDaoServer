package common

import (
	"context"
	"errors"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	unisms "github.com/apistd/uni-go-sdk/sms"
	"go.uber.org/zap"
)

type UnismsProvider struct {
	ctx *config.Context
	log.Log
}

// NewUnismsProvider 创建短信服务
func NewUnismsProvider(ctx *config.Context) ISMSProvider {
	return &UnismsProvider{
		ctx: ctx,
		Log: log.NewTLog("UnismsProvider"),
	}
}

func (u *UnismsProvider) SendSMS(ctx context.Context, zone, phone string, code string) error {
	ph := phone
	if zone != "0086" {
		if len(zone) > 2 {
			ph = strings.Replace(zone, "00", "", 1) + phone
		}
	}

	cli := unisms.NewClient(u.ctx.GetConfig().UniSMS.AccessKeyID, u.ctx.GetConfig().UniSMS.AccessKeySecret)

	// 构建信息
	message := unisms.BuildMessage()
	message.SetTo(ph)
	message.SetSignature(u.ctx.GetConfig().UniSMS.Signature)
	message.SetTemplateId(u.ctx.GetConfig().UniSMS.TemplateId)
	message.SetTemplateData(map[string]string{"code": code}) // 设置自定义参数 (变量短信)

	// 发送短信
	res, err := cli.Send(message)
	if err != nil {
		u.Error("发送短信失败！", zap.Error(err))
		return err
	}
	if res.Code != "0" {
		u.Error("发送短信失败！", zap.String("message", res.Message))
		return errors.New(res.Message)
	}
	return nil
}
