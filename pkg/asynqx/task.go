package asynqx

import hibikenAsynq "github.com/hibiken/asynq"

type Task interface {
	TaskType() string
	TaskQueue() string
	TaskOptions() []hibikenAsynq.Option
}
