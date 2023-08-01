package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
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
	resp, err := network.Post(fmt.Sprintf("https://uni.apistd.com/?action=sms.message.send&accessKeyId=%s", u.ctx.GetConfig().UniSMS.AccessKeyID), []byte(util.ToJson(map[string]interface{}{
		"to":         ph,
		"signature":  u.ctx.GetConfig().UniSMS.Signature,
		"templateId": "pub_verif_ttl2",
		"templateData": map[string]interface{}{
			"code": code,
			"ttl":  "10",
		},
	})), nil)
	if err != nil {
		u.Error("发送短信失败！", zap.Error(err))
		return err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			var resultMap map[string]interface{}
			util.ReadJsonByByte([]byte(resp.Body), &resultMap)
			u.Error("短信提供商返回错误！", zap.String("message", resultMap["message"].(string)))
			return errors.New(resultMap["message"].(string))
		}
		u.Error("发送短信返回状态码失败！", zap.Int("httpCode", resp.StatusCode))
		return errors.New("发送短信返回状态码失败！")
	}
	if resp.StatusCode == http.StatusOK {
		var resultMap map[string]interface{}
		util.ReadJsonByByte([]byte(resp.Body), &resultMap)
		if resultMap["code"] != "0" {
			return errors.New(resultMap["message"].(string))
		}
		return nil
	}

	return nil
}
