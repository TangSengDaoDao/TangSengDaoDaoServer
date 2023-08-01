package user

import (
	"net/http"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// 退出pc登录
func (u *User) pcQuit(c *wkhttp.Context) {

	err := u.ctx.QuitUserDevice(c.GetLoginUID(), int(config.Web)) // 退出web
	if err != nil {
		u.Error("退出web设备失败", zap.Error(err))
		c.ResponseError(errors.New("退出web设备失败"))
		return
	}

	err = u.ctx.QuitUserDevice(c.GetLoginUID(), int(config.PC))
	if err != nil {
		u.Error("退出PC设备失败", zap.Error(err))
		c.ResponseError(errors.New("退出PC设备失败"))
		return
	}

	err = u.ctx.SendCMD(config.MsgCMDReq{
		NoPersist:   true,
		ChannelID:   c.GetLoginUID(),
		ChannelType: common.ChannelTypePerson.Uint8(),
		CMD:         common.CMDPCQuit,
	})
	if err != nil {
		c.ResponseErrorf("发送指令失败！", err)
		return
	}

	c.ResponseOK()
}

func (u *User) onlinelistWithUIDs(c *wkhttp.Context) {
	var uids []string
	if err := c.BindJSON(&uids); err != nil {
		c.ResponseError(err)
		return
	}
	onlineResps := make([]*userOnlineResp, 0)
	if len(uids) > 0 {
		onlines, err := u.onlineDB.queryUserOnlineRecets(uids)
		if err != nil {
			u.Error("查询用户在线状态失败！", zap.Error(err))
			c.ResponseError(errors.New("查询用户在线状态失败！"))
			return
		}
		if len(onlines) > 0 {
			for _, online := range onlines {
				onlineResps = append(onlineResps, newUserOnlineResp(online))
			}
		}
	}
	c.JSON(http.StatusOK, onlineResps)

}

// onlineList 查询在线用户 包含我的pc设备
func (u *User) onlineList(c *wkhttp.Context) {
	if !u.ctx.GetConfig().OnlineStatusOn {
		c.Response(make([]string, 0))
		return
	}
	loginUID := c.MustGet("uid").(string)
	friends, err := u.friendDB.QueryFriends(loginUID)
	if err != nil {
		u.Error("查询用户好友失败", zap.Error(err))
		c.ResponseError(errors.New("查询用户好友失败"))
		return
	}
	uids := make([]string, 0, len(friends))
	for _, friend := range friends {
		uids = append(uids, friend.ToUID)
	}
	resps, err := u.onlineService.GetUserLastOnlineStatus(uids)
	if err != nil {
		c.ResponseErrorf("获取用户在线状态失败！", err)
		return
	}
	pcOnlineB, err := u.onlineDB.exist(c.GetLoginUID(), config.PC.Uint8(), 1)
	if err != nil {
		c.ResponseErrorf("查询指定在线设备失败！", err)
		return
	}
	webOnline := 0
	if !pcOnlineB {
		webOnlineB, err := u.onlineDB.exist(c.GetLoginUID(), config.Web.Uint8(), 1)
		if err != nil {
			c.ResponseErrorf("查询指定在线设备失败！", err)
			return
		}
		if webOnlineB {
			webOnline = 1
		}
	}
	var pcResp *pcOnlineResp
	if pcOnlineB || webOnline == 1 {
		myM, err := u.db.QueryByUID(c.GetLoginUID())
		if err != nil {
			c.ResponseErrorf("获取我的个人数据失败！", err)
			return
		}
		deviceFlag := config.Web
		if pcOnlineB {
			deviceFlag = config.PC
		}
		pcResp = &pcOnlineResp{
			Online:     1,
			MuteOfApp:  myM.MuteOfApp,
			DeviceFlag: deviceFlag.Uint8(),
		}
	}

	c.Response(onlineFriendAndDeviceResp{
		Friends: resps,
		PC:      pcResp,
	})
}

func (u *User) onlineStatusCheck() {

	u.Debug("开始检查在线状态...")

	onlines, err := u.onlineDB.queryOnlinesMoreThan(time.Minute, 1000)
	if err != nil {
		u.Error("【在线状态检查】查询在线用户数失败！", zap.Error(err))
		return
	}
	if len(onlines) == 0 {
		return
	}
	u.Debug("检查到需要矫正的在线数量", zap.Int("onlines", len(onlines)))

	onlineUIDs := make([]string, 0, len(onlines))
	for _, online := range onlines {
		onlineUIDs = append(onlineUIDs, online.UID)
	}
	makeOfflines := make([]*onlineStatusModel, 0, len(onlines)) // 需要离线的id
	onlineStatusResps, err := u.ctx.IMSOnlineStatus(onlineUIDs)
	if err != nil {
		u.Error("【在线状态检查】获取在线状态失败！", zap.Error(err))
		return
	}
	u.Debug("检查到需要矫正的在线数量-->", zap.Int("onlineStatusResps", len(onlineStatusResps)))

	if len(onlines) > 0 {
		for _, online := range onlines {
			var exist bool
			for _, onlineStatusResp := range onlineStatusResps {
				if online.UID == onlineStatusResp.UID && onlineStatusResp.DeviceFlag == online.DeviceFlag {
					exist = true
					break
				}
			}
			if !exist {
				makeOfflines = append(makeOfflines, online)
			}
		}
	}
	if len(makeOfflines) > 0 {
		u.Debug("改变在线状态！", zap.Int("offlineCount", len(makeOfflines)))
		tx, _ := u.ctx.DB().Begin()
		defer func() {
			if err := recover(); err != nil {
				tx.RollbackUnlessCommitted()
				panic(err)
			}
		}()
		for _, onlineStatusResp := range makeOfflines {
			err := u.onlineDB.insertOrUpdateUserOnlineTx(&onlineStatusModel{
				UID:         onlineStatusResp.UID,
				DeviceFlag:  onlineStatusResp.DeviceFlag,
				LastOffline: int(time.Now().Unix()),
				LastOnline:  int(time.Now().Unix()),
				Online:      0,
				Version:     time.Now().UnixNano() / 1000,
			}, tx)
			if err != nil {
				tx.Rollback()
				u.Error("【在线状态检查】添加或更新用户在线状态失败！", zap.Error(err))
				return
			}
		}
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			u.Error("【在线状态检查】提交在线状态数据库的事务失败！！", zap.Error(err))
			return
		}
	}

}

type onlineFriendAndDeviceResp struct {
	PC      *pcOnlineResp              `json:"pc,omitempty"`
	Friends []*config.OnlinestatusResp `json:"friends,omitempty"` // 我的最近在线的好友
}

type pcOnlineResp struct {
	Online     int   `json:"online"`      // pc是否在线
	DeviceFlag uint8 `json:"device_flag"` // 设备类型
	MuteOfApp  int   `json:"mute_of_app"` // app是否开启禁音
}
