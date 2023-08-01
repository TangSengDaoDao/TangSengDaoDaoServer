package webhook

import (
	"errors"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

// 数据源
func (w *Webhook) datasource(c *wkhttp.Context) {
	var cmdReq struct {
		CMD  string                 `json:"cmd"`
		Data map[string]interface{} `json:"data"`
	}
	if err := c.BindJSON(&cmdReq); err != nil {
		w.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(err)
		return
	}
	if strings.TrimSpace(cmdReq.CMD) == "" {
		c.ResponseError(errors.New("cmd不能为空！"))
		return
	}
	w.Debug("请求数据源", zap.Any("cmd", cmdReq))
	var result interface{}
	var err error
	switch cmdReq.CMD {
	case "getChannelInfo":
		result, err = w.getChannelInfo(cmdReq.Data)
	case "getSubscribers":
		result, err = w.getSubscribers(cmdReq.Data)
	case "getBlacklist":
		result, err = w.getBlacklist(cmdReq.Data)
	case "getWhitelist":
		result, err = w.getWhitelist(cmdReq.Data)
	case "getSystemUIDs":
		result, err = w.getSystemUIDs()
	}

	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Response(result)
}

func (w *Webhook) getChannelInfo(data map[string]interface{}) (interface{}, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}

	modules := register.GetModules(w.ctx)

	if len(modules) > 0 {
		for _, m := range modules {
			if m.IMDatasource.HasData == nil {
				continue
			}
			hasData := m.IMDatasource.HasData(channelReq.ChannelID, channelReq.ChannelType)
			if m.IMDatasource.ChannelInfo != nil && hasData.Has(register.IMDatasourceTypeChannelInfo) {
				channelInfoMap, err := m.IMDatasource.ChannelInfo(channelReq.ChannelID, channelReq.ChannelType)
				if err != nil {
					if errors.Is(err, register.ErrDatasourceNotProcess) {
						continue
					}
					return nil, err
				}
				return channelInfoMap, nil
			}
		}
	}
	return map[string]interface{}{}, nil
}

func (w *Webhook) getSubscribers(data map[string]interface{}) ([]string, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}

	if channelReq.ChannelType == common.ChannelTypePerson.Uint8() {
		return make([]string, 0), nil
	}

	modules := register.GetModules(w.ctx)

	if len(modules) > 0 {
		for _, m := range modules {
			if m.IMDatasource.HasData == nil {
				continue
			}
			hasData := m.IMDatasource.HasData(channelReq.ChannelID, channelReq.ChannelType)
			if m.IMDatasource.Subscribers != nil && hasData.Has(register.IMDatasourceTypeSubscribers) {
				subscribers, err := m.IMDatasource.Subscribers(channelReq.ChannelID, channelReq.ChannelType)
				if err != nil {
					return nil, err
				}
				return subscribers, nil
			}
		}
	}
	return make([]string, 0), nil

}

func (w *Webhook) getBlacklist(data map[string]interface{}) ([]string, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}
	if channelReq.ChannelType == uint8(common.ChannelTypePerson) && common.IsFakeChannel(channelReq.ChannelID) {
		uids := strings.Split(channelReq.ChannelID, "@")
		exist, err := w.userService.ExistBlacklist(uids[0], uids[1])
		if err != nil {
			return nil, err
		}
		if exist {
			return uids, nil
		}
	}
	modules := register.GetModules(w.ctx)
	if len(modules) > 0 {
		for _, m := range modules {
			if m.IMDatasource.HasData == nil {
				continue
			}
			hasData := m.IMDatasource.HasData(channelReq.ChannelID, channelReq.ChannelType)
			if m.IMDatasource.Blacklist != nil && hasData.Has(register.IMDatasourceTypeBlacklist) {
				data, err := m.IMDatasource.Blacklist(channelReq.ChannelID, channelReq.ChannelType)
				if err != nil {
					return nil, err
				}
				return data, nil
			}
		}
	}
	return make([]string, 0), nil
}

func (w *Webhook) getWhitelist(data map[string]interface{}) ([]string, error) {
	var channelReq ChannelReq
	if err := util.ReadJsonByByte([]byte(util.ToJson(data)), &channelReq); err != nil {
		return nil, err
	}

	modules := register.GetModules(w.ctx)
	if len(modules) > 0 {
		for _, m := range modules {
			if m.IMDatasource.HasData == nil {
				continue
			}
			hasData := m.IMDatasource.HasData(channelReq.ChannelID, channelReq.ChannelType)
			if m.IMDatasource.Whitelist != nil && hasData.Has(register.IMDatasourceTypeWhitelist) {
				data, err := m.IMDatasource.Whitelist(channelReq.ChannelID, channelReq.ChannelType)
				if err != nil {
					return nil, err
				}
				return data, nil
			}
		}
	}
	return make([]string, 0), nil

}

func (w *Webhook) getSystemUIDs() ([]string, error) {

	modules := register.GetModules(w.ctx)

	uids := make([]string, 0)
	if len(modules) > 0 {
		for _, m := range modules {
			if m.IMDatasource.SystemUIDs != nil {
				systemUIDs, err := m.IMDatasource.SystemUIDs()
				if err != nil {
					return nil, err
				}
				if len(systemUIDs) > 0 {
					uids = append(uids, systemUIDs...)
				}
			}
		}
	}
	return uids, nil
}

type ChannelReq struct {
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}
