package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/file"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/source"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/gocraft/dbr/v2"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/app"
	commonapi "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/base/event"
	common2 "github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkevent"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrUserNeedVerification = errors.New("user need verification") // 用户需要验证
)

var qrcodeChanMap = map[string]chan *common.QRCodeModel{}
var qrcodeChanLock sync.RWMutex

// User 用户相关API
type User struct {
	db            *DB
	friendDB      *friendDB
	deviceDB      *deviceDB
	smsServie     commonapi.ISMSService
	fileService   file.IService
	settingDB     *SettingDB
	onlineDB      *onlineDB
	userService   IService
	onlineService *OnlineService
	giteeDB       *giteeDB
	githubDB      *githubDB

	setting *Setting
	log.Log
	ctx                      *config.Context
	userDeviceTokenPrefix    string
	loginUUIDPrefix          string
	openapiAuthcodePrefix    string
	openapiAccessTokenPrefix string
	loginLog                 *LoginLog
	identitieDB              *identitieDB
	onetimePrekeysDB         *onetimePrekeysDB
	maillistDB               *maillistDB
	commonService            common2.IService
	deviceFlagDB             *deviceFlagDB
	deviceFlagsCache         []*deviceFlagModel
	appService               app.IService
}

// New New
func New(ctx *config.Context) *User {
	u := &User{
		ctx:                      ctx,
		db:                       NewDB(ctx),
		deviceDB:                 newDeviceDB(ctx),
		friendDB:                 newFriendDB(ctx),
		smsServie:                commonapi.NewSMSService(ctx),
		settingDB:                NewSettingDB(ctx.DB()),
		setting:                  NewSetting(ctx),
		userDeviceTokenPrefix:    common.UserDeviceTokenPrefix,
		loginUUIDPrefix:          "loginUUID:",
		openapiAuthcodePrefix:    "openapi:authcodePrefix:",
		openapiAccessTokenPrefix: "openapi:accessTokenPrefix:",
		onlineDB:                 newOnlineDB(ctx),
		onlineService:            NewOnlineService(ctx),
		Log:                      log.NewTLog("User"),
		fileService:              file.NewService(ctx),
		userService:              NewService(ctx),
		loginLog:                 NewLoginLog(ctx),
		identitieDB:              newIdentitieDB(ctx),
		onetimePrekeysDB:         newOnetimePrekeysDB(ctx),
		maillistDB:               newMaillistDB(ctx),
		deviceFlagDB:             newDeviceFlagDB(ctx),
		giteeDB:                  newGiteeDB(ctx),
		githubDB:                 newGithubDB(ctx),
		commonService:            common2.NewService(ctx),
		appService:               app.NewService(ctx),
	}
	u.updateSystemUserToken()
	source.SetUserProvider(u)
	return u
}

// Route 路由配置
func (u *User) Route(r *wkhttp.WKHttp) {
	auth := r.Group("/v1", u.ctx.AuthMiddleware(r))
	{

		auth.GET("/users/:uid", u.get) // 根据uid查询用户信息
		// 获取用户的会话信息
		// auth.GET("/users/:uid/conversation", u.userConversationInfoGet)

		auth.POST("/users/:uid/avatar", u.uploadAvatar)              //上传用户头像
		auth.PUT("/users/:uid/setting", u.setting.userSettingUpdate) // 更新用户设置
	}

	user := r.Group("/v1/user", u.ctx.AuthMiddleware(r))
	{
		user.POST("/device_token", u.registerUserDeviceToken)      // 注册用户设备
		user.DELETE("/device_token", u.unregisterUserDeviceToken)  // 卸载用户设备
		user.POST("/device_badge", u.registerUserDeviceBadge)      // 上传设备红点数量
		user.GET("/grant_login", u.grantLogin)                     // 授权登录
		user.PUT("/current", u.userUpdateWithField)                //修改用户信息
		user.GET("/qrcode", u.qrcodeMy)                            // 我的二维码
		user.PUT("/my/setting", u.userUpdateSetting)               // 更新我的设置
		user.POST("/blacklist/:uid", u.addBlacklist)               //添加黑名单
		user.DELETE("/blacklist/:uid", u.removeBlacklist)          //移除黑名单
		user.GET("/blacklists", u.blacklists)                      //黑名单列表
		user.POST("/chatpwd", u.setChatPwd)                        //设置聊天密码
		user.POST("/lockscreenpwd", u.setLockScreenPwd)            //设置锁屏密码
		user.PUT("/lock_after_minute", u.lockScreenAfterMinuteSet) // 设置多久后锁屏
		user.DELETE("/lockscreenpwd", u.closeLockScreenPwd)        //关闭锁屏密码
		user.GET("/customerservices", u.customerservices)          //客服列表
		user.DELETE("/destroy/:code", u.destroyAccount)            // 注销用户
		user.POST("/sms/destroy", u.sendDestroyCode)               //获取注销账号短信验证码
		user.PUT("/updatepassword", u.updatePwd)                   // 修改登录密码
		user.POST("/web3publickey", u.uploadWeb3PublicKey)         // 上传web3公钥
		// #################### 登录设备管理 ####################
		user.GET("/devices", u.deviceList)                 // 用户登录设备
		user.DELETE("/devices/:device_id", u.deviceDelete) // 删除登录设备
		user.GET("/online", u.onlineList)                  // 用户在线列表（我的设备和我的好友）
		user.POST("/online", u.onlinelistWithUIDs)         // 获取指定的uid在线状态
		user.POST("/pc/quit", u.pcQuit)                    // 退出pc登录

		// #################### 用户通讯录 ####################
		user.POST("/maillist", u.addMaillist)
		user.GET("/maillist", u.getMailList)

		// #################### 用户红点 ####################
		user.GET("/reddot/:category", u.getRedDot)      // 获取用户红点
		user.DELETE("/reddot/:category", u.clearRedDot) // 清除红点
	}
	v := r.Group("/v1")
	{

		v.POST("/user/register", u.register)                 //用户注册
		v.POST("/user/login", u.login)                       // 用户登录
		v.POST("/user/usernamelogin", u.usernameLogin)       // 用户名登录
		v.POST("/user/usernameregister", u.usernameRegister) // 用户名注册

		v.POST("/user/pwdforget_web3", u.resetPwdWithWeb3PublicKey) // 通过web3公钥重置密码
		v.GET("/user/web3verifytext", u.getVerifyText)              // 获取验证字符串
		v.POST("/user/web3verifysign", u.web3verifySignature)       // 验证签名
		// v.POST("user/wxlogin", u.wxLogin)
		v.POST("/user/sms/forgetpwd", u.getForgetPwdSMS) //获取忘记密码验证码
		v.POST("/user/pwdforget", u.pwdforget)           //重置登录密码
		v.GET("/user/search", u.search)                  // 搜索用户
		v.GET("/users/:uid/avatar", u.UserAvatar)        // 用户头像
		v.GET("/users/:uid/im", u.userIM)                // 获取用户所在IM节点信息
		v.GET("/user/loginuuid", u.getLoginUUID)         // 获取扫描用的登录uuid
		v.GET("/user/loginstatus", u.getloginStatus)
		v.POST("/user/sms/registercode", u.sendRegisterCode)             //获取注册短信验证码
		v.POST("/user/login_authcode/:auth_code", u.loginWithAuthCode)   // 通过认证码登录
		v.POST("/user/sms/login_check_phone", u.sendLoginCheckPhoneCode) //发送登录设备验证验证码
		v.POST("/user/login/check_phone", u.loginCheckPhone)             //登录验证设备手机号

		// #################### 第三方授权 ####################
		v.GET("/user/thirdlogin/authcode", u.thirdAuthcode)     // 第三方授权码获取
		v.GET("/user/thirdlogin/authstatus", u.thirdAuthStatus) // github认证页面
		// github
		v.GET("/user/github", u.github)            // github认证页面
		v.GET("/user/oauth/github", u.githubOAuth) // github登录
		// gitee
		v.GET("/user/gitee", u.gitee)            // gitee认证页面
		v.GET("/user/oauth/gitee", u.giteeOAuth) // gitee登录

	}

	u.ctx.AddOnlineStatusListener(u.onlineService.listenOnlineStatus) // 监听在线状态
	u.ctx.AddOnlineStatusListener(u.handleOnlineStatus)               // 需要放在listenOnlineStatus之后
	u.ctx.Schedule(time.Minute*5, u.onlineStatusCheck)                // 在线状态定时检查

}

// 清除红点
func (u *User) clearRedDot(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	category := c.Param("category")
	if category == "" {
		c.ResponseError(errors.New("分类不能为空"))
		return
	}
	userRedDot, err := u.db.queryUserRedDot(loginUID, category)
	if err != nil {
		u.Error("查询用户红点错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户红点错误"))
		return
	}
	if userRedDot != nil {
		userRedDot.Count = 0
		err = u.db.updateUserRedDot(userRedDot)
		if err != nil {
			u.Error("修改用户红点错误", zap.Error(err))
			c.ResponseError(errors.New("查询用户红点错误"))
			return
		}
	}
	c.ResponseOK()
}

// 获取用户红点
func (u *User) getRedDot(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	category := c.Param("category")
	if category == "" {
		c.ResponseError(errors.New("分类不能为空"))
		return
	}
	userRedDot, err := u.db.queryUserRedDot(loginUID, UserRedDotCategoryFriendApply)
	if err != nil {
		u.Error("查询用户红点错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户红点错误"))
		return
	}
	count := 0
	isDot := 0
	if userRedDot != nil {
		count = userRedDot.Count
		isDot = userRedDot.IsDot
	}
	c.Response(map[string]interface{}{
		"count":  count,
		"is_dot": isDot,
	})
}

// updateSystemUserToken 更新系统账号token
func (u *User) updateSystemUserToken() {
	_, err := u.ctx.UpdateIMToken(config.UpdateIMTokenReq{
		UID:         u.ctx.GetConfig().Account.SystemUID,
		DeviceFlag:  config.APP,
		DeviceLevel: config.DeviceLevelMaster,
		Token:       util.GenerUUID(),
	})
	if err != nil {
		u.Error("更新IM的token失败！", zap.Error(err))
	}

	_, err = u.ctx.UpdateIMToken(config.UpdateIMTokenReq{
		UID:         u.ctx.GetConfig().Account.FileHelperUID,
		DeviceFlag:  config.APP,
		DeviceLevel: config.DeviceLevelMaster,
		Token:       util.GenerUUID(),
	})
	if err != nil {
		u.Error("更新IM的token失败！", zap.Error(err))
	}

	// 系统管理员
	_, err = u.ctx.UpdateIMToken(config.UpdateIMTokenReq{
		UID:         u.ctx.GetConfig().Account.AdminUID,
		DeviceFlag:  config.APP,
		DeviceLevel: config.DeviceLevelMaster,
		Token:       util.GenerUUID(),
	})
	if err != nil {
		u.Error("更新IM的token失败！", zap.Error(err))
	}

}

