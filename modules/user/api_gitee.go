package user

import (
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	ThirdAuthcodePrefix = "thirdlogin:authcode:"
)

func (u *User) thirdAuthcode(c *wkhttp.Context) {
	authcode := util.GenerUUID()
	err := u.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", ThirdAuthcodePrefix, authcode), "1", time.Minute*5)
	if err != nil {
		u.Error("redis set error", zap.Error(err))
		c.ResponseError(errors.New("redis set error"))
		return
	}

	c.Response(gin.H{
		"authcode": authcode,
	})
}

func (u *User) thirdAuthStatus(c *wkhttp.Context) {
	authcode := c.Query("authcode")
	key := fmt.Sprintf("%s%s", ThirdAuthcodePrefix, authcode)
	result, err := u.ctx.GetRedisConn().GetString(key)
	if err != nil {
		u.Error("获取登录状态失败！", zap.Error(err))
		c.ResponseError(errors.New("获取登录状态失败！"))
		return
	}
	if len(result) == 0 {
		c.ResponseError(errors.New("登录状态已过期！"))
		return
	}
	if result == "1" {
		c.Response(gin.H{
			"status": 0, // 等待登录
		})
		return
	}
	if result == "0" {
		c.Response(gin.H{
			"status": 2, // 登录失败
		})
		return
	}

	err = u.ctx.GetRedisConn().Del(key)
	if err != nil {
		u.Error("redis del error", zap.Error(err))
	}

	var loginResp *loginUserDetailResp
	err = util.ReadJsonByByte([]byte(result), &loginResp)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Response(gin.H{
		"status": 1, // 登录成功
		"result": loginResp,
	})
}

// 获取gitee授权地址
func (u *User) gitee(c *wkhttp.Context) {
	cfg := u.ctx.GetConfig()
	authcode := c.Query("authcode")
	redirectURL := fmt.Sprintf("%s%s", cfg.External.APIBaseURL, "/user/oauth/gitee")
	oauthURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s", cfg.Gitee.OAuthURL, cfg.Gitee.ClientID, url.QueryEscape(redirectURL), authcode)
	c.Redirect(http.StatusFound, oauthURL)
}

// giteeOAuth gitee授权
func (u *User) giteeOAuth(c *wkhttp.Context) {
	code := c.Query("code")
	if len(code) == 0 {
		c.ResponseError(errors.New("code不能为空"))
		return
	}
	authcode := c.Query("state")
	accessToken, err := u.requestGiteeAccessToken(code)
	if err != nil {
		c.ResponseError(err)
		return
	}
	userInfo, err := u.requestGiteeUserInfo(accessToken)
	if err != nil {
		c.ResponseError(err)
		return
	}
	if userInfo == nil {
		c.ResponseError(errors.New("获取gitee用户信息失败"))
		return
	}
	userInfoM, err := u.db.queryWithGiteeUID(userInfo.Login)
	if err != nil {
		u.Error("查询gitee用户信息失败！", zap.String("login", userInfo.Login))
		c.ResponseError(errors.New("查询gitee用户信息失败！"))
		return
	}
	loginSpan := u.ctx.Tracer().StartSpan(
		"giteelogin",
		opentracing.ChildOf(c.GetSpanContext()),
	)

	deviceFlag := config.APP
	loginSpanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), loginSpan)
	loginSpan.SetTag("username", userInfo.Login)
	defer loginSpan.Finish()

	var loginResp *loginUserDetailResp
	if userInfoM != nil { // 存在就登录
		if userInfo == nil || userInfoM.IsDestroy == 1 {
			c.ResponseError(errors.New("用户不存在"))
			return
		}
		loginResp, err = u.execLogin(userInfoM, deviceFlag, nil, loginSpanCtx)
		if err != nil {
			c.ResponseError(err)
			return
		}
		// 发送登录消息
		publicIP := util.GetClientPublicIP(c.Request)
		go u.sentWelcomeMsg(publicIP, userInfoM.UID)
	} else {
		// 创建用户
		uid := util.GenerUUID()
		name := userInfo.Name
		if strings.TrimSpace(name) == "" {
			name = userInfo.Login
		}
		var model = &createUserModel{
			UID:      uid,
			Zone:     "",
			Phone:    "",
			Password: "",
			Name:     name,
			GiteeUID: userInfo.Login,
			Flag:     int(deviceFlag.Uint8()),
		}
		if userInfo.AvatarURL != "" && !strings.HasSuffix(userInfo.AvatarURL, "no_portrait.png") {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			imgReader, _ := u.fileService.DownloadImage(userInfo.AvatarURL, timeoutCtx)
			cancel()
			if imgReader != nil {
				avatarID := crc32.ChecksumIEEE([]byte(uid)) % uint32(u.ctx.GetConfig().Avatar.Partition)
				_, err = u.fileService.UploadFile(fmt.Sprintf("avatar/%d/%s.png", avatarID, uid), "image/png", func(w io.Writer) error {
					_, err := io.Copy(w, imgReader)
					return err
				})
				defer imgReader.Close()
				if err == nil {
					model.IsUploadAvatar = 1
				}
			}
		}
		tx, err := u.ctx.DB().Begin()
		defer func() {
			if err := recover(); err != nil {
				tx.Rollback()
				panic(err)
			}
		}()
		if err != nil {
			u.Error("开启事务失败！", zap.Error(err))
			c.ResponseError(errors.New("开启事务失败！"))
			return
		}

		err = u.giteeDB.insertTx(userInfo.toModel(), tx)
		if err != nil {
			tx.Rollback()
			u.Error("插入gitee user失败！", zap.Error(err))
			c.ResponseError(errors.New("插入gitee user失败！"))
			return
		}
		// 发送登录消息
		publicIP := util.GetClientPublicIP(c.Request)
		loginResp, err = u.createUserWithRespAndTx(loginSpanCtx, model, publicIP, tx, func() error {
			err := tx.Commit()
			if err != nil {
				tx.Rollback()
				u.Error("数据库事物提交失败", zap.Error(err))
				c.ResponseError(errors.New("数据库事物提交失败"))
				return nil
			}
			return nil
		})
		if err != nil {
			tx.Rollback()
			c.ResponseError(err)
			return
		}
	}
	var loginRespStr string
	if loginResp != nil {
		loginRespStr = util.ToJson(loginResp)
	} else {
		loginRespStr = "0"
	}
	err = u.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", ThirdAuthcodePrefix, authcode), loginRespStr, time.Minute*1)
	if err != nil {
		u.Error("redis set error", zap.Error(err))
		c.ResponseError(errors.New("redis set error"))
		return
	}
	time.Sleep(time.Second * 3)      // 这里等待2秒，让前端有足够的时间跳转到登录成功页面。
	c.String(http.StatusOK, "登录失败！") // 如果一切正常，理论上是看不到这个返回结果的
}

