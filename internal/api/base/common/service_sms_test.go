package common

import (
	"context"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAliyunSendVerifyCode(t *testing.T) {
	cfg := config.New()
	ctx := testutil.NewTestContext(cfg)
	smsService := NewSMSService(ctx)
	err := smsService.SendVerifyCode(context.Background(), "0086", "15921615913", CodeTypeRegister)
	assert.NoError(t, err)
}

func TestUnismsSendVerifyCode(t *testing.T) {
	cfg := config.New()
	cfg.UniSMS.Signature = "飞船"
	cfg.SMSProvider = config.SMSProviderUnisms
	cfg.UniSMS.AccessKeyID = "jtrNcLPQF1tA3JQuGQq7NyT3Aj7N9Tg9xmYDjRW8ehqoAMqcK"
	ctx := testutil.NewTestContext(cfg)
	smsService := NewSMSService(ctx)
	err := smsService.SendVerifyCode(context.Background(), "0086", "15921615913", CodeTypeRegister)
	assert.NoError(t, err)
}
