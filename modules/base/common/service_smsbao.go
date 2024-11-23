// smsbao
package common

import (
	"context"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"

	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type SmsbaoProvider struct {
	ctx *config.Context
	log.Log
}

// NewSmsbaoProvider 创建短信服务
func NewSmsbaoProvider(ctx *config.Context) ISMSProvider {
	return &SmsbaoProvider{
		ctx: ctx,
		Log: log.NewTLog("SmsbaoProvider"),
	}
}

func (u *SmsbaoProvider) SendSMS(ctx context.Context, zone, phone string, code string) error {
	ph := phone
	if zone != "0086" {
		if len(zone) > 2 {
			ph = strings.Replace(zone, "00", "", 1) + phone
		}
	}

	//u.ctx.GetConfig().UniSMS.AccessKeyID, u.ctx.GetConfig().UniSMS.AccessKeySecret
	//u.ctx.GetConfig().UniSMS.Signature
	//u.ctx.GetConfig().UniSMS.TemplateId

	// 状态码与提示信息的映射
	statusStr := map[string]string{
		"0":  "短信发送成功",
		"-1": "参数不全",
		"-2": "服务器空间不支持, 请确认支持curl或者fsocket，联系您的空间商解决或者更换空间！",
		"30": "密码错误",
		"40": "账号不存在",
		"41": "余额不足",
		"42": "帐户已过期",
		"43": "IP地址限制",
		"50": "内容含有敏感词",
	}

	smsapi := "https://api.smsbao.com/" // 短信API
	//user := "shyuke1688"                       // 短信平台帐号
	//pass := "74cb45173e9642478c2c07f29b82850e" // 短信平台密码，未加密的原文

	user := u.ctx.GetConfig().Smsbao.Account
	pass := u.ctx.GetConfig().Smsbao.APIKey
	tpl := u.ctx.GetConfig().Smsbao.Template
	// 加密密码为MD5
	hash := md5.New()
	hash.Write([]byte(pass))
	passMd5 := hex.EncodeToString(hash.Sum(nil))

	content := strings.Replace(tpl, "{code}", code, -1) // "您好！您的验证码是: " + code + " 。五分钟内有效。注意验证码打死也不要告诉别人哦！"
	//phone := "13714715608" // 要发送短信的手机号码
	fmt.Printf("Template: %s\n", tpl)
	fmt.Printf("Content: %s\n", content)
	// 构建请求URL
	params := url.Values{}
	params.Set("u", user)
	params.Set("p", passMd5)
	params.Set("m", ph)
	params.Set("c", content)

	sendurl := smsapi + "sms?" + params.Encode()

	// 发送HTTP请求
	resp, err := http.Get(sendurl)
	if err != nil {
		fmt.Println("HTTP请求失败:", err)
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return err
	}

	result := string(body)
	fmt.Println("响应代码:", result)
	u.Error("短信发送结果:" + result)
	// 显示对应的状态信息
	if msg, ok := statusStr[result]; ok {
		fmt.Println("状态信息:", msg)
	} else {
		fmt.Println("未知状态码:", result)
	}
	return nil
}
