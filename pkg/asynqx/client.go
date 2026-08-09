package asynqx

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	hibikenAsynq "github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

var initOnce sync.Once
var globalClient *hibikenAsynq.Client

func InitClient(rdb redis.UniversalClient) {
	initOnce.Do(func() {
		globalClient = hibikenAsynq.NewClientFromRedisClient(rdb)
	})
}

func AddTaskWithCtx(ctx context.Context, data Task, options ...hibikenAsynq.Option) (*hibikenAsynq.TaskInfo, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	taskOptions := append([]hibikenAsynq.Option(nil), data.TaskOptions()...)
	taskOptions = append(taskOptions, hibikenAsynq.Queue(data.TaskQueue()))
	task := hibikenAsynq.NewTask(data.TaskType(), payload, taskOptions...)
	return globalClient.EnqueueContext(ctx, task, options...)
}

func Enqueue(ctx context.Context, data Task, options ...hibikenAsynq.Option) error {
	_, err := AddTaskWithCtx(ctx, data, options...)
	if errors.Is(err, hibikenAsynq.ErrDuplicateTask) {
		return nil
	}
	return err
}