// UserAvatar 用户头像
func (u *User) UserAvatar(c *wkhttp.Context) {
	uid := c.Param("uid")
	v := c.Query("v")
	if u.ctx.GetConfig().IsVisitor(uid) {
		c.Header("Content-Type", "image/jpeg")
		avatarBytes, err := ioutil.ReadFile("assets/assets/visitor.png")
		if err != nil {
			u.Error("头像读取失败！", zap.Error(err))
			c.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		c.Writer.Write(avatarBytes)
		return
	}
	if uid == u.ctx.GetConfig().Account.SystemUID {
		c.Header("Content-Type", "image/jpeg")
		avatarBytes, err := ioutil.ReadFile("assets/assets/u_10000.png")
		if err != nil {
			u.Error("系统用户头像读取失败！", zap.Error(err))
			c.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		c.Writer.Write(avatarBytes)
		return
	}
	if uid == u.ctx.GetConfig().Account.FileHelperUID {
		c.Header("Content-Type", "image/jpeg")
		avatarBytes, err := ioutil.ReadFile("assets/assets/fileHelper.jpeg")
		if err != nil {
			u.Error("文件传输助手头像读取失败！", zap.Error(err))
			c.Writer.WriteHeader(http.StatusNotFound)
			return
		}
		c.Writer.Write(avatarBytes)
		return
	}

	userInfo, err := u.db.QueryByUID(uid)
	if err != nil {
		u.Error("查询用户信息错误", zap.Error(err))
		c.Writer.WriteHeader(http.StatusNotFound)
		return
	}
	if userInfo == nil {
		u.Error("用户不存在", zap.Error(err))
		c.Writer.WriteHeader(http.StatusNotFound)
		return
	}
	ph := ""
	fileName := fmt.Sprintf("%s.png", uid)
	downloadUrl := ""
	if userInfo.IsUploadAvatar == 1 {
		avatarID := crc32.ChecksumIEEE([]byte(uid)) % uint32(u.ctx.GetConfig().Avatar.Partition)
		ph = fmt.Sprintf("/avatar/%d/%s.png", avatarID, uid)
	} else {
		//访问默认头像
		avatarID := crc32.ChecksumIEEE([]byte(uid)) % uint32(u.ctx.GetConfig().Avatar.DefaultCount)
		ph = fmt.Sprintf("/avatar/default/test (%d).jpg", avatarID)
		if strings.TrimSpace(u.ctx.GetConfig().Avatar.DefaultBaseURL) != "" {
			downloadUrl = strings.ReplaceAll(u.ctx.GetConfig().Avatar.DefaultBaseURL, "{avatar}", fmt.Sprintf("%d", avatarID))
		}
	}
	if downloadUrl == "" {
		downloadUrl, err = u.fileService.DownloadURL(ph, fileName)
		if err != nil {
			u.Error("获取文件下载地址失败", zap.Error(err))
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

	}

	c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s#%s", downloadUrl, v))

}

// uploadAvatar 上传用户头像
func (u *User) uploadAvatar(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	if c.Request.MultipartForm == nil {
		err := c.Request.ParseMultipartForm(1024 * 1024 * 20) // 20M
		if err != nil {
			u.Error("数据格式不正确！", zap.Error(err))
			c.ResponseError(errors.New("数据格式不正确！"))
			return
		}
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		u.Error("读取文件失败！", zap.Error(err))
		c.ResponseError(errors.New("读取文件失败！"))
		return
	}
	avatarID := crc32.ChecksumIEEE([]byte(loginUID)) % uint32(u.ctx.GetConfig().Avatar.Partition)
	_, err = u.fileService.UploadFile(fmt.Sprintf("avatar/%d/%s.png", avatarID, loginUID), "image/png", func(w io.Writer) error {
		_, err := io.Copy(w, file)
		return err
	})
	defer file.Close()
	if err != nil {
		u.Error("上传文件失败！", zap.Error(err))
		c.ResponseError(errors.New("上传文件失败！"))
		return
	}
	friends, err := u.friendDB.QueryFriends(loginUID)
	if err != nil {
		u.Error("查询用户好友失败")
		return
	}
	if len(friends) > 0 {
		uids := make([]string, 0)
		for _, friend := range friends {
			uids = append(uids, friend.ToUID)
		}
		// 发送头像更新命令
		err = u.ctx.SendCMD(config.MsgCMDReq{
			CMD:         common.CMDUserAvatarUpdate,
			Subscribers: uids,
			Param: map[string]interface{}{
				"uid": loginUID,
			},
		})
		if err != nil {
			u.Error("发送个人头像更新命令失败！")
			return
		}
	}
	//更改用户上传头像状态
	err = u.db.UpdateUsersWithField("is_upload_avatar", "1", loginUID)
	if err != nil {
		u.Error("修改用户是否修改头像错误！", zap.Error(err))
		c.ResponseError(errors.New("修改用户是否修改头像错误！"))
		return
	}
	c.ResponseOK()
}

// 获取用户的IM连接地址
func (u *User) userIM(c *wkhttp.Context) {
	uid := c.Param("uid")
	resp, err := network.Get(fmt.Sprintf("%s/route?uid=%s", u.ctx.GetConfig().WuKongIM.APIURL, uid), nil, nil)
	if err != nil {
		u.Error("调用IM服务失败！", zap.Error(err))
		c.ResponseError(errors.New("调用IM服务失败！"))
		return
	}
	var resultMap map[string]interface{}
	err = util.ReadJsonByByte([]byte(resp.Body), &resultMap)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.JSON(resp.StatusCode, resultMap)
}

func (u *User) qrcodeMy(c *wkhttp.Context) {
	userModel, err := u.db.QueryByUID(c.GetLoginUID())
	if err != nil {
		c.ResponseErrorf("查询当前用户信息失败！", err)
		return
	}
	if userModel == nil {
		c.ResponseError(errors.New("登录用户不存在！"))
		return
	}
	if userModel.QRVercode == "" {
		c.ResponseError(errors.New("用户没有QRVercode，非法操作！"))
		return
	}
	path := strings.ReplaceAll(u.ctx.GetConfig().QRCodeInfoURL, ":code", fmt.Sprintf("vercode_%s", userModel.QRVercode))
	c.Response(gin.H{
		"data": fmt.Sprintf("%s/%s", u.ctx.GetConfig().External.BaseURL, path),
	})
}

// 修改用户信息
func (u *User) userUpdateWithField(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()

	var reqMap map[string]interface{}
	if err := c.BindJSON(&reqMap); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	// 查询用户信息
	users, err := u.db.QueryByUID(loginUID)
	if err != nil {
		c.ResponseError(errors.New("查询用户信息出错！"))
		return
	}
	if users == nil {
		c.ResponseError(errors.New("用户信息不存在！"))
		return
	}

	for key, value := range reqMap {
		//是否允许更新此field
		if !allowUpdateUserField(key) {
			c.ResponseError(errors.New("不允许更新【" + key + "】"))
			return
		}
		if key == "short_no" {
			if u.ctx.GetConfig().ShortNo.EditOff {
				c.ResponseError(errors.New("不允许编辑！"))
				return
			}
			if users.ShortStatus == 1 {
				c.ResponseError(errors.New("用户短编号只能修改一次"))
				return
			}
			if len(fmt.Sprintf("%s", value)) < 6 || len(fmt.Sprintf("%s", value)) > 20 {
				c.ResponseError(errors.New("短号须以字母开头，仅支持使用6～20个字母、数字、下划线、减号自由组合"))
				return
			}
			isLetter := true
			isIncludeNum := false
			for index, r := range fmt.Sprintf("%s", value) {
				if !unicode.IsLetter(r) && index == 0 {
					isLetter = false
					break
				}
				if unicode.Is(unicode.Han, r) {
					isLetter = false
					break
				}
				if unicode.IsDigit(r) {
					isIncludeNum = true
				}
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
					isLetter = false
					break
				}
			}
			if !isLetter || !isIncludeNum {
				c.ResponseError(errors.New("短号须以字母开头，仅支持使用6～20个字母、数字、下划线、减号自由组合"))
				return
			}
			users, err = u.db.QueryUserWithOnlyShortNo(fmt.Sprintf("%s", value))
			if err != nil {
				u.Error("通过short_no查询用户失败！", zap.Error(err), zap.String("shortNo", key))
				c.ResponseError(errors.New("通过short_no查询用户失败！"))
				return
			}
			if users != nil {
				c.ResponseError(errors.New("已存在，请换一个！"))
				return
			}

			tx, _ := u.db.session.Begin()
			defer func() {
				if err := recover(); err != nil {
					tx.Rollback()
					panic(err)
				}
			}()
			err = u.db.UpdateUsersWithField(key, fmt.Sprintf("%s", value), loginUID)
			if err != nil {
				c.ResponseError(errors.New("修改用户资料失败"))
				tx.Rollback()
				return
			}
			err = u.db.UpdateUsersWithField("short_status", "1", loginUID)
			if err != nil {
				u.Error("修改用户资料失败", zap.Error(err), zap.Any(key, value))
				c.ResponseError(errors.New("修改用户资料失败"))
				tx.Rollback()
				return
			}
			err = tx.Commit()
			if err != nil {
				u.Error("数据库事物提交失败", zap.Error(err))
				c.ResponseError(errors.New("数据库事物提交失败"))
				tx.Rollback()
				return
			}
			c.ResponseOK()
			return
		}
		//修改用户信息
		if key == "name" && value != nil && value.(string) == "" { // 修改名字
			c.ResponseError(errors.New("名字不能为空！"))
			return
		}

		err = u.db.UpdateUsersWithField(key, fmt.Sprintf("%s", value), loginUID)
		if err != nil {
			u.Error("修改用户资料失败", zap.Error(err))
			c.ResponseError(errors.New("修改用户资料失败"))
			return
		}
		if key == "name" {
			// 将重新设置token设置到缓存（这里主要是更新登录者的name）
			err = u.ctx.Cache().Set(u.ctx.GetConfig().Cache.TokenCachePrefix+c.GetHeader("token"), fmt.Sprintf("%s@%s@%s", loginUID, value, c.GetLoginRole()))
			if err != nil {
				u.Error("重新设置token缓存失败！", zap.Error(err))
				c.ResponseError(errors.New("重新设置token缓存失败！"))
				return
			}
		}
	}
	// 发送频道刚刚消息给登录好友
	friends, err := u.friendDB.QueryFriends(loginUID)
	if err != nil {
		u.Error("查询用户好友错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户好友错误"))
		return
	}
	if len(friends) > 0 {
		uids := make([]string, 0)
		for _, friend := range friends {
			uids = append(uids, friend.ToUID)
		}
		err = u.ctx.SendCMD(config.MsgCMDReq{
			CMD:         common.CMDChannelUpdate,
			ChannelID:   loginUID,
			ChannelType: common.ChannelTypePerson.Uint8(),
			Subscribers: uids,
			Param: map[string]interface{}{
				"channel_id":   loginUID,
				"channel_type": common.ChannelTypePerson,
			},
		})
		if err != nil {
			u.Error("发送频道更改消息错误！", zap.Error(err))
			c.ResponseError(errors.New("发送频道更改消息错误！"))
			return
		}
	}

	c.ResponseOK()
}

func (u *User) userUpdateSetting(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()

	var reqMap map[string]interface{}
	if err := c.BindJSON(&reqMap); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	// 查询用户信息
	users, err := u.db.QueryByUID(loginUID)
	if err != nil {
		c.ResponseError(errors.New("查询用户信息出错！"))
		return
	}
	if users == nil {
		c.ResponseError(errors.New("用户信息不存在！"))
		return
	}

	for key, value := range reqMap {
		if key == "device_lock" ||
			key == "search_by_phone" ||
			key == "search_by_short" ||
			key == "new_msg_notice" ||
			key == "msg_show_detail" ||
			key == "offline_protection" ||
			key == "voice_on" ||
			key == "shock_on" ||
			key == "mute_of_app" {
			err = u.db.UpdateUsersWithField(key, fmt.Sprintf("%v", value), loginUID)
			if err != nil {
				u.Error("修改用户资料失败", zap.Error(err))
				c.ResponseError(errors.New("修改用户资料失败"))
				return
			}
			c.ResponseOK()
		}
	}
}

// 获取用户详情
func (u *User) get(c *wkhttp.Context) {
	uid := c.Param("uid")
	loginUID := c.MustGet("uid").(string)

	if u.ctx.GetConfig().IsVisitor(uid) { // 访客频道
		c.Request.URL.Path = fmt.Sprintf("/v1/hotline/visitors/%s/im", uid)
		u.ctx.GetHttpRoute().HandleContext(c)
		return
	}

	userDetailResp, err := u.userService.GetUserDetail(uid, loginUID)
	if err != nil {
		u.Error("获取用户详情失败！", zap.Error(err))
		c.ResponseError(errors.New("获取用户详情失败！"))
		return
	}
	if userDetailResp == nil {
		c.ResponseError(errors.New("用户不存在！"))
		return
	}
	c.Response(userDetailResp)
}

//	获取用户详情
//
//	func (u *User) userConversationInfoGet(c *wkhttp.Context) {
//		uid := c.Param("uid")
//		loginUID := c.MustGet("uid").(string)
//		model, err := u.db.QueryDetailByUID(uid, loginUID)
//		if err != nil {
//			u.Error("查询用户信息失败！", zap.Error(err), zap.String("uid", uid))
//			c.ResponseError(errors.New("查询用户信息失败！"))
//			return
//		}
//		if model == nil {
//			c.ResponseError(errors.New("用户信息不存在！"))
//			return
//		}
//		userDetailResp := newUserDetailResp(model)
//		if uid == loginUID {
//			userDetailResp.Name = u.ctx.GetConfig().FileHelperName
//		}
//		c.Response(userDetailResp)
//	}
//
// 微信登录
func (u *User) wxLogin(c *wkhttp.Context) {
	type wxLoginReq struct {
		Code   string     `json:"code"`
		Flag   int        `json:"flag"`
		Device *deviceReq `json:"device"`
	}
	var req wxLoginReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if req.Code == "" {
		c.ResponseError(errors.New("微信code不能为空"))
		return
	}
	accessTokenResp, err := network.Get("https://api.weixin.qq.com/sns/oauth2/access_token", map[string]string{
		"appid":      u.ctx.GetConfig().Wechat.AppID,
		"secret":     u.ctx.GetConfig().Wechat.AppSecret,
		"code":       req.Code,
		"grant_type": "authorization_code",
	}, nil)
	if err != nil {
		u.Error("获取微信access_token错误", zap.Error(err))
		c.ResponseError(errors.New("获取微信access_token错误"))
		return
	}
	if accessTokenResp.StatusCode != http.StatusOK {
		c.ResponseErrorf("请求验证微信access_token错误", fmt.Errorf("错误代码-> %d", accessTokenResp.StatusCode))
		return
	}
	var bodyMap map[string]interface{}
	if err = util.ReadJsonByByte([]byte(accessTokenResp.Body), &bodyMap); err != nil {
		c.ResponseErrorf("解码微信access_token返回数据失败！", err)
		return
	}
	var accessToken = bodyMap["access_token"].(string)
	var openid = bodyMap["openid"].(string)
	wxUserInfoResp, err := network.Get("https://api.weixin.qq.com/sns/userinfo", map[string]string{
		"access_token": accessToken,
		"openid":       openid,
	}, nil)
	if err != nil {
		u.Error("获取微信用户资料错误", zap.Error(err))
		c.ResponseError(errors.New("获取微信用户资料错误"))
		return
	}

	if wxUserInfoResp.StatusCode != http.StatusOK {
		c.ResponseErrorf("获取微信用户资料请求错误", fmt.Errorf("错误代码-> %d", wxUserInfoResp.StatusCode))
		return
	}

	var wxUserInfoBodyMap map[string]interface{}
	if err = util.ReadJsonByByte([]byte(wxUserInfoResp.Body), &wxUserInfoBodyMap); err != nil {
		c.ResponseErrorf("解码微信用户信息返回数据失败！", err)
		return
	}

	var unionid = wxUserInfoBodyMap["unionid"].(string)
	var nickname = wxUserInfoBodyMap["nickname"].(string)
	sex, _ := wxUserInfoBodyMap["sex"].(json.Number).Int64()
	var headimgurl = wxUserInfoBodyMap["headimgurl"].(string)
	// 验证该用户是否存在
	loginSpan := u.ctx.Tracer().StartSpan(
		"login",
		opentracing.ChildOf(c.GetSpanContext()),
	)
	loginSpanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), loginSpan)
	loginSpan.SetTag("username", nickname)
	defer loginSpan.Finish()

	userInfo, err := u.db.queryWithWXOpenIDAndWxUnionidCtx(loginSpanCtx, openid, unionid)
	if err != nil {
		u.Error("通过微信openid查询用户是否存在错误", zap.Error(err))
		c.ResponseError(errors.New("通过微信openid查询用户是否存在错误"))
		return
	}
	if userInfo != nil {
		if userInfo == nil || userInfo.IsDestroy == 1 {
			c.ResponseError(errors.New("用户不存在"))
			return
		}
		u.execLoginAndRespose(userInfo, config.DeviceFlag(req.Flag), req.Device, loginSpanCtx, c)
	} else {
		// 创建用户
		uid := util.GenerUUID()
		var model = &createUserModel{
			UID:       uid,
			Zone:      "",
			Phone:     "",
			Password:  "",
			Sex:       int(sex),
			Name:      nickname,
			WXOpenid:  openid,
			WXUnionid: unionid,
			Flag:      req.Flag,
			Device:    req.Device,
		}
		// 下载微信用户头像并上传
		if headimgurl != "" {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			imgReader, _ := u.fileService.DownloadImage(headimgurl, timeoutCtx)
			cancel()
			if imgReader != nil {
				avatarID := crc32.ChecksumIEEE([]byte(uid)) % uint32(u.ctx.GetConfig().Avatar.Partition)
				_, err = u.fileService.UploadFile(fmt.Sprintf("avatar/%d/%s.png", avatarID, uid), "image/png", func(w io.Writer) error {
					_, err := io.Copy(w, imgReader)
					return err
				})
				defer imgReader.Close()
				if err == nil {
					// u.Error("上传文件失败！", zap.Error(err))
					// c.ResponseError(errors.New("上传文件失败！"))
					// return
					model.IsUploadAvatar = 1
				}
			}
		}
		u.createUser(loginSpanCtx, model, c)
	}
}

