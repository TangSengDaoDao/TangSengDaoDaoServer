package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
)

// IService 服务接口
type IService interface {
	// GetApp 获取app
	GetApp(appID string) (*Resp, error)
	// 创建app
	CreateApp(r Req) (*Resp, error)
}

// Service app服务
type Service struct {
	ctx *config.Context
	db  *DB
}

// NewService NewService
func NewService(ctx *config.Context) IService {
	return &Service{
		ctx: ctx,
		db:  newDB(ctx.DB()),
	}
}

// GetApp 获取APP信息
func (s *Service) GetApp(appID string) (*Resp, error) {
	appM, err := s.db.queryWithAppID(appID)
	if err != nil {
		return nil, err
	}
	if appM == nil {
		return nil, fmt.Errorf("app[%s]不存在！", appID)
	}
	return &Resp{
		AppID:   appM.AppID,
		AppName: appM.AppName,
		AppLogo: appM.AppLogo,
		AppKey:  appM.AppKey,
		Status:  Status(appM.Status),
	}, nil
}

// CreateApp 创建APP 幂等
func (s *Service) CreateApp(r Req) (*Resp, error) {
	if err := r.Check(); err != nil {
		return nil, err
	}
	appM, err := s.db.queryWithAppID(r.AppID)
	if err != nil {
		return nil, err
	}

	var appKey string
	var appID string
	if appM == nil {
		appKey = util.GenerUUID()
		appID = r.AppID
		err = s.db.insert(&model{
			AppID:  r.AppID,
			Status: StatusEnable.Int(),
			AppKey: appKey,
		})
		if err != nil {
			return nil, err
		}
	} else {
		appID = appM.AppID
		appKey = appM.AppKey
	}

	return &Resp{
		AppID:  appID,
		AppKey: appKey,
		Status: StatusEnable,
	}, nil

}

type Resp struct {
	AppID   string
	AppKey  string
	AppName string
	AppLogo string
	Status  Status
}

type Req struct {
	AppID string
}

func (r Req) Check() error {
	if len(strings.TrimSpace(r.AppID)) <= 0 {
		return errors.New("appID不能为空！")
	}
	return nil
}