func (u *User) requestGiteeUserInfo(accessToken string) (*giteeUserInfo, error) {
	userInfo := &giteeUserInfo{}
	resp, err := network.Get(fmt.Sprintf("https://gitee.com/api/v5/user?access_token=%s", accessToken), nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("获取gitee用户信息失败，状态码：%d", resp.StatusCode)
	}
	err = util.ReadJsonByByte([]byte(resp.Body), &userInfo)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

func (u *User) requestGiteeAccessToken(code string) (string, error) {
	cfg := u.ctx.GetConfig()

	result, err := network.PostForWWWForm("https://gitee.com/oauth/token?grant_type=authorization_code", map[string]string{
		"code":          code,
		"client_id":     cfg.Gitee.ClientID,
		"redirect_uri":  fmt.Sprintf("%s%s", cfg.External.APIBaseURL, "/user/oauth/gitee"),
		"client_secret": cfg.Gitee.ClientSecret,
	}, nil)
	if err != nil {
		return "", err
	}
	fmt.Println("getGiteeAccessToken-result-->", result)

	accessToken := ""
	if result["access_token"] != nil {
		accessToken = result["access_token"].(string)
	}

	return accessToken, nil
}

type giteeUserInfo struct {
	AvatarURL         string `json:"avatar_url"`
	Bio               string `json:"bio"`
	Blog              string `json:"blog"`
	CreatedAt         string `json:"created_at"`
	Email             string `json:"email"`
	EventsURL         string `json:"events_url"`
	Followers         int    `json:"followers"`
	FollowersURL      string `json:"followers_url"`
	Following         int    `json:"following"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	HtmlURL           string `json:"html_url"`
	ID                int64  `json:"id"`
	Login             string `json:"login"`
	MemberRole        string `json:"member_role"`
	Name              string `json:"name"`
	OrganizationsURL  string `json:"organizations_url"`
	PublicGists       int    `json:"public_gists"`
	PublicRepos       int    `json:"public_repos"`
	ReceivedEventsURL string `json:"received_events_url"`
	Remark            string `json:"remark"` // 企业备注名
	ReposURL          string `json:"repos_url"`
	Stared            int    `json:"stared"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	Type              string `json:"type"`
	UpdatedAt         string `json:"updated_at"`
	URL               string `json:"url"`
	Watched           int    `json:"watched"`
	Weibo             string `json:"weibo"`
}

func (g *giteeUserInfo) toModel() *gitUserInfoModel {
	m := &gitUserInfoModel{
		Login:             g.Login,
		Name:              g.Name,
		AvatarURL:         g.AvatarURL,
		Bio:               g.Bio,
		Blog:              g.Blog,
		Email:             g.Email,
		Remark:            g.Remark,
		EventsURL:         g.EventsURL,
		Followers:         g.Followers,
		Following:         g.Following,
		GistsURL:          g.GistsURL,
		HtmlURL:           g.HtmlURL,
		MemberRole:        g.MemberRole,
		OrganizationsURL:  g.OrganizationsURL,
		PublicGists:       g.PublicGists,
		PublicRepos:       g.PublicRepos,
		ReceivedEventsURL: g.ReceivedEventsURL,
		ReposURL:          g.ReposURL,
		Stared:            g.Stared,
		StarredURL:        g.StarredURL,
		SubscriptionsURL:  g.SubscriptionsURL,
		Type:              g.Type,
		Weibo:             g.Weibo,
		Watched:           g.Watched,
		GiteeCreatedAt:    g.CreatedAt,
		GiteeUpdatedAt:    g.UpdatedAt,
	}
	m.Id = g.ID

	return m
}