// 登录
func (u *User) login(c *wkhttp.Context) {

	var req loginReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	loginSpan := u.ctx.Tracer().StartSpan(
		"login",
		opentracing.ChildOf(c.GetSpanContext()),
	)
	loginSpanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), loginSpan)
	loginSpan.SetTag("username", req.Username)
	defer loginSpan.Finish()

	userInfo, err := u.db.QueryByUsernameCxt(loginSpanCtx, req.Username)
	if err != nil {
		u.Error("查询用户信息失败！", zap.String("username", req.Username))
		c.ResponseError(err)
		return
	}
	if userInfo == nil || userInfo.IsDestroy == 1 {
		c.ResponseError(errors.New("用户不存在"))
		return
	}
	if userInfo.Password == "" {
		c.ResponseError(errors.New("此账号不允许登录"))
		return
	}
	if util.MD5(util.MD5(req.Password)) != userInfo.Password {
		c.ResponseError(errors.New("密码不正确！"))
		return
	}
	u.execLoginAndRespose(userInfo, config.DeviceFlag(req.Flag), req.Device, loginSpanCtx, c)
}

// 验证登录用户信息
func (u *User) execLoginAndRespose(userInfo *Model, flag config.DeviceFlag, device *deviceReq, loginSpanCtx context.Context, c *wkhttp.Context) {

	result, err := u.execLogin(userInfo, flag, device, loginSpanCtx)
	if err != nil {
		if errors.Is(err, ErrUserNeedVerification) {
			phone := ""
			if len(userInfo.Phone) > 5 {
				phone = fmt.Sprintf("%s******%s", userInfo.Phone[0:3], userInfo.Phone[len(userInfo.Phone)-2:])
			}
			c.ResponseWithStatus(http.StatusBadRequest, map[string]interface{}{
				"status": 110,
				"msg":    "需要验证手机号码！",
				"uid":    userInfo.UID,
				"phone":  phone,
			})
			return
		}
		c.ResponseError(err)
		return
	}

	c.Response(result)

	publicIP := util.GetClientPublicIP(c.Request)
	go u.sentWelcomeMsg(publicIP, userInfo.UID)
}

