package common

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Common Common
type Common struct {
	ctx *config.Context
	log.Log
	db          *db
	appConfigDB *appConfigDB
}

// New New
func New(ctx *config.Context) *Common {
	return &Common{
		ctx:         ctx,
		db:          newDB(ctx.DB()),
		appConfigDB: newAppConfigDB(ctx),
		Log:         log.NewTLog("common"),
	}
}

// Route 路由配置
func (cn *Common) Route(r *wkhttp.WKHttp) {
	common := r.Group("/v1/common", cn.ctx.AuthMiddleware(r))
	{
		common.POST("/appversion", cn.addAppVersion)             // 添加APP版本
		common.GET("/appversion/:os/:version", cn.getNewVersion) //获取最新版本
		common.GET("/appversion/list", cn.appVersionList)        // 版本列表
		common.GET("/chatbg", cn.chatBgList)                     // 聊天背景列表
		common.GET("/appmodule", cn.appModule)                   // app模块列表
	}
	commonNoAuth := r.Group("/v1/common")
	{
		commonNoAuth.GET("/countries", cn.countriesList)

		commonNoAuth.GET("/appconfig", cn.appConfig) // app配置

		commonNoAuth.GET("/updater/:os/:version", cn.updater) // 版本更新检查（兼容tauri）
	}

	r.GET("/v1/health", func(c *wkhttp.Context) {
		var (
			statusMap = map[string]string{
				"status": "up",
				"db":     "up",
				"redis":  "up",
			}
			lastError error
		)

		err := cn.db.session.Ping()
		if err != nil {
			cn.Error("db ping error", zap.Error(err))
			lastError = err
			statusMap["db"] = "down"
		}

		_, err = cn.ctx.GetRedisConn().Ping()
		if err != nil {
			cn.Error("redis ping error", zap.Error(err))
			lastError = err
			statusMap["redis"] = "down"
		}

		if lastError != nil {
			statusMap["status"] = "down"
			statusMap["error"] = lastError.Error()
		}

		c.JSON(http.StatusOK, statusMap)
	})

	appConfigM, err := cn.insertAppConfigIfNeed()
	if err != nil {
		panic(err)
	}
	// 设置系统私钥
	cn.ctx.GetConfig().AppRSAPrivateKey = appConfigM.RSAPrivateKey
	cn.ctx.GetConfig().AppRSAPubKey = appConfigM.RSAPublicKey
}

func (cn *Common) updater(c *wkhttp.Context) {
	os := c.Param("os")
	oldVersion := c.Param("version")

	model, err := cn.db.queryNewVersion(os)
	if err != nil {
		cn.Error("查询最新版本错误", zap.Error(err))
		c.ResponseError(errors.New("查询最新版本错误"))
		return
	}
	if model == nil || model.AppVersion == oldVersion {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":       model.DownloadURL,
		"version":   model.AppVersion,
		"notes":     model.UpdateDesc,
		"pub_date":  time.Time(model.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		"signature": model.Signature,
	})
}

// 查询app模块
func (cn *Common) appModule(c *wkhttp.Context) {
	modules, err := cn.db.queryAppModule()
	if err != nil {
		cn.Error("查询所有app模块错误", zap.Error(err))
		c.ResponseError(errors.New("查询所有app模块错误"))
		return
	}
	list := make([]*appModuleResp, 0)
	if len(modules) > 0 {
		for _, module := range modules {
			list = append(list, &appModuleResp{
				SID:    module.SID,
				Name:   module.Name,
				Desc:   module.Desc,
				Status: module.Status,
			})
		}
	}
	c.Response(list)
}

