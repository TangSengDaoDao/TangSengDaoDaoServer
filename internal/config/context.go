package config

import (
	"time"

	"github.com/RussellLuo/timingwheel"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/server"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/cache"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/pool"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/redis"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/wkevent"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/wkhttp"
	"github.com/bwmarrin/snowflake"
	"github.com/gocraft/dbr/v2"
	"github.com/olivere/elastic"
	"github.com/opentracing/opentracing-go"
)

// Context 配置上下文
type Context struct {
	cfg          *Config
	mySQLSession *dbr.Session
	redisCache   *common.RedisCache
	memoryCache  cache.Cache
	log.Log
	EventPool      pool.Collector
	PushPool       pool.Collector // 离线push
	RobotEventPool pool.Collector // 机器人事件pool
	Event          wkevent.Event
	elasticClient  *elastic.Client
	UserIDGen      *snowflake.Node // 消息ID生成器
	Server         *server.Server
	tracer         *Tracer                  // 调用链追踪
	aysncTask      *AsyncTask               // 异步任务
	timingWheel    *timingwheel.TimingWheel // Time wheel delay task

}

// NewContext NewContext
func NewContext(cfg *Config) *Context {
	userIDGen, err := snowflake.NewNode(int64(cfg.Cluster.NodeID))
	if err != nil {
		panic(err)
	}
	c := &Context{
		cfg:            cfg,
		UserIDGen:      userIDGen,
		Log:            log.NewTLog("Context"),
		EventPool:      pool.StartDispatcher(cfg.EventPoolSize),
		PushPool:       pool.StartDispatcher(cfg.Push.PushPoolSize),
		RobotEventPool: pool.StartDispatcher(cfg.Robot.EventPoolSize),
		aysncTask:      NewAsyncTask(cfg),
		timingWheel:    timingwheel.NewTimingWheel(cfg.TimingWheelTick.Duration, cfg.TimingWheelSize),
	}
	c.tracer, err = NewTracer(cfg)
	if err != nil {
		panic(err)
	}
	opentracing.SetGlobalTracer(c.tracer)
	c.timingWheel.Start()
	return c
}

// GetConfig 获取配置信息
func (c *Context) GetConfig() *Config {
	return c.cfg
}

// NewMySQL 创建mysql数据库实例
func (c *Context) NewMySQL() *dbr.Session {

	if c.mySQLSession == nil {
		c.mySQLSession = db.NewMySQL(c.cfg.DB.MySQLAddr, c.cfg.DB.SQLDir, c.cfg.DB.Migration)
	}

	return c.mySQLSession
}

// AsyncTask 异步任务
func (c *Context) AsyncTask() *AsyncTask {
	return c.aysncTask
}

// Tracer Tracer
func (c *Context) Tracer() *Tracer {
	return c.tracer
}

// DB DB
func (c *Context) DB() *dbr.Session {
	return c.NewMySQL()
}

// NewRedisCache 创建一个redis缓存
func (c *Context) NewRedisCache() *common.RedisCache {
	if c.redisCache == nil {
		c.redisCache = common.NewRedisCache(c.cfg.DB.RedisAddr)
	}
	return c.redisCache
}

// NewMemoryCache 创建一个内存缓存
func (c *Context) NewMemoryCache() cache.Cache {
	if c.memoryCache == nil {
		c.memoryCache = common.NewMemoryCache()
	}
	return c.memoryCache
}

// Cache 缓存
func (c *Context) Cache() cache.Cache {
	return c.NewRedisCache()
}

// 认证中间件
func (c *Context) AuthMiddleware(r *wkhttp.WKHttp) wkhttp.HandlerFunc {

	return r.AuthMiddleware(c.Cache(), c.cfg.Cache.TokenCachePrefix)
}

// GetRedisConn GetRedisConn
func (c *Context) GetRedisConn() *redis.Conn {
	return c.NewRedisCache().GetRedisConn()
}

// EventBegin 开启事件
func (c *Context) EventBegin(data *wkevent.Data, tx *dbr.Tx) (int64, error) {
	return c.Event.Begin(data, tx)
}

// EventCommit 提交事件
func (c *Context) EventCommit(eventID int64) {
	c.Event.Commit(eventID)
}

// Schedule 延迟任务
func (c *Context) Schedule(interval time.Duration, f func()) *timingwheel.Timer {
	return c.timingWheel.ScheduleFunc(&everyScheduler{
		Interval: interval,
	}, f)
}

// OnlineStatus 在线状态
type OnlineStatus struct {
	UID              string // 用户uid
	DeviceFlag       uint8  // 设备标记
	Online           bool   // 是否在线
	SocketID         int64  // 当前设备在wukongim中的在线/离线的socketID
	OnlineCount      int    //在线数量 当前DeviceFlag下的在线设备数量
	TotalOnlineCount int    // 当前用户所有在线设备数量
}

// OnlineStatusListener 在线状态监听
type OnlineStatusListener func(onlineStatusList []OnlineStatus)

var onlinStatusListeners = make([]OnlineStatusListener, 0)

// AddOnlineStatusListener 添加在线状态监听
func (c *Context) AddOnlineStatusListener(listener OnlineStatusListener) {
	onlinStatusListeners = append(onlinStatusListeners, listener)
}

// GetAllOnlineStatusListeners 获取所有在线监听者
func (c *Context) GetAllOnlineStatusListeners() []OnlineStatusListener {
	return onlinStatusListeners
}

// EventCommit 事件提交
type EventCommit func(err error)

// EventListener EventListener
type EventListener func(data []byte, commit EventCommit)

var eventListeners = map[string][]EventListener{}

// AddEventListener  添加事件监听
func (c *Context) AddEventListener(event string, listener EventListener) {
	listeners := eventListeners[event]
	if listeners == nil {
		listeners = make([]EventListener, 0)
	}
	listeners = append(listeners, listener)
	eventListeners[event] = listeners
}

// GetEventListeners 获取某个事件
func (c *Context) GetEventListeners(event string) []EventListener {
	return eventListeners[event]
}

// MessagesListener 消息监听者
type MessagesListener func(messages []*MessageResp)

var messagesListeners = make([]MessagesListener, 0)

// AddMessagesListener 添加消息监听者
func (c *Context) AddMessagesListener(listener MessagesListener) {
	messagesListeners = append(messagesListeners, listener)
}

// NotifyMessagesListeners 通知消息监听者
func (c *Context) NotifyMessagesListeners(messages []*MessageResp) {
	for _, messagesListener := range messagesListeners {
		messagesListener(messages)
	}
}

type everyScheduler struct {
	Interval time.Duration
}

func (s *everyScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}