func (u *User) execLogin(userInfo *Model, flag config.DeviceFlag, device *deviceReq, loginSpanCtx context.Context) (*loginUserDetailResp, error) {
	if userInfo.Status == int(common.UserDisable) {
		return nil, errors.New("该用户已被禁用")
	}
	deviceLevel := config.DeviceLevelSlave
	if flag == config.APP {
		deviceLevel = config.DeviceLevelMaster
	}
	//app登录验证设备锁
	if flag == 0 && userInfo.DeviceLock == 1 {
		if device == nil {
			return nil, errors.New("登录设备信息不能为空！")
		}
		var existDevice bool
		var err error
		if device != nil {
			existDevice, err = u.deviceDB.existDeviceWithDeviceIDAndUIDCtx(loginSpanCtx, device.DeviceID, userInfo.UID)
			if err != nil {
				u.Error("查询是否存在的设备失败", zap.Error(err))
				return nil, errors.New("查询是否存在的设备失败")
			}
			if existDevice {
				err = u.deviceDB.updateDeviceLastLoginCtx(loginSpanCtx, time.Now().Unix(), device.DeviceID, userInfo.UID)
				if err != nil {
					u.Error("更新用户登录设备失败", zap.Error(err))
					return nil, errors.New("更新用户登录设备失败")
				}
			}
		}
		if !existDevice {
			err := u.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", u.ctx.GetConfig().Cache.LoginDeviceCachePrefix, userInfo.UID), util.ToJson(device), u.ctx.GetConfig().Cache.LoginDeviceCacheExpire)
			if err != nil {
				u.Error("缓存登录设备失败！", zap.Error(err))
				return nil, errors.New("缓存登录设备失败！")
			}
			return nil, ErrUserNeedVerification
		}
	}
	//更新最后一次登录设备信息
	if flag == config.APP && device != nil {
		err := u.deviceDB.insertOrUpdateDeviceCtx(loginSpanCtx, &deviceModel{
			UID:         userInfo.UID,
			DeviceID:    device.DeviceID,
			DeviceName:  device.DeviceName,
			DeviceModel: device.DeviceModel,
			LastLogin:   time.Now().Unix(),
		})
		if err != nil {
			u.Error("更新用户登录设备失败", zap.Error(err))
			return nil, errors.New("更新用户登录设备失败")
		}
	}
	token := util.GenerUUID()
	// 将token设置到缓存
	tokenSpan, _ := u.ctx.Tracer().StartSpanFromContext(loginSpanCtx, "SetAndExpire")
	tokenSpan.SetTag("key", "token")
	// 获取老的token并清除老token数据
	oldToken, err := u.ctx.Cache().Get(fmt.Sprintf("%s%d%s", u.ctx.GetConfig().Cache.UIDTokenCachePrefix, flag, userInfo.UID))
	if err != nil {
		u.Error("获取旧token错误", zap.Error(err))
		tokenSpan.Finish()
		return nil, errors.New("获取旧token错误")
	}
	if flag == config.APP {
		if oldToken != "" {
			err = u.ctx.Cache().Delete(u.ctx.GetConfig().Cache.TokenCachePrefix + oldToken)
			if err != nil {
				u.Error("清除旧token数据错误", zap.Error(err))
				tokenSpan.Finish()
				return nil, errors.New("清除旧token数据错误")
			}
		}
	} else { // PC暂时不执行删除操作，因为PC可以同时登陆
		if strings.TrimSpace(oldToken) != "" { // 如果是web或pc类设备 因为支持多登所以这里依然使用老token
			token = oldToken
		}
	}

	err = u.ctx.Cache().SetAndExpire(u.ctx.GetConfig().Cache.TokenCachePrefix+token, fmt.Sprintf("%s@%s@%s", userInfo.UID, userInfo.Name, userInfo.Role), u.ctx.GetConfig().Cache.TokenExpire)
	if err != nil {
		u.Error("设置token缓存失败！", zap.Error(err))
		tokenSpan.Finish()
		return nil, errors.New("设置token缓存失败！")
	}
	err = u.ctx.Cache().SetAndExpire(fmt.Sprintf("%s%d%s", u.ctx.GetConfig().Cache.UIDTokenCachePrefix, flag, userInfo.UID), token, u.ctx.GetConfig().Cache.TokenExpire)
	if err != nil {
		u.Error("设置uidtoken缓存失败！", zap.Error(err))
		tokenSpan.Finish()
		return nil, errors.New("设置uidtoken缓存失败！")
	}
	tokenSpan.Finish()

	updateTokenSpan, _ := u.ctx.Tracer().StartSpanFromContext(loginSpanCtx, "UpdateIMToken")

	imTokenReq := config.UpdateIMTokenReq{
		UID:         userInfo.UID,
		Token:       token,
		DeviceFlag:  config.DeviceFlag(flag),
		DeviceLevel: deviceLevel,
	}
	imResp, err := u.ctx.UpdateIMToken(imTokenReq)
	if err != nil {
		u.Error("更新IM的token失败！", zap.Error(err))
		updateTokenSpan.SetTag("err", err)
		updateTokenSpan.Finish()
		return nil, errors.New("更新IM的token失败！")
	}
	updateTokenSpan.Finish()

	if imResp.Status == config.UpdateTokenStatusBan {
		return nil, errors.New("此账号已经被封禁！")
	}

	return newLoginUserDetailResp(userInfo, token, u.ctx), nil
}

// sendWelcomeMsg 发送欢迎语
func (u *User) sentWelcomeMsg(publicIP, uid string) {
	time.Sleep(time.Second * 2)
	//发送登录欢迎消息
	lastLoginLog := u.loginLog.getLastLoginIP(uid)
	content := u.ctx.GetConfig().WelcomeMessage
	var sentContent string
	appconfig, err := u.commonService.GetAppConfig()
	if err != nil {
		u.Error("获取应用配置错误", zap.Error(err))
	}
	if appconfig != nil && appconfig.WelcomeMessage != "" {
		content = appconfig.WelcomeMessage
	}
	if lastLoginLog != nil {
		ipStr := fmt.Sprintf("上次的登录信息：%s %s\n本次登录的信息：%s %s", lastLoginLog.LoginIP, lastLoginLog.CreateAt, publicIP, util.ToyyyyMMddHHmmss(time.Now()))
		sentContent = fmt.Sprintf("%s\n%s", content, ipStr)
	} else {
		ipStr := fmt.Sprintf("本次登录的信息：%s %s", publicIP, util.ToyyyyMMddHHmmss(time.Now()))
		sentContent = fmt.Sprintf("%s\n%s", content, ipStr)
	}
	err = u.ctx.SendMessage(&config.MsgSendReq{
		FromUID:     u.ctx.GetConfig().Account.SystemUID,
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"content": sentContent,
			"type":    common.Text,
		})),
		Header: config.MsgHeader{
			RedDot: 1,
		},
	})
	if err != nil {
		u.Error("发送登录消息欢迎消息失败", zap.Error(err))
	}
	//保存登录日志
	u.loginLog.add(uid, publicIP)
}

// 注册
func (u *User) register(c *wkhttp.Context) {
	var req registerReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if err := req.CheckRegister(); err != nil {
		c.ResponseError(err)
		return
	}

	if u.ctx.GetConfig().Register.Off {
		c.ResponseError(errors.New("注册通道暂不开放"))
		return
	}

	registerSpan := u.ctx.Tracer().StartSpan(
		"user.register",
		opentracing.ChildOf(c.GetSpanContext()),
	)
	defer registerSpan.Finish()
	registerSpanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), registerSpan)

	registerSpan.SetTag("username", fmt.Sprintf("%s%s", req.Zone, req.Phone))
	//验证手机号是否注册
	userInfo, err := u.db.QueryByUsernameCxt(registerSpanCtx, fmt.Sprintf("%s%s", req.Zone, req.Phone))
	if err != nil {
		u.Error("查询用户信息失败！", zap.String("username", req.Phone))
		c.ResponseError(err)
		return
	}
	if userInfo != nil {
		c.ResponseError(errors.New("该用户已存在"))
		return
	}
	//测试模式
	if strings.TrimSpace(u.ctx.GetConfig().SMSCode) != "" {
		if strings.TrimSpace(u.ctx.GetConfig().SMSCode) != req.Code {
			c.ResponseError(errors.New("验证码错误"))
			return
		}
	} else {
		//线上验证短信验证码
		err = u.smsServie.Verify(registerSpanCtx, req.Zone, req.Phone, req.Code, commonapi.CodeTypeRegister)
		if err != nil {
			c.ResponseError(err)
			return
		}
	}
	uid := util.GenerUUID()
	var model = &createUserModel{
		UID:      uid,
		Sex:      1,
		Name:     req.Name,
		Zone:     req.Zone,
		Phone:    req.Phone,
		Password: req.Password,
		Flag:     int(req.Flag),
		Device:   req.Device,
	}
	u.createUser(registerSpanCtx, model, c)
}

// 搜索用户
func (u *User) search(c *wkhttp.Context) {
	keyword := c.Query("keyword")
	useModel, err := u.db.QueryByKeyword(keyword)
	if err != nil {
		u.Error("查询用户信息失败！", zap.Error(err), zap.String("keyword", keyword))
		c.ResponseError(errors.New("查询用户信息失败！"))
		return
	}
	if useModel == nil {
		c.JSON(http.StatusOK, gin.H{
			"exist": 0,
		})
		return
	}
	appconfig, _ := u.commonService.GetAppConfig()

	if keyword == useModel.Phone {
		//关闭了手机号搜索
		if useModel.SearchByPhone == 0 || (appconfig != nil && appconfig.SearchByPhone == 0) || u.ctx.GetConfig().PhoneSearchOff {
			c.JSON(http.StatusOK, gin.H{
				"exist": 0,
			})
			return
		}
	}

	if useModel.SearchByShort == 0 {
		//关闭了短编号搜索
		if keyword != useModel.ShortNo {
			c.JSON(http.StatusOK, gin.H{
				"exist": 0,
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"exist": 1,
		"data":  newUserResp(useModel),
	})
}

// 注册用户设备token
func (u *User) registerUserDeviceToken(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	var req struct {
		DeviceToken string `json:"device_token"` // 设备token
		DeviceType  string `json:"device_type"`  // 设备类型 IOS，MI，HMS
		BundleID    string `json:"bundle_id"`    // app的唯一ID标示
	}
	if err := c.BindJSON(&req); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if strings.TrimSpace(req.DeviceToken) == "" {
		c.ResponseError(errors.New("设备token不能为空！"))
		return
	}
	if strings.TrimSpace(req.DeviceType) == "" {
		c.ResponseError(errors.New("设备类型不能为空！"))
		return
	}
	if strings.TrimSpace(req.BundleID) == "" {
		c.ResponseError(errors.New("bundleID不能为空！"))
		return
	}
	err := u.ctx.GetRedisConn().Hmset(fmt.Sprintf("%s%s", u.userDeviceTokenPrefix, loginUID), "device_type", req.DeviceType, "device_token", req.DeviceToken, "bundle_id", req.BundleID)
	if err != nil {
		u.Error("存储用户设备token失败！", zap.Error(err))
		c.ResponseError(errors.New("存储用户设备token失败！"))
		return
	}
	c.ResponseOK()
}

// 注册用户设备红点数量
func (u *User) registerUserDeviceBadge(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	var req struct {
		Badge int `json:"badge"` // 设备红点数量
	}
	if err := c.BindJSON(&req); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	err := u.setUserBadge(loginUID, int64(req.Badge))
	if err != nil {
		u.Error("存储用户红点失败！", zap.Error(err))
		c.ResponseError(errors.New("存储用户红点失败！"))
		return
	}
	c.ResponseOK()
}

func (u *User) setUserBadge(uid string, badge int64) error {
	err := u.ctx.GetRedisConn().Hset(common.UserDeviceBadgePrefix, uid, fmt.Sprintf("%d", badge))
	if err != nil {
		return err
	}
	return nil
}

// 卸载注册设备token
func (u *User) unregisterUserDeviceToken(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)

	err := u.ctx.GetRedisConn().Del(fmt.Sprintf("%s%s", u.userDeviceTokenPrefix, loginUID))
	if err != nil {
		u.Error("删除设备token失败！", zap.Error(err))
		c.ResponseError(errors.New("删除设备token失败！"))
		return
	}
	c.ResponseOK()
}

// 获取登录的uuid（web登录）
func (u *User) getLoginUUID(c *wkhttp.Context) {
	uuid := util.GenerUUID()
	err := u.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", common.QRCodeCachePrefix, uuid), util.ToJson(common.NewQRCodeModel(common.QRCodeTypeScanLogin, map[string]interface{}{
		"app_id":  "wukongchat",
		"status":  common.ScanLoginStatusWaitScan,
		"pub_key": c.Query("pub_key"),
	})), time.Minute*1)
	if err != nil {
		u.Error("设置登录uuid失败！", zap.Error(err))
		c.ResponseError(errors.New("设置登录uuid失败！"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"uuid":   uuid,
		"qrcode": fmt.Sprintf("%s/%s", u.ctx.GetConfig().External.BaseURL, strings.ReplaceAll(u.ctx.GetConfig().QRCodeInfoURL, ":code", uuid)),
	})
}