// 查询聊天背景列表
func (cn *Common) chatBgList(c *wkhttp.Context) {
	list, err := cn.db.queryChatBgs()
	if err != nil {
		cn.Error("查询所有聊天背景错误", zap.Error(err))
		c.ResponseError(errors.New("查询所有聊天背景错误"))
		return
	}
	resps := make([]*chatBgResp, 0)
	if len(list) == 0 {
		c.Response(resps)
		return
	}
	for index, model := range list {
		var lightColors = make([]string, 0)
		var darkColors = make([]string, 0)
		if model.IsSvg == 1 && index < len(defaultColorsLight) {
			lightColors = defaultColorsLight[index]
		}
		if model.IsSvg == 1 && index < len(defaultColorsDark) {
			darkColors = defaultColorsDark[index]
		}
		resps = append(resps, &chatBgResp{
			Cover:       model.Cover,
			Url:         model.Url,
			IsSvg:       model.IsSvg,
			LightColors: lightColors,
			DarkColors:  darkColors,
		})
	}
	c.Response(resps)
}
func (cn *Common) insertAppConfigIfNeed() (*appConfigModel, error) {

	appConfigM, err := cn.appConfigDB.query()
	if err != nil {
		return nil, err
	}
	if appConfigM != nil {
		return appConfigM, nil
	}

	privateKeyBuff := new(bytes.Buffer)
	publicKeyBuff := new(bytes.Buffer)

	bits := 2048
	// 生成私钥文件
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	err = pem.Encode(privateKeyBuff, block)
	if err != nil {
		return nil, err
	}
	// 生成公钥文件
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	err = pem.Encode(publicKeyBuff, block)
	if err != nil {
		return nil, err
	}

	appConfigM = &appConfigModel{
		RSAPrivateKey: privateKeyBuff.String(),
		RSAPublicKey:  publicKeyBuff.String(),
		Version:       1,
		SuperToken:    util.GenerUUID(),
		SuperTokenOn:  0,
		SearchByPhone: 1,
	}
	err = cn.appConfigDB.insert(appConfigM)
	return appConfigM, err
}

func (cn *Common) appConfig(c *wkhttp.Context) {
	versionStr := c.Query("version")
	appConfigM, err := cn.appConfigDB.query()
	if err != nil {
		cn.Error("查询应用配置失败！", zap.Error(err))
		c.ResponseError(errors.New("查询应用配置失败！"))
		return
	}
	versionI64, _ := strconv.ParseInt(versionStr, 10, 64)
	if versionI64 != 0 && int(versionI64) >= appConfigM.Version {
		c.JSON(http.StatusOK, &appConfigResp{
			Version: appConfigM.Version,
		})
		return
	}
	var phoneSearchOff int
	var shortnoEditOff int
	var revokeSecond int
	if cn.ctx.GetConfig().PhoneSearchOff {
		phoneSearchOff = 1
	}
	if cn.ctx.GetConfig().ShortNo.EditOff {
		shortnoEditOff = 1
	}
	if appConfigM.RevokeSecond == 0 {
		revokeSecond = -1
	} else {
		revokeSecond = appConfigM.RevokeSecond
	}

	c.JSON(http.StatusOK, &appConfigResp{
		Version:        appConfigM.Version,
		PhoneSearchOff: phoneSearchOff,
		ShortnoEditOff: shortnoEditOff,
		WebURL:         cn.ctx.GetConfig().External.WebLoginURL,
		RevokeSecond:   revokeSecond,
	})
}

func (cn *Common) countriesList(c *wkhttp.Context) {
	c.JSON(http.StatusOK, Countrys())
}

// 添加app版本
func (cn *Common) addAppVersion(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	var req appVersionReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	err = cn.check(req)
	if err != nil {
		c.ResponseError(err)
		return
	}
	_, err = cn.db.insertAppVersion(&appVersionModel{
		AppVersion:  req.AppVersion,
		OS:          req.OS,
		IsForce:     req.IsForce,
		UpdateDesc:  req.UpdateDesc,
		DownloadURL: req.DownloadURL,
	})
	if err != nil {
		cn.Error("添加更新记录错误", zap.Error(err))
		c.ResponseError(errors.New("添加更新记录错误"))
		return
	}
	c.ResponseOK()
}

