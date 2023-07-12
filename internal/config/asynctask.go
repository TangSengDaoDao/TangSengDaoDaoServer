package config

import (
	"context"

	"github.com/RichardKnop/machinery/v2"
	redisbackend "github.com/RichardKnop/machinery/v2/backends/redis"
	"github.com/RichardKnop/machinery/v2/backends/result"
	redisbroker "github.com/RichardKnop/machinery/v2/brokers/redis"
	"github.com/RichardKnop/machinery/v2/config"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
	"github.com/RichardKnop/machinery/v2/tasks"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"go.uber.org/zap"
)

// AsyncTask 异步任务
type AsyncTask struct {
	server *machinery.Server
	log.Log
}

// NewAsyncTask 创建一个异步任务
func NewAsyncTask(cfg *Config) *AsyncTask {
	cnf := &config.Config{
		DefaultQueue:    "machinery_tasks",
		ResultsExpireIn: 3600,
		Redis: &config.RedisConfig{
			MaxIdle:                3,
			IdleTimeout:            240,
			ReadTimeout:            15,
			WriteTimeout:           15,
			ConnectTimeout:         15,
			NormalTasksPollPeriod:  1000,
			DelayedTasksPollPeriod: 500,
		},
	}
	broker := redisbroker.NewGR(cnf, []string{cfg.DB.AsynctaskRedisAddr}, 0)
	backend := redisbackend.NewGR(cnf, []string{cfg.DB.AsynctaskRedisAddr}, 0)
	lock := eagerlock.New()

	t := &AsyncTask{
		Log: log.NewTLog("AsyncTask"),
	}
	server := machinery.NewServer(cnf, broker, backend, lock)
	t.server = server
	return t
}

// RegisterTask 注册任务
func (a *AsyncTask) RegisterTask(name string, taskFunc interface{}) error {
	return a.server.RegisterTask(name, taskFunc)
}

// RegisterTasks 注册任务
func (a *AsyncTask) RegisterTasks(namedTaskFuncs map[string]interface{}) error {
	return a.server.RegisterTasks(namedTaskFuncs)
}

// SendTask 发送任务
func (a *AsyncTask) SendTask(task *tasks.Signature) (*result.AsyncResult, error) {
	return a.server.SendTask(task)
}

// SendTaskWithContext 发送任务
func (a *AsyncTask) SendTaskWithContext(ctx context.Context, task *tasks.Signature) (*result.AsyncResult, error) {
	return a.server.SendTaskWithContext(ctx, task)
}

// LaunchWorker 启动消费worker  [concurrency]并发数
func (a *AsyncTask) LaunchWorker(consumerTag string, concurrency int) error {
	worker := a.server.NewWorker(consumerTag, concurrency)
	errorhandler := func(err error) {
		a.Debug("I am an error handler", zap.Error(err))
	}

	pretaskhandler := func(signature *tasks.Signature) {
		a.Debug("I am a start of task handler for", zap.String("name", signature.Name))
	}

	posttaskhandler := func(signature *tasks.Signature) {
		a.Debug("I am an end of task handler for", zap.String("name", signature.Name))
	}

	worker.SetPostTaskHandler(posttaskhandler)
	worker.SetErrorHandler(errorhandler)
	worker.SetPreTaskHandler(pretaskhandler)

	return worker.Launch()
}