// 通过loginUUID获取登录状态
func (u *User) getloginStatus(c *wkhttp.Context) {
	uuid := c.Query("uuid")
	qrcodeInfo, err := u.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", common.QRCodeCachePrefix, uuid))
	if err != nil {
		u.Error("获取uuid绑定的二维码信息失败！", zap.Error(err))
		c.ResponseError(errors.New("获取uuid绑定的二维码信息失败！"))
		return
	}
	if qrcodeInfo == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": common.ScanLoginStatusExpired,
		})
		return
	}
	var qrcodeModel *common.QRCodeModel
	err = util.ReadJsonByByte([]byte(qrcodeInfo), &qrcodeModel)
	if err != nil {
		u.Error("解码二维码信息失败！", zap.Error(err))
		c.ResponseError(errors.New("解码二维码信息失败！"))
		return
	}
	if qrcodeModel == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": common.ScanLoginStatusExpired,
		})
		return
	}
	qrcodeChan := u.getQRCodeModelChan(uuid)
	select {
	case qrcodeModel := <-qrcodeChan:
		u.removeQRCodeChan(uuid)
		if qrcodeModel == nil {
			break
		}
		c.JSON(http.StatusOK, qrcodeModel.Data)
		break
	case <-time.After(10 * time.Second):
		u.removeQRCodeChan(uuid)
		c.JSON(http.StatusOK, qrcodeModel.Data)
		break

	}
}

// 通过authCode登录
func (u *User) loginWithAuthCode(c *wkhttp.Context) {
	authCode := c.Param("auth_code")
	authCodeKey := fmt.Sprintf("%s%s", common.AuthCodeCachePrefix, authCode)
	flagI64, _ := strconv.ParseInt(c.Query("flag"), 10, 64)
	var flag config.DeviceFlag
	if flagI64 == 0 {
		flag = config.Web // loginWithAuthCode 默认为web登陆
	} else {
		flag = config.DeviceFlag(flag)
	}
	authInfo, err := u.ctx.GetRedisConn().GetString(authCodeKey)
	if err != nil {
		u.Error("获取授权信息失败！", zap.Error(err))
		c.ResponseError(errors.New("获取授权信息失败！"))
		return
	}
	if authInfo == "" {
		c.ResponseError(errors.New("授权码失效或不存在！"))
		return
	}
	var authInfoMap map[string]interface{}
	err = util.ReadJsonByByte([]byte(authInfo), &authInfoMap)
	if err != nil {
		u.Error("解码授权信息失败！", zap.Error(err))
		c.ResponseError(errors.New("解码授权信息失败！"))
		return
	}
	authType := authInfoMap["type"].(string)
	if authType != string(common.AuthCodeTypeScanLogin) {
		c.ResponseError(errors.New("授权码不是登录授权码！"))
		return
	}
	scaner := authInfoMap["scaner"].(string)
	// 获取老的token
	token, err := u.ctx.Cache().Get(fmt.Sprintf("%s%d%s", u.ctx.GetConfig().Cache.UIDTokenCachePrefix, flag, scaner))
	if err != nil {
		u.Error("获取旧token错误", zap.Error(err))
		c.ResponseError(errors.New("获取旧token错误"))
		return
	}
	if strings.TrimSpace(token) == "" {
		token = util.GenerUUID()
	}

	userModel, err := u.db.QueryByUID(scaner)
	if err != nil {
		u.Error("用户不存在！", zap.String("uid", scaner), zap.Error(err))
		c.ResponseError(errors.New("用户不存在！"))
		return
	}

	imResp, err := u.ctx.UpdateIMToken(config.UpdateIMTokenReq{
		UID:         scaner,
		Token:       token,
		DeviceFlag:  flag,
		DeviceLevel: config.DeviceLevelSlave,
	})
	if err != nil {
		u.Error("更新IM的token失败！", zap.Error(err))
		c.ResponseError(errors.New("更新IM的token失败！"))
		return
	}
	if imResp.Status == config.UpdateTokenStatusBan {
		c.ResponseError(errors.New("此账号已经被封禁！"))
		return
	}

	// 将token设置到缓存
	err = u.ctx.Cache().SetAndExpire(u.ctx.GetConfig().Cache.TokenCachePrefix+token, fmt.Sprintf("%s@%s", userModel.UID, userModel.Name), u.ctx.GetConfig().Cache.TokenExpire)
	if err != nil {
		u.Error("设置token缓存失败！", zap.Error(err))
		c.ResponseError(errors.New("设置token缓存失败！"))
		return
	}
	err = u.ctx.GetRedisConn().Del(authCodeKey)
	if err != nil {
		u.Error("删除授权码失败！", zap.Error(err))
		c.ResponseError(errors.New("删除授权码失败！"))
		return
	}

	err = u.ctx.Cache().SetAndExpire(fmt.Sprintf("%s%d%s", u.ctx.GetConfig().Cache.UIDTokenCachePrefix, flag, userModel.UID), token, u.ctx.GetConfig().Cache.TokenExpire)
	if err != nil {
		u.Error("设置uidtoken缓存失败！", zap.Error(err))
		c.ResponseError(errors.New("设置uidtoken缓存失败！"))
		return
	}

	c.Response(map[string]interface{}{
		"app_id":     userModel.AppID,
		"name":       userModel.Name,
		"username":   userModel.Username,
		"uid":        userModel.UID,
		"token":      token,
		"short_no":   userModel.ShortNo,
		"avatar":     u.ctx.GetConfig().GetAvatarPath(userModel.UID),
		"im_pub_key": "",
	})
}

// 获取二维码数据的管道
func (u *User) getQRCodeModelChan(uuid string) <-chan *common.QRCodeModel {
	qrcodeModelChan := make(chan *common.QRCodeModel)
	qrcodeChanLock.Lock()
	qrcodeChanMap[uuid] = qrcodeModelChan
	qrcodeChanLock.Unlock()
	return qrcodeModelChan
}
func (u *User) removeQRCodeChan(uuid string) {
	qrcodeChanLock.Lock()
	defer qrcodeChanLock.Unlock()
	_, exist := qrcodeChanMap[uuid]
	if exist {
		delete(qrcodeChanMap, uuid)
	}
}

// SendQRCodeInfo 发送二维码数据
func SendQRCodeInfo(uuid string, qrcode *common.QRCodeModel) {
	qrcodeChanLock.Lock()
	qrcodeChan := qrcodeChanMap[uuid]
	qrcodeChanLock.Unlock()
	if qrcodeChan != nil {
		qrcodeChan <- qrcode
	}
}

// 授权登录
func (u *User) grantLogin(c *wkhttp.Context) {
	authCode := c.Query("auth_code")
	loginUID := c.MustGet("uid").(string)
	encrypt := c.Query("encrypt") // signal相关密钥
	if authCode == "" {
		c.ResponseError(errors.New("授权码不能为空！"))
		return
	}
	authInfo, err := u.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", common.AuthCodeCachePrefix, authCode))
	if err != nil {
		u.Error("获取授权信息失败！", zap.Error(err))
		c.ResponseError(errors.New("获取授权信息失败！"))
		return
	}
	if authInfo == "" {
		c.ResponseError(errors.New("授权码失效或不存在！"))
		return
	}
	var authInfoMap map[string]interface{}
	err = util.ReadJsonByByte([]byte(authInfo), &authInfoMap)
	if err != nil {
		u.Error("解码授权信息失败！", zap.Error(err))
		c.ResponseError(errors.New("解码授权信息失败！"))
		return
	}
	authType := authInfoMap["type"].(string)
	if authType != string(common.AuthCodeTypeScanLogin) {
		c.ResponseError(errors.New("授权码不是登录授权码！"))
		return
	}
	scaner := authInfoMap["scaner"].(string)
	if scaner != loginUID {
		c.ResponseError(errors.New("扫描者与授权者不是同一个用户！"))
		return
	}
	uuid := authInfoMap["uuid"].(string)
	qrcodeInfo := common.NewQRCodeModel(common.QRCodeTypeScanLogin, map[string]interface{}{
		"app_id":    "wukongchat",
		"status":    common.ScanLoginStatusAuthed,
		"uid":       loginUID,
		"auth_code": authCode,
		"encrypt":   encrypt,
	})
	err = u.ctx.GetRedisConn().SetAndExpire(fmt.Sprintf("%s%s", common.QRCodeCachePrefix, uuid), util.ToJson(qrcodeInfo), time.Minute*5)
	if err != nil {
		u.Error("更新二维码信息失败！", zap.Error(err))
		c.ResponseError(errors.New("更新二维码信息失败！"))
		return
	}
	SendQRCodeInfo(uuid, qrcodeInfo)
	c.ResponseOK()
}

// addBlacklist 添加黑名单
func (u *User) addBlacklist(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	uid := c.Param("uid")
	if strings.TrimSpace(uid) == "" {
		c.ResponseError(errors.New("添加黑名单的用户ID不能空！"))
		return
	}
	model, err := u.settingDB.QueryUserSettingModel(uid, loginUID)
	if err != nil {
		u.Error("查询用户设置失败", zap.Error(err))
		c.ResponseError(errors.New("查询用户设置失败！"))
		return
	}
	//如果没有设置记录先添加一条记录
	if model == nil || strings.TrimSpace(model.UID) == "" {
		userSettingModel := &SettingModel{
			UID:   loginUID,
			ToUID: uid,
		}
		err = u.settingDB.InsertUserSettingModel(userSettingModel)
		if err != nil {
			u.Error("添加用户设置失败", zap.Error(err))
			c.ResponseError(errors.New("添加用户设置失败！"))
			return
		}
	}

	// 请求im服务器设置黑名单
	err = u.ctx.IMBlacklistSet(config.ChannelBlacklistReq{
		ChannelReq: config.ChannelReq{
			ChannelID:   common.GetFakeChannelIDWith(loginUID, uid),
			ChannelType: common.ChannelTypePerson.Uint8(),
		},
		UIDs: []string{loginUID, uid},
	})
	if err != nil {
		u.Error("设置黑名单失败！", zap.Error(err))
		c.ResponseError(errors.New("设置黑名单失败！"))
		return
	}
	//添加黑名单
	version := u.ctx.GenSeq(common.UserSettingSeqKey)
	friendVersion := u.ctx.GenSeq(common.FriendSeqKey)
	tx, _ := u.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err = u.db.AddOrRemoveBlacklistTx(loginUID, uid, 1, version, tx)
	if err != nil {
		tx.Rollback()
		u.Error("添加黑名单失败！", zap.Error(err))
		c.ResponseError(errors.New("添加黑名单失败！"))
		return
	}
	err = u.friendDB.updateVersionTx(friendVersion, loginUID, uid, tx)
	if err != nil {
		tx.Rollback()
		u.Error("更新好友的版本号失败！", zap.Error(err))
		c.ResponseError(errors.New("更新好友的版本号失败！"))
		return
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		u.Error("提交数据库失败！", zap.Error(err))
		c.ResponseError(errors.New("提交数据库失败！"))
		return
	}

	// 发送给被拉黑的人去更新拉黑人的频道
	err = u.ctx.SendChannelUpdate(config.ChannelReq{
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
	}, config.ChannelReq{
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
	})
	if err != nil {
		u.Warn("发送频道更新命令失败！", zap.Error(err))
	}

	// 发送给操作者，去更新被拉黑的人的频道
	err = u.ctx.SendChannelUpdate(config.ChannelReq{
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
	}, config.ChannelReq{
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
	})
	if err != nil {
		u.Warn("发送频道更新命令失败！", zap.Error(err))
	}

	c.ResponseOK()
}