// 获取最新版本
func (cn *Common) getNewVersion(c *wkhttp.Context) {
	os := c.Param("os")
	version := c.Param("version")
	if os == "" {
		c.ResponseError(errors.New("平台类型不能为空"))
		return
	}
	if version == "" {
		c.ResponseError(errors.New("版本号不能为空"))
		return
	}
	model, err := cn.db.queryNewVersion(os)
	if err != nil {
		cn.Error("查询最新版本错误", zap.Error(err))
		c.ResponseError(errors.New("查询最新版本错误"))
		return
	}
	if model == nil || model.AppVersion == version {
		c.Response(map[string]interface{}{})
		return
	}
	c.Response(&appVersionResp{
		AppVersion:  model.AppVersion,
		OS:          model.OS,
		DownloadURL: model.DownloadURL,
		IsForce:     model.IsForce,
		UpdateDesc:  model.UpdateDesc,
		CreatedAt:   model.CreatedAt.String(),
	})
}

// 查询总记录
func (cn *Common) appVersionList(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	pageIndex, pageSize := c.GetPage()
	list, err := cn.db.queryAppVersionListWithPage(uint64(pageSize), uint64(pageIndex))
	if err != nil {
		cn.Error("查询版本列表错误", zap.Error(err))
		c.ResponseError(errors.New("查询版本列表错误"))
		return
	}
	count, err := cn.db.queryCount()
	if err != nil {
		cn.Error("查询总数量错误", zap.Error(err))
		c.ResponseError(errors.New("查询总数量错误"))
		return
	}
	resps := make([]*appVersionResp, 0)
	if len(list) == 0 {
		c.Response(map[string]interface{}{
			"count": count,
			"list":  resps,
		})
		return
	}

	for _, model := range list {
		resps = append(resps, &appVersionResp{
			AppVersion:  model.AppVersion,
			OS:          model.OS,
			IsForce:     model.IsForce,
			UpdateDesc:  model.UpdateDesc,
			DownloadURL: model.DownloadURL,
			CreatedAt:   model.CreatedAt.String(),
		})
	}
	c.Response(map[string]interface{}{
		"count": count,
		"list":  resps,
	})
}

func (cn *Common) check(req appVersionReq) error {
	if req.AppVersion == "" {
		return errors.New("请输入版本号")
	}
	if req.UpdateDesc == "" {
		return errors.New("请输入更新说明")
	}
	if req.OS == "" {
		return errors.New("请输入升级平台")
	}
	if req.OS == "android" && req.DownloadURL == "" {
		return errors.New("Android平台请传入下载地址")
	}
	return nil
}

