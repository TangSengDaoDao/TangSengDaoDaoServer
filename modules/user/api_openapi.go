package user

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/app"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
)

func (u *User) authcodeGet(c *wkhttp.Context) {
	uid := c.GetLoginUID()

	appID := c.Query("app_id")

	authcode := util.GenerUUID()

	err := u.setOpenapiAuthcodeCache(uid, appID, authcode)
	if err != nil {
		c.ResponseError(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authcode": authcode,
	})

}

func (u *User) accessTokenGet(c *wkhttp.Context) {
	authcode := c.Query("authcode")

	appKey := c.Query("app_key")
	appID, uid, err := u.getOpenapiAuthcodeCache(authcode)
	if err != nil {
		c.ResponseError(err)
		return
	}
	appResp, err := u.appService.GetApp(appID)
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

	err = u.setOpenapiAccessToken(uid, appID, accessToken, time.Second*time.Duration(second))
	if err != nil {
		c.ResponseError(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expire":       second,
	})

}

func (u *User) userinfoGet(c *wkhttp.Context) {
	accessToken := c.Query("access_token")

	appID, uid, err := u.getOpenapiAccessToken(accessToken)
	if err != nil {
		c.ResponseError(err)
		return
	}
	if appID == "" || uid == "" {
		c.ResponseError(fmt.Errorf("invalid accessToken: %s", accessToken))
		return
	}
	user, err := u.userService.GetUser(uid)
	if err != nil {
		c.ResponseError(err)
		return
	}
	if user == nil {
		c.ResponseError(fmt.Errorf("user: %s not found", uid))
		return
	}
	avatarURL := fmt.Sprintf("%s/%s", u.ctx.GetConfig().External.APIBaseURL, u.ctx.GetConfig().GetAvatarPath(user.UID))
	c.JSON(http.StatusOK, gin.H{
		"uid":    user.UID,
		"name":   user.Name,
		"avatar": avatarURL,
		"app_id": appID,
	})
}

func (u *User) setOpenapiAuthcodeCache(uid, appID, authcode string) error {
	return u.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", u.openapiAuthcodePrefix, authcode), fmt.Sprintf("%s@%s", appID, uid), time.Minute*5)
}

func (u *User) getOpenapiAuthcodeCache(authcode string) (string, string, error) {
	appIDAndUID, err := u.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", u.openapiAuthcodePrefix, authcode))
	if err != nil {
		return "", "", err
	}
	appIDAndUIDArr := strings.Split(appIDAndUID, "@")
	if len(appIDAndUIDArr) != 2 {
		return "", "", fmt.Errorf("invalid appIDAndUIDArr: %s", appIDAndUID)
	}
	return appIDAndUIDArr[0], appIDAndUIDArr[1], nil
}

func (u *User) setOpenapiAccessToken(uid, appID, accessToken string, expire time.Duration) error {
	return u.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", u.openapiAccessTokenPrefix, accessToken), fmt.Sprintf("%s@%s", appID, uid), expire)
}

func (u *User) getOpenapiAccessToken(accessToken string) (string, string, error) {
	appIDAndUID, err := u.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", u.openapiAccessTokenPrefix, accessToken))
	if err != nil {
		return "", "", err
	}
	appIDAndUIDArr := strings.Split(appIDAndUID, "@")
	if len(appIDAndUIDArr) != 2 {
		return "", "", fmt.Errorf("invalid appIDAndUIDArr: %s", appIDAndUID)
	}
	return appIDAndUIDArr[0], appIDAndUIDArr[1], nil
}