// removeBlacklist 移除黑名单
func (u *User) removeBlacklist(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	uid := c.Param("uid")
	if strings.TrimSpace(uid) == "" {
		c.ResponseError(errors.New("移除黑名单的用户ID不能空！"))
		return
	}

	version := u.ctx.GenSeq(common.UserSettingSeqKey)
	friendVersion := u.ctx.GenSeq(common.FriendSeqKey)

	tx, _ := u.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err := u.db.AddOrRemoveBlacklistTx(loginUID, uid, 0, version, tx)
	if err != nil {
		tx.Rollback()
		u.Error("移除黑名单失败！", zap.Error(err))
		c.ResponseError(errors.New("移除黑名单失败！"))
		return
	}
	err = u.friendDB.updateVersionTx(friendVersion, loginUID, uid, tx)
	if err != nil {
		tx.Rollback()
		u.Error("更新好友的版本号失败！", zap.Error(err))
		c.ResponseError(errors.New("更新好友的版本号失败！"))
		return
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		u.Error("提交数据库失败！", zap.Error(err))
		c.ResponseError(errors.New("提交数据库失败！"))
		return
	}

	userSetting, err := u.settingDB.querySettingByUIDAndToUID(uid, loginUID)
	if err != nil {
		u.Error("查询用户设置错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户设置错误"))
		return
	}

	// 双方都不在黑名单后才能设置IM黑名单
	if userSetting == nil || userSetting.Blacklist == 0 {
		// 请求im服务器设置黑名单
		err = u.ctx.IMBlacklistSet(config.ChannelBlacklistReq{
			ChannelReq: config.ChannelReq{
				ChannelID:   common.GetFakeChannelIDWith(loginUID, uid),
				ChannelType: common.ChannelTypePerson.Uint8(),
			},
			UIDs: make([]string, 0),
		})
		if err != nil {
			u.Error("设置黑名单失败！", zap.Error(err))
			c.ResponseError(errors.New("设置黑名单失败！"))
			return
		}
	}

	// 发送给被拉黑的人去更新拉黑人的频道
	err = u.ctx.SendChannelUpdate(config.ChannelReq{
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
	}, config.ChannelReq{
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
	})
	if err != nil {
		u.Warn("发送频道更新命令失败！", zap.Error(err))
	}

	// 发送给操作者，去更新被拉黑的人的频道
	err = u.ctx.SendChannelUpdate(config.ChannelReq{
		ChannelID:   loginUID,
		ChannelType: common.ChannelTypePerson.Uint8(),
	}, config.ChannelReq{
		ChannelID:   uid,
		ChannelType: common.ChannelTypePerson.Uint8(),
	})
	if err != nil {
		u.Warn("发送频道更新命令失败！", zap.Error(err))
	}

	c.ResponseOK()
}

// blacklists 获取黑名单列表
func (u *User) blacklists(c *wkhttp.Context) {
	loginUID := c.MustGet("uid").(string)
	list, err := u.db.Blacklists(loginUID)
	if err != nil {
		u.Error("查询黑名单列表失败！", zap.Error(err))
		c.ResponseError(errors.New("查询黑名单列表失败！"))
		return
	}
	blacklists := []*blacklistResp{}
	for _, result := range list {
		blacklists = append(blacklists, &blacklistResp{
			UID:      result.UID,
			Name:     result.Name,
			Username: result.Username,
		})
	}
	c.Response(blacklists)
}

// sendRegisterCode 发送注册短信
func (u *User) sendRegisterCode(c *wkhttp.Context) {
	var req codeReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if strings.TrimSpace(req.Zone) == "" {
		c.ResponseError(errors.New("区号不能为空！"))
		return
	}
	if strings.TrimSpace(req.Phone) == "" {
		c.ResponseError(errors.New("手机号不能为空！"))
		return
	}
	if u.ctx.GetConfig().Register.OnlyChina {
		if strings.TrimSpace(req.Zone) != "0086" {
			c.ResponseError(errors.New("仅仅支持中国大陆手机号注册！"))
			return
		}
	}

	span := u.ctx.Tracer().StartSpan(
		"user.sendRegisterCode",
		opentracing.ChildOf(c.GetSpanContext()),
	)
	defer span.Finish()
	spanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), span)

	model, err := u.db.QueryByPhone(req.Zone, req.Phone)
	if err != nil {
		u.Error("查询用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询用户信息失败！"))
		return
	}
	if model != nil {
		c.Response(map[string]interface{}{
			"exist": 1,
		})
		return
	}
	err = u.smsServie.SendVerifyCode(spanCtx, req.Zone, req.Phone, commonapi.CodeTypeRegister)
	if err != nil {
		u.Error("发送短信验证码失败", zap.Error(err))
		c.ResponseError(errors.New("发送短信验证码失败！"))
		return
	}
	c.Response(map[string]interface{}{
		"exist": 0,
	})
}

// setChatPwd 修改用户聊天密码
func (u *User) setChatPwd(c *wkhttp.Context) {
	var req chatPwdReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if strings.TrimSpace(req.ChatPwd) == "" {
		c.ResponseError(errors.New("聊天密码不能为空"))
		return
	}
	if strings.TrimSpace(req.LoginPwd) == "" {
		c.ResponseError(errors.New("登录密码不能为空！"))
		return
	}
	loginUID := c.MustGet("uid").(string)
	user, err := u.db.QueryByUID(loginUID)
	if err != nil {
		u.Error("查询用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询用户信息失败"))
		return
	}
	if user.Password != util.MD5(util.MD5(req.LoginPwd)) {
		c.ResponseError(errors.New("登录密码错误"))
		return
	}
	//修改用户聊天密码
	err = u.db.UpdateUsersWithField("chat_pwd", req.ChatPwd, loginUID)
	if err != nil {
		u.Error("查询用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("修改聊天密码失败"))
		return
	}
	c.ResponseOK()
}

