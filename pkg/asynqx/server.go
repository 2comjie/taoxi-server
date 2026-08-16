package asynqx

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"sync"
	"time"

	"github.com/2comjie/nova/logx"
	hibikenAsynq "github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

type RedisProvider func() redis.UniversalClient

type Server struct {
	redisProvider RedisProvider
	queues        map[string]int

	mu      sync.Mutex
	started bool
	server  *hibikenAsynq.Server
	tasks   map[string]*taskRegistryInfo
}

func NewServer(redisProvider RedisProvider, queues map[string]int) *Server {
	return &Server{
		redisProvider: redisProvider,
		queues:        maps.Clone(queues),
		tasks:         make(map[string]*taskRegistryInfo),
	}
}

func (s *Server) Name() string {
	return "asyncq-server"
}

func (s *Server) register(taskType string, info *taskRegistryInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		panic("asynqx: Server启动后不能再注册任务")
	}
	if _, exists := s.tasks[taskType]; exists {
		panic("asynqx: 重复注册任务 " + taskType)
	}
	s.tasks[taskType] = info
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return errors.New("asynqx: Server已经启动")
	}
	if s.redisProvider == nil {
		return errors.New("asynqx: RedisProvider不能为空")
	}
	rdb := s.redisProvider()
	if rdb == nil {
		return errors.New("asynqx: Redis客户端不能为空")
	}
	if len(s.queues) == 0 {
		return errors.New("asynqx: 消费队列不能为空")
	}
	for queue, weight := range s.queues {
		if queue == "" || weight <= 0 {
			return fmt.Errorf("asynqx: 无效队列 queue=%q weight=%d", queue, weight)
		}
	}

	mux := hibikenAsynq.NewServeMux()
	for taskType, info := range s.tasks {
		mux.HandleFunc(taskType, info.process)
	}

	concurrency := runtime.NumCPU() * 4
	worker := hibikenAsynq.NewServerFromRedisClient(rdb, hibikenAsynq.Config{
		Concurrency:              concurrency,
		Queues:                   maps.Clone(s.queues),
		StrictPriority:           true,
		RetryDelayFunc:           s.retryDelay,
		ShutdownTimeout:          10 * time.Second,
		DelayedTaskCheckInterval: time.Second,
		Logger:                   asyncqLogger{},
		LogLevel:                 hibikenAsynq.InfoLevel,
		ErrorHandler: hibikenAsynq.ErrorHandlerFunc(func(ctx context.Context, task *hibikenAsynq.Task, taskErr error) {
			retried, _ := hibikenAsynq.GetRetryCount(ctx)
			maxRetry, _ := hibikenAsynq.GetMaxRetry(ctx)
			logx.Errorf(
				"asyncq: 任务执行失败 type=%s retry=%d max_retry=%d err=%v",
				task.Type(),
				retried,
				maxRetry,
				taskErr,
			)
		}),
	})
	if err := worker.Start(mux); err != nil {
		return err
	}

	s.server = worker
	s.started = true
	logx.Infof("asyncq: Server启动 queues=%v concurrency=%d", s.queues, concurrency)
	return nil
}

func (s *Server) retryDelay(retryCount int, err error, task *hibikenAsynq.Task) time.Duration {
	s.mu.Lock()
	info := s.tasks[task.Type()]
	s.mu.Unlock()
	if info == nil || info.retryDelayFunc == nil {
		return hibikenAsynq.DefaultRetryDelayFunc(retryCount, err, task)
	}
	return info.retryDelayFunc(retryCount, err, task)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	worker := s.server
	s.server = nil
	s.mu.Unlock()
	if worker == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		worker.Shutdown()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type asyncqLogger struct{}

func (asyncqLogger) Debug(args ...interface{}) { logx.Debug(args...) }
func (asyncqLogger) Info(args ...interface{})  { logx.Info(args...) }
func (asyncqLogger) Warn(args ...interface{})  { logx.Warn(args...) }
func (asyncqLogger) Error(args ...interface{}) { logx.Error(args...) }
func (asyncqLogger) Fatal(args ...interface{}) { logx.Error(args...) }
