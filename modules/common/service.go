package common

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"go.uber.org/zap"
)

var onceSerce sync.Once

// IService IService
type IService interface {
	GetAppConfig() (*AppConfigResp, error)
	// 获取短编号
	GetShortno() (string, error)
	SetShortnoUsed(shortno string, business string) error
}

// NewService NewService
func NewService(ctx *config.Context) IService {
	return newService(ctx)
}

type service struct {
	ctx         *config.Context
	appConfigDB *appConfigDB
	shortnoDB   *shortnoDB
	shortnoLock sync.RWMutex
}

func newService(ctx *config.Context) *service {
	if ctx.GetConfig().ShortNo.NumOn {
		onceSerce.Do(func() {
			go runGenShortnoTask(ctx)
		})
	}

	return &service{
		ctx:         ctx,
		appConfigDB: newAppConfigDB(ctx),
		shortnoDB:   newShortnoDB(ctx),
	}
}

// GetAppConfig GetAppConfig
func (s *service) GetAppConfig() (*AppConfigResp, error) {
	appConfigM, err := s.appConfigDB.query()
	if err != nil {
		return nil, err
	}

	return &AppConfigResp{
		RSAPublicKey:           appConfigM.RSAPublicKey,
		Version:                appConfigM.Version,
		SuperToken:             appConfigM.SuperToken,
		SuperTokenOn:           appConfigM.SuperTokenOn,
		WelcomeMessage:         appConfigM.WelcomeMessage,
		NewUserJoinSystemGroup: appConfigM.NewUserJoinSystemGroup,
		SearchByPhone:          appConfigM.SearchByPhone,
	}, nil
}

func (s *service) GetShortno() (string, error) {

	s.shortnoLock.Lock() // 这里需要加锁 要不然多线程下会出现shortNo重复的问题
	defer s.shortnoLock.Unlock()

	shortnoM, err := s.shortnoDB.queryVail()
	if err != nil {
		return "", err
	}
	if shortnoM == nil {
		return "", errors.New("没有短编号可分配")
	}
	err = s.shortnoDB.updateLock(shortnoM.Shortno, 1)
	if err != nil {
		return "", err
	}
	return shortnoM.Shortno, nil
}

func (s *service) SetShortnoUsed(shortno string, business string) error {
	return s.shortnoDB.updateUsed(shortno, 1, business)
}

// 开启生成短编号任务
func runGenShortnoTask(ctx *config.Context) {
	shortnoDB := newShortnoDB(ctx)
	errorSleep := time.Second * 2
	for {
		count, err := shortnoDB.queryVailCount()
		if err != nil {
			time.Sleep(errorSleep)
			continue
		}
		if count < 10000 {
			shortnos := generateNums(ctx.GetConfig().ShortNo.NumLen, 100)
			if len(shortnos) > 0 {
				err = shortnoDB.inserts(shortnos)
				if err != nil {
					ctx.Error("添加短编号失败！", zap.Error(err))
				}
			}
		}
		time.Sleep(time.Second * 30)
	}
}

func generateNums(len int, count int) []string {
	var nums = make([]string, 0, count)
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := count; i > 0; i-- {
		var num = rd.Int63n(1e16)
		nums = append(nums, fmt.Sprintf("%016d", num)[0:len])
	}
	return nums

}

type AppConfigResp struct {
	RSAPublicKey           string
	Version                int
	SuperToken             string
	SuperTokenOn           int
	WelcomeMessage         string // 登录欢迎语
	NewUserJoinSystemGroup int    // 新用户是否加入系统群聊
	SearchByPhone          int    // 是否可通过手机号搜索
}
