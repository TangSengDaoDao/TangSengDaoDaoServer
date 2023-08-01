package user

import (
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"go.uber.org/zap"
)

type IOnlineService interface {
	// 获取用户最新在线状态
	GetUserLastOnlineStatus(uids []string) ([]*config.OnlinestatusResp, error)

	// 判断用户设备是否在线
	DeviceOnline(uid string, device config.DeviceFlag) (bool, error)

	// 总在线人数
	GetOnlineCount() (int64, error)
}

type OnlineService struct {
	ctx      *config.Context
	onlineDB *onlineDB
	log.Log
}

func NewOnlineService(ctx *config.Context) *OnlineService {
	return &OnlineService{
		ctx:      ctx,
		onlineDB: newOnlineDB(ctx),
		Log:      log.NewTLog("OnlineService"),
	}
}

func (o *OnlineService) listenOnlineStatus(onlineStatusList []config.OnlineStatus) {
	if len(onlineStatusList) <= 0 {
		return
	}

	tx, _ := o.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()

	for _, onlineStatus := range onlineStatusList {

		if !onlineStatus.Online && onlineStatus.OnlineCount > 0 { // 如果离线，但是还有设备在线 则不更新数据库的状态
			continue
		}
		status := 0
		if onlineStatus.Online {
			status = 1
		}
		err := o.onlineDB.insertOrUpdateUserOnlineTx(&onlineStatusModel{
			UID:         onlineStatus.UID,
			DeviceFlag:  onlineStatus.DeviceFlag,
			LastOffline: int(time.Now().Unix()),
			LastOnline:  int(time.Now().Unix()),
			Online:      status,
			Version:     time.Now().UnixNano() / 1000,
		}, tx)
		if err != nil {
			tx.Rollback()
			o.Error("添加或更新用户在线状态失败！", zap.Error(err))
			return
		}
	}
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		o.Error("提交在线状态数据库的事务失败！！", zap.Error(err))
		return
	}
}

func (o *OnlineService) GetUserLastOnlineStatus(uids []string) ([]*config.OnlinestatusResp, error) {
	onlineModels, err := o.onlineDB.queryUserOnlineRecets(uids)
	if err != nil {
		return nil, err
	}
	resps := make([]*config.OnlinestatusResp, 0, len(onlineModels))
	if len(onlineModels) > 0 {
		for _, onlineM := range onlineModels {
			resps = append(resps, &config.OnlinestatusResp{
				UID:         onlineM.UID,
				DeviceFlag:  onlineM.DeviceFlag,
				LastOffline: onlineM.LastOffline,
				Online:      onlineM.Online,
			})
		}
	}
	return resps, nil

}

func (o *OnlineService) DeviceOnline(uid string, device config.DeviceFlag) (bool, error) {

	return o.onlineDB.exist(uid, device.Uint8(), 1)
}

func (o *OnlineService) GetOnlineCount() (int64, error) {
	return o.onlineDB.queryOnlineCount()
}
