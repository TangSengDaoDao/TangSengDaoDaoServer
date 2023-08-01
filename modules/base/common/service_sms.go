package common

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"go.uber.org/zap"
)

type ISMSProvider interface {
	SendSMS(ctx context.Context, zone, phone string, code string) error
}

// ISMSService ISMSService
type ISMSService interface {
	// 发送验证码
	SendVerifyCode(ctx context.Context, zone, phone string, codeType CodeType) error
	// 验证验证码(销毁缓存)
	Verify(ctx context.Context, zone, phone, code string, codeType CodeType) error
}

// SMSService 短信服务
type SMSService struct {
	ctx *config.Context
	log.Log
}

// NewSMSService 创建短信服务
func NewSMSService(ctx *config.Context) *SMSService {
	return &SMSService{
		ctx: ctx,
		Log: log.NewTLog("SMSService"),
	}
}

// SendVerifyCode 发送验证码
func (s *SMSService) SendVerifyCode(ctx context.Context, zone, phone string, codeType CodeType) error {
	var smsProvider ISMSProvider

	smsProviderName := s.ctx.GetConfig().SMSProvider
	if smsProviderName == config.SMSProviderAliyun {
		if zone != "0086" && s.ctx.GetConfig().AliyunInternationalSMS.AccessKeyID != "" {
			smsProvider = NewAliyunInternationalProvider(s.ctx)
		} else {
			smsProvider = NewAliyunProvider(s.ctx)
		}
	} else if smsProviderName == config.SMSProviderUnisms {
		smsProvider = NewUnismsProvider(s.ctx)
	}

	if smsProvider == nil {
		return errors.New("没有找到短信提供商！")
	}

	verifyCode := ""
	rand.Seed(int64(time.Now().Nanosecond()))
	for i := 0; i < 4; i++ {
		verifyCode += fmt.Sprintf("%v", rand.Intn(10))
	}
	s.Info("发送验证码", zap.String("code", verifyCode))
	cacheKey := fmt.Sprintf("%s%d@%s@%s", CacheKeySMSCode, codeType, zone, phone)
	err := s.ctx.GetRedisConn().SetAndExpire(cacheKey, verifyCode, time.Minute*5)
	if err != nil {
		return err
	}
	err = smsProvider.SendSMS(ctx, zone, phone, verifyCode)
	return err
}

// Verify 验证验证码
func (s *SMSService) Verify(ctx context.Context, zone, phone, code string, codeType CodeType) error {
	span, _ := s.ctx.Tracer().StartSpanFromContext(ctx, "smsService.Verify")
	defer span.Finish()

	cacheKey := fmt.Sprintf("%s%d@%s@%s", CacheKeySMSCode, codeType, zone, phone)
	sysCode, err := s.ctx.GetRedisConn().GetString(cacheKey)
	if err != nil {
		return err
	}
	if sysCode != "" && sysCode == code {
		s.ctx.GetRedisConn().Del(cacheKey)
		return nil
	}
	return errors.New("验证码无效！")
}
