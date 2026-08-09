package asynqx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	hibikenAsynq "github.com/hibiken/asynq"
)

type ProcessFunc[T Task] func(context.Context, T, *hibikenAsynq.Task) error

type RetryDelayFunc[T Task] func(retryCount int, err error, task T, rawTask *hibikenAsynq.Task) time.Duration

type taskRegistryInfo struct {
	process        hibikenAsynq.HandlerFunc
	retryDelayFunc hibikenAsynq.RetryDelayFunc
}

func RegisterTaskType[T Task](server *Server, sample T, process ProcessFunc[T], retryDelayFunc RetryDelayFunc[T]) {
	if server == nil {
		panic("asynqx: Server不能为空")
	}
	if process == nil {
		panic("asynqx: 任务处理方法不能为空")
	}

	taskType := reflect.TypeOf(sample)
	if taskType == nil || taskType.Kind() != reflect.Pointer {
		panic("asynqx: 注册任务必须传入指针类型")
	}

	newTask := func() T {
		return reflect.New(taskType.Elem()).Interface().(T)
	}
	taskName := newTask().TaskType()
	if taskName == "" {
		panic("asynqx: 任务类型不能为空")
	}

	info := &taskRegistryInfo{
		process: func(ctx context.Context, rawTask *hibikenAsynq.Task) error {
			task := newTask()
			if err := json.Unmarshal(rawTask.Payload(), task); err != nil {
				return errors.Join(fmt.Errorf("asynqx: 解析任务失败 type=%s: %w", taskName, err), hibikenAsynq.SkipRetry)
			}
			return process(ctx, task, rawTask)
		},
	}
	if retryDelayFunc != nil {
		info.retryDelayFunc = func(retryCount int, err error, rawTask *hibikenAsynq.Task) time.Duration {
			task := newTask()
			if decodeErr := json.Unmarshal(rawTask.Payload(), task); decodeErr != nil {
				return hibikenAsynq.DefaultRetryDelayFunc(retryCount, err, rawTask)
			}
			return retryDelayFunc(retryCount, err, task, rawTask)
		}
	}
	server.register(taskName, info)
}
