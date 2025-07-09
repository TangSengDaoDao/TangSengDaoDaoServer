package user

import (
	"fmt"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// 获取某个设备信息
func (u *User) getDevice(c *wkhttp.Context) {
	deviceID := c.Param("device_id")
	loginUID := c.GetLoginUID()
	device, err := u.deviceDB.queryDeviceWithUIDAndDeviceID(deviceID, loginUID)
	if err != nil {
		u.Error("获取设备信息失败！", zap.Error(err))
		c.ResponseError(errors.New("获取设备信息失败！"))
		return
	}
	if device == nil {
		c.ResponseError(errors.New("未查询到该设备"))
		return
	}
	c.Response(&deviceResp{
		ID:          device.Id,
		DeviceID:    device.DeviceID,
		DeviceName:  device.DeviceName,
		DeviceModel: device.DeviceModel,
		LastLogin:   util.ToyyyyMMddHHmm(time.Unix(device.LastLogin, 0)),
	})
}
func (u *User) deviceDelete(c *wkhttp.Context) {
	deviceID := c.Param("device_id")

	err := u.deviceDB.deleteDeviceWithDeviceIDAndUID(deviceID, c.GetLoginUID())
	if err != nil {
		u.Error("删除设备失败！", zap.Error(err))
		c.ResponseError(errors.New("删除设备失败！"))
		return
	}
	c.ResponseOK()
}

// 登录设备列表
func (u *User) deviceList(c *wkhttp.Context) {
	// 获取分页参数
	pageIndex, pageSize := c.GetPage()

	// 如果没有传入分页参数，则查询全部
	var devices []*deviceModel
	var err error

	if pageIndex == 1 && pageSize == 15 { // 默认值，表示没有传入分页参数
		devices, err = u.deviceDB.queryDeviceWithUID(c.GetLoginUID())
	} else {
		// 分页查询
		devices, err = u.deviceDB.queryDeviceWithUIDAndPage(c.GetLoginUID(), pageIndex, pageSize)
	}

	if err != nil {
		u.Error("查询设备列表失败！", zap.Error(err))
		c.ResponseError(errors.New("查询设备列表失败！"))
		return
	}

	var deviceResps = make([]deviceResp, 0, len(devices))
	if len(devices) > 0 {
		for index, device := range devices {
			var selft int
			if index == 0 {
				selft = 1
			}
			deviceName := device.DeviceName
			if selft == 1 && pageIndex == 1 {
				deviceName = fmt.Sprintf("%s（本机）", device.DeviceName)
			}
			deviceResps = append(deviceResps, deviceResp{
				ID:          device.Id,
				DeviceID:    device.DeviceID,
				DeviceName:  deviceName,
				DeviceModel: device.DeviceModel,
				Self:        selft,
				LastLogin:   util.ToyyyyMMddHHmm(time.Unix(device.LastLogin, 0)),
			})
		}
	}
	c.Response(deviceResps)
}

type deviceResp struct {
	ID          int64  `json:"id"`
	DeviceID    string `json:"device_id"`    // 设备ID
	DeviceName  string `json:"device_name"`  // 设备名称
	DeviceModel string `json:"device_model"` // 设备型号
	LastLogin   string `json:"last_login"`   // 设备最后一次登录时间
	Self        int    `json:"self"`         // 是否是本机
}