type appModuleResp struct {
	SID    string `json:"sid"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Status int    `json:"status"` // 模块状态 1.可选 0.不可选 2.选中不可编辑
}

type chatBgResp struct {
	Cover       string   `json:"cover"`
	Url         string   `json:"url"`
	IsSvg       int      `json:"is_svg"`
	LightColors []string `json:"light_colors"`
	DarkColors  []string `json:"dark_colors"`
}

type appConfigResp struct {
	Version        int    `json:"version"`
	WebURL         string `json:"web_url"`
	PhoneSearchOff int    `json:"phone_search_off"`
	ShortnoEditOff int    `json:"shortno_edit_off"`
	RevokeSecond   int    `json:"revoke_second"`
	AppleSignIn    int    `json:"apple_sign_in"`
}

type appVersionReq struct {
	AppVersion  string `json:"app_version"`  // 版本号
	OS          string `json:"os"`           // 平台 android｜ios
	IsForce     int    `json:"is_force"`     // 是否强制更新
	UpdateDesc  string `json:"update_desc"`  // 更新说明
	DownloadURL string `json:"download_url"` // 下载地址
}

type appVersionResp struct {
	AppVersion  string `json:"app_version"`  // 版本号
	OS          string `json:"os"`           // 平台 android｜ios
	IsForce     int    `json:"is_force"`     // 是否强制更新
	UpdateDesc  string `json:"update_desc"`  // 更新说明
	DownloadURL string `json:"download_url"` // 下载地址
	CreatedAt   string `json:"created_at"`   //更新时间
}

// Country Country
type Country struct {
	Code string `json:"code"`
	Icon string `json:"icon"`
	Name string `json:"name"`
}

var defaultColorsLight = [][]string{
	{"a6B0CDEB", "a69FB0EA", "a6BBEAD5", "a6B2E3DD"},
	{"a640CDDE", "a6AC86ED", "a6E984D8", "a6EFD359"},
	{"a6DBDDBB", "a66BA587", "a6D5D88D", "a688B884"},
	{"a6DAEACB", "a6A2B4FF", "a6ECCBFF", "a6B9E2FF"},
	{"a6B2B1EE", "a6D4A7C9", "a66C8CD4", "a64CA3D4"},
	{"a6DCEB92", "a68FE1D6", "a667A3F2", "a685D685"},
	{"a68ADBF2", "a6888DEC", "a6E39FEA", "a6679CED"},
	{"a6FFC3B2", "a6E2C0FF", "a6FFE7B2", "a6FDFF8C"},
	{"a697BEEB", "a6B1E9EA", "a6C6B1EF", "a6EFB7DC"},
	{"a6E4B2EA", "a68376C2", "a6EAB9D9", "a6B493E6"},
	{"a6D1A3E2", "a6EDD594", "a6E5A1D0", "a6ECD893"},
	{"a6EAA36E", "a6F0E486", "a6F29EBF", "a6E8C06E"},
	{"a67EC289", "a6E4D573", "a6AFD677", "a6F0C07A"},
}
var defaultColorsDark = [][]string{
	{"a6A4DBFF", "a6009FDD", "a6527BDD", "a673B6DD"},
	{"a6FEC496", "a6DD6CB9", "a6962FBF", "a64F5BD5"},
	{"a6E4B2EA", "a68376C2", "a6EAB9D9", "a6B493E6"},
	{"a6EAA36E", "a6F0E486", "a6F29EBF", "a6E8C06E"},
	{"a68ADBF2", "a6888DEC", "a6E39FEA", "a6679CED"},
	{"a6E4B2EA", "a68376C2", "a6EAB9D9", "a6B493E6"},
	{"a627FF03", "a6FC31FF", "a600FEFF", "a6FFFC00"},
	{"a6FEC496", "a6DD6CB9", "a6962FBF", "a64F5BD5"},
	{"a6EAA36E", "a6F0E486", "a6F29EBF", "a6E8C06E"},
	{"a6FAF4D2", "a6CEA668", "a6DDB56D", "a6BAA161"},
	{"a6A4DBFF", "a6009FDD", "a6527BDD", "a673B6DD"},
	{"a6E4B2EA", "a68376C2", "a6EAB9D9", "a6B493E6"},
	{"a6EAA36E", "a6F0E486", "a6F29EBF", "a6E8C06E"},
}

// Countrys Countrys
func Countrys() []*Country {

	return []*Country{
		{
			Code: "0086",
			Icon: "🇨🇳",
			Name: "中国",
		},
		{
			Code: "001",
			Icon: "🇺🇸",
			Name: "美国",
		},
		{
			Code: "00853",
			Icon: "🇲🇴",
			Name: "中国澳门",
		},
		{
			Code: "001",
			Icon: "🇨🇦",
			Name: "加拿大",
		},
		{
			Code: "007",
			Icon: "🇰🇿",
			Name: "哈萨克斯坦",
		},
		{
			Code: "00998",
			Icon: "🇺🇿",
			Name: "乌兹别克斯坦",
		},
		{
			Code: "00996",
			Icon: "🇰🇬",
			Name: "吉尔吉斯斯坦",
		},
		{
			Code: "0090",
			Icon: "🇹🇷",
			Name: "土耳其",
		},
		{
			Code: "0033",
			Icon: "🇫🇷",
			Name: "法国",
		},
		{
			Code: "0049",
			Icon: "🇩🇪",
			Name: "德国",
		},
		{
			Code: "0044",
			Icon: "🇬🇧",
			Name: "英国",
		},
		{
			Code: "0039",
			Icon: "🇮🇹",
			Name: "意大利",
		},
		{
			Code: "00886",
			Icon: "🇹🇼",
			Name: "中国台湾",
		},
		{
			Code: "0060",
			Icon: "🇲🇾",
			Name: "马来西亚",
		},
		{
			Code: "0062",
			Icon: "🇮🇩",
			Name: "印度尼西亚",
		},
		{
			Code: "0061",
			Icon: "🇦🇺",
			Name: "澳大利亚",
		},
		{
			Code: "0064",
			Icon: "🇳🇿",
			Name: "新西兰",
		},
		{
			Code: "0063",
			Icon: "🇵🇭",
			Name: "菲律宾",
		},
		{
			Code: "0065",
			Icon: "🇸🇬",
			Name: "新加坡",
		},
		{
			Code: "0066",
			Icon: "🇹🇭",
			Name: "泰国",
		},
		{
			Code: "00673",
			Icon: "🇧🇳",
			Name: "文莱",
		},
		{
			Code: "0081",
			Icon: "🇯🇵",
			Name: "日本",
		},
		{
			Code: "0082",
			Icon: "🇰🇷",
			Name: "韩国",
		},
		{
			Code: "0084",
			Icon: "🇻🇳",
			Name: "越南",
		},
		{
			Code: "00852",
			Icon: "🇭🇰",
			Name: "中国香港",
		},
		{
			Code: "00855",
			Icon: "🇰🇭",
			Name: "柬埔寨",
		},
		{
			Code: "00856",
			Icon: "🇱🇦",
			Name: "老挝",
		},
		{
			Code: "00880",
			Icon: "🇧🇩",
			Name: "孟加拉国",
		},
		{
			Code: "0091",
			Icon: "🇮🇳",
			Name: "印度",
		},
		{
			Code: "0094",
			Icon: "🇱🇰",
			Name: "斯里兰卡",
		},
		{
			Code: "0095",
			Icon: "🇲🇲",
			Name: "缅甸",
		},
		{
			Code: "00960",
			Icon: "🇲🇻",
			Name: "马尔代夫",
		},
		{
			Code: "00976",
			Icon: "🇲🇳",
			Name: "蒙古",
		},
		{
			Code: "00975",
			Icon: "🇧🇹",
			Name: "不丹",
		},
		{
			Code: "007",
			Icon: "🇷🇺",
			Name: "俄罗斯",
		},
		{
			Code: "0030",
			Icon: "🇬🇷",
			Name: "希腊",
		},
		{
			Code: "0031",
			Icon: "🇳🇱",
			Name: "荷兰",
		},
		{
			Code: "0034",
			Icon: "🇪🇸",
			Name: "西班牙",
		},
		{
			Code: "00351",
			Icon: "🇵🇹",
			Name: "葡萄牙",
		},
		{
			Code: "00353",
			Icon: "🇮🇪",
			Name: "爱尔兰",
		},
		{
			Code: "0041",
			Icon: "🇨🇭",
			Name: "瑞士",
		},
		{
			Code: "0045",
			Icon: "🇩🇰",
			Name: "丹麦",
		},
		{
			Code: "0046",
			Icon: "🇸🇪",
			Name: "瑞典",
		},
		{
			Code: "0047",
			Icon: "🇳🇴",
			Name: "挪威",
		},
		{
			Code: "0055",
			Icon: "🇧🇷",
			Name: "巴西",
		},
	}
}
