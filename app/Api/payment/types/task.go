package paymentTypes

import (
	"encoding/json"
	"fmt"
	"time"

	hibikenAsynq "github.com/hibiken/asynq"
)

type GrantTask struct {
	OrderId uint64          `json:"order_id"`
	Uid     uint64          `json:"uid"`
	Rewards json.RawMessage `json:"rewards"`
}

func (g *GrantTask) TaskType() string {
	return "payment:grant_rewards"
}

func (g *GrantTask) TaskQueue() string {
	return "default"
}

func (g *GrantTask) TaskOptions() []hibikenAsynq.Option {
	return []hibikenAsynq.Option{
		hibikenAsynq.TaskID(fmt.Sprintf("payment-grant-rewards-%d", g.OrderId)),
		hibikenAsynq.MaxRetry(1_000_000_000),
		hibikenAsynq.Timeout(30 * time.Second),
	}
}
