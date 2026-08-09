package paymentService

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/2comjie/taoxi-server/app/Api/items"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
	"github.com/2comjie/taoxi-server/internal/config/shared"
	"github.com/2comjie/taoxi-server/pkg/asynqx"
	"github.com/2comjie/wali/logx"
	hibikenAsynq "github.com/hibiken/asynq"
	"time"
)

func RegisterTasks(server *asynqx.Server) {
	asynqx.RegisterTaskType[*paymentTypes.GrantTask](server, &paymentTypes.GrantTask{}, processGrantTask, grantTaskRetryDelay)
}

func processGrantTask(ctx context.Context, task *paymentTypes.GrantTask, _ *hibikenAsynq.Task) error {
	logCtx := logx.WithField("action", "支付发货任务").WithField("order_id", task.OrderId).WithField("uid", task.Uid)
	rewards := make([]*shared.Reward, 0)
	if len(task.Rewards) > 0 {
		err := json.Unmarshal(task.Rewards, &rewards)
		if err != nil {
			logCtx.Errorf("解析奖励错误 %v %s", err, task.Rewards)
			return err
		}
	}

	nonce := fmt.Sprintf("订单发货-%d", task.OrderId)
	stdErr := items.AddItems(ctx, task.Uid, nonce, rewards)
	if stdErr != nil {
		logCtx.Errorf("添加道具失败 %v %s", stdErr, task.Rewards)
		return errors.New(stdErr.Msg)
	}
	logCtx.Infof("道具发放成功 %d %s", task.Uid, task.Rewards)
	return nil
}

func grantTaskRetryDelay(retryCount int, err error, _ *paymentTypes.GrantTask, rawTask *hibikenAsynq.Task) time.Duration {
	delay := hibikenAsynq.DefaultRetryDelayFunc(retryCount, err, rawTask)
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}
