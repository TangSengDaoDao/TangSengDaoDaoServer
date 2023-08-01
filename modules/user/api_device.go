package user

import (
	"fmt"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

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

	devices, err := u.deviceDB.queryDeviceWithUID(c.GetLoginUID())
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
			if selft == 1 {
				deviceName = fmt.Sprintf("%s（本机）", device.DeviceName)
			}
			deviceResps = append(deviceResps, deviceResp{
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
	DeviceID    string `json:"device_id"`    // 设备ID
	DeviceName  string `json:"device_name"`  // 设备名称
	DeviceModel string `json:"device_model"` // 设备型号
	LastLogin   string `json:"last_login"`   // 设备最后一次登录时间
	Self        int    `json:"self"`         // 是否是本机
}
