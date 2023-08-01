package common

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	sms_intl20180501 "github.com/alibabacloud-go/sms-intl-20180501/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
	"github.com/opentracing/opentracing-go/ext"
)

type AliyunProvider struct {
	ctx *config.Context
	log.Log
}

// NewAliyunProvider 创建短信服务
func NewAliyunProvider(ctx *config.Context) ISMSProvider {
	return &AliyunProvider{
		ctx: ctx,
		Log: log.NewTLog("AliyunProvider"),
	}
}

func (a *AliyunProvider) SendSMS(ctx context.Context, zone, phone string, code string) error {
	fmt.Println("AliyunProvider......")
	span, _ := a.ctx.Tracer().StartSpanFromContext(ctx, "smsService.SendVerifyCode")
	defer span.Finish()
	client, err := dysmsapi.NewClientWithAccessKey("cn-hangzhou", a.ctx.GetConfig().AliyunSMS.AccessKeyID, a.ctx.GetConfig().AliyunSMS.AccessSecret)
	if err != nil {
		return err
	}
	request := dysmsapi.CreateSendSmsRequest()
	request.Scheme = "https"
	request.PhoneNumbers = phone
	request.SignName = a.ctx.GetConfig().AliyunSMS.SignName
	request.TemplateCode = a.ctx.GetConfig().AliyunSMS.TemplateCode
	request.TemplateParam = util.ToJson(map[string]interface{}{
		"code": code,
	})
	response, err := client.SendSms(request)
	if err != nil {
		ext.LogError(span, err)
		return err
	}
	if response.Code == "OK" {
		fmt.Println("AliyunProvider......ok...")
		return nil
	}
	return errors.New(response.Message)
}

type AliyunInternationalProvider struct {
	ctx *config.Context
	log.Log
}

// NewAliyunInternationalProvider 创建短信服务
func NewAliyunInternationalProvider(ctx *config.Context) ISMSProvider {
	return &AliyunInternationalProvider{
		ctx: ctx,
		Log: log.NewTLog("AliyunInternationalProvider"),
	}
}

func (a *AliyunInternationalProvider) SendSMS(ctx context.Context, zone, phone string, code string) error {
	span, _ := a.ctx.Tracer().StartSpanFromContext(ctx, "smsService.SendVerifyCode")
	defer span.Finish()

	sendMessageToGlobeRequest := &sms_intl20180501.SendMessageToGlobeRequest{
		To:      tea.String(fmt.Sprintf("%s%s", strings.TrimLeft(zone, "00"), phone)),
		Message: tea.String(fmt.Sprintf("【%s】您的验证码%s，该验证码5分钟内有效，请勿泄漏于他人！", a.ctx.GetConfig().AppName, code)),
		Type:    tea.String("OTP"),
	}
	client, err := a.createClient()
	if err != nil {
		return err
	}
	response, err := client.SendMessageToGlobe(sendMessageToGlobeRequest)
	if err != nil {
		return err
	}

	if *response.Body.ResponseCode == "OK" {
		return nil
	}
	return nil
}

// 初始化账号Client
func (s *AliyunInternationalProvider) createClient() (_result *sms_intl20180501.Client, _err error) {
	config := &openapi.Config{
		// 您的AccessKey ID
		AccessKeyId: &s.ctx.GetConfig().AliyunInternationalSMS.AccessKeyID,
		// 您的AccessKey Secret
		AccessKeySecret: &s.ctx.GetConfig().AliyunInternationalSMS.AccessSecret,
	}
	// 访问的域名
	config.Endpoint = tea.String("dysmsapi.ap-southeast-1.aliyuncs.com")
	_result, _err = sms_intl20180501.NewClient(config)
	return _result, _err
}