// 设置锁屏密码
func (u *User) lockScreenAfterMinuteSet(c *wkhttp.Context) {
	var req struct {
		LockAfterMinute int `json:"lock_after_minute"` // 在几分钟后锁屏
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if req.LockAfterMinute < 0 {
		c.ResponseError(errors.New("锁屏时间不能小于0"))
		return
	}
	if req.LockAfterMinute > 60 {
		c.ResponseError(errors.New("锁屏时间不能大于60分钟"))
		return
	}
	loginUID := c.GetLoginUID()
	err := u.db.UpdateUsersWithField("lock_after_minute", strconv.FormatInt(int64(req.LockAfterMinute), 10), loginUID)
	if err != nil {
		u.Error("修改用户锁屏密码错误", zap.Error(err))
		c.ResponseError(errors.New("修改用户锁屏密码错误"))
		return
	}
	c.ResponseOK()
}

// 设置锁屏密码
func (u *User) setLockScreenPwd(c *wkhttp.Context) {
	var req struct {
		LockScreenPwd string `json:"lock_screen_pwd"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if strings.TrimSpace(req.LockScreenPwd) == "" {
		c.ResponseError(errors.New("锁屏密码不能为空"))
		return
	}

	loginUID := c.GetLoginUID()
	err := u.db.UpdateUsersWithField("lock_screen_pwd", req.LockScreenPwd, loginUID)
	if err != nil {
		u.Error("修改用户锁屏密码错误", zap.Error(err))
		c.ResponseError(errors.New("修改用户锁屏密码错误"))
		return
	}
	c.ResponseOK()
}

// 关闭锁屏密码
func (u *User) closeLockScreenPwd(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	err := u.db.UpdateUsersWithField("lock_screen_pwd", "", loginUID)
	if err != nil {
		u.Error("修改用户锁屏密码错误", zap.Error(err))
		c.ResponseError(errors.New("修改用户锁屏密码错误"))
		return
	}
	c.ResponseOK()
}

// sendLoginCheckPhoneCode 发送登录验证短信
func (u *User) sendLoginCheckPhoneCode(c *wkhttp.Context) {
	var req struct {
		UID string `json:"uid"`
	}
	if err := c.BindJSON(&req); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if req.UID == "" {
		c.ResponseError(errors.New("uid不能为空！"))
		return
	}

	span := u.ctx.Tracer().StartSpan(
		"user.sendLoginCheckPhoneCode",
		opentracing.ChildOf(c.GetSpanContext()),
	)
	defer span.Finish()
	spanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), span)

	userinfo, err := u.db.QueryByUID(req.UID)
	if err != nil {
		u.Error("查询用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("修改聊天密码失败"))
		return
	}
	if userinfo == nil {
		u.Error("该用户不存在", zap.Error(err))
		c.ResponseError(errors.New("该用户不存在"))
		return
	}
	//发送短信
	// if u.ctx.GetConfig().Test {
	// 	c.ResponseOK()
	// 	return
	// }
	err = u.smsServie.SendVerifyCode(spanCtx, userinfo.Zone, userinfo.Phone, commonapi.CodeTypeCheckMobile)
	if err != nil {
		u.Error("发送短信失败", zap.Error(err))
		ext.LogError(span, err)
		c.ResponseError(errors.New("发送短信失败"))
		return
	}
	c.ResponseOK()
}

// loginCheckPhone 登录验证设备短信
func (u *User) loginCheckPhone(c *wkhttp.Context) {
	var req struct {
		UID  string `json:"uid"`
		Code string `json:"code"`
	}
	if err := c.BindJSON(&req); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if req.UID == "" {
		c.ResponseError(errors.New("uid不能为空！"))
		return
	}
	if req.Code == "" {
		c.ResponseError(errors.New("验证码不能为空！"))
		return
	}
	span := u.ctx.Tracer().StartSpan(
		"user.loginCheckPhone",
		opentracing.ChildOf(c.GetSpanContext()),
	)
	defer span.Finish()
	spanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), span)

	userInfo, err := u.db.QueryByUID(req.UID)
	if err != nil {
		u.Error("查询用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("修改聊天密码失败"))
		return
	}
	if userInfo == nil {
		u.Error("该用户不存在", zap.Error(err))
		c.ResponseError(errors.New("该用户不存在"))
		return
	}
	err = u.smsServie.Verify(spanCtx, userInfo.Zone, userInfo.Phone, req.Code, commonapi.CodeTypeCheckMobile)
	if err != nil {
		u.Error("验证短信失败", zap.Error(err))
		c.ResponseError(err)
		return
	}

	loginDeviceJsonStr, err := u.ctx.GetRedisConn().GetString(fmt.Sprintf("%s%s", u.ctx.GetConfig().Cache.LoginDeviceCachePrefix, req.UID))
	if err != nil {
		u.Error("获取登录设备缓存失败！", zap.Error(err))
		c.ResponseError(errors.New("获取登录设备缓存失败！"))
		return
	}
	if loginDeviceJsonStr == "" {
		c.ResponseError(errors.New("登录设备已过期，请重新登录"))
		return
	}
	var loginDeivce *deviceReq
	err = util.ReadJsonByByte([]byte(loginDeviceJsonStr), &loginDeivce)
	if err != nil {
		u.Error("解码登录设备信息失败！", zap.Error(err), zap.String("uid", req.UID))
		c.ResponseError(errors.New("解码登录设备信息失败！"))
		return
	}
	err = u.deviceDB.insertOrUpdateDeviceCtx(spanCtx, &deviceModel{
		UID:         userInfo.UID,
		DeviceID:    loginDeivce.DeviceID,
		DeviceName:  loginDeivce.DeviceName,
		DeviceModel: loginDeivce.DeviceModel,
		LastLogin:   time.Now().Unix(),
	})
	if err != nil {
		u.Error("添加或更新登录设备信息失败！", zap.Error(err))
		c.ResponseError(errors.New("添加或更新登录设备信息失败！"))
		return
	}
	token := util.GenerUUID()
	// 将token设置到缓存
	err = u.ctx.Cache().SetAndExpire(u.ctx.GetConfig().Cache.TokenCachePrefix+token, fmt.Sprintf("%s@%s", userInfo.UID, userInfo.Name), u.ctx.GetConfig().Cache.TokenExpire)
	if err != nil {
		u.Error("设置token缓存失败！", zap.Error(err))
		c.ResponseError(errors.New("设置token缓存失败！"))
		return
	}
	// err = u.ctx.UpdateIMToken(userInfo.UID, token, config.DeviceFlag(0), config.DeviceLevelMaster)
	imResp, err := u.ctx.UpdateIMToken(config.UpdateIMTokenReq{
		UID:         userInfo.UID,
		Token:       token,
		DeviceFlag:  config.APP,
		DeviceLevel: config.DeviceLevelMaster,
	})
	if err != nil {
		u.Error("更新IM的token失败！", zap.Error(err))
		c.ResponseError(errors.New("更新IM的token失败！"))
		return
	}
	if imResp.Status == config.UpdateTokenStatusBan {
		c.ResponseError(errors.New("此账号已经被封禁！"))
		return
	}
	c.Response(newLoginUserDetailResp(userInfo, token, u.ctx))
}

// customerservices 客服列表
func (u *User) customerservices(c *wkhttp.Context) {
	list, err := u.db.QueryByCategory("service")
	if err != nil {
		u.Error("查询客服列表失败", zap.Error(err))
		c.ResponseError(errors.New("查询客服列表失败"))
		return
	}
	results := []*customerservicesResp{}
	if len(list) > 0 {
		for _, user := range list {
			results = append(results, &customerservicesResp{
				UID:  user.UID,
				Name: user.Name,
			})
		}
	}
	c.Response(results)
}

// 发送注销账号验证吗
func (u *User) sendDestroyCode(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	userInfo, err := u.db.QueryByUID(loginUID)
	if err != nil {
		u.Error("查询登录用户信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询登录用户信息错误"))
		return
	}
	if userInfo == nil || userInfo.IsDestroy == 1 {
		c.ResponseError(errors.New("登录用户不存在"))
		return
	}
	err = u.smsServie.SendVerifyCode(c.Context, userInfo.Zone, userInfo.Phone, commonapi.CodeTypeDestroyAccount)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.ResponseOK()
}

// 注销账号
func (u *User) destroyAccount(c *wkhttp.Context) {
	code := c.Param("code")
	loginUID := c.GetLoginUID()
	if code == "" {
		c.ResponseError(errors.New("验证码不能为空"))
		return
	}
	userInfo, err := u.db.QueryByUID(loginUID)
	if err != nil {
		u.Error("查询登录用户信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询登录用户信息错误"))
		return
	}
	if userInfo == nil || userInfo.IsDestroy == 1 {
		c.ResponseError(errors.New("登录用户不存在"))
		return
	}
	// 校验验证码
	err = u.smsServie.Verify(c.Context, userInfo.Zone, userInfo.Phone, code, commonapi.CodeTypeDestroyAccount)
	if err != nil {
		c.ResponseError(err)
		return
	}
	t := time.Now()
	time := fmt.Sprintf("%d%d%d%d%d", t.Year(), t.Month(), t.Day(), t.Minute(), t.Second())
	phone := fmt.Sprintf("%s@%s@delete", userInfo.Phone, time)
	username := fmt.Sprintf("%s%s", userInfo.Zone, phone)
	err = u.db.destroyAccount(loginUID, username, phone)
	if err != nil {
		u.Error("注销账号错误", zap.Error(err))
		c.ResponseError(errors.New("注销账号错误"))
		return
	}
	err = u.ctx.QuitUserDevice(c.GetLoginUID(), -1) // 退出全部登陆设备
	if err != nil {
		u.Error("退出登陆设备失败", zap.Error(err))
		c.ResponseError(errors.New("退出登陆设备失败"))
		return
	}

	c.ResponseOK()
}

// 处理注册用户和文件助手互为好友
func (u *User) addFileHelperFriend(uid string) error {
	if uid == "" {
		u.Error("用户ID不能为空")
		return errors.New("用户ID不能为空")
	}
	isFriend, err := u.friendDB.IsFriend(uid, u.ctx.GetConfig().Account.FileHelperUID)
	if err != nil {
		u.Error("查询用户关系失败")
		return err
	}
	if !isFriend {
		version := u.ctx.GenSeq(common.FriendSeqKey)
		err := u.friendDB.Insert(&FriendModel{
			UID:     uid,
			ToUID:   u.ctx.GetConfig().Account.FileHelperUID,
			Version: version,
		})
		if err != nil {
			u.Error("注册用户和文件助手成为好友失败")
			return err
		}
	}
	return nil
}

// addSystemFriend 处理注册用户和系统账号互为好友
func (u *User) addSystemFriend(uid string) error {

	if uid == "" {
		u.Error("用户ID不能为空")
		return errors.New("用户ID不能为空")
	}
	isFriend, err := u.friendDB.IsFriend(uid, u.ctx.GetConfig().Account.SystemUID)
	if err != nil {
		u.Error("查询用户关系失败")
		return err
	}
	tx, _ := u.friendDB.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	if !isFriend {
		version := u.ctx.GenSeq(common.FriendSeqKey)
		err := u.friendDB.InsertTx(&FriendModel{
			UID:     uid,
			ToUID:   u.ctx.GetConfig().Account.SystemUID,
			Version: version,
		}, tx)
		if err != nil {
			u.Error("注册用户和系统账号成为好友失败")
			tx.Rollback()
			return err
		}
	}
	// systemIsFriend, err := u.friendDB.IsFriend(u.ctx.GetConfig().SystemUID, uid)
	// if err != nil {
	// 	u.Error("查询系统账号和注册用户关系失败")
	// 	tx.Rollback()
	// 	return err
	// }
	// if !systemIsFriend {
	// 	version := u.ctx.GenSeq(common.FriendSeqKey)
	// 	err := u.friendDB.InsertTx(&FriendModel{
	// 		UID:     u.ctx.GetConfig().SystemUID,
	// 		ToUID:   uid,
	// 		Version: version,
	// 	}, tx)
	// 	if err != nil {
	// 		u.Error("系统账号和注册用户成为好友失败")
	// 		tx.Rollback()
	// 		return err
	// 	}
	// }
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		u.Error("用户注册数据库事物提交失败", zap.Error(err))
		return err
	}
	return nil
}

// 重置登录密码
func (u *User) pwdforget(c *wkhttp.Context) {
	var req resetPwdReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if strings.TrimSpace(req.Zone) == "" {
		c.ResponseError(errors.New("区号不能为空！"))
		return
	}
	if strings.TrimSpace(req.Phone) == "" {
		c.ResponseError(errors.New("手机号不能为空！"))
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		c.ResponseError(errors.New("验证码不能为空！"))
		return
	}
	if strings.TrimSpace(req.Pwd) == "" {
		c.ResponseError(errors.New("密码不能为空！"))
		return
	}
	userInfo, err := u.db.QueryByPhone(req.Zone, req.Phone)
	if err != nil {
		u.Error("查询用户信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询用户信息错误"))
		return
	}
	if userInfo == nil {
		c.ResponseError(errors.New("该账号不存在"))
		return
	}
	//测试模式
	if strings.TrimSpace(u.ctx.GetConfig().SMSCode) != "" {
		if strings.TrimSpace(u.ctx.GetConfig().SMSCode) != req.Code {
			c.ResponseError(errors.New("验证码错误"))
			return
		}
	} else {
		//线上验证短信验证码
		err = u.smsServie.Verify(context.Background(), req.Zone, req.Phone, req.Code, commonapi.CodeTypeForgetLoginPWD)
		if err != nil {
			c.ResponseError(err)
			return
		}
	}

	err = u.db.UpdateUsersWithField("password", util.MD5(util.MD5(req.Pwd)), userInfo.UID)
	if err != nil {
		u.Error("修改登录密码错误", zap.Error(err))
		c.ResponseError(errors.New("修改登录密码错误"))
		return
	}
	c.ResponseOK()
}

// 获取忘记密码验证码
func (u *User) getForgetPwdSMS(c *wkhttp.Context) {
	var req codeReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	if strings.TrimSpace(req.Zone) == "" {
		c.ResponseError(errors.New("区号不能为空！"))
		return
	}
	if strings.TrimSpace(req.Phone) == "" {
		c.ResponseError(errors.New("手机号不能为空！"))
		return
	}

	span := u.ctx.Tracer().StartSpan(
		"user.sendForgetPwdCode",
		opentracing.ChildOf(c.GetSpanContext()),
	)
	defer span.Finish()
	spanCtx := u.ctx.Tracer().ContextWithSpan(context.Background(), span)

	model, err := u.db.QueryByPhone(req.Zone, req.Phone)
	if err != nil {
		u.Error("查询用户信息失败！", zap.Error(err))
		c.ResponseError(errors.New("查询用户信息失败！"))
		return
	}
	if model == nil {
		c.ResponseError(errors.New("该手机号未注册"))
		return
	}
	err = u.smsServie.SendVerifyCode(spanCtx, req.Zone, req.Phone, commonapi.CodeTypeForgetLoginPWD)
	if err != nil {
		u.Error("发送短信验证码失败", zap.Error(err))
		c.ResponseError(errors.New("发送短信验证码失败！"))
		return
	}
	c.ResponseOK()
}

// 是否允许更新
func allowUpdateUserField(field string) bool {
	allowfields := []string{"sex", "short_no", "name", "search_by_phone", "search_by_short", "new_msg_notice", "msg_show_detail", "voice_on", "shock_on", "msg_expire_second"}
	for _, allowFiled := range allowfields {
		if field == allowFiled {
			return true
		}
	}
	return false
}

func (u *User) createUser(registerSpanCtx context.Context, createUser *createUserModel, c *wkhttp.Context) {
	tx, _ := u.db.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	publicIP := util.GetClientPublicIP(c.Request)
	resp, err := u.createUserWithRespAndTx(registerSpanCtx, createUser, publicIP, tx, func() error {
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
		c.ResponseError(errors.New("注册失败！"))
		return
	}
	c.Response(resp)
}

func (u *User) createUserTx(registerSpanCtx context.Context, createUser *createUserModel, c *wkhttp.Context, commitCallback func() error, tx *dbr.Tx) {
	publicIP := util.GetClientPublicIP(c.Request)
	resp, err := u.createUserWithRespAndTx(registerSpanCtx, createUser, publicIP, tx, commitCallback)
	if err != nil {
		c.ResponseError(errors.New("注册失败！"))
		return
	}
	c.Response(resp)
}

func (u *User) createUserWithRespAndTx(registerSpanCtx context.Context, createUser *createUserModel, publicIP string, tx *dbr.Tx, commitCallback func() error) (*loginUserDetailResp, error) {
	var (
		shortNo = ""
		err     error
	)
	if u.ctx.GetConfig().ShortNo.NumOn {
		shortNo, err = u.commonService.GetShortno()
		if err != nil {
			u.Error("获取短编号失败！", zap.Error(err))
			return nil, err
		}
	} else {
		shortNo = util.Ten2Hex(time.Now().UnixNano())
	}

	userModel := &Model{}
	userModel.UID = createUser.UID
	rand.Seed(time.Now().Unix())
	if createUser.Name != "" {
		userModel.Name = createUser.Name
	} else {
		userModel.Name = Names[rand.Intn(len(Names)-1)]
	}
	userModel.Sex = createUser.Sex
	userModel.Vercode = fmt.Sprintf("%s@%d", util.GenerUUID(), common.User)
	userModel.QRVercode = fmt.Sprintf("%s@%d", util.GenerUUID(), common.QRCode)
	userModel.Phone = createUser.Phone
	userModel.Zone = createUser.Zone
	if createUser.Phone != "" {
		userModel.Username = fmt.Sprintf("%s%s", createUser.Zone, createUser.Phone)
	}
	if createUser.Password != "" {
		userModel.Password = util.MD5(util.MD5(createUser.Password))
	}
	if createUser.Username != "" {
		userModel.Username = createUser.Username
	}

	userModel.ShortNo = shortNo
	userModel.OfflineProtection = 0
	userModel.NewMsgNotice = 1
	userModel.MsgShowDetail = 1
	userModel.SearchByPhone = 1
	userModel.SearchByShort = 1
	userModel.VoiceOn = 1
	userModel.ShockOn = 1
	userModel.IsUploadAvatar = createUser.IsUploadAvatar
	userModel.WXOpenid = createUser.WXOpenid
	userModel.WXUnionid = createUser.WXUnionid
	userModel.GiteeUID = createUser.GiteeUID
	userModel.GithubUID = createUser.GithubUID

	userModel.Status = int(common.UserAvailable)
	err = u.db.insertTx(userModel, tx)
	if err != nil {
		u.Error("注册用户失败", zap.Error(err))
		return nil, err
	}
	if createUser.Device != nil {
		err = u.deviceDB.insertOrUpdateDeviceTx(&deviceModel{
			UID:         createUser.UID,
			DeviceID:    createUser.Device.DeviceID,
			DeviceName:  createUser.Device.DeviceName,
			DeviceModel: createUser.Device.DeviceModel,
			LastLogin:   time.Now().Unix(),
		}, tx)
		if err != nil {
			u.Error("添加用户设备信息失败", zap.Error(err))
			return nil, err
		}
	}
	err = u.addSystemFriend(createUser.UID)
	if err != nil {
		u.Error("添加注册用户和系统账号为好友关系失败", zap.Error(err))
		return nil, err
	}
	err = u.addFileHelperFriend(createUser.UID)
	if err != nil {
		u.Error("添加注册用户和文件助手为好友关系失败", zap.Error(err))
		return nil, err
	}
	//发送用户注册事件
	eventID, err := u.ctx.EventBegin(&wkevent.Data{
		Event: event.EventUserRegister,
		Type:  wkevent.Message,
		Data: map[string]interface{}{
			"uid": createUser.UID,
		},
	}, tx)
	if err != nil {
		u.Error("开启事件失败！", zap.Error(err))
		return nil, err
	}

	if commitCallback != nil {
		commitCallback()
	}
	u.ctx.EventCommit(eventID)
	token := util.GenerUUID()
	// 将token设置到缓存
	err = u.ctx.Cache().SetAndExpire(u.ctx.GetConfig().Cache.TokenCachePrefix+token, fmt.Sprintf("%s@%s@%s", userModel.UID, userModel.Name, userModel.Role), u.ctx.GetConfig().Cache.TokenExpire)
	if err != nil {
		u.Error("设置token缓存失败！", zap.Error(err))
		return nil, err
	}
	_, err = u.ctx.UpdateIMToken(config.UpdateIMTokenReq{
		UID:         createUser.UID,
		Token:       token,
		DeviceFlag:  config.DeviceFlag(createUser.Flag),
		DeviceLevel: config.DeviceLevelSlave,
	})
	if err != nil {
		u.Error("更新IM的token失败！", zap.Error(err))
		return nil, err
	}
	go u.sentWelcomeMsg(publicIP, createUser.UID)

	if u.ctx.GetConfig().ShortNo.NumOn {
		err = u.commonService.SetShortnoUsed(userModel.ShortNo, "user")
		if err != nil {
			u.Error("设置短编号被使用失败！", zap.Error(err), zap.String("shortNo", userModel.ShortNo))
		}
	}

	return newLoginUserDetailResp(userModel, token, u.ctx), nil
}

// ---------- vo ----------
type createUserModel struct {
	UID            string
	Name           string
	Zone           string
	Phone          string
	Sex            int
	Password       string
	WXOpenid       string
	WXUnionid      string
	GiteeUID       string
	GithubUID      string
	Username       string
	Flag           int
	IsUploadAvatar int
	Device         *deviceReq
}

// 重置登录密码
type resetPwdReq struct {
	Zone  string `json:"zone"`  //区号
	Phone string `json:"phone"` //手机号
	Code  string `json:"code"`  //验证码
	Pwd   string `json:"pwd"`   //密码
}
type customerservicesResp struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}
type registerReq struct {
	Name     string     `json:"name"`
	Zone     string     `json:"zone"`
	Phone    string     `json:"phone"`
	Code     string     `json:"code"`
	Password string     `json:"password"`
	Flag     uint8      `json:"flag"`   // 注册设备的标记 0.APP 1.PC
	Device   *deviceReq `json:"device"` //注册用户设备信息
}

func (r registerReq) CheckRegister() error {
	if strings.TrimSpace(r.Zone) == "" {
		return errors.New("区号不能为空！")
	}
	if strings.TrimSpace(r.Phone) == "" {
		return errors.New("手机号不能为空！")
	}
	if strings.TrimSpace(r.Code) == "" {
		return errors.New("验证码不能为空！")
	}
	if strings.TrimSpace(r.Password) == "" {
		return errors.New("密码不能为空！")
	}
	if len(r.Password) < 6 {
		return errors.New("密码长度必须大于6位！")
	}
	return nil
}

// 设置聊天密码请求
type chatPwdReq struct {
	ChatPwd  string `json:"chat_pwd"`  //聊天密码
	LoginPwd string `json:"login_pwd"` //登录密码
}

// 注册验证码请求
type codeReq struct {
	Zone  string `json:"zone"`
	Phone string `json:"phone"`
}
type loginReq struct {
	Username string     `json:"username"`
	Password string     `json:"password"`
	Flag     int        `json:"flag"`   // 设备标示 0.APP 1.PC
	Device   *deviceReq `json:"device"` //登录设备信息
}

func (r loginReq) Check() error {
	if strings.TrimSpace(r.Username) == "" {
		return errors.New("用户名不能为空！")
	}
	if strings.TrimSpace(r.Password) == "" {
		return errors.New("密码不能为空！")
	}
	return nil
}

type userResp struct {
	UID     string `json:"uid"`
	Name    string `json:"name"`
	Vercode string `json:"vercode"`
}

func newUserResp(m *Model) userResp {
	return userResp{
		UID:     m.UID,
		Name:    m.Name,
		Vercode: m.Vercode,
	}
}

type deviceReq struct {
	DeviceID    string `json:"device_id"`    //设备唯一ID
	DeviceName  string `json:"device_name"`  //设备名称
	DeviceModel string `json:"device_model"` //设备model
}

type loginUserDetailResp struct {
	UID             string  `json:"uid"`
	AppID           string  `json:"app_id"`
	Name            string  `json:"name"`
	Username        string  `json:"username"`
	Sex             int     `json:"sex"`               //性别1:男
	Category        string  `json:"category"`          //用户分类 '客服'
	ShortNo         string  `json:"short_no"`          // 用户唯一短编号
	Zone            string  `json:"zone"`              //区号
	Phone           string  `json:"phone"`             //手机号
	Token           string  `json:"token"`             //token
	ChatPwd         string  `json:"chat_pwd"`          //聊天密码
	LockScreenPwd   string  `json:"lock_screen_pwd"`   // 锁屏密码
	LockAfterMinute int     `json:"lock_after_minute"` // 在N分钟后锁屏
	Setting         setting `json:"setting"`
	RSAPublicKey    string  `json:"rsa_public_key"` // 应用公钥做一些消息验证 base64编码
	ShortStatus     int     `json:"short_status"`
	MsgExpireSecond int64   `json:"msg_expire_second"` // 消息过期时长
}

type setting struct {
	SearchByPhone     int `json:"search_by_phone"`    //是否可以通过手机号搜索0.否1.是
	SearchByShort     int `json:"search_by_short"`    //是否可以通过短编号搜索0.否1.是
	NewMsgNotice      int `json:"new_msg_notice"`     //新消息通知0.否1.是
	MsgShowDetail     int `json:"msg_show_detail"`    //显示消息通知详情0.否1.是
	VoiceOn           int `json:"voice_on"`           //声音0.否1.是
	ShockOn           int `json:"shock_on"`           //震动0.否1.是
	OfflineProtection int `json:"offline_protection"` //离线保护，断网屏保
	DeviceLock        int `json:"device_lock"`        // 设备锁
	MuteOfApp         int `json:"mute_of_app"`        // web登录 app是否静音
}

type blacklistResp struct {
	UID      string `json:"uid"`
	Name     string `json:"name"`
	Username string `json:"usename"`
}

func newLoginUserDetailResp(m *Model, token string, ctx *config.Context) *loginUserDetailResp {

	return &loginUserDetailResp{
		UID:             m.UID,
		AppID:           m.AppID,
		Name:            m.Name,
		Username:        m.Username,
		Sex:             m.Sex,
		Category:        m.Category,
		ShortNo:         m.ShortNo,
		Zone:            m.Zone,
		Phone:           m.Phone,
		Token:           token,
		ChatPwd:         m.ChatPwd,
		LockScreenPwd:   m.LockScreenPwd,
		LockAfterMinute: m.LockAfterMinute,
		ShortStatus:     m.ShortStatus,
		RSAPublicKey:    base64.StdEncoding.EncodeToString([]byte(ctx.GetConfig().AppRSAPubKey)),
		MsgExpireSecond: m.MsgExpireSecond,
		Setting: setting{
			SearchByPhone:     m.SearchByPhone,
			SearchByShort:     m.SearchByShort,
			NewMsgNotice:      m.NewMsgNotice,
			MsgShowDetail:     m.MsgShowDetail,
			VoiceOn:           m.VoiceOn,
			ShockOn:           m.ShockOn,
			OfflineProtection: m.OfflineProtection,
			DeviceLock:        m.DeviceLock,
			MuteOfApp:         m.MuteOfApp,
		},
	}
}
