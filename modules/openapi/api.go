package openapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/app"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
)

type OpenAPI struct {
	ctx                      *config.Context
	appService               app.IService
	openapiAuthcodePrefix    string
	openapiAccessTokenPrefix string
	userService              user.IService
}

func New(ctx *config.Context) *OpenAPI {

	return &OpenAPI{
		ctx:                      ctx,
		appService:               app.NewService(ctx),
		openapiAuthcodePrefix:    "openapi:authcodePrefix:",
		openapiAccessTokenPrefix: "openapi:accessTokenPrefix:",
		userService:              user.NewService(ctx),
	}
}

// Route 路由配置
func (o *OpenAPI) Route(r *wkhttp.WKHttp) {
	// 不需要认证
	openapinoauth := r.Group("/v1")
	{
		// #################### openapi ####################
		openapinoauth.GET("/openapi/access_token", o.accessTokenGet) // 获取用户的授权access_token
		openapinoauth.GET("/openapi/userinfo", o.userinfoGet)        // 获取用户信息
	}
	// 需要用户认证
	openapi := r.Group("/v1", o.ctx.AuthMiddleware(r))
	{
		// #################### openapi ####################
		openapi.GET("/openapi/authcode", o.authcodeGet) // 获取用户的授权authcode
	}
}

func (o *OpenAPI) accessTokenGet(c *wkhttp.Context) {
	authcode := c.Query("authcode")

	appKey := c.Query("app_key")
	appID, uid, err := o.getOpenapiAuthcodeCache(authcode)
	if err != nil {
		c.ResponseError(err)
		return
	}
	appResp, err := o.appService.GetApp(appID)
	if err != nil {
		c.ResponseError(err)
		return
	}
	if appResp == nil {
		c.ResponseError(fmt.Errorf("appID: %s not found", appID))
		return
	}
	if appResp.Status != app.StatusEnable {
		c.ResponseError(fmt.Errorf("appID: %s status: %s", appID, appResp.Status.String()))
		return
	}
	if appResp.AppKey != appKey {
		c.ResponseError(fmt.Errorf("appKey: %s not match", appKey))
		return
	}
	accessToken := util.GenerUUID()

	second := 24 * 7 * 3600

	err = o.setOpenapiAccessToken(uid, appID, accessToken, time.Second*time.Duration(second))
	if err != nil {
		c.ResponseError(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expire":       second,
	})

}

func (o *OpenAPI) userinfoGet(c *wkhttp.Context) {
	accessToken := c.Query("access_token")

	appID, uid, err := o.getOpenapiAccessToken(accessToken)
	if err != nil {
		c.ResponseError(err)
		return
	}
	if appID == "" || uid == "" {
		c.ResponseError(fmt.Errorf("invalid accessToken: %s", accessToken))
		return
	}
	user, err := o.userService.GetUser(uid)
	if err != nil {
		c.ResponseError(err)
		return
	}
	if user == nil {
		c.ResponseError(fmt.Errorf("user: %s not found", uid))
		return
	}
	avatarURL := fmt.Sprintf("%s/%s", o.ctx.GetConfig().External.APIBaseURL, o.ctx.GetConfig().GetAvatarPath(user.UID))
	c.JSON(http.StatusOK, gin.H{
		"uid":    user.UID,
		"name":   user.Name,
		"avatar": avatarURL,
		"app_id": appID,
	})
}

func (o *OpenAPI) authcodeGet(c *wkhttp.Context) {
	uid := c.GetLoginUID()

	appID := c.Query("app_id")

	authcode := util.GenerUUID()

	err := o.setOpenapiAuthcodeCache(uid, appID, authcode)
	if err != nil {
		c.ResponseError(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authcode": authcode,
	})

}

func (o *OpenAPI) setOpenapiAuthcodeCache(uid, appID, authcode string) error {
	return o.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", o.openapiAuthcodePrefix, authcode), fmt.Sprintf("%s@%s", appID, uid), time.Minute*5)
}

func (o *OpenAPI) getOpenapiAuthcodeCache(authcode string) (string, string, error) {
	appIDAndUID, err := o.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", o.openapiAuthcodePrefix, authcode))
	if err != nil {
		return "", "", err
	}
	appIDAndUIDArr := strings.Split(appIDAndUID, "@")
	if len(appIDAndUIDArr) != 2 {
		return "", "", fmt.Errorf("invalid appIDAndUIDArr: %s", appIDAndUID)
	}
	return appIDAndUIDArr[0], appIDAndUIDArr[1], nil
}

func (o *OpenAPI) setOpenapiAccessToken(uid, appID, accessToken string, expire time.Duration) error {
	return o.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", o.openapiAccessTokenPrefix, accessToken), fmt.Sprintf("%s@%s", appID, uid), expire)
}

func (o *OpenAPI) getOpenapiAccessToken(accessToken string) (string, string, error) {
	appIDAndUID, err := o.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", o.openapiAccessTokenPrefix, accessToken))
	if err != nil {
		return "", "", err
	}
	appIDAndUIDArr := strings.Split(appIDAndUID, "@")
	if len(appIDAndUIDArr) != 2 {
		return "", "", fmt.Errorf("invalid appIDAndUIDArr: %s", appIDAndUID)
	}
	return appIDAndUIDArr[0], appIDAndUIDArr[1], nil
}
